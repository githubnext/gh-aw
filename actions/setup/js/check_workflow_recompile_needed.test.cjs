// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

describe("check_workflow_recompile_needed", () => {
  let mockCore;
  let mockGithub;
  let mockContext;
  let mockExec;
  let originalGlobals;
  let originalEnv;
  const testPromptsDir = path.join(os.tmpdir(), "gh-aw-test", "prompts");
  const templatePath = path.join(testPromptsDir, "workflow_recompile_issue.md");

  beforeEach(() => {
    // Save original environment
    originalEnv = process.env.GH_AW_PROMPTS_DIR;

    // Set test prompts directory
    process.env.GH_AW_PROMPTS_DIR = testPromptsDir;

    // Create the template file for testing
    const templateDir = path.dirname(templatePath);
    if (!fs.existsSync(templateDir)) {
      fs.mkdirSync(templateDir, { recursive: true });
    }

    const templateContent = `## Problem

The workflow lock files (\`.lock.yml\`) are out of sync with their source markdown files (\`.md\`). This means the workflows that run in GitHub Actions are not using the latest configuration.

## What needs to be done

The workflows need to be recompiled to regenerate the lock files from the markdown sources.

## Instructions

Recompile all workflows using one of the following methods:

### Using gh aw CLI

\`\`\`bash
gh aw compile --validate --verbose
\`\`\`

### Using gh-aw MCP Server

If you have the gh-aw MCP server configured, use the \`compile\` tool:

\`\`\`json
{
  "tool": "compile",
  "arguments": {
    "validate": true,
    "verbose": true
  }
}
\`\`\`

This will:
1. Build the latest version of \`gh-aw\`
2. Compile all workflow markdown files to YAML lock files
3. Ensure all workflows are up to date

After recompiling, commit the changes with a message like:
\`\`\`
Recompile workflows to update lock files
\`\`\`

## Detected Changes

The following workflow lock files have changes:

<details>
<summary>View diff</summary>

\`\`\`diff
{DIFF_CONTENT}
\`\`\`

</details>

## References

- **Repository:** {REPOSITORY}
`;

    fs.writeFileSync(templatePath, templateContent, "utf8");

    // Save original globals
    originalGlobals = {
      core: global.core,
      github: global.github,
      context: global.context,
      exec: global.exec,
    };

    // Setup mock core module
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      summary: {
        addHeading: vi.fn().mockReturnThis(),
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };

    // Setup mock github module
    mockGithub = {
      rest: {
        search: {
          issuesAndPullRequests: vi.fn(),
        },
        issues: {
          create: vi.fn(),
          createComment: vi.fn(),
        },
      },
    };

    // Setup mock context
    mockContext = {
      repo: {
        owner: "testowner",
        repo: "testrepo",
      },
      runId: 123456,
      payload: {
        repository: {
          html_url: "https://github.com/testowner/testrepo",
        },
      },
    };

    // Setup mock exec module
    mockExec = {
      exec: vi.fn(),
    };

    // Set globals for the module
    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
    global.exec = mockExec;
  });

  afterEach(() => {
    // Restore environment variable
    if (originalEnv !== undefined) {
      process.env.GH_AW_PROMPTS_DIR = originalEnv;
    } else {
      delete process.env.GH_AW_PROMPTS_DIR;
    }

    // Clean up the test directory
    const testDir = path.join(os.tmpdir(), "gh-aw-test");
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }

    // Restore original globals
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
    global.exec = originalGlobals.exec;

    // Clear all mocks
    vi.clearAllMocks();
  });

  it("should report no changes when workflows are up to date", async () => {
    // Mock exec to return no changes (empty diff output)
    mockExec.exec.mockResolvedValue(0);

    const { main } = await import("./check_workflow_recompile_needed.cjs");
    await main();

    expect(mockCore.info).toHaveBeenCalledWith("✓ All workflow lock files are up to date");
    expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
  });

  it("should add comment to existing issue when workflows are out of sync", async () => {
    // Mock exec to return changes (non-empty diff output)
    mockExec.exec
      .mockImplementationOnce(async (cmd, args, options) => {
        if (options?.listeners?.stdout) {
          options.listeners.stdout(Buffer.from("diff content"));
        }
        return 1; // Non-zero exit code indicates changes
      })
      .mockImplementationOnce(async (cmd, args, options) => {
        if (options?.listeners?.stdout) {
          options.listeners.stdout(Buffer.from("detailed diff content"));
        }
        return 0;
      });

    // Mock search to return existing issue
    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: {
        total_count: 1,
        items: [
          {
            number: 42,
            html_url: "https://github.com/testowner/testrepo/issues/42",
          },
        ],
      },
    });

    mockGithub.rest.issues.createComment.mockResolvedValue({});

    const { main } = await import("./check_workflow_recompile_needed.cjs");
    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Found existing issue"));
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 42,
      body: expect.stringContaining("Workflows are still out of sync"),
    });
    expect(mockGithub.rest.issues.create).not.toHaveBeenCalled();
  });

  it("should create new issue when workflows are out of sync and no issue exists", async () => {
    // Mock exec to return changes (non-empty diff output)
    mockExec.exec
      .mockImplementationOnce(async (cmd, args, options) => {
        if (options?.listeners?.stdout) {
          options.listeners.stdout(Buffer.from("diff content"));
        }
        return 1;
      })
      .mockImplementationOnce(async (cmd, args, options) => {
        if (options?.listeners?.stdout) {
          options.listeners.stdout(Buffer.from("detailed diff content"));
        }
        return 0;
      });

    // Mock search to return no existing issue
    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: {
        total_count: 0,
        items: [],
      },
    });

    mockGithub.rest.issues.create.mockResolvedValue({
      data: {
        number: 43,
        html_url: "https://github.com/testowner/testrepo/issues/43",
      },
    });

    const { main } = await import("./check_workflow_recompile_needed.cjs");
    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No existing issue found"));
    expect(mockGithub.rest.issues.create).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      title: "[agentics] agentic workflows out of sync",
      body: expect.stringContaining("Using gh aw CLI"),
      labels: ["maintenance", "workflows"],
    });
  });

  it("should handle errors gracefully", async () => {
    // Mock exec to throw error
    mockExec.exec.mockRejectedValue(new Error("Git command failed"));

    const { main } = await import("./check_workflow_recompile_needed.cjs");

    await expect(main()).rejects.toThrow("Git command failed");
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to check for workflow changes"));
  });
});
