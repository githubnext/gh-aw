// @ts-check

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

const noopIssueTemplateBody = `This issue tracks no-op runs from agentic workflows in this repository. A maintained rollup below summarizes workflow/root-cause buckets with current counts, and new runs update that rollup in place instead of adding an unbounded stream of comments.

<details>
<summary>📘 What is a No-Op?</summary>

A no-op (no operation) occurs when an agentic workflow runs successfully but determines that no action is required. For example:
- A security scanner that finds no issues
- An update checker that finds nothing to update
- A monitoring workflow that finds everything is healthy

These are successful outcomes, not failures, and help provide transparency into workflow behavior.

</details>

<details>
<summary>🎯 How This Helps</summary>

This issue helps you:
- Track workflows that ran but determined no action was needed
- Distinguish between failures and intentional no-ops
- Monitor workflow health by seeing when workflows decide not to act

</details>

<details>
<summary>📚 Resources</summary>

- [GitHub Agentic Workflows Documentation](https://github.com/github/gh-aw)

</details>

> [!TIP]
> To stop a workflow from posting here, set \`report-as-issue: false\` in its frontmatter:
> \`\`\`yaml
> safe-outputs:
>   noop:
>     report-as-issue: false
> \`\`\`

---

> This issue is automatically managed by GitHub Agentic Workflows. Do not close this issue manually.
> 
> **No action to take** - Do not assign to an agent.

<!-- gh-aw-noop-runs -->`;

describe("handle_noop_message", () => {
  let mockCore;
  let mockGithub;
  let mockContext;
  let originalEnv;
  let tempDir;
  let originalReadFileSync;

  beforeEach(async () => {
    // Save original environment
    originalEnv = { ...process.env };

    // Create temp directory for test files
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "handle-noop-test-"));

    // getPromptPath() throws unless GH_AW_PROMPTS_DIR or RUNNER_TEMP is set.
    // On CI RUNNER_TEMP is ambient, but it is not set in local dev, so set the
    // prompts dir explicitly to make the suite environment-independent. The
    // actual path is irrelevant because fs.readFileSync is mocked below.
    process.env.GH_AW_PROMPTS_DIR = tempDir;

    // Mock fs.readFileSync to return template content
    originalReadFileSync = fs.readFileSync;
    fs.readFileSync = vi.fn((filePath, encoding) => {
      if (filePath.includes("noop_runs_issue.md")) {
        return noopIssueTemplateBody;
      }
      if (filePath.includes("noop_comment.md")) {
        return `### {workflow_name}

{message}

> Generated from [{workflow_name}]({run_url}){aic_suffix}{ambient_context_suffix}{history_link}`;
      }
      return originalReadFileSync.call(fs, filePath, encoding);
    });

    // Mock core
    const mockSummary = {
      addRaw: vi.fn(),
      write: vi.fn().mockResolvedValue(undefined),
    };
    mockSummary.addRaw.mockReturnValue(mockSummary);
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setOutput: vi.fn(),
      summary: mockSummary,
    };

    // Mock GitHub API
    mockGithub = {
      rest: {
        search: {
          issuesAndPullRequests: vi.fn(),
        },
        issues: {
          create: vi.fn(),
          get: vi.fn().mockResolvedValue({
            data: {
              body: noopIssueTemplateBody,
            },
          }),
          update: vi.fn().mockResolvedValue({ data: {} }),
          listComments: vi.fn().mockResolvedValue({ data: [] }),
          createComment: vi.fn(),
        },
      },
    };

    // Mock context
    mockContext = {
      repo: {
        owner: "test-owner",
        repo: "test-repo",
      },
    };

    // Setup globals
    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
  });

  afterEach(() => {
    // Restore environment by mutating process.env in place
    for (const key of Object.keys(process.env)) {
      if (!(key in originalEnv)) {
        delete process.env[key];
      }
    }
    Object.assign(process.env, originalEnv);

    // Restore fs.readFileSync
    if (originalReadFileSync) {
      fs.readFileSync = originalReadFileSync;
    }

    // Clean up temp directory
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }

    vi.clearAllMocks();
  });

  it("should skip if no noop items in agent output", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/123";
    process.env.GH_AW_AGENT_CONCLUSION = "success";

    // Create agent output file with no noop items
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [{ type: "create_issue", title: "Some issue" }],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No noop items found in agent output"));
    expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
  });

  it("should skip if report-as-issue is set to false", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/123";
    process.env.GH_AW_NOOP_MESSAGE = "Some message";
    process.env.GH_AW_AGENT_CONCLUSION = "success";
    process.env.GH_AW_NOOP_REPORT_AS_ISSUE = "false";

    // Create agent output file with only noop outputs
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [{ type: "noop", message: "Some message" }],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("report-as-issue is disabled"));
    expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
  });

  it("should proceed if report-as-issue is set to true", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/123";
    process.env.GH_AW_NOOP_MESSAGE = "Some message";
    process.env.GH_AW_AGENT_CONCLUSION = "success";
    process.env.GH_AW_NOOP_REPORT_AS_ISSUE = "true";

    // Create agent output file with only noop outputs
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [{ type: "noop", message: "Some message" }],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    // Mock search to return existing issue
    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: {
        total_count: 1,
        items: [
          {
            number: 42,
            node_id: "MDU6SXNzdWU0Mg==",
            html_url: "https://github.com/test-owner/test-repo/issues/42",
          },
        ],
      },
    });

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Report as issue: true"));
    expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalled();
    expect(mockGithub.rest.issues.get).toHaveBeenCalled();
    expect(mockGithub.rest.issues.update).toHaveBeenCalled();
  });

  it("should default to true if report-as-issue is not set", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/123";
    process.env.GH_AW_NOOP_MESSAGE = "Some message";
    process.env.GH_AW_AGENT_CONCLUSION = "success";
    // Don't set GH_AW_NOOP_REPORT_AS_ISSUE at all

    // Create agent output file with only noop outputs
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [{ type: "noop", message: "Some message" }],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    // Mock search to return existing issue
    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: {
        total_count: 1,
        items: [
          {
            number: 42,
            node_id: "MDU6SXNzdWU0Mg==",
            html_url: "https://github.com/test-owner/test-repo/issues/42",
          },
        ],
      },
    });

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Report as issue: true"));
    expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalled();
    expect(mockGithub.rest.issues.update).toHaveBeenCalled();
  });

  it("should skip if agent conclusion is cancelled (not success or failure)", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/123";
    process.env.GH_AW_AGENT_CONCLUSION = "cancelled";

    // Create agent output file with noop items so we reach the conclusion check
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [{ type: "noop", message: "Some message" }],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Agent did not succeed"));
    expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
  });

  it("should skip if agent failed with no agent output file", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/123";
    process.env.GH_AW_NOOP_MESSAGE = "Some message";
    process.env.GH_AW_AGENT_CONCLUSION = "failure";

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Could not load agent output, skipping"));
    expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
  });

  it("should post noop message when agent failed with only noop outputs (transient error)", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/123";
    process.env.GH_AW_NOOP_MESSAGE = "No issues found";
    process.env.GH_AW_AGENT_CONCLUSION = "failure";

    // Create agent output file with only noop outputs
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [{ type: "noop", message: "No issues found" }],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 1, items: [{ number: 1, node_id: "ID", html_url: "url" }] },
    });

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Agent failed but produced only noop outputs (transient AI model error"));
    expect(mockGithub.rest.issues.update).toHaveBeenCalled();
  });

  it("should skip if agent failed with non-noop outputs", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/123";
    process.env.GH_AW_NOOP_MESSAGE = "Some message";
    process.env.GH_AW_AGENT_CONCLUSION = "failure";

    // Create agent output file with noop + non-noop outputs
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [
          { type: "noop", message: "No action needed" },
          { type: "create_issue", title: "Some issue" },
        ],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Found 1 non-noop output(s)"));
    expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
  });

  it("should skip if there are non-noop outputs", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/123";
    process.env.GH_AW_NOOP_MESSAGE = "Some message";
    process.env.GH_AW_AGENT_CONCLUSION = "success";

    // Create agent output file with noop + other outputs
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [
          { type: "noop", message: "No action needed" },
          { type: "create_issue", title: "Some issue" },
        ],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Found 1 non-noop output(s)"));
    expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
  });

  it("should create no-op runs issue if it doesn't exist", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/123456";
    process.env.GH_AW_NOOP_MESSAGE = "No updates needed";
    process.env.GH_AW_AGENT_CONCLUSION = "success";

    // Create agent output file with only noop outputs
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [{ type: "noop", message: "No updates needed" }],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    // Mock search to return no results
    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: {
        total_count: 0,
        items: [],
      },
    });

    // Mock issue creation
    mockGithub.rest.issues.create.mockResolvedValue({
      data: {
        number: 42,
        node_id: "MDU6SXNzdWU0Mg==",
        html_url: "https://github.com/test-owner/test-repo/issues/42",
      },
    });

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    // Verify search was performed
    expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalledWith({
      q: expect.stringContaining("[aw] No-Op Runs"),
      per_page: 1,
    });

    // Verify issue was created with correct title
    const createCall = mockGithub.rest.issues.create.mock.calls[0][0];
    expect(createCall.title).toBe("[aw] No-Op Runs");
    expect(createCall.labels).toContain("agentic-workflows");
    expect(createCall.body).toContain("maintained rollup");

    const updateCall = mockGithub.rest.issues.update.mock.calls[0][0];
    expect(updateCall.issue_number).toBe(42);
    expect(updateCall.body).toContain("## No-Op Rollup");
    expect(updateCall.body).toContain("Test Workflow");
    expect(updateCall.body).toContain("No updates needed");
    expect(updateCall.body).toContain("https://github.com/test-owner/test-repo/actions/runs/123456");
  });

  it("should use existing no-op runs issue if it exists", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Another Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/789";
    process.env.GH_AW_NOOP_MESSAGE = "Everything is up to date";
    process.env.GH_AW_AGENT_CONCLUSION = "success";

    // Create agent output file with only noop outputs
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [{ type: "noop", message: "Everything is up to date" }],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    // Mock search to return existing issue
    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: {
        total_count: 1,
        items: [
          {
            number: 99,
            node_id: "MDU6SXNzdWU5OQ==",
            html_url: "https://github.com/test-owner/test-repo/issues/99",
          },
        ],
      },
    });

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    // Verify issue was not created
    expect(mockGithub.rest.issues.create).not.toHaveBeenCalled();

    const updateCall = mockGithub.rest.issues.update.mock.calls[0][0];
    expect(updateCall.issue_number).toBe(99);
    expect(updateCall.body).toContain("Another Workflow");
    expect(updateCall.body).toContain("Everything is up to date");
  });

  it("should seed the rollup from legacy per-run comments when no rollup exists yet", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Smoke CI";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/999";
    process.env.GH_AW_AGENT_CONCLUSION = "success";

    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "noop", message: "push event - no PR context" }] }));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 1, items: [{ number: 77, node_id: "ID", html_url: "url" }] },
    });
    mockGithub.rest.issues.listComments.mockResolvedValue({
      data: [
        {
          body: "### Smoke CI\n\npush event - no PR context\n\n> Generated from [Smoke CI](https://github.com/test-owner/test-repo/actions/runs/100)",
          created_at: "2026-07-01T00:00:00Z",
          updated_at: "2026-07-01T00:00:00Z",
        },
        {
          body: "### Smoke CI\n\npush event - no PR context\n\n> Generated from [Smoke CI](https://github.com/test-owner/test-repo/actions/runs/101)",
          created_at: "2026-07-02T00:00:00Z",
          updated_at: "2026-07-02T00:00:00Z",
        },
        {
          body: "### Auto-Triage Issues\n\nalready labeled\n\n> Generated from [Auto-Triage Issues](https://github.com/test-owner/test-repo/actions/runs/102)",
          created_at: "2026-07-03T00:00:00Z",
          updated_at: "2026-07-03T00:00:00Z",
        },
      ],
    });

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    const updateCall = mockGithub.rest.issues.update.mock.calls[0][0];
    expect(mockGithub.rest.issues.listComments).toHaveBeenCalledTimes(1);
    expect(updateCall.issue_number).toBe(77);
    expect(updateCall.body).toContain("Tracked no-op runs: **4**");
    expect(updateCall.body).toContain("| Smoke CI | 3 |");
    expect(updateCall.body).toContain("| Auto-Triage Issues | 1 |");
    expect(updateCall.body).toContain("| Smoke CI | push event - no PR context | 3 |");
  });

  it("should skip legacy comment seeding when rollup state already exists", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Smoke CI";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/1000";
    process.env.GH_AW_AGENT_CONCLUSION = "success";

    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "noop", message: "push event - no PR context" }] }));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    const { buildNoopRollupSection, upsertNoopRollupIntoIssueBody } = await import("./handle_noop_message.cjs?t=" + Date.now());
    const existingRollupBody = upsertNoopRollupIntoIssueBody(
      noopIssueTemplateBody,
      buildNoopRollupSection({
        schemaVersion: 1,
        totalRuns: 2,
        updatedAt: "2026-07-03T00:00:00Z",
        latestRunUrl: "https://github.com/test-owner/test-repo/actions/runs/998",
        buckets: [
          {
            workflowName: "Smoke CI",
            message: "push event - no PR context",
            count: 2,
            lastSeenAt: "2026-07-03T00:00:00Z",
            lastRunUrl: "https://github.com/test-owner/test-repo/actions/runs/998",
          },
        ],
      })
    );

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 1, items: [{ number: 78, node_id: "ID", html_url: "url" }] },
    });
    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        body: existingRollupBody,
      },
    });

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    expect(mockGithub.rest.issues.listComments).not.toHaveBeenCalled();
    const updateCall = mockGithub.rest.issues.update.mock.calls[0][0];
    expect(updateCall.issue_number).toBe(78);
    expect(updateCall.body).toContain("Tracked no-op runs: **3**");
  });

  it("should handle rollup update failure gracefully", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/456";
    process.env.GH_AW_NOOP_MESSAGE = "No action required";
    process.env.GH_AW_AGENT_CONCLUSION = "success";

    // Create agent output file with only noop outputs
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [{ type: "noop", message: "No action required" }],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    // Mock existing issue
    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: {
        total_count: 1,
        items: [{ number: 10, node_id: "MDU6SXNzdWUxMA==", html_url: "https://github.com/test-owner/test-repo/issues/10" }],
      },
    });

    mockGithub.rest.issues.update.mockRejectedValue(new Error("API rate limit exceeded"));

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    // Verify warning was logged but workflow didn't fail
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to update no-op rollup issue"));
  });

  it("should handle issue creation failure gracefully", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/789";
    process.env.GH_AW_NOOP_MESSAGE = "All checks passed";
    process.env.GH_AW_AGENT_CONCLUSION = "success";

    // Create agent output file with only noop outputs
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [{ type: "noop", message: "All checks passed" }],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    // Mock no existing issue
    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 0, items: [] },
    });

    // Mock issue creation failure
    mockGithub.rest.issues.create.mockRejectedValue(new Error("Insufficient permissions"));

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    // Verify warning was logged but workflow didn't fail
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not create no-op runs issue"));
  });

  it("should not create issue when search throws (prevents duplicate issues)", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/789";
    process.env.GH_AW_NOOP_MESSAGE = "All checks passed";
    process.env.GH_AW_AGENT_CONCLUSION = "success";

    // Create agent output file with only noop outputs
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [{ type: "noop", message: "All checks passed" }],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    // Mock search failure (e.g. transient API error)
    mockGithub.rest.search.issuesAndPullRequests.mockRejectedValue(new Error("API rate limit exceeded"));

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    // Search error should be caught and logged as a warning (via ensureAgentRunsIssue throw → main catch)
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not create no-op runs issue"));
    // Issue must NOT be created to prevent duplicates
    expect(mockGithub.rest.issues.create).not.toHaveBeenCalled();
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
  });

  it("should extract run ID from URL correctly", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Test";
    process.env.GH_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/987654321";
    process.env.GH_AW_NOOP_MESSAGE = "Done";
    process.env.GH_AW_AGENT_CONCLUSION = "success";

    // Create agent output file with only noop outputs
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [{ type: "noop", message: "Done" }],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 1, items: [{ number: 1, node_id: "ID", html_url: "url" }] },
    });

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    const updateCall = mockGithub.rest.issues.update.mock.calls[0][0];
    expect(updateCall.body).toContain("987654321");
  });

  it("should sanitize workflow name in comment", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Test <script>alert('xss')</script> Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test/test/actions/runs/123";
    process.env.GH_AW_NOOP_MESSAGE = "Clean";
    process.env.GH_AW_AGENT_CONCLUSION = "success";

    // Create agent output file with only noop outputs
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [{ type: "noop", message: "Clean" }],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 1, items: [{ number: 1, node_id: "ID", html_url: "url" }] },
    });

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    const commentCall = mockGithub.rest.issues.update.mock.calls[0][0];
    // Verify XSS attempt was sanitized (specific behavior depends on sanitizeContent implementation)
    expect(commentCall.body).not.toContain("<script>");
  });

  it("should not include effective token count in footer when GH_AW_EFFECTIVE_TOKENS is set", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Token Test Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test/test/actions/runs/123";
    process.env.GH_AW_AGENT_CONCLUSION = "success";
    process.env.GH_AW_EFFECTIVE_TOKENS = "12500";

    // Create agent output file with only noop outputs
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [{ type: "noop", message: "No action needed" }],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 1, items: [{ number: 1, node_id: "ID", html_url: "url" }] },
    });

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    const commentCall = mockGithub.rest.issues.update.mock.calls[0][0];
    expect(commentCall.body).not.toContain("12.5K");
  });

  it("should not include effective token count in footer when GH_AW_EFFECTIVE_TOKENS is not set", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "No Token Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test/test/actions/runs/456";
    process.env.GH_AW_AGENT_CONCLUSION = "success";
    delete process.env.GH_AW_EFFECTIVE_TOKENS;

    // Create agent output file with only noop outputs
    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({
        items: [{ type: "noop", message: "Nothing to do" }],
      })
    );
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 1, items: [{ number: 1, node_id: "ID", html_url: "url" }] },
    });

    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    const commentCall = mockGithub.rest.issues.update.mock.calls[0][0];
    expect(commentCall.body).not.toContain("●");
    expect(commentCall.body).toContain("> Generated from [No Token Workflow](https://github.com/test/test/actions/runs/456)");
  });

  it("should include AIC suffix in comment footer when GH_AW_AIC is set", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "AIC Test Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test/test/actions/runs/123";
    process.env.GH_AW_AGENT_CONCLUSION = "success";
    process.env.GH_AW_AIC = "0.025";

    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "noop", message: "No action needed" }] }));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 1, items: [{ number: 1, node_id: "ID", html_url: "url" }] },
    });
    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    const commentCall = mockGithub.rest.issues.update.mock.calls[0][0];
    expect(commentCall.body).toContain("AIC");
  });

  it("should include detection AIC in the footer total when GH_AW_THREAT_DETECTION_AIC is set", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Detection AIC Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test/test/actions/runs/123";
    process.env.GH_AW_AGENT_CONCLUSION = "success";
    process.env.GH_AW_AIC = "0.100";
    process.env.GH_AW_THREAT_DETECTION_AIC = "0.025";

    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "noop", message: "No action needed" }] }));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 1, items: [{ number: 1, node_id: "ID", html_url: "url" }] },
    });
    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    const commentCall = mockGithub.rest.issues.update.mock.calls[0][0];
    expect(commentCall.body).toContain("0.125 AIC");
  });

  it("should not include AIC suffix in comment footer when GH_AW_AIC is not set", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "No Credits Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test/test/actions/runs/123";
    process.env.GH_AW_AGENT_CONCLUSION = "success";
    delete process.env.GH_AW_AIC;
    delete process.env.GH_AW_THREAT_DETECTION_AIC;

    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "noop", message: "No action needed" }] }));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 1, items: [{ number: 1, node_id: "ID", html_url: "url" }] },
    });
    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    const commentCall = mockGithub.rest.issues.update.mock.calls[0][0];
    expect(commentCall.body).not.toContain(" AIC");
  });

  it("should include ambient context suffix in comment footer when GH_AW_AMBIENT_CONTEXT is set", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "Ambient Context Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test/test/actions/runs/123";
    process.env.GH_AW_AGENT_CONCLUSION = "success";
    process.env.GH_AW_AMBIENT_CONTEXT = "1200";

    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "noop", message: "No action needed" }] }));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 1, items: [{ number: 1, node_id: "ID", html_url: "url" }] },
    });
    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    const commentCall = mockGithub.rest.issues.update.mock.calls[0][0];
    expect(commentCall.body).toContain("⊞");
    expect(commentCall.body).toContain("1.2K");
  });

  it("should include history link in comment footer when GH_AW_WORKFLOW_ID is set", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "History Link Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test/test/actions/runs/123";
    process.env.GH_AW_AGENT_CONCLUSION = "success";
    process.env.GH_AW_WORKFLOW_ID = "history-link-workflow";

    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "noop", message: "No action needed" }] }));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 1, items: [{ number: 1, node_id: "ID", html_url: "url" }] },
    });
    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    const commentCall = mockGithub.rest.issues.update.mock.calls[0][0];
    expect(commentCall.body).toContain("◷");
    expect(commentCall.body).toContain("history-link-workflow");
  });

  it("should not include history link when GH_AW_WORKFLOW_ID is not set", async () => {
    process.env.GH_AW_WORKFLOW_NAME = "No History Workflow";
    process.env.GH_AW_RUN_URL = "https://github.com/test/test/actions/runs/123";
    process.env.GH_AW_AGENT_CONCLUSION = "success";
    delete process.env.GH_AW_WORKFLOW_ID;

    const outputFile = path.join(tempDir, "agent_output.json");
    fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "noop", message: "No action needed" }] }));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 1, items: [{ number: 1, node_id: "ID", html_url: "url" }] },
    });
    const { main } = await import("./handle_noop_message.cjs?t=" + Date.now());
    await main();

    const commentCall = mockGithub.rest.issues.update.mock.calls[0][0];
    expect(commentCall.body).not.toContain("◷");
  });
});
