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

  it("should handle missing patch file with default warn behavior", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.fs.existsSync.mockReturnValue(false);

    const mainFunction = createMainFunction(mockDependencies);

    await mainFunction();

    expect(mockDependencies.core.warning).toHaveBeenCalledWith(
      "No patch file found - cannot create pull request without changes"
    );
    expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
  });

  it("should handle empty patch with default warn behavior when patch file is empty", async () => {
    mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
    mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
    mockDependencies.fs.readFileSync.mockReturnValue("   ");

    const mainFunction = createMainFunction(mockDependencies);

    await mainFunction();

    expect(mockDependencies.core.warning).toHaveBeenCalledWith(
      "Patch file is empty - no changes to apply (noop operation)"
    );
    expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
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
    expect(mockDependencies.execSync).toHaveBeenCalledWith(
      "git checkout -b test-workflow-1234567890abcdef",
      { stdio: "inherit" }
    );
    expect(mockDependencies.execSync).toHaveBeenCalledWith(
      "git am /tmp/aw.patch",
      { stdio: "inherit" }
    );
    expect(mockDependencies.execSync).toHaveBeenCalledWith(
      "git push origin test-workflow-1234567890abcdef",
      { stdio: "inherit" }
    );

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
      "test-workflow-1234567890abcdef"
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
          type: "create-pull-request",
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
          type: "create-pull-request",
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
    expect(callArgs.body).toContain(
      "Test PR content with detailed body information."
    );
    expect(callArgs.body).toContain("Generated by Agentic Workflow");
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
          type: "create-pull-request",
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

  describe("if-no-changes configuration", () => {
    beforeEach(() => {
      mockDependencies.process.env.GITHUB_AW_WORKFLOW_ID = "test-workflow";
      mockDependencies.process.env.GITHUB_AW_BASE_BRANCH = "main";
      mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "create-pull-request",
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

      expect(mockDependencies.core.warning).toHaveBeenCalledWith(
        "Patch file is empty - no changes to apply (noop operation)"
      );
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
    });

    it("should handle empty patch with ignore behavior", async () => {
      mockDependencies.fs.readFileSync.mockReturnValue("");
      mockDependencies.process.env.GITHUB_AW_PR_IF_NO_CHANGES = "ignore";

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      expect(mockDependencies.core.info).not.toHaveBeenCalledWith(
        expect.stringContaining("Patch file is empty")
      );
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
    });

    it("should handle empty patch with error behavior", async () => {
      mockDependencies.fs.readFileSync.mockReturnValue("");
      mockDependencies.process.env.GITHUB_AW_PR_IF_NO_CHANGES = "error";

      const mainFunction = createMainFunction(mockDependencies);

      await expect(mainFunction()).rejects.toThrow(
        "No changes to push - failing as configured by if-no-changes: error"
      );
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
    });

    it("should handle missing patch file with warn behavior", async () => {
      mockDependencies.fs.existsSync.mockReturnValue(false);
      mockDependencies.process.env.GITHUB_AW_PR_IF_NO_CHANGES = "warn";

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      expect(mockDependencies.core.warning).toHaveBeenCalledWith(
        "No patch file found - cannot create pull request without changes"
      );
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
    });

    it("should handle missing patch file with ignore behavior", async () => {
      mockDependencies.fs.existsSync.mockReturnValue(false);
      mockDependencies.process.env.GITHUB_AW_PR_IF_NO_CHANGES = "ignore";

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      expect(mockDependencies.core.info).not.toHaveBeenCalledWith(
        expect.stringContaining("No patch file found")
      );
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
    });

    it("should handle missing patch file with error behavior", async () => {
      mockDependencies.fs.existsSync.mockReturnValue(false);
      mockDependencies.process.env.GITHUB_AW_PR_IF_NO_CHANGES = "error";

      const mainFunction = createMainFunction(mockDependencies);

      await expect(mainFunction()).rejects.toThrow(
        "No patch file found - cannot create pull request without changes"
      );
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
    });

    it("should handle patch with error message with warn behavior", async () => {
      mockDependencies.fs.readFileSync.mockReturnValue(
        "Failed to generate patch: some error"
      );
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

      expect(mockDependencies.core.warning).toHaveBeenCalledWith(
        "Patch file is empty - no changes to apply (noop operation)"
      );
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
            type: "create-pull-request",
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
        expect.stringContaining(
          "## 🎭 Staged Mode: Create Pull Request Preview"
        )
      );
      expect(mockDependencies.core.summary.write).toHaveBeenCalled();

      // Verify console log for staged mode
      expect(mockDependencies.core.info).toHaveBeenCalledWith(
        "📝 Pull request creation preview written to step summary"
      );

      // Verify that actual PR creation was not called
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
      expect(mockDependencies.execSync).not.toHaveBeenCalled();
    });

    it("should include patch information in staged mode summary", async () => {
      mockDependencies.process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true";
      mockDependencies.fs.readFileSync.mockReturnValue(
        "diff --git a/test.txt b/test.txt\n+added line\n-removed line"
      );

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      const summaryCall = mockDependencies.core.summary.addRaw.mock.calls[0][0];
      expect(summaryCall).toContain("**Title:** Staged Mode Test PR");
      expect(summaryCall).toContain("**Branch:** feature-test");
      expect(summaryCall).toContain("**Base:** main");
      expect(summaryCall).toContain("**Body:**");
      expect(summaryCall).toContain(
        "This is a test PR for staged mode functionality."
      );
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
            type: "create-pull-request",
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
      expect(mockDependencies.execSync).not.toHaveBeenCalledWith(
        expect.stringContaining("git"),
        expect.anything()
      );

      // Verify no GitHub API calls were made
      expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled();
      expect(
        mockDependencies.github.rest.issues.addLabels
      ).not.toHaveBeenCalled();

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
      expect(summaryCall).toContain(
        "No patch file found - cannot create pull request without changes"
      );

      // Verify console log for staged mode
      expect(mockDependencies.core.info).toHaveBeenCalledWith(
        "📝 Pull request creation preview written to step summary (no patch file)"
      );
    });

    it("should handle patch error in staged mode", async () => {
      mockDependencies.process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true";
      mockDependencies.fs.readFileSync.mockReturnValue(
        "Failed to generate patch: some error occurred"
      );

      const mainFunction = createMainFunction(mockDependencies);

      await mainFunction();

      // Verify that step summary was written showing the patch error
      expect(mockDependencies.core.summary.addRaw).toHaveBeenCalled();
      const summaryCall = mockDependencies.core.summary.addRaw.mock.calls[0][0];
      expect(summaryCall).toContain("⚠️ Patch file contains error");
      expect(summaryCall).toContain(
        "Patch file contains error message - cannot create pull request without changes"
      );

      // Verify console log for staged mode
      expect(mockDependencies.core.info).toHaveBeenCalledWith(
        "📝 Pull request creation preview written to step summary (patch error)"
      );
    });

    it("should validate patch size within limit", async () => {
      mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "create-pull-request",
            title: "Test PR",
            body: "This is a test PR",
            branch: "test-branch",
          },
        ],
      });
      mockDependencies.process.env.GITHUB_AW_MAX_PATCH_SIZE = "10"; // 10 KB limit

      mockDependencies.fs.existsSync.mockReturnValue(true);
      // Create patch content under 10 KB (approximately 5 KB)
      const patchContent =
        "diff --git a/file.txt b/file.txt\n+new content\n".repeat(100);
      mockDependencies.fs.readFileSync.mockReturnValue(patchContent);

      const main = createMainFunction(mockDependencies);
      await main();

      expect(mockDependencies.core.info).toHaveBeenCalledWith(
        expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 10 KB\)/)
      );
      expect(mockDependencies.core.info).toHaveBeenCalledWith(
        "Patch size validation passed"
      );
      // Should not throw an error
    });

    it("should fail when patch size exceeds limit", async () => {
      mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "create-pull-request",
            title: "Test PR",
            body: "This is a test PR",
            branch: "test-branch",
          },
        ],
      });
      mockDependencies.process.env.GITHUB_AW_MAX_PATCH_SIZE = "1"; // 1 KB limit

      mockDependencies.fs.existsSync.mockReturnValue(true);
      // Create patch content over 1 KB (approximately 5 KB)
      const patchContent =
        "diff --git a/file.txt b/file.txt\n+new content\n".repeat(100);
      mockDependencies.fs.readFileSync.mockReturnValue(patchContent);

      const main = createMainFunction(mockDependencies);

      await expect(main()).rejects.toThrow(
        expect.stringMatching(
          /Patch size \(\d+ KB\) exceeds maximum allowed size \(1 KB\)/
        )
      );

      expect(mockDependencies.core.info).toHaveBeenCalledWith(
        expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 1 KB\)/)
      );
    });

    it("should show staged preview when patch size exceeds limit in staged mode", async () => {
      mockDependencies.process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true";
      mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "create-pull-request",
            title: "Test PR",
            body: "This is a test PR",
            branch: "test-branch",
          },
        ],
      });
      mockDependencies.process.env.GITHUB_AW_MAX_PATCH_SIZE = "1"; // 1 KB limit

      mockDependencies.fs.existsSync.mockReturnValue(true);
      // Create patch content over 1 KB (approximately 5 KB)
      const patchContent =
        "diff --git a/file.txt b/file.txt\n+new content\n".repeat(100);
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
            type: "create-pull-request",
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

      const main = createMainFunction(mockDependencies);
      await main();

      expect(mockDependencies.core.info).toHaveBeenCalledWith(
        expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 1024 KB\)/)
      );
      expect(mockDependencies.core.info).toHaveBeenCalledWith(
        "Patch size validation passed"
      );
    });

    it("should skip patch size validation for empty patches", async () => {
      mockDependencies.process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "create-pull-request",
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
      expect(mockDependencies.core.info).not.toHaveBeenCalledWith(
        expect.stringMatching(/Patch size:/)
      );
      // Should proceed with empty patch handling
    });
  });
});
