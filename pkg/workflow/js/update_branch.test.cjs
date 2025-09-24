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
  eventName: "workflow_dispatch",
  payload: {
    repository: { html_url: "https://github.com/testowner/testrepo" },
  },
  repo: { owner: "testowner", repo: "testrepo" },
};

// Set up global variables
global.core = mockCore;
global.context = mockContext;

describe("update_branch.cjs", () => {
  let updateBranchScript;
  let mockExec;

  // Helper function to execute the script with proper globals
  const executeScript = async () => {
    // Set globals just before execution
    global.core = mockCore;
    global.context = mockContext;
    global.exec = mockExec;

    // Execute the script
    return await eval(`(async () => { ${updateBranchScript} })()`);
  };

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Create fresh mock for exec
    mockExec = {
      exec: vi.fn().mockImplementation((command, args, options) => {
        // Handle git branch --show-current command
        if (command === "git" && args && args[0] === "branch" && args[1] === "--show-current") {
          if (options && options.listeners && options.listeners.stdout) {
            options.listeners.stdout(Buffer.from("feature-branch\n"));
          }
          return Promise.resolve(0);
        }

        // Handle gh pr view command
        if (command === "gh" && args && args[0] === "pr" && args[1] === "view") {
          if (options && options.listeners && options.listeners.stdout) {
            const prData = {
              base: "main",
              number: 123,
              title: "Test PR",
            };
            options.listeners.stdout(Buffer.from(JSON.stringify(prData) + "\n"));
          }
          return Promise.resolve(0);
        }

        // Handle git config commands
        if (command === "git" && args && args[0] === "config") {
          return Promise.resolve(0);
        }

        // Handle git fetch command
        if (command === "git" && args && args[0] === "fetch") {
          return Promise.resolve(0);
        }

        // Handle git checkout command
        if (command === "git" && args && args[0] === "checkout") {
          return Promise.resolve(0);
        }

        // Handle git merge command - default to success
        if (command === "git" && args && args[0] === "merge") {
          return Promise.resolve(0);
        }

        // Handle git status commands
        if (command === "git" && args && args[0] === "status") {
          if (args.includes("--porcelain")) {
            if (options && options.listeners && options.listeners.stdout) {
              // Mock no changes by default
              options.listeners.stdout(Buffer.from(""));
            }
          } else if (args.includes("-b")) {
            if (options && options.listeners && options.listeners.stdout) {
              // Mock branch status showing ahead
              options.listeners.stdout(Buffer.from("## feature-branch...origin/feature-branch [ahead 1]\n"));
            }
          }
          return Promise.resolve(0);
        }

        // Handle git push command
        if (command === "git" && args && args[0] === "push") {
          return Promise.resolve(0);
        }

        // Default success for other git commands
        return Promise.resolve(0);
      }),
    };

    // Read the actual script file
    const scriptPath = path.join(__dirname, "update_branch.cjs");
    updateBranchScript = fs.readFileSync(scriptPath, "utf8");

    // Replace the import/require pattern to work with our test environment
    updateBranchScript = updateBranchScript.replace('const exec = require("@actions/exec");', "const exec = global.exec;");
  });

  afterEach(() => {
    // Clean up globals safely
    if (typeof global !== "undefined") {
      delete global.core;
      delete global.context;
      delete global.exec;
    }
  });

  describe("Script execution", () => {
    it("should successfully update branch when merge is successful and changes exist", async () => {
      // Mock git status to show changes (ahead of origin)
      mockExec.exec.mockImplementation((command, args, options) => {
        if (command === "git" && args && args[0] === "branch" && args[1] === "--show-current") {
          if (options && options.listeners && options.listeners.stdout) {
            options.listeners.stdout(Buffer.from("feature-branch\n"));
          }
          return Promise.resolve(0);
        }

        if (command === "gh" && args && args[0] === "pr" && args[1] === "view") {
          if (options && options.listeners && options.listeners.stdout) {
            const prData = { base: "main", number: 123, title: "Test PR" };
            options.listeners.stdout(Buffer.from(JSON.stringify(prData) + "\n"));
          }
          return Promise.resolve(0);
        }

        if (command === "git" && args && args[0] === "status" && args.includes("-b")) {
          if (options && options.listeners && options.listeners.stdout) {
            options.listeners.stdout(Buffer.from("## feature-branch...origin/feature-branch [ahead 1]\n"));
          }
          return Promise.resolve(0);
        }

        return Promise.resolve(0);
      });

      await executeScript();

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.info).toHaveBeenCalledWith("Starting Update Branch workflow");
      expect(mockCore.info).toHaveBeenCalledWith("Current branch: feature-branch");
      expect(mockCore.info).toHaveBeenCalledWith("Pull Request #123: Test PR");
      expect(mockCore.info).toHaveBeenCalledWith("Base branch: main");
      expect(mockCore.info).toHaveBeenCalledWith("Successfully pushed updated branch");
      expect(mockCore.setOutput).toHaveBeenCalledWith("updated", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("branch", "feature-branch");
      expect(mockCore.setOutput).toHaveBeenCalledWith("base_branch", "main");
      expect(mockCore.setOutput).toHaveBeenCalledWith("pr_number", "123");
    });

    it("should handle case when branch is already up to date", async () => {
      // Mock git status to show no changes
      mockExec.exec.mockImplementation((command, args, options) => {
        if (command === "git" && args && args[0] === "branch" && args[1] === "--show-current") {
          if (options && options.listeners && options.listeners.stdout) {
            options.listeners.stdout(Buffer.from("feature-branch\n"));
          }
          return Promise.resolve(0);
        }

        if (command === "gh" && args && args[0] === "pr" && args[1] === "view") {
          if (options && options.listeners && options.listeners.stdout) {
            const prData = { base: "main", number: 123, title: "Test PR" };
            options.listeners.stdout(Buffer.from(JSON.stringify(prData) + "\n"));
          }
          return Promise.resolve(0);
        }

        if (command === "git" && args && args[0] === "status") {
          if (options && options.listeners && options.listeners.stdout) {
            options.listeners.stdout(Buffer.from(""));
          }
          return Promise.resolve(0);
        }

        return Promise.resolve(0);
      });

      await executeScript();

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.info).toHaveBeenCalledWith("No changes to push - branch is already up to date");
      expect(mockCore.setOutput).toHaveBeenCalledWith("updated", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("branch", "feature-branch");
    });

    it("should fail when no pull request is found for current branch", async () => {
      // Mock gh pr view to fail
      mockExec.exec.mockImplementation((command, args, options) => {
        if (command === "git" && args && args[0] === "branch" && args[1] === "--show-current") {
          if (options && options.listeners && options.listeners.stdout) {
            options.listeners.stdout(Buffer.from("feature-branch\n"));
          }
          return Promise.resolve(0);
        }

        if (command === "gh" && args && args[0] === "pr" && args[1] === "view") {
          return Promise.resolve(1); // Non-zero exit code indicates failure
        }

        return Promise.resolve(0);
      });

      await executeScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith("No pull request found for branch feature-branch");
    });

    it("should fail when merge conflicts occur", async () => {
      // Mock git merge to fail and git status to show conflicts
      mockExec.exec.mockImplementation((command, args, options) => {
        if (command === "git" && args && args[0] === "branch" && args[1] === "--show-current") {
          if (options && options.listeners && options.listeners.stdout) {
            options.listeners.stdout(Buffer.from("feature-branch\n"));
          }
          return Promise.resolve(0);
        }

        if (command === "gh" && args && args[0] === "pr" && args[1] === "view") {
          if (options && options.listeners && options.listeners.stdout) {
            const prData = { base: "main", number: 123, title: "Test PR" };
            options.listeners.stdout(Buffer.from(JSON.stringify(prData) + "\n"));
          }
          return Promise.resolve(0);
        }

        if (command === "git" && args && args[0] === "merge") {
          return Promise.resolve(1); // Merge failed
        }

        if (command === "git" && args && args[0] === "status" && args.includes("--porcelain")) {
          if (options && options.listeners && options.listeners.stdout) {
            options.listeners.stdout(Buffer.from("UU conflicted-file.txt\n"));
          }
          return Promise.resolve(0);
        }

        return Promise.resolve(0);
      });

      await executeScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "Merge conflict detected when merging main into feature-branch. Manual resolution required."
      );
    });

    it("should fail when git branch --show-current fails", async () => {
      // Mock git branch command to fail
      mockExec.exec.mockImplementation((command, args, options) => {
        if (command === "git" && args && args[0] === "branch" && args[1] === "--show-current") {
          return Promise.resolve(1); // Non-zero exit code
        }
        return Promise.resolve(0);
      });

      await executeScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to get current branch name");
    });

    it("should fail when push fails", async () => {
      // Mock git push to fail
      mockExec.exec.mockImplementation((command, args, options) => {
        if (command === "git" && args && args[0] === "branch" && args[1] === "--show-current") {
          if (options && options.listeners && options.listeners.stdout) {
            options.listeners.stdout(Buffer.from("feature-branch\n"));
          }
          return Promise.resolve(0);
        }

        if (command === "gh" && args && args[0] === "pr" && args[1] === "view") {
          if (options && options.listeners && options.listeners.stdout) {
            const prData = { base: "main", number: 123, title: "Test PR" };
            options.listeners.stdout(Buffer.from(JSON.stringify(prData) + "\n"));
          }
          return Promise.resolve(0);
        }

        if (command === "git" && args && args[0] === "status" && args.includes("-b")) {
          if (options && options.listeners && options.listeners.stdout) {
            options.listeners.stdout(Buffer.from("## feature-branch...origin/feature-branch [ahead 1]\n"));
          }
          return Promise.resolve(0);
        }

        if (command === "git" && args && args[0] === "push") {
          return Promise.resolve(1); // Push failed
        }

        return Promise.resolve(0);
      });

      await executeScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to push changes to feature-branch");
    });
  });
});
