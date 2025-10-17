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

describe("push_to_pull_request_branch.cjs", () => {
  let pushToPrBranchScript;
  let mockFs;
  let mockExec;
  let tempFilePath;

  // Helper function to set agent output via file
  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === 'string' ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GITHUB_AW_AGENT_OUTPUT = tempFilePath;
  };

  // Helper function to execute the script with proper globals
  const executeScript = async () => {
    // Set globals just before execution
    global.core = mockCore;
    global.context = mockContext;
    global.mockFs = mockFs;
    global.exec = mockExec;

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
    delete process.env.GITHUB_AW_PR_TITLE_PREFIX;
    delete process.env.GITHUB_AW_PR_LABELS;

    // Create fresh mock objects for each test
    mockFs = {
      existsSync: vi.fn(),
      readFileSync: vi.fn(),
    };

    // Create fresh mock for exec
    mockExec = {
      exec: vi.fn().mockResolvedValue(0), // For commands that don't read output
      getExecOutput: vi.fn().mockImplementation((command, args) => {
        // Handle the gh pr view command specifically
        if (command === "gh" && args && args[0] === "pr" && args[1] === "view") {
          // Check if this is the JSON query for PR details
          if (args.includes("--json") && args.includes("headRefName,title,labels")) {
            const prData = JSON.stringify({
              headRefName: "feature-branch",
              title: "Test PR Title",
              labels: ["bug", "enhancement"],
            });
            return Promise.resolve({
              exitCode: 0,
              stdout: prData + "\n",
              stderr: "",
            });
          }
        }
        // Handle git rev-parse HEAD
        if (command === "git" && args && args[0] === "rev-parse" && args[1] === "HEAD") {
          return Promise.resolve({
            exitCode: 0,
            stdout: "abc123def456\n",
            stderr: "",
          });
        }
        // For other commands, return empty success
        return Promise.resolve({
          exitCode: 0,
          stdout: "",
          stderr: "",
        });
      }),
    };

    // Reset mockCore calls
    mockCore.setFailed.mockReset();
    mockCore.setOutput.mockReset();
    mockCore.warning.mockReset();
    mockCore.error.mockReset();

    // Read the script content
    const scriptPath = path.join(process.cwd(), "push_to_pull_request_branch.cjs");
    pushToPrBranchScript = fs.readFileSync(scriptPath, "utf8");

    // Modify the script to inject our mocks and make core available
    pushToPrBranchScript = pushToPrBranchScript.replace(
      /\/\*\* @type \{typeof import\("fs"\)\} \*\/\nconst fs = require\("fs"\);/,
      `const core = global.core;
const context = global.context || {};
const fs = global.mockFs;
const exec = global.exec;`
    );
  });

  afterEach(() => {
    // Clean up temporary file
    if (tempFilePath && require("fs").existsSync(tempFilePath)) {
      require("fs").unlinkSync(tempFilePath);
      tempFilePath = undefined;
    }

    // Clean up globals safely
    if (typeof global !== "undefined") {
      delete global.core;
      delete global.context;
      delete global.mockFs;
      delete global.exec;
    }
  });

  describe("Script execution", () => {
    it("should skip when no agent output is provided", async () => {
      // Remove the output content environment variable
      delete process.env.GITHUB_AW_AGENT_OUTPUT;

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should skip when agent output is empty", async () => {
      setAgentOutput("");

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle missing patch file with default 'warn' behavior", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [{ type: "push_to_pull_request_branch", content: "test" }],
      });

      mockFs.existsSync.mockReturnValue(false);

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith("No patch file found - cannot push without changes");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should fail when patch file missing and if-no-changes is 'error'", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [{ type: "push_to_pull_request_branch", content: "test" }],
      });
      process.env.GITHUB_AW_PUSH_IF_NO_CHANGES = "error";

      mockFs.existsSync.mockReturnValue(false);

      // Execute the script
      await executeScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith("No patch file found - cannot push without changes");
    });

    it("should silently succeed when patch file missing and if-no-changes is 'ignore'", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [{ type: "push_to_pull_request_branch", content: "test" }],
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
        items: [{ type: "push_to_pull_request_branch", content: "test" }],
      });

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("Failed to generate patch: some error");

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith("Patch file contains error message - cannot push without changes");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle empty patch file with default 'warn' behavior", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [{ type: "push_to_pull_request_branch", content: "test" }],
      });

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("");

      // Mock the git command to return a branch name

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith("Patch file is empty - no changes to apply (noop operation)");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringMatching(/Agent output content length: \d+/));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should fail when empty patch and if-no-changes is 'error'", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [{ type: "push_to_pull_request_branch", content: "test" }],
      });
      process.env.GITHUB_AW_PUSH_IF_NO_CHANGES = "error";

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("   ");

      // Execute the script
      await executeScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith("No changes to push - failing as configured by if-no-changes: error");
    });

    it("should handle valid patch content and parse JSON output", async () => {
      const validOutput = {
        items: [
          {
            type: "push_to_pull_request_branch",
            content: "some changes to push",
          },
        ],
      };

      setAgentOutput(validOutput);

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("diff --git a/file.txt b/file.txt\n+new content");

      // Mock the git commands that will be called

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringMatching(/Agent output content length: \d+/));
      expect(mockCore.info).toHaveBeenCalledWith("Patch content validation passed");
      expect(mockCore.info).toHaveBeenCalledWith("Target configuration: triggering");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle invalid JSON in agent output", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = "invalid json content";

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("some patch content");

      // Execute the script
      await executeScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringMatching(/Error parsing agent output JSON:/));
    });

    it("should handle agent output without valid items array", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: "not an array",
      });

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("some patch content");

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith("No valid items found in agent output");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should use custom target configuration", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [{ type: "push_to_pull_request_branch", content: "test" }],
      });
      process.env.GITHUB_AW_PUSH_TARGET = "custom-target";

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("some patch content");

      // Mock the git commands

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(`Target configuration: ${"custom-target"}`);
    });
  });

  describe("Script validation", () => {
    it("should have valid JavaScript syntax", () => {
      const scriptPath = path.join(__dirname, "push_to_pull_request_branch.cjs");
      const scriptContent = fs.readFileSync(scriptPath, "utf8");

      // Basic syntax validation - should not contain obvious errors
      expect(scriptContent).toContain("async function main()");
      expect(scriptContent).toContain("core.setFailed");
      expect(scriptContent).toContain("/tmp/gh-aw/aw.patch");
      expect(scriptContent).toContain("await main()");
    });

    it("should export a main function", () => {
      const scriptPath = path.join(__dirname, "push_to_pull_request_branch.cjs");
      const scriptContent = fs.readFileSync(scriptPath, "utf8");

      // Check that the script has the expected structure
      expect(scriptContent).toMatch(/async function main\(\) \{[\s\S]*\}/);
    });

    it("should validate patch size within limit", async () => {
      const validOutput = {
        items: [
          {
            type: "push_to_pull_request_branch",
            content: "some changes to push",
          },
        ],
      };

      setAgentOutput(validOutput);
      process.env.GITHUB_AW_MAX_PATCH_SIZE = "10"; // 10 KB limit

      mockFs.existsSync.mockReturnValue(true);
      // Create patch content under 10 KB (approximately 5 KB)
      const patchContent = "diff --git a/file.txt b/file.txt\n+new content\n".repeat(100);
      mockFs.readFileSync.mockReturnValue(patchContent);

      // Mock the git commands that will be called

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 10 KB\)/));
      expect(mockCore.info).toHaveBeenCalledWith("Patch size validation passed");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should fail when patch size exceeds limit", async () => {
      const validOutput = {
        items: [
          {
            type: "push_to_pull_request_branch",
            content: "some changes to push",
          },
        ],
      };

      setAgentOutput(validOutput);
      process.env.GITHUB_AW_MAX_PATCH_SIZE = "1"; // 1 KB limit

      mockFs.existsSync.mockReturnValue(true);
      // Create patch content over 1 KB (approximately 5 KB)
      const patchContent = "diff --git a/file.txt b/file.txt\n+new content\n".repeat(100);
      mockFs.readFileSync.mockReturnValue(patchContent);

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 1 KB\)/));
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringMatching(/Patch size \(\d+ KB\) exceeds maximum allowed size \(1 KB\)/));
    });

    it("should use default 1024 KB limit when env var not set", async () => {
      const validOutput = {
        items: [
          {
            type: "push_to_pull_request_branch",
            content: "some changes to push",
          },
        ],
      };

      setAgentOutput(validOutput);
      delete process.env.GITHUB_AW_MAX_PATCH_SIZE; // No limit set

      mockFs.existsSync.mockReturnValue(true);
      const patchContent = "diff --git a/file.txt b/file.txt\n+new content\n";
      mockFs.readFileSync.mockReturnValue(patchContent);

      // Mock the git commands that will be called

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 1024 KB\)/));
      expect(mockCore.info).toHaveBeenCalledWith("Patch size validation passed");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should skip patch size validation for empty patches", async () => {
      const validOutput = {
        items: [
          {
            type: "push_to_pull_request_branch",
            content: "some changes to push",
          },
        ],
      };

      setAgentOutput(validOutput);
      process.env.GITHUB_AW_MAX_PATCH_SIZE = "1"; // 1 KB limit

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue(""); // Empty patch

      // Mock the git commands that will be called

      // Execute the script
      await executeScript();

      // Should not check patch size for empty patches
      expect(mockCore.info).not.toHaveBeenCalledWith(expect.stringMatching(/Patch size:/));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should validate PR title prefix when specified", async () => {
      const validOutput = {
        items: [
          {
            type: "push_to_pull_request_branch",
            content: "some changes to push",
          },
        ],
      };

      setAgentOutput(validOutput);
      process.env.GITHUB_AW_PR_TITLE_PREFIX = "[bot] ";

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("diff --git a/file.txt b/file.txt\n+new content");

      // Mock the gh pr view command to return PR data with matching title prefix
      mockExec.getExecOutput.mockImplementation((command, args) => {
        if (command === "gh" && args && args[0] === "pr" && args[1] === "view") {
          const prData = {
            headRefName: "feature-branch",
            title: "[bot] Add new feature",
            labels: [],
          };
          return Promise.resolve({
            exitCode: 0,
            stdout: JSON.stringify(prData),
            stderr: "",
          });
        }
        if (command === "git" && args && args[0] === "rev-parse" && args[1] === "HEAD") {
          return Promise.resolve({
            exitCode: 0,
            stdout: "abc123def456\n",
            stderr: "",
          });
        }
        return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
      });

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith('✓ Title prefix validation passed: "[bot] "');
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should fail when PR title doesn't match required prefix", async () => {
      const validOutput = {
        items: [
          {
            type: "push_to_pull_request_branch",
            content: "some changes to push",
          },
        ],
      };

      setAgentOutput(validOutput);
      process.env.GITHUB_AW_PR_TITLE_PREFIX = "[bot] ";

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("diff --git a/file.txt b/file.txt\n+new content");

      // Mock the gh pr view command to return PR data without matching title prefix
      mockExec.getExecOutput.mockImplementation((command, args) => {
        if (command === "gh" && args && args[0] === "pr" && args[1] === "view") {
          const prData = {
            headRefName: "feature-branch",
            title: "Add new feature",
            labels: [],
          };
          return Promise.resolve({
            exitCode: 0,
            stdout: JSON.stringify(prData),
            stderr: "",
          });
        }
        if (command === "git" && args && args[0] === "rev-parse" && args[1] === "HEAD") {
          return Promise.resolve({
            exitCode: 0,
            stdout: "abc123def456\n",
            stderr: "",
          });
        }
        return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
      });

      // Execute the script
      await executeScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith('Pull request title "Add new feature" does not start with required prefix "[bot] "');
    });

    it("should validate PR labels when specified", async () => {
      const validOutput = {
        items: [
          {
            type: "push_to_pull_request_branch",
            content: "some changes to push",
          },
        ],
      };

      setAgentOutput(validOutput);
      process.env.GITHUB_AW_PR_LABELS = "automation,enhancement";

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("diff --git a/file.txt b/file.txt\n+new content");

      // Mock the gh pr view command to return PR data with required labels
      mockExec.getExecOutput.mockImplementation((command, args) => {
        if (command === "gh" && args && args[0] === "pr" && args[1] === "view") {
          const prData = {
            headRefName: "feature-branch",
            title: "Add new feature",
            labels: ["automation", "enhancement", "feature"],
          };
          return Promise.resolve({
            exitCode: 0,
            stdout: JSON.stringify(prData),
            stderr: "",
          });
        }
        if (command === "git" && args && args[0] === "rev-parse" && args[1] === "HEAD") {
          return Promise.resolve({
            exitCode: 0,
            stdout: "abc123def456\n",
            stderr: "",
          });
        }
        return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
      });

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith("✓ Labels validation passed: automation,enhancement");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should fail when PR is missing required labels", async () => {
      const validOutput = {
        items: [
          {
            type: "push_to_pull_request_branch",
            content: "some changes to push",
          },
        ],
      };

      setAgentOutput(validOutput);
      process.env.GITHUB_AW_PR_LABELS = "automation,enhancement";

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("diff --git a/file.txt b/file.txt\n+new content");

      // Mock the gh pr view command to return PR data missing required labels
      mockExec.getExecOutput.mockImplementation((command, args) => {
        if (command === "gh" && args && args[0] === "pr" && args[1] === "view") {
          const prData = {
            headRefName: "feature-branch",
            title: "Add new feature",
            labels: ["feature"], // Missing "automation" and "enhancement"
          };
          return Promise.resolve({
            exitCode: 0,
            stdout: JSON.stringify(prData),
            stderr: "",
          });
        }
        if (command === "git" && args && args[0] === "rev-parse" && args[1] === "HEAD") {
          return Promise.resolve({
            exitCode: 0,
            stdout: "abc123def456\n",
            stderr: "",
          });
        }
        return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
      });

      // Execute the script
      await executeScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "Pull request is missing required labels: automation, enhancement. Current labels: feature"
      );
    });

    it("should validate both title prefix and labels when both are specified", async () => {
      const validOutput = {
        items: [
          {
            type: "push_to_pull_request_branch",
            content: "some changes to push",
          },
        ],
      };

      setAgentOutput(validOutput);
      process.env.GITHUB_AW_PR_TITLE_PREFIX = "[automated] ";
      process.env.GITHUB_AW_PR_LABELS = "bot,feature";

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readFileSync.mockReturnValue("diff --git a/file.txt b/file.txt\n+new content");

      // Mock the gh pr view command to return PR data with both valid title and labels
      mockExec.getExecOutput.mockImplementation((command, args) => {
        if (command === "gh" && args && args[0] === "pr" && args[1] === "view") {
          const prData = {
            headRefName: "feature-branch",
            title: "[automated] Add new feature",
            labels: ["bot", "feature", "enhancement"],
          };
          return Promise.resolve({
            exitCode: 0,
            stdout: JSON.stringify(prData),
            stderr: "",
          });
        }
        if (command === "git" && args && args[0] === "rev-parse" && args[1] === "HEAD") {
          return Promise.resolve({
            exitCode: 0,
            stdout: "abc123def456\n",
            stderr: "",
          });
        }
        return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
      });

      // Execute the script
      await executeScript();

      expect(mockCore.info).toHaveBeenCalledWith('✓ Title prefix validation passed: "[automated] "');
      expect(mockCore.info).toHaveBeenCalledWith("✓ Labels validation passed: bot,feature");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });
});
