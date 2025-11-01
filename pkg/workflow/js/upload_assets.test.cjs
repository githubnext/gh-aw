import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(undefined),
  },
};

// Set up global variables
global.core = mockCore;

describe("upload_assets.cjs", () => {
  let uploadAssetsScript;
  let mockExec;
  let tempFilePath;

  // Helper function to set agent output via file
  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GH_AW_AGENT_OUTPUT = tempFilePath;
  };

  // Helper function to execute the script with proper globals
  // NOTE: Using eval() here is safe because the script content is read from a local file
  // and is never derived from user input. This pattern is used consistently across all
  // test files in this directory to test GitHub Actions scripts.
  const executeScript = async () => {
    global.core = mockCore;
    global.exec = mockExec;
    return await eval(`(async () => { ${uploadAssetsScript} })()`);
  };

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Clear environment variables
    delete process.env.GH_AW_ASSETS_BRANCH;
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;

    // Read the script content
    const scriptPath = path.join(__dirname, "upload_assets.cjs");
    uploadAssetsScript = fs.readFileSync(scriptPath, "utf8");

    // Create fresh mock for exec
    mockExec = {
      exec: vi.fn().mockResolvedValue(0),
    };
  });

  afterEach(() => {
    // Clean up temp files
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
    }
  });

  describe("git commit command - vulnerability fix", () => {
    it("should not wrap commit message in extra quotes to prevent command injection", async () => {
      // Set up environment
      process.env.GH_AW_ASSETS_BRANCH = "assets/test-workflow";
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "false";

      // Create a temp directory for the asset
      const assetDir = "/tmp/gh-aw/safeoutputs/assets";
      if (!fs.existsSync(assetDir)) {
        fs.mkdirSync(assetDir, { recursive: true });
      }

      // Create a temp file for the asset
      const assetPath = path.join(assetDir, "test.png");
      fs.writeFileSync(assetPath, "fake png data");

      // Calculate the SHA of the file
      const crypto = require("crypto");
      const fileContent = fs.readFileSync(assetPath);
      const actualSha = crypto.createHash("sha256").update(fileContent).digest("hex");

      // Set up agent output with a valid upload-asset item
      const agentOutput = {
        items: [
          {
            type: "upload_asset",
            fileName: "test.png",
            sha: actualSha,
            size: fileContent.length,
            targetFileName: "test.png",
            url: "https://example.com/test.png",
          },
        ],
      };
      setAgentOutput(agentOutput);

      // Mock git commands to succeed, but track calls
      let gitCheckoutCalled = false;
      mockExec.exec.mockImplementation(async (command, args) => {
        const fullCommand = Array.isArray(args) ? `${command} ${args.join(" ")}` : command;

        // Track if git checkout was called (indicates branch creation)
        if (fullCommand.includes("checkout")) {
          gitCheckoutCalled = true;
        }

        // Mock git rev-parse to fail (branch doesn't exist yet)
        if (fullCommand.includes("rev-parse")) {
          throw new Error("Branch does not exist");
        }

        return 0;
      });

      // Execute the script
      await executeScript();

      // Verify git checkout was called (sanity check)
      expect(gitCheckoutCalled).toBe(true);

      // Find the git commit call
      const allCalls = mockExec.exec.mock.calls;
      const gitCommitCall = allCalls.find(call => {
        if (Array.isArray(call[1])) {
          return call[0] === "git" && call[1].includes("commit");
        }
        return false;
      });

      // Verify git commit was called
      expect(gitCommitCall).toBeDefined();

      if (gitCommitCall) {
        const commitArgs = gitCommitCall[1];
        const messageArgIndex = commitArgs.indexOf("-m");
        const commitMessage = commitArgs[messageArgIndex + 1];

        // SECURITY: Verify the commit message does NOT have extra quotes wrapping it
        // The message should be a plain string like: [skip-ci] Add 1 asset(s)
        // NOT wrapped in quotes like: "[skip-ci] Add 1 asset(s)"
        // Extra quotes can lead to command injection vulnerabilities
        expect(commitMessage).toBeDefined();
        expect(typeof commitMessage).toBe("string");
        expect(commitMessage).not.toMatch(/^"/);
        expect(commitMessage).not.toMatch(/"$/);
        expect(commitMessage).toContain("[skip-ci]");
        expect(commitMessage).toContain("asset(s)");
      }

      // Cleanup
      if (fs.existsSync(assetPath)) {
        fs.unlinkSync(assetPath);
      }
    });
  });

  describe("normalizeBranchName function", () => {
    it("should normalize branch names correctly", async () => {
      // Set up environment with a branch name that needs normalization
      process.env.GH_AW_ASSETS_BRANCH = "assets/My Branch!@#$%";
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "false";

      // Set up empty agent output
      setAgentOutput({ items: [] });

      // Execute the script
      await executeScript();

      // Verify that setOutput was called with normalized branch name
      const outputCalls = mockCore.setOutput.mock.calls;
      const branchNameCall = outputCalls.find(call => call[0] === "branch_name");

      expect(branchNameCall).toBeDefined();
      // Should be normalized: forward slashes are allowed, but special chars replaced by dash
      // The slash in "assets/" is kept because it's a valid character for branch names
      expect(branchNameCall[1]).toBe("assets/my-branch");
    });
  });
});
