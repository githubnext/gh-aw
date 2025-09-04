import { describe, it, expect, beforeEach, vi } from "vitest";
import { readFileSync } from "fs";
import path from "path";

// Create standalone test functions by extracting parts of the script
const createTestableFunction = scriptContent => {
  // Extract just the main function content and wrap it properly
  const mainFunctionMatch = scriptContent.match(
    /async function main\(\) \{([\s\S]*?)\}\s*await main\(\);?\s*$/
  );
  if (!mainFunctionMatch) {
    throw new Error("Could not extract main function from script");
  }

  const mainFunctionBody = mainFunctionMatch[1];

  // Create a testable function that has the same logic but can be called with dependencies
  return new Function(`
    const { fs, crypto, execSync, github, core, context, process, console } = arguments[0];
    
    return async function main() {
      ${mainFunctionBody}
    };
  `);
};

describe("create_pull_request.cjs", () => {
  let createMainFunction;
  let mockDependencies;

  beforeEach(() => {
    // Read the script content
    const scriptPath = path.join(
      process.cwd(),
      "pkg/workflow/js/create_pull_request.cjs"
    );
    const scriptContent = readFileSync(scriptPath, "utf8");

    // Create testable function
    createMainFunction = createTestableFunction(scriptContent);

    // Set up mock dependencies
    mockDependencies = {
      fs: {
        existsSync: vi.fn().mockReturnValue(true),
        readFileSync: vi
          .fn()
          .mockReturnValue("diff --git a/file.txt b/file.txt\n+new content"),
      },
      crypto: {
        randomBytes: vi
          .fn()
          .mockReturnValue(Buffer.from("1234567890abcdef", "hex")),
      },
      execSync: vi.fn(),
      github: {
        rest: {
          pulls: {
            create: vi.fn(),
          },
          issues: {
            addLabels: vi.fn(),
          },
        },
      },
      core: {
        setOutput: vi.fn(),
        summary: {
          addRaw: vi.fn().mockReturnThis(),
          write: vi.fn(),
        },
      },
      context: {
        runId: 12345,
        repo: {
          owner: "testowner",
          repo: "testrepo",
        },
        payload: {
          repository: {
            html_url: "https://github.com/testowner/testrepo",
          },
        },
      },
      process: {
        env: {},
      },
      console: {
        log: vi.fn(),
      },
    };
  });

  it("should throw error when GITHUB_AW_WORKFLOW_ID is missing", async () => {
    const mainFunction = createMainFunction(mockDependencies);

    await expect(mainFunction()).rejects.toThrow(
      "GITHUB_AW_WORKFLOW_ID environment variable is required"
    );
  });

  it("should throw error when GITHUB_AW_BASE_BRANCH is missing", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";

    const mainFunction = createMainFunction(mockDependencies);

    await expect(mainFunction()).rejects.toThrow(
      "GITHUB_AW_BASE_BRANCH environment variable is required"
    );
  });

  it("should throw error when patch file does not exist", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.fs.existsSync.mockReturnValue(false);

    const mainFunction = createMainFunction(mockDependencies);

    await expect(mainFunction()).rejects.toThrow(
      "No patch file found - cannot create pull request without changes"
    );
  });

  it("should throw error when patch file is empty", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.fs.readFileSync.mockReturnValue("   ");

    const mainFunction = createMainFunction(mockDependencies);

    await expect(mainFunction()).rejects.toThrow(
      "Patch file is empty or contains error message - cannot create pull request without changes"
    );
  });

  it("should create pull request successfully with valid input", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create-pull-request",
          title: "New Feature",
          body: "This adds a new feature to the codebase.",
        },
      ],
    });

    const mockPullRequest = {
      number: 123,
      html_url: "https://github.com/testowner/testrepo/pull/123",
    };

    mockDependencies.github.rest.pulls.create.mockResolvedValue({
      data: mockPullRequest,
    });

    const mainFunction = createMainFunction(mockDependencies);

    await mainFunction();

    // Verify git operations
    expect(mockDependencies.execSync).toHaveBeenCalledWith(
      'git config --global user.email "action@github.com"',
      { stdio: "inherit" }
    );
    expect(mockDependencies.execSync).toHaveBeenCalledWith(
      'git config --global user.name "GitHub Action"',
      { stdio: "inherit" }
    );
    expect(mockDependencies.execSync).toHaveBeenCalledWith(
      "git checkout -b test-workflow/1234567890abcdef",
      { stdio: "inherit" }
    );
    expect(mockDependencies.execSync).toHaveBeenCalledWith(
      "git apply /tmp/aw.patch",
      { stdio: "inherit" }
    );
    expect(mockDependencies.execSync).toHaveBeenCalledWith("git add .", {
      stdio: "inherit",
    });
    expect(mockDependencies.execSync).toHaveBeenCalledWith(
      'git commit -m "Add agent output: New Feature"',
      { stdio: "inherit" }
    );
    expect(mockDependencies.execSync).toHaveBeenCalledWith(
      "git push origin test-workflow/1234567890abcdef",
      { stdio: "inherit" }
    );

    // Verify PR creation
    expect(mockDependencies.github.rest.pulls.create).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      title: "New Feature",
      body: expect.stringContaining("This adds a new feature to the codebase."),
      head: "test-workflow/1234567890abcdef",
      base: "main",
      draft: true, // default value
    });

    expect(mockDependencies.core.setOutput).toHaveBeenCalledWith(
      "pull_request_number",
      123
    );
    expect(mockDependencies.core.setOutput).toHaveBeenCalledWith(
      "pull_request_url",
      mockPullRequest.html_url
    );
    expect(mockDependencies.core.setOutput).toHaveBeenCalledWith(
      "branch_name",
      "test-workflow/1234567890abcdef"
    );
  });

  it("should handle labels correctly", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create-pull-request",
          title: "PR with labels",
          body: "PR with labels",
        },
      ],
    });
    mockDependencies.process.env.GITHUB_AW_PR_LABELS =
      "enhancement, automated, needs-review";

    const mockPullRequest = {
      number: 456,
      html_url: "https://github.com/testowner/testrepo/pull/456",
    };

    mockDependencies.github.rest.pulls.create.mockResolvedValue({
      data: mockPullRequest,
    });
    mockDependencies.github.rest.issues.addLabels.mockResolvedValue({});

    const mainFunction = createMainFunction(mockDependencies);

    await mainFunction();

    // Verify labels were added
    expect(mockDependencies.github.rest.issues.addLabels).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 456,
      labels: ["enhancement", "automated", "needs-review"],
    });
  });

  it("should respect draft setting from environment", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create-pull-request",
          title: "Non-draft PR",
          body: "Non-draft PR",
        },
      ],
    });
    mockDependencies.process.env.GITHUB_AW_PR_DRAFT = "false";

    const mockPullRequest = {
      number: 789,
      html_url: "https://github.com/testowner/testrepo/pull/789",
    };

    mockDependencies.github.rest.pulls.create.mockResolvedValue({
      data: mockPullRequest,
    });

    const mainFunction = createMainFunction(mockDependencies);

    await mainFunction();

    const callArgs = mockDependencies.github.rest.pulls.create.mock.calls[0][0];
    expect(callArgs.draft).toBe(false);
  });

  it("should include run information in PR body", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create-pull-request",
          title: "Test PR Title",
          body: "Test PR content with detailed body information.",
        },
      ],
    });

    const mockPullRequest = {
      number: 202,
      html_url: "https://github.com/testowner/testrepo/pull/202",
    };

    mockDependencies.github.rest.pulls.create.mockResolvedValue({
      data: mockPullRequest,
    });

    const mainFunction = createMainFunction(mockDependencies);

    await mainFunction();

    const callArgs = mockDependencies.github.rest.pulls.create.mock.calls[0][0];
    expect(callArgs.title).toBe("Test PR Title");
    expect(callArgs.body).toContain(
      "Test PR content with detailed body information."
    );
    expect(callArgs.body).toContain("Generated by Agentic Workflow Run");
    expect(callArgs.body).toContain("[12345]");
    expect(callArgs.body).toContain(
      "https://github.com/testowner/testrepo/actions/runs/12345"
    );
  });

  it("should apply title prefix when provided", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create-pull-request",
          title: "Simple PR title",
          body: "Simple PR body content",
        },
      ],
    });
    mockDependencies.process.env.GITHUB_AW_PR_TITLE_PREFIX = "[BOT] ";

    const mockPullRequest = {
      number: 987,
      html_url: "https://github.com/testowner/testrepo/pull/987",
    };

    mockDependencies.github.rest.pulls.create.mockResolvedValue({
      data: mockPullRequest,
    });

    const mainFunction = createMainFunction(mockDependencies);

    await mainFunction();

    const callArgs = mockDependencies.github.rest.pulls.create.mock.calls[0][0];
    expect(callArgs.title).toBe("[BOT] Simple PR title");
  });

  it("should not duplicate title prefix when already present", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create-pull-request",
          title: "[BOT] PR title already prefixed",
          body: "PR body content",
        },
      ],
    });
    mockDependencies.process.env.GITHUB_AW_PR_TITLE_PREFIX = "[BOT] ";

    const mockPullRequest = {
      number: 988,
      html_url: "https://github.com/testowner/testrepo/pull/988",
    };

    mockDependencies.github.rest.pulls.create.mockResolvedValue({
      data: mockPullRequest,
    });

    const mainFunction = createMainFunction(mockDependencies);

    await mainFunction();

    const callArgs = mockDependencies.github.rest.pulls.create.mock.calls[0][0];
    expect(callArgs.title).toBe("[BOT] PR title already prefixed"); // Should not be duplicated
  });
});
