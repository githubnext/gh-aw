import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
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
};

const mockContext = {
  eventName: "pull_request",
  payload: {
    pull_request: { number: 123 },
    repository: { html_url: "https://github.com/testowner/testrepo" },
  },
  repo: { owner: "testowner", repo: "testrepo" },
};

// Set up global variables
global.core = mockCore;
global.context = mockContext;

describe("push_to_pr_branch.cjs", () => {
  let pushToPrBranchScript;
  let mockFs;
  let mockExecSync;

  // Helper function to execute the script with proper globals
  const executeScript = async () => {
    // Set globals just before execution
    global.core = mockCore;
    global.context = mockContext;
    global.mockFs = mockFs;
    global.mockExecSync = mockExecSync;

    // Execute the script
    return await eval(`(async () => { ${pushToPrBranchScript} })()`);
  };

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Clear environment variables
    delete process.env.GITHUB_AW_PUSH_TARGET;
    delete process.env.GITHUB_AW_AGENT_OUTPUT;
    delete process.env.GITHUB_AW_PUSH_IF_NO_CHANGES;

    // Create fresh mock objects for each test
    mockFs = {
      existsSync: vi.fn(),
      readFileSync: vi.fn(),
    };

    // Create fresh mock for execSync
    mockExecSync = vi.fn();

    // Reset mockCore calls
    mockCore.setFailed.mockReset();
    mockCore.setOutput.mockReset();
    mockCore.warning.mockReset();
    mockCore.error.mockReset();

    // Read the script content
    const scriptPath = path.join(
      process.cwd(),
      "pkg/workflow/js/push_to_pr_branch.cjs"
    );
    pushToPrBranchScript = fs.readFileSync(scriptPath, "utf8");

    // Modify the script to inject our mocks and make core available
    pushToPrBranchScript = pushToPrBranchScript.replace(
      'async function main() {\n  /** @type {typeof import("fs")} */\n  const fs = require("fs");\n  const { execSync } = require("child_process");',
      `async function main() {
  const core = global.core;
  const context = global.context || {};
  const fs = global.mockFs;
  const execSync = global.mockExecSync;`
    );
  });

  afterEach(() => {
    // Clean up globals safely
    if (typeof global !== "undefined") {
      delete global.core;
      delete global.context;
      delete global.mockFs;
      delete global.mockExecSync;
    }
  });

  describe("Script execution", () => {
    it("should skip when no agent output is provided", async () => {
      // Remove the output content environment variable
      delete process.env.GITHUB_AW_AGENT_OUTPUT;

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(
        "Agent output content is empty"
      );
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should skip when agent output is empty", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = "   ";

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(
        "Agent output content is empty"
      );
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle missing patch file with default 'warn' behavior", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [{ type: "push-to-pr-branch", content: "test" }],
      });

      mockFs.existsSync.mockReturnValue(false);

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(
        "No patch file found - cannot push without changes"
      );
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should fail when patch file missing and if-no-changes is 'error'", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [{ type: "push-to-pr-branch", content: "test" }],
      });
      process.env.GITHUB_AW_PUSH_IF_NO_CHANGES = "error";

      mockFs.existsSync.mockReturnValue(false);

      // Execute the script
      await executeScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "No patch file found - cannot push without changes"
      );
    });

    it("should silently succeed when patch file missing and if-no-changes is 'ignore'", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [{ type: "push-to-pr-branch", content: "test" }],
      });
      process.env.GITHUB_AW_PUSH_IF_NO_CHANGES = "ignore";

      mockFs.existsSync.mockReturnValue(false);

      // Execute the script
      await executeScript();

      expect(mockCore.info).not.toHaveBeenCalled();
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle patch file with error content", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [{ type: "push-to-pr-branch", content: "test" }],
      });

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue(
        "Failed to generate patch: some error"
      );

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(
        "Patch file contains error message - cannot push without changes"
      );
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle empty patch file with default 'warn' behavior", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [{ type: "push-to-pr-branch", content: "test" }],
      });

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("");

      // Mock the git command to return a branch name
      mockExecSync.mockReturnValue("feature-branch");

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(
        "Patch file is empty - no changes to apply (noop operation)"
      );
      expect(mockCore.info).toHaveBeenCalledWith(
        expect.stringMatching(/Agent output content length: \d+/)
      );
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should fail when empty patch and if-no-changes is 'error'", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [{ type: "push-to-pr-branch", content: "test" }],
      });
      process.env.GITHUB_AW_PUSH_IF_NO_CHANGES = "error";

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("   ");

      // Execute the script
      await executeScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "No changes to push - failing as configured by if-no-changes: error"
      );
    });

    it("should handle valid patch content and parse JSON output", async () => {
      const validOutput = {
        items: [
          {
            type: "push-to-pr-branch",
            content: "some changes to push",
          },
        ],
      };

      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue(
        "diff --git a/file.txt b/file.txt\n+new content"
      );

      // Mock the git commands that will be called
      mockExecSync.mockReturnValue("feature-branch");

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(
        expect.stringMatching(/Agent output content length: \d+/)
      );
      expect(mockCore.info).toHaveBeenCalledWith(
        "Patch content validation passed"
      );
      expect(mockCore.info).toHaveBeenCalledWith(
        "Target configuration: triggering"
      );
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle invalid JSON in agent output", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = "invalid json content";

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("some patch content");

      // Execute the script
      await executeScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        expect.stringMatching(/Error parsing agent output JSON:/)
      );
    });

    it("should handle agent output without valid items array", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: "not an array",
      });

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("some patch content");

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(
        "No valid items found in agent output"
      );
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should use custom target configuration", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [{ type: "push-to-pr-branch", content: "test" }],
      });
      process.env.GITHUB_AW_PUSH_TARGET = "custom-target";

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("some patch content");

      // Mock the git commands
      mockExecSync.mockReturnValue("feature-branch");

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(
        `Target configuration: ${"custom-target"}`
      );
    });
  });

  describe("Script validation", () => {
    it("should have valid JavaScript syntax", () => {
      const scriptPath = path.join(__dirname, "push_to_pr_branch.cjs");
      const scriptContent = fs.readFileSync(scriptPath, "utf8");

      // Basic syntax validation - should not contain obvious errors
      expect(scriptContent).toContain("async function main()");
      expect(scriptContent).toContain("core.setFailed");
      expect(scriptContent).toContain("/tmp/aw.patch");
      expect(scriptContent).toContain("await main()");
    });

    it("should export a main function", () => {
      const scriptPath = path.join(__dirname, "push_to_pr_branch.cjs");
      const scriptContent = fs.readFileSync(scriptPath, "utf8");

      // Check that the script has the expected structure
      expect(scriptContent).toMatch(/async function main\(\) \{[\s\S]*\}/);
    });
  });
});
