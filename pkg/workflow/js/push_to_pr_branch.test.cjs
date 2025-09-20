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
  let mockExec;

  // Helper function to execute the script with proper globals
  const executeScript = async () => {
    // Set globals just before execution
    global.core = mockCore;
    global.context = mockContext;
    global.mockFs = mockFs;
    global.mockExec = mockExec;

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

    // Create fresh mock for exec
    mockExec = {
      exec: vi.fn().mockImplementation((command, args, options) => {
        // Handle the gh pr view command specifically
        if (command === "gh" && args && args[0] === "pr" && args[1] === "view") {
          // Simulate the stdout listener being called with branch name
          if (options && options.listeners && options.listeners.stdout) {
            options.listeners.stdout(Buffer.from("feature-branch\n"));
          }
          return Promise.resolve(0); // Return exit code directly, not an object
        }
        // For other commands, just return success
        return Promise.resolve(0);
      }),
    };

    // Reset mockCore calls
    mockCore.setFailed.mockReset();
    mockCore.setOutput.mockReset();
    mockCore.warning.mockReset();
    mockCore.error.mockReset();

    // Read the script content
    const scriptPath = path.join(process.cwd(), "push_to_pr_branch.cjs");
    pushToPrBranchScript = fs.readFileSync(scriptPath, "utf8");

    // Modify the script to inject our mocks and make core available
    pushToPrBranchScript = pushToPrBranchScript.replace(
      /\/\*\* @type \{typeof import\("fs"\)\} \*\/\nconst fs = require\("fs"\);\n\/\*\* @type \{typeof import\("@actions\/exec"\)\} \*\/\nconst exec = require\("@actions\/exec"\);/,
      `const core = global.core;
const context = global.context || {};
const fs = global.mockFs;
const exec = global.mockExec;`
    );
  });

  afterEach(() => {
    // Clean up globals safely
    if (typeof global !== "undefined") {
      delete global.core;
      delete global.context;
      delete global.mockFs;
      delete global.mockExec;
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

    it("should validate patch size within limit", async () => {
      const validOutput = {
        items: [
          {
            type: "push-to-pr-branch",
            content: "some changes to push",
          },
        ],
      };

      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);
      process.env.GITHUB_AW_MAX_PATCH_SIZE = "10"; // 10 KB limit

      mockFs.existsSync.mockReturnValue(true);
      // Create patch content under 10 KB (approximately 5 KB)
      const patchContent =
        "diff --git a/file.txt b/file.txt\n+new content\n".repeat(100);
      mockFs.readFileSync.mockReturnValue(patchContent);

      // Mock the git commands that will be called

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(
        expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 10 KB\)/)
      );
      expect(mockCore.info).toHaveBeenCalledWith(
        "Patch size validation passed"
      );
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should fail when patch size exceeds limit", async () => {
      const validOutput = {
        items: [
          {
            type: "push-to-pr-branch",
            content: "some changes to push",
          },
        ],
      };

      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);
      process.env.GITHUB_AW_MAX_PATCH_SIZE = "1"; // 1 KB limit

      mockFs.existsSync.mockReturnValue(true);
      // Create patch content over 1 KB (approximately 5 KB)
      const patchContent =
        "diff --git a/file.txt b/file.txt\n+new content\n".repeat(100);
      mockFs.readFileSync.mockReturnValue(patchContent);

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(
        expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 1 KB\)/)
      );
      expect(mockCore.setFailed).toHaveBeenCalledWith(
        expect.stringMatching(
          /Patch size \(\d+ KB\) exceeds maximum allowed size \(1 KB\)/
        )
      );
    });

    it("should use default 1024 KB limit when env var not set", async () => {
      const validOutput = {
        items: [
          {
            type: "push-to-pr-branch",
            content: "some changes to push",
          },
        ],
      };

      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);
      delete process.env.GITHUB_AW_MAX_PATCH_SIZE; // No limit set

      mockFs.existsSync.mockReturnValue(true);
      const patchContent = "diff --git a/file.txt b/file.txt\n+new content\n";
      mockFs.readFileSync.mockReturnValue(patchContent);

      // Mock the git commands that will be called

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(
        expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 1024 KB\)/)
      );
      expect(mockCore.info).toHaveBeenCalledWith(
        "Patch size validation passed"
      );
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should skip patch size validation for empty patches", async () => {
      const validOutput = {
        items: [
          {
            type: "push-to-pr-branch",
            content: "some changes to push",
          },
        ],
      };

      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);
      process.env.GITHUB_AW_MAX_PATCH_SIZE = "1"; // 1 KB limit

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue(""); // Empty patch

      // Mock the git commands that will be called

      // Execute the script
      await executeScript();

      // Should not check patch size for empty patches
      expect(mockCore.info).not.toHaveBeenCalledWith(
        expect.stringMatching(/Patch size:/)
      );
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });
});
