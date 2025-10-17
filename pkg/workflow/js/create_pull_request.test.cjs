import { describe, it, expect, beforeEach, vi } from "vitest";
import { readFileSync } from "fs";
import path from "path";

// Create standalone test functions by extracting parts of the script
const createTestableFunction = scriptContent => {
  // Extract everything before await main() - this includes helper functions and the main function
  const beforeMainCall = scriptContent.match(/^([\s\S]*?)\s*await main\(\);?\s*$/);
  if (!beforeMainCall) {
    throw new Error("Could not extract script content before await main()");
  }

  let scriptBody = beforeMainCall[1];

  // Remove const declarations for fs and crypto since they'll be provided as parameters
  scriptBody = scriptBody.replace(/\/\*\* @type \{typeof import\("fs"\)\} \*\/\s*const fs = require\("fs"\);?\s*/g, "");
  scriptBody = scriptBody.replace(/\/\*\* @type \{typeof import\("crypto"\)\} \*\/\s*const crypto = require\("crypto"\);?\s*/g, "");

  // Create a testable function that has the same logic but can be called with dependencies
  return new Function(`
    const { fs, crypto, github, core, context, process, console } = arguments[0];
    
    ${scriptBody}
    
    return main;
  `);
};

describe("create_pull_request.cjs", () => {
  let createMainFunction;
  let mockDependencies;

  beforeEach(() => {
    // Read the script content
    const scriptPath = path.join(process.cwd(), "create_pull_request.cjs");
    const scriptContent = readFileSync(scriptPath, "utf8");

    // Create testable function
    createMainFunction = createTestableFunction(scriptContent);

    // Set up global exec mock
    global.exec = {
      exec: vi.fn().mockResolvedValue(0), // Return exit code directly
      getExecOutput: vi.fn().mockResolvedValue({ exitCode: 0, stdout: "", stderr: "" }),
    };

    // Set up mock dependencies
    mockDependencies = {
      fs: {
        existsSync: vi.fn().mockReturnValue(true),
        readFileSync: vi.fn().mockReturnValue("diff --git a/file.txt b/file.txt\n+new content"),
      },
      crypto: {
        randomBytes: vi.fn().mockReturnValue(Buffer.from("1234567890abcdef", "hex")),
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
        // Core logging functions
        debug: vi.fn(),
        info: vi.fn(),
        notice: vi.fn(),
        warning: vi.fn(),
        error: vi.fn(),

        // Core workflow functions
        setFailed: vi.fn(),
        setOutput: vi.fn(),
        exportVariable: vi.fn(),
        setSecret: vi.fn(),

        // Input/state functions (less commonly used but included for completeness)
        getInput: vi.fn(),
        getBooleanInput: vi.fn(),
        getMultilineInput: vi.fn(),
        getState: vi.fn(),
        saveState: vi.fn(),

        // Group functions
        startGroup: vi.fn(),
        endGroup: vi.fn(),
        group: vi.fn(),

        // Other utility functions
        addPath: vi.fn(),
        setCommandEcho: vi.fn(),
        isDebug: vi.fn().mockReturnValue(false),
        getIDToken: vi.fn(),
        toPlatformPath: vi.fn(),
        toPosixPath: vi.fn(),
        toWin32Path: vi.fn(),

        // Summary object with chainable methods
        summary: {
          addRaw: vi.fn().mockReturnThis(),
          write: vi.fn().mockResolvedValue(),
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

  afterEach(() => {
    // Clean up temporary file
    if (tempFilePath && require("fs").existsSync(tempFilePath)) {
      require("fs").unlinkSync(tempFilePath);
      tempFilePath = undefined;
    }

    // Clean up global exec mock
    if (typeof global !== "undefined") {
      delete global.exec;
    }
  });

  it("should throw error when GITHUB_AW_WORKFLOW_ID is missing", async () => {
    const mainFunction = createMainFunction(mockDependencies);

    await expect(mainFunction()).rejects.toThrow("GITHUB_AW_WORKFLOW_ID environment variable is required");
  });

  it("should throw error when GITHUB_AW_BASE_BRANCH is missing", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";

    const mainFunction = createMainFunction(mockDependencies);

    await expect(mainFunction()).rejects.toThrow("GITHUB_AW_BASE_BRANCH environment variable is required");
  });

  it("should handle missing patch file with default warn behavior", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.fs.existsSync.mockReturnValue(false);

    const mainFunction = createMainFunction(mockDependencies);

    await mainFunction();

    expect(mockDependencies.core.warning).toHaveBeenCalledWith("No patch file found - cannot create pull request without changes");
    expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
  });

  it("should handle empty patch with default warn behavior when patch file is empty", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.fs.readFileSync.mockReturnValue("   ");

    const mainFunction = createMainFunction(mockDependencies);

    await mainFunction();

    expect(mockDependencies.core.warning).toHaveBeenCalledWith("Patch file is empty - no changes to apply (noop operation)");
    expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
  });

  it("should create pull request successfully with valid input", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create_pull_request",
          title: "New Feature",
          body: "This adds a new feature to the codebase.",
        },
      ],
    });

    // Mock execSync to simulate git behavior with changes
    mockDependencies.execSync.mockImplementation(command => {
      if (command === "git diff --cached --exit-code") {
        // Throw to indicate changes are present (non-zero exit code)
        const error = new Error("Changes exist");
        error.status = 1;
        throw error;
      }
      if (command === "git rev-parse HEAD") {
        return "abc123456";
      }
      // For all other git commands, just return normally
      return "";
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

    // Verify git operations (excluding git config which is handled by workflow)
    expect(global.exec.exec).toHaveBeenCalledWith("git fetch origin");
    expect(global.exec.exec).toHaveBeenCalledWith("git checkout main");
    expect(global.exec.exec).toHaveBeenCalledWith("git checkout -b test-workflow-1234567890abcdef");
    expect(global.exec.exec).toHaveBeenCalledWith("git am /tmp/gh-aw/aw.patch");
    expect(global.exec.exec).toHaveBeenCalledWith("git push origin test-workflow-1234567890abcdef");

    // Verify PR creation
    expect(mockDependencies.github.rest.pulls.create).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      title: "New Feature",
      body: expect.stringContaining("This adds a new feature to the codebase."),
      head: "test-workflow-1234567890abcdef",
      base: "main",
      draft: true, // default value
    });

    expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("pull_request_number", 123);
    expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("pull_request_url", mockPullRequest.html_url);
    expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("branch_name", "test-workflow-1234567890abcdef");
  });

  it("should handle labels correctly", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create_pull_request",
          title: "PR with labels",
          body: "PR with labels",
        },
      ],
    });
    mockDependencies.process.env.GITHUB_AW_PR_LABELS = "enhancement, automated, needs-review";

    // Mock execSync to simulate git behavior with changes
    mockDependencies.execSync.mockImplementation(command => {
      if (command === "git diff --cached --exit-code") {
        // Throw to indicate changes are present (non-zero exit code)
        const error = new Error("Changes exist");
        error.status = 1;
        throw error;
      }
      return "";
    });

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
          type: "create_pull_request",
          title: "Non-draft PR",
          body: "Non-draft PR",
        },
      ],
    });
    mockDependencies.process.env.GITHUB_AW_PR_DRAFT = "false";

    // Mock execSync to simulate git behavior with changes
    mockDependencies.execSync.mockImplementation(command => {
      if (command === "git diff --cached --exit-code") {
        // Throw to indicate changes are present (non-zero exit code)
        const error = new Error("Changes exist");
        error.status = 1;
        throw error;
      }
      return "";
    });

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
          type: "create_pull_request",
          title: "Test PR Title",
          body: "Test PR content with detailed body information.",
        },
      ],
    });

    // Mock execSync to simulate git behavior with changes
    mockDependencies.execSync.mockImplementation(command => {
      if (command === "git diff --cached --exit-code") {
        // Throw to indicate changes are present (non-zero exit code)
        const error = new Error("Changes exist");
        error.status = 1;
        throw error;
      }
      return "";
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
    expect(callArgs.body).toContain("Test PR content with detailed body information.");
    expect(callArgs.body).toContain("AI generated by");
    expect(callArgs.body).toContain("https://github.com/testowner/testrepo/actions/runs/12345");
  });

  it("should apply title prefix when provided", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create_pull_request",
          title: "Simple PR title",
          body: "Simple PR body content",
        },
      ],
    });
    mockDependencies.process.env.GITHUB_AW_PR_TITLE_PREFIX = "[BOT] ";

    // Mock execSync to simulate git behavior with changes
    mockDependencies.execSync.mockImplementation(command => {
      if (command === "git diff --cached --exit-code") {
        // Throw to indicate changes are present (non-zero exit code)
        const error = new Error("Changes exist");
        error.status = 1;
        throw error;
      }
      return "";
    });

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
          type: "create_pull_request",
          title: "[BOT] PR title already prefixed",
          body: "PR body content",
        },
      ],
    });
    mockDependencies.process.env.GITHUB_AW_PR_TITLE_PREFIX = "[BOT] ";

    // Mock execSync to simulate git behavior with changes
    mockDependencies.execSync.mockImplementation(command => {
      if (command === "git diff --cached --exit-code") {
        // Throw to indicate changes are present (non-zero exit code)
        const error = new Error("Changes exist");
        error.status = 1;
        throw error;
      }
      return "";
    });

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

  it("should fallback to creating issue when PR creation fails", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create_pull_request",
          title: "PR that will fail",
          body: "This PR creation will fail and fallback to an issue.",
        },
      ],
    });
    mockDependencies.process.env.GITHUB_AW_PR_LABELS = "enhancement, automated";

    // Mock execSync to simulate git behavior with changes
    mockDependencies.execSync.mockImplementation(command => {
      if (command === "git diff --cached --exit-code") {
        const error = new Error("Changes exist");
        error.status = 1;
        throw error;
      }
      return "";
    });

    // Mock PR creation to fail
    const prError = new Error("Pull request creation is disabled by organization policy");
    mockDependencies.github.rest.pulls.create.mockRejectedValue(prError);

    // Mock issue creation to succeed
    const mockIssue = {
      number: 456,
      html_url: "https://github.com/testowner/testrepo/issues/456",
    };
    mockDependencies.github.rest.issues = {
      ...mockDependencies.github.rest.issues,
      create: vi.fn().mockResolvedValue({ data: mockIssue }),
    };

    const mainFunction = createMainFunction(mockDependencies);

    await mainFunction();

    // Verify PR creation was attempted
    expect(mockDependencies.github.rest.pulls.create).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      title: "PR that will fail",
      body: expect.stringContaining("This PR creation will fail and fallback to an issue."),
      head: "test-workflow-1234567890abcdef",
      base: "main",
      draft: true,
    });

    // Verify fallback issue was created with enhanced body
    expect(mockDependencies.github.rest.issues.create).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      title: "PR that will fail",
      body: expect.stringMatching(
        /This PR creation will fail and fallback to an issue\.[\s\S]*---[\s\S]*Note.*originally intended as a pull request[\s\S]*Original error.*Pull request creation is disabled by organization policy[\s\S]*You can manually create a pull request from the branch if needed/
      ),
      labels: ["enhancement", "automated"],
    });

    // Verify warning was logged
    expect(mockDependencies.core.warning).toHaveBeenCalledWith(
      "Failed to create pull request: Pull request creation is disabled by organization policy"
    );
    expect(mockDependencies.core.info).toHaveBeenCalledWith("Falling back to creating an issue instead");
    expect(mockDependencies.core.info).toHaveBeenCalledWith(
      "Created fallback issue #456: https://github.com/testowner/testrepo/issues/456"
    );

    // Verify fallback outputs were set
    expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("issue_number", 456);
    expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("issue_url", mockIssue.html_url);
    expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("branch_name", "test-workflow-1234567890abcdef");
    expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("fallback_used", "true");

    // Verify fallback summary was written
    expect(mockDependencies.core.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("## Fallback Issue Created"));
  });

  it("should include patch preview in fallback issue when PR creation fails", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create_pull_request",
          title: "PR with patch preview",
          body: "This PR will fail and create an issue with patch preview.",
        },
      ],
    });

    // Create a patch with multiple lines (over 500 lines to test truncation)
    const patchLines = ["diff --git a/file.txt b/file.txt", "--- a/file.txt", "+++ b/file.txt", "@@ -1,1 +1,1 @@"];
    for (let i = 0; i < 600; i++) {
      patchLines.push(`+Line ${i}`);
    }
    const largePatch = patchLines.join("\n");
    mockDependencies.fs.readFileSync.mockReturnValue(largePatch);

    // Mock PR creation to fail
    const prError = new Error("Pull request creation is disabled");
    mockDependencies.github.rest.pulls.create.mockRejectedValue(prError);

    // Mock issue creation to succeed
    const mockIssue = {
      number: 789,
      html_url: "https://github.com/testowner/testrepo/issues/789",
    };
    mockDependencies.github.rest.issues = {
      ...mockDependencies.github.rest.issues,
      create: vi.fn().mockResolvedValue({ data: mockIssue }),
    };

    const mainFunction = createMainFunction(mockDependencies);
    await mainFunction();

    // Verify fallback issue was created with patch preview
    expect(mockDependencies.github.rest.issues.create).toHaveBeenCalled();
    const issueCreateCall = mockDependencies.github.rest.issues.create.mock.calls[0][0];

    // Should include patch preview with details tag and diff code block
    expect(issueCreateCall.body).toMatch(/<details><summary>Show patch preview \(500 of 604 lines\)<\/summary>/);
    expect(issueCreateCall.body).toMatch(/```diff/);
    expect(issueCreateCall.body).toMatch(/\.\.\. \(truncated\)/);

    // Should include first few lines but not all 604 lines
    expect(issueCreateCall.body).toContain("+Line 0");
    // Due to 2000 char limit, it won't contain line 100
    expect(issueCreateCall.body).not.toContain("+Line 550"); // Should be truncated before this
  });

  it("should truncate patch by character limit when it exceeds 2000 chars", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create_pull_request",
          title: "PR with large patch",
          body: "This PR will fail and create an issue with char-limited patch.",
        },
      ],
    });

    // Create a patch that exceeds 2000 chars but has fewer than 500 lines
    const patchLines = ["diff --git a/file.txt b/file.txt", "--- a/file.txt", "+++ b/file.txt", "@@ -1,1 +1,100 @@"];
    // Add lines with enough content to exceed 2000 chars
    for (let i = 0; i < 100; i++) {
      patchLines.push(`+This is a longer line ${i} with more content to trigger character limit truncation`);
    }
    const largePatch = patchLines.join("\n");
    mockDependencies.fs.readFileSync.mockReturnValue(largePatch);

    // Mock PR creation to fail
    const prError = new Error("Pull request creation is disabled");
    mockDependencies.github.rest.pulls.create.mockRejectedValue(prError);

    // Mock issue creation to succeed
    const mockIssue = {
      number: 790,
      html_url: "https://github.com/testowner/testrepo/issues/790",
    };
    mockDependencies.github.rest.issues = {
      ...mockDependencies.github.rest.issues,
      create: vi.fn().mockResolvedValue({ data: mockIssue }),
    };

    const mainFunction = createMainFunction(mockDependencies);
    await mainFunction();

    // Verify fallback issue was created with patch preview
    expect(mockDependencies.github.rest.issues.create).toHaveBeenCalled();
    const issueCreateCall = mockDependencies.github.rest.issues.create.mock.calls[0][0];

    // Should include patch preview with truncation
    expect(issueCreateCall.body).toMatch(/<details><summary>Show patch preview/);
    expect(issueCreateCall.body).toMatch(/```diff/);
    expect(issueCreateCall.body).toMatch(/\.\.\. \(truncated\)/);

    // Verify the patch content in the issue body is limited to 2000 chars
    const patchMatch = issueCreateCall.body.match(/```diff\n([\s\S]*?)\n\.\.\. \(truncated\)/);
    if (patchMatch) {
      const patchInBody = patchMatch[1];
      expect(patchInBody.length).toBeLessThanOrEqual(2000);
    }
  });

  it("should include full patch when under 500 lines in fallback issue", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create_pull_request",
          title: "PR with small patch",
          body: "This PR will fail and create an issue with full patch.",
        },
      ],
    });

    // Create a small patch (under 500 lines)
    const smallPatch = "diff --git a/file.txt b/file.txt\n--- a/file.txt\n+++ b/file.txt\n@@ -1,1 +1,1 @@\n+Small change";
    mockDependencies.fs.readFileSync.mockReturnValue(smallPatch);

    // Mock PR creation to fail
    const prError = new Error("Pull request creation is disabled");
    mockDependencies.github.rest.pulls.create.mockRejectedValue(prError);

    // Mock issue creation to succeed
    const mockIssue = {
      number: 790,
      html_url: "https://github.com/testowner/testrepo/issues/790",
    };
    mockDependencies.github.rest.issues = {
      ...mockDependencies.github.rest.issues,
      create: vi.fn().mockResolvedValue({ data: mockIssue }),
    };

    const mainFunction = createMainFunction(mockDependencies);
    await mainFunction();

    // Verify fallback issue was created with full patch
    expect(mockDependencies.github.rest.issues.create).toHaveBeenCalled();
    const issueCreateCall = mockDependencies.github.rest.issues.create.mock.calls[0][0];

    // Should include full patch without truncation
    expect(issueCreateCall.body).toMatch(/<details><summary>Show patch \(5 lines\)<\/summary>/);
    expect(issueCreateCall.body).toMatch(/```diff/);
    expect(issueCreateCall.body).toContain("+Small change");
    expect(issueCreateCall.body).not.toContain("... (truncated)");
  });

  it("should fail when both PR and fallback issue creation fail", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create_pull_request",
          title: "PR that will fail",
          body: "Both PR and issue creation will fail.",
        },
      ],
    });

    // Mock execSync to simulate git behavior with changes
    mockDependencies.execSync.mockImplementation(command => {
      if (command === "git diff --cached --exit-code") {
        const error = new Error("Changes exist");
        error.status = 1;
        throw error;
      }
      return "";
    });

    // Mock both PR and issue creation to fail
    const prError = new Error("Pull request creation failed");
    const issueError = new Error("Issue creation also failed");
    mockDependencies.github.rest.pulls.create.mockRejectedValue(prError);
    mockDependencies.github.rest.issues = {
      ...mockDependencies.github.rest.issues,
      create: vi.fn().mockRejectedValue(issueError),
    };

    const mainFunction = createMainFunction(mockDependencies);

    await mainFunction();

    // Verify both API calls were attempted
    expect(mockDependencies.github.rest.pulls.create).toHaveBeenCalled();
    expect(mockDependencies.github.rest.issues.create).toHaveBeenCalled();

    // Verify setFailed was called with combined error message
    expect(mockDependencies.core.setFailed).toHaveBeenCalledWith(
      "Failed to create both pull request and fallback issue. PR error: Pull request creation failed. Issue error: Issue creation also failed"
    );
  });

  it("should fallback to creating issue when git push fails", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create_pull_request",
          title: "Push will fail",
          body: "Git push will fail and fallback to an issue.",
        },
      ],
    });
    mockDependencies.process.env.GITHUB_AW_PR_LABELS = "automation";

    // Mock execSync to simulate git behavior with changes
    mockDependencies.execSync.mockImplementation(command => {
      if (command === "git diff --cached --exit-code") {
        const error = new Error("Changes exist");
        error.status = 1;
        throw error;
      }
      return "";
    });

    // Track exec.exec calls to fail on git push
    let execCallCount = 0;
    global.exec.exec = vi.fn().mockImplementation(async (cmd, args, options) => {
      execCallCount++;

      // Let git commands succeed except push
      if (typeof cmd === "string" && cmd.includes("git push")) {
        throw new Error("Permission denied (publickey)");
      }

      // For git ls-remote, return empty (no remote branch exists)
      if (typeof cmd === "string" && cmd.includes("git ls-remote")) {
        return 0;
      }

      return 0;
    });

    // Mock issue creation to succeed
    const mockIssue = {
      number: 789,
      html_url: "https://github.com/testowner/testrepo/issues/789",
    };
    mockDependencies.github.rest.issues = {
      ...mockDependencies.github.rest.issues,
      create: vi.fn().mockResolvedValue({ data: mockIssue }),
    };

    const mainFunction = createMainFunction(mockDependencies);

    await mainFunction();

    // Verify push was attempted (check if git push was called)
    const pushCallsForTest = global.exec.exec.mock.calls.filter(call => call[0] && call[0].includes("git push"));
    expect(pushCallsForTest.length).toBeGreaterThan(0);

    // Verify fallback issue was created with artifact link
    expect(mockDependencies.github.rest.issues.create).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      title: "Push will fail",
      body: expect.stringMatching(
        /Git push will fail[\s\S]*\[!NOTE\][\s\S]*git push operation failed[\s\S]*gh run download[\s\S]*git am aw\.patch/
      ),
      labels: ["automation"],
    });

    // Verify push failure outputs were set
    expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("issue_number", 789);
    expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("issue_url", mockIssue.html_url);
    expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("branch_name", "test-workflow-1234567890abcdef");
    expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("fallback_used", "true");
    expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("push_failed", "true");

    // Verify push failure summary was written
    expect(mockDependencies.core.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("## Push Failure Fallback"));
    expect(mockDependencies.core.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Permission denied"));

    // Verify appropriate logging
    expect(mockDependencies.core.error).toHaveBeenCalledWith(expect.stringContaining("Git push failed"));
    expect(mockDependencies.core.warning).toHaveBeenCalledWith(expect.stringContaining("Git push operation failed"));
  });

  it("should include patch preview in fallback issue when git push fails", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create_pull_request",
          title: "Push will fail with patch",
          body: "Git push will fail and create issue with patch preview.",
        },
      ],
    });

    // Create a patch with multiple lines to test preview
    const patchLines = ["diff --git a/test.js b/test.js", "--- a/test.js", "+++ b/test.js", "@@ -1,1 +1,1 @@"];
    for (let i = 0; i < 100; i++) {
      patchLines.push(`+Test line ${i}`);
    }
    const testPatch = patchLines.join("\n");
    mockDependencies.fs.readFileSync.mockReturnValue(testPatch);

    // Mock git push to fail
    global.exec.exec = vi.fn().mockImplementation(async (cmd, args, options) => {
      if (typeof cmd === "string" && cmd.includes("git push")) {
        throw new Error("Permission denied (publickey)");
      }
      if (typeof cmd === "string" && cmd.includes("git ls-remote")) {
        return 0;
      }
      return 0;
    });

    // Mock issue creation to succeed
    const mockIssue = {
      number: 890,
      html_url: "https://github.com/testowner/testrepo/issues/890",
    };
    mockDependencies.github.rest.issues = {
      ...mockDependencies.github.rest.issues,
      create: vi.fn().mockResolvedValue({ data: mockIssue }),
    };

    const mainFunction = createMainFunction(mockDependencies);
    await mainFunction();

    // Verify fallback issue was created with patch preview
    expect(mockDependencies.github.rest.issues.create).toHaveBeenCalled();
    const issueCreateCall = mockDependencies.github.rest.issues.create.mock.calls[0][0];

    // Should include patch preview in the issue body
    expect(issueCreateCall.body).toMatch(/<details><summary>Show patch \(104 lines\)<\/summary>/);
    expect(issueCreateCall.body).toMatch(/```diff/);
    expect(issueCreateCall.body).toContain("diff --git a/test.js b/test.js");
    expect(issueCreateCall.body).toContain("+Test line 0");
  });

  it("should fail when both git push and fallback issue creation fail", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create_pull_request",
          title: "Push and issue will fail",
          body: "Both git push and issue creation will fail.",
        },
      ],
    });

    // Mock execSync to simulate git behavior with changes
    mockDependencies.execSync.mockImplementation(command => {
      if (command === "git diff --cached --exit-code") {
        const error = new Error("Changes exist");
        error.status = 1;
        throw error;
      }
      return "";
    });

    // Mock git push to fail
    global.exec.exec = vi.fn().mockImplementation(async (cmd, args, options) => {
      if (typeof cmd === "string" && cmd.includes("git push")) {
        throw new Error("Network error: Connection timeout");
      }
      if (typeof cmd === "string" && cmd.includes("git ls-remote")) {
        return 0;
      }
      return 0;
    });

    // Mock issue creation to also fail
    const issueError = new Error("GitHub API rate limit exceeded");
    mockDependencies.github.rest.issues = {
      ...mockDependencies.github.rest.issues,
      create: vi.fn().mockRejectedValue(issueError),
    };

    const mainFunction = createMainFunction(mockDependencies);

    await mainFunction();

    // Verify push was attempted (check for git push calls)
    const pushCallsInFailTest = global.exec.exec.mock.calls.filter(call => call[0] && call[0].includes("git push"));
    expect(pushCallsInFailTest.length).toBeGreaterThan(0);

    // Verify issue creation was attempted
    expect(mockDependencies.github.rest.issues.create).toHaveBeenCalled();

    // Verify setFailed was called with combined error message
    expect(mockDependencies.core.setFailed).toHaveBeenCalledWith(
      expect.stringMatching(/Failed to push and failed to create fallback issue.*Network error.*GitHub API rate limit/)
    );
  });

  it("should handle remote branch collision by appending random suffix", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create_pull_request",
          title: "Test PR with branch collision",
          body: "This will handle remote branch collision.",
        },
      ],
    });

    // Mock execSync to simulate git behavior with changes
    mockDependencies.execSync.mockImplementation(command => {
      if (command === "git diff --cached --exit-code") {
        const error = new Error("Changes exist");
        error.status = 1;
        throw error;
      }
      return "";
    });

    // Mock crypto to return predictable values
    let randomBytesCallCount = 0;
    mockDependencies.crypto.randomBytes = vi.fn().mockImplementation(size => {
      randomBytesCallCount++;
      if (randomBytesCallCount === 1) {
        return Buffer.from("1234567890abcdef", "hex");
      } else {
        return Buffer.from("fedcba09", "hex");
      }
    });

    // Mock git commands
    global.exec.getExecOutput = vi.fn().mockImplementation(async cmd => {
      // git ls-remote should indicate remote branch exists
      if (typeof cmd === "string" && cmd.includes("git ls-remote")) {
        return {
          exitCode: 0,
          stdout: "abc123 refs/heads/test-workflow-1234567890abcdef\n",
          stderr: "",
        };
      }
      return { exitCode: 0, stdout: "", stderr: "" };
    });

    global.exec.exec = vi.fn().mockImplementation(async (cmd, args, options) => {
      // git branch -m should succeed
      if (typeof cmd === "string" && cmd.includes("git branch -m")) {
        return 0;
      }

      // git push should succeed
      if (typeof cmd === "string" && cmd.includes("git push")) {
        return 0;
      }

      return 0;
    });

    // Mock PR creation to succeed
    const mockPR = {
      number: 123,
      html_url: "https://github.com/testowner/testrepo/pull/123",
    };
    mockDependencies.github.rest.pulls.create.mockResolvedValue({ data: mockPR });

    const mainFunction = createMainFunction(mockDependencies);

    await mainFunction();

    // Verify branch rename was called
    const branchRenameCalls = global.exec.exec.mock.calls.filter(call => call[0] && call[0].includes("git branch -m"));
    expect(branchRenameCalls.length).toBeGreaterThan(0);

    // Verify warning about branch collision
    expect(mockDependencies.core.warning).toHaveBeenCalledWith(expect.stringContaining("already exists"));

    // Verify PR was created with renamed branch
    expect(mockDependencies.github.rest.pulls.create).toHaveBeenCalledWith(
      expect.objectContaining({
        head: expect.stringMatching(/test-workflow-1234567890abcdef-fedcba09/),
      })
    );
  });

  describe("if-no-changes configuration", () => {
    beforeEach(() => {
      mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
      mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
      mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "create_pull_request",
            title: "Test PR",
            body: "Test PR body",
          },
        ],
      });
    });

    it("should handle empty patch with warn (default) behavior", async () => {
      mockDependencies.fs.readFileSync.mockReturnValue("");
      mockDependencies.process.env.GITHUB_AW_PR_IF_NO_CHANGES = "warn";

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      expect(mockDependencies.core.warning).toHaveBeenCalledWith("Patch file is empty - no changes to apply (noop operation)");
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
    });

    it("should handle empty patch with ignore behavior", async () => {
      mockDependencies.fs.readFileSync.mockReturnValue("");
      mockDependencies.process.env.GITHUB_AW_PR_IF_NO_CHANGES = "ignore";

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      expect(mockDependencies.core.info).not.toHaveBeenCalledWith(expect.stringContaining("Patch file is empty"));
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
    });

    it("should handle empty patch with error behavior", async () => {
      mockDependencies.fs.readFileSync.mockReturnValue("");
      mockDependencies.process.env.GITHUB_AW_PR_IF_NO_CHANGES = "error";

      const mainFunction = createMainFunction(mockDependencies);

      await expect(mainFunction()).rejects.toThrow("No changes to push - failing as configured by if-no-changes: error");
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
    });

    it("should handle missing patch file with warn behavior", async () => {
      mockDependencies.fs.existsSync.mockReturnValue(false);
      mockDependencies.process.env.GITHUB_AW_PR_IF_NO_CHANGES = "warn";

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      expect(mockDependencies.core.warning).toHaveBeenCalledWith("No patch file found - cannot create pull request without changes");
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
    });

    it("should handle missing patch file with ignore behavior", async () => {
      mockDependencies.fs.existsSync.mockReturnValue(false);
      mockDependencies.process.env.GITHUB_AW_PR_IF_NO_CHANGES = "ignore";

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      expect(mockDependencies.core.info).not.toHaveBeenCalledWith(expect.stringContaining("No patch file found"));
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
    });

    it("should handle missing patch file with error behavior", async () => {
      mockDependencies.fs.existsSync.mockReturnValue(false);
      mockDependencies.process.env.GITHUB_AW_PR_IF_NO_CHANGES = "error";

      const mainFunction = createMainFunction(mockDependencies);

      await expect(mainFunction()).rejects.toThrow("No patch file found - cannot create pull request without changes");
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
    });

    it("should handle patch with error message with warn behavior", async () => {
      mockDependencies.fs.readFileSync.mockReturnValue("Failed to generate patch: some error");
      mockDependencies.process.env.GITHUB_AW_PR_IF_NO_CHANGES = "warn";

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      expect(mockDependencies.core.warning).toHaveBeenCalledWith(
        "Patch file contains error message - cannot create pull request without changes"
      );
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
    });

    it("should default to warn when if-no-changes is not specified", async () => {
      mockDependencies.fs.readFileSync.mockReturnValue("");
      // Don't set GITHUB_AW_PR_IF_NO_CHANGES env var

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      expect(mockDependencies.core.warning).toHaveBeenCalledWith("Patch file is empty - no changes to apply (noop operation)");
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
    });
  });

  describe("staged mode functionality", () => {
    beforeEach(() => {
      mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
      mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
      mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "create_pull_request",
            title: "Staged Mode Test PR",
            body: "This is a test PR for staged mode functionality.",
            branch: "feature-test",
          },
        ],
      });
    });

    it("should write step summary instead of creating PR when in staged mode", async () => {
      mockDependencies.process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true";

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      // Verify that step summary was written
      expect(mockDependencies.core.summary.addRaw).toHaveBeenCalledWith(
        expect.stringContaining("## 🎭 Staged Mode: Create Pull Request Preview")
      );
      expect(mockDependencies.core.summary.write).toHaveBeenCalled();

      // Verify console log for staged mode
      expect(mockDependencies.core.info).toHaveBeenCalledWith("📝 Pull request creation preview written to step summary");

      // Verify that actual PR creation was not called
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
      expect(mockDependencies.execSync).not.toHaveBeenCalled();
    });

    it("should include patch information in staged mode summary", async () => {
      mockDependencies.process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true";
      mockDependencies.fs.readFileSync.mockReturnValue("diff --git a/test.txt b/test.txt\n+added line\n-removed line");

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      const summaryCall = mockDependencies.core.summary.addRaw.mock.calls[0][0];
      expect(summaryCall).toContain("**Title:** Staged Mode Test PR");
      expect(summaryCall).toContain("**Branch:** feature-test");
      expect(summaryCall).toContain("**Base:** main");
      expect(summaryCall).toContain("**Body:**");
      expect(summaryCall).toContain("This is a test PR for staged mode functionality.");
      expect(summaryCall).toContain("**Changes:** Patch file exists with");
      expect(summaryCall).toContain("Show patch preview");
      expect(summaryCall).toContain("diff --git a/test.txt");
    });

    it("should handle empty patch in staged mode", async () => {
      mockDependencies.process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true";
      mockDependencies.fs.readFileSync.mockReturnValue("");

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      // Verify that step summary was written
      expect(mockDependencies.core.summary.addRaw).toHaveBeenCalled();
      const summaryCall = mockDependencies.core.summary.addRaw.mock.calls[0][0];
      expect(summaryCall).toContain("**Changes:** No changes (empty patch)");
      expect(summaryCall).not.toContain("Show patch preview");
    });

    it("should use auto-generated branch when no branch specified in staged mode", async () => {
      mockDependencies.process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true";
      mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "create_pull_request",
            title: "PR without branch",
            body: "Test PR body",
          },
        ],
      });

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      const summaryCall = mockDependencies.core.summary.addRaw.mock.calls[0][0];
      expect(summaryCall).toContain("**Branch:** auto-generated");
    });

    it("should not execute git operations in staged mode", async () => {
      mockDependencies.process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true";

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      // Verify no git operations were performed
      expect(mockDependencies.execSync).not.toHaveBeenCalledWith(expect.stringContaining("git"), expect.anything());

      // Verify no GitHub API calls were made
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
      expect(mockDependencies.github.rest.issues.addLabels).not.toHaveBeenCalled();

      // Verify no outputs were set
      expect(mockDependencies.core.setOutput).not.toHaveBeenCalled();
    });

    it("should handle missing patch file in staged mode", async () => {
      mockDependencies.process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true";
      mockDependencies.fs.existsSync.mockReturnValue(false);

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      // Verify that step summary was written showing the missing patch file
      expect(mockDependencies.core.summary.addRaw).toHaveBeenCalled();
      const summaryCall = mockDependencies.core.summary.addRaw.mock.calls[0][0];
      expect(summaryCall).toContain("⚠️ No patch file found");
      expect(summaryCall).toContain("No patch file found - cannot create pull request without changes");

      // Verify console log for staged mode
      expect(mockDependencies.core.info).toHaveBeenCalledWith("📝 Pull request creation preview written to step summary (no patch file)");
    });

    it("should handle patch error in staged mode", async () => {
      mockDependencies.process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true";
      mockDependencies.fs.readFileSync.mockReturnValue("Failed to generate patch: some error occurred");

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      // Verify that step summary was written showing the patch error
      expect(mockDependencies.core.summary.addRaw).toHaveBeenCalled();
      const summaryCall = mockDependencies.core.summary.addRaw.mock.calls[0][0];
      expect(summaryCall).toContain("⚠️ Patch file contains error");
      expect(summaryCall).toContain("Patch file contains error message - cannot create pull request without changes");

      // Verify console log for staged mode
      expect(mockDependencies.core.info).toHaveBeenCalledWith("📝 Pull request creation preview written to step summary (patch error)");
    });

    it("should validate patch size within limit", async () => {
      mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "create_pull_request",
            title: "Test PR",
            body: "This is a test PR",
            branch: "test-branch",
          },
        ],
      });
      mockDependencies.process.env.GITHUB_AW_MAX_PATCH_SIZE = "10"; // 10 KB limit

      mockDependencies.fs.existsSync.mockReturnValue(true);
      // Create patch content under 10 KB (approximately 5 KB)
      const patchContent = "diff --git a/file.txt b/file.txt\n+new content\n".repeat(100);
      mockDependencies.fs.readFileSync.mockReturnValue(patchContent);

      const mockPullRequest = {
        number: 123,
        html_url: "https://github.com/testowner/testrepo/pull/123",
      };

      mockDependencies.github.rest.pulls.create.mockResolvedValue({
        data: mockPullRequest,
      });

      const main = createMainFunction(mockDependencies);
      await main();

      expect(mockDependencies.core.info).toHaveBeenCalledWith(expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 10 KB\)/));
      expect(mockDependencies.core.info).toHaveBeenCalledWith("Patch size validation passed");
      // Should not throw an error
    });

    it("should fail when patch size exceeds limit", async () => {
      mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "create_pull_request",
            title: "Test PR",
            body: "This is a test PR",
            branch: "test-branch",
          },
        ],
      });
      mockDependencies.process.env.GITHUB_AW_MAX_PATCH_SIZE = "1"; // 1 KB limit

      mockDependencies.fs.existsSync.mockReturnValue(true);
      // Create patch content over 1 KB (approximately 5 KB)
      const patchContent = "diff --git a/file.txt b/file.txt\n+new content\n".repeat(100);
      mockDependencies.fs.readFileSync.mockReturnValue(patchContent);

      const main = createMainFunction(mockDependencies);

      await expect(main()).rejects.toThrow(/Patch size \(\d+ KB\) exceeds maximum allowed size \(1 KB\)/);

      expect(mockDependencies.core.info).toHaveBeenCalledWith(expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 1 KB\)/));
    });

    it("should show staged preview when patch size exceeds limit in staged mode", async () => {
      mockDependencies.process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true";
      mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "create_pull_request",
            title: "Test PR",
            body: "This is a test PR",
            branch: "test-branch",
          },
        ],
      });
      mockDependencies.process.env.GITHUB_AW_MAX_PATCH_SIZE = "1"; // 1 KB limit

      mockDependencies.fs.existsSync.mockReturnValue(true);
      // Create patch content over 1 KB (approximately 5 KB)
      const patchContent = "diff --git a/file.txt b/file.txt\n+new content\n".repeat(100);
      mockDependencies.fs.readFileSync.mockReturnValue(patchContent);

      const main = createMainFunction(mockDependencies);
      await main();

      // Should show staged preview instead of throwing error
      expect(mockDependencies.core.summary.addRaw).toHaveBeenCalled();
      const summaryCall = mockDependencies.core.summary.addRaw.mock.calls[0][0];
      expect(summaryCall).toContain("❌ Patch size exceeded");
      expect(summaryCall).toContain("exceeds maximum allowed size");

      // Verify console log for staged mode
      expect(mockDependencies.core.info).toHaveBeenCalledWith(
        "📝 Pull request creation preview written to step summary (patch size error)"
      );
    });

    it("should use default 1024 KB limit when env var not set", async () => {
      mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "create_pull_request",
            title: "Test PR",
            body: "This is a test PR",
            branch: "test-branch",
          },
        ],
      });
      delete mockDependencies.process.env.GITHUB_AW_MAX_PATCH_SIZE; // No limit set

      mockDependencies.fs.existsSync.mockReturnValue(true);
      const patchContent = "diff --git a/file.txt b/file.txt\n+new content\n";
      mockDependencies.fs.readFileSync.mockReturnValue(patchContent);

      const mockPullRequest = {
        number: 123,
        html_url: "https://github.com/testowner/testrepo/pull/123",
      };

      mockDependencies.github.rest.pulls.create.mockResolvedValue({
        data: mockPullRequest,
      });

      const main = createMainFunction(mockDependencies);
      await main();

      expect(mockDependencies.core.info).toHaveBeenCalledWith(expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 1024 KB\)/));
      expect(mockDependencies.core.info).toHaveBeenCalledWith("Patch size validation passed");
    });

    it("should skip patch size validation for empty patches", async () => {
      mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "create_pull_request",
            title: "Test PR",
            body: "This is a test PR",
            branch: "test-branch",
          },
        ],
      });
      mockDependencies.process.env.GITHUB_AW_MAX_PATCH_SIZE = "1"; // 1 KB limit

      mockDependencies.fs.existsSync.mockReturnValue(true);
      mockDependencies.fs.readFileSync.mockReturnValue(""); // Empty patch

      const main = createMainFunction(mockDependencies);
      await main();

      // Should not check patch size for empty patches
      expect(mockDependencies.core.info).not.toHaveBeenCalledWith(expect.stringMatching(/Patch size:/));
      // Should proceed with empty patch handling
    });
  });
});
