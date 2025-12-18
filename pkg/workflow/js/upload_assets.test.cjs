import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
// Mock the global objects that GitHub Actions provides
const mockCore = { debug: vi.fn(), info: vi.fn(), notice: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn(), summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue(void 0) } };
// Set up global variables
((global.core = mockCore),
  describe("upload_assets.cjs", () => {
    let uploadAssetsScript, mockExec, tempFilePath;
    // Helper function to set agent output via file
    const setAgentOutput = data => {
        tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
        const content = "string" == typeof data ? data : JSON.stringify(data);
        (fs.writeFileSync(tempFilePath, content), (process.env.GH_AW_AGENT_OUTPUT = tempFilePath));
      },
      executeScript = async () => ((global.core = mockCore), (global.exec = mockExec), await eval(`(async () => { ${uploadAssetsScript} })()`));
    // Helper function to execute the script with proper globals
    // NOTE: Using eval() here is safe because the script content is read from a local file
    // and is never derived from user input. This pattern is used consistently across all
    // test files in this directory to test GitHub Actions scripts.
    (beforeEach(() => {
      // Reset all mocks
      (vi.clearAllMocks(),
        // Clear environment variables
        delete process.env.GH_AW_ASSETS_BRANCH,
        delete process.env.GH_AW_AGENT_OUTPUT,
        delete process.env.GH_AW_SAFE_OUTPUTS_STAGED);
      // Read the script content
      const scriptPath = path.join(__dirname, "upload_assets.cjs");
      ((uploadAssetsScript = fs.readFileSync(scriptPath, "utf8")),
        // Create fresh mock for exec
        (mockExec = { exec: vi.fn().mockResolvedValue(0) }));
    }),
      afterEach(() => {
        // Clean up temp files
        tempFilePath && fs.existsSync(tempFilePath) && fs.unlinkSync(tempFilePath);
      }),
      describe("git commit command - vulnerability fix", () => {
        it("should not wrap commit message in extra quotes to prevent command injection", async () => {
          (fs.existsSync("test.png") && fs.unlinkSync("test.png"),
            // Set up environment
            (process.env.GH_AW_ASSETS_BRANCH = "assets/test-workflow"),
            (process.env.GH_AW_SAFE_OUTPUTS_STAGED = "false"));
          // Create a temp directory for the asset
          const assetDir = "/tmp/gh-aw/safeoutputs/assets";
          fs.existsSync(assetDir) || fs.mkdirSync(assetDir, { recursive: !0 });
          // Create a temp file for the asset
          const assetPath = path.join(assetDir, "test.png");
          fs.writeFileSync(assetPath, "fake png data");
          // Calculate the SHA of the file
          const crypto = require("crypto"),
            fileContent = fs.readFileSync(assetPath),
            agentOutput = {
              items: [{ type: "upload_asset", fileName: "test.png", sha: crypto.createHash("sha256").update(fileContent).digest("hex"), size: fileContent.length, targetFileName: "test.png", url: "https://example.com/test.png" }],
            };
          setAgentOutput(agentOutput);
          // Mock git commands to succeed, but track calls
          let gitCheckoutCalled = !1;
          (mockExec.exec.mockImplementation(async (command, args) => {
            const fullCommand = Array.isArray(args) ? `${command} ${args.join(" ")}` : command;
            // Track if git checkout was called (indicates branch creation)
            // Mock git rev-parse to fail (branch doesn't exist yet)
            if ((fullCommand.includes("checkout") && (gitCheckoutCalled = !0), fullCommand.includes("rev-parse"))) throw new Error("Branch does not exist");
            return 0;
          }),
            // Execute the script
            await executeScript(),
            // Verify git checkout was called (sanity check)
            expect(gitCheckoutCalled).toBe(!0));
          // Find the git commit call
          const gitCommitCall = mockExec.exec.mock.calls.find(call => !!Array.isArray(call[1]) && "git" === call[0] && call[1].includes("commit"));
          if (
            // Verify git commit was called
            (expect(gitCommitCall).toBeDefined(), gitCommitCall)
          ) {
            const commitArgs = gitCommitCall[1],
              messageArgIndex = commitArgs.indexOf("-m"),
              commitMessage = commitArgs[messageArgIndex + 1];
            // SECURITY: Verify the commit message does NOT have extra quotes wrapping it
            // The message should be a plain string like: [skip-ci] Add 1 asset(s)
            // NOT wrapped in quotes like: "[skip-ci] Add 1 asset(s)"
            // Extra quotes can lead to command injection vulnerabilities
            (expect(commitMessage).toBeDefined(),
              expect(typeof commitMessage).toBe("string"),
              expect(commitMessage).not.toMatch(/^"/),
              expect(commitMessage).not.toMatch(/"$/),
              expect(commitMessage).toContain("[skip-ci]"),
              expect(commitMessage).toContain("asset(s)"));
          }
          // Cleanup
          (fs.existsSync(assetPath) && fs.unlinkSync(assetPath), fs.existsSync("test.png") && fs.unlinkSync("test.png"));
        });
      }),
      describe("normalizeBranchName function", () => {
        it("should normalize branch names correctly", async () => {
          // Set up environment with a branch name that needs normalization
          ((process.env.GH_AW_ASSETS_BRANCH = "assets/My Branch!@#$%"),
            (process.env.GH_AW_SAFE_OUTPUTS_STAGED = "false"),
            // Set up empty agent output
            setAgentOutput({ items: [] }),
            // Execute the script
            await executeScript());
          // Verify that setOutput was called with normalized branch name
          const branchNameCall = mockCore.setOutput.mock.calls.find(call => "branch_name" === call[0]);
          (expect(branchNameCall).toBeDefined(),
            // Should be normalized: forward slashes are allowed, but special chars replaced by dash
            // The slash in "assets/" is kept because it's a valid character for branch names
            expect(branchNameCall[1]).toBe("assets/my-branch"));
        });
      }),
      describe("branch prefix validation", () => {
        (it("should allow creating orphaned branch with 'assets/' prefix when branch doesn't exist", async () => {
          (fs.existsSync("test.png") && fs.unlinkSync("test.png"),
            // Set up environment with valid assets/ prefix
            (process.env.GH_AW_ASSETS_BRANCH = "assets/test-workflow"),
            (process.env.GH_AW_SAFE_OUTPUTS_STAGED = "false"));
          // Create a temp directory for the asset
          const assetDir = "/tmp/gh-aw/safeoutputs/assets";
          fs.existsSync(assetDir) || fs.mkdirSync(assetDir, { recursive: !0 });
          // Create a temp file for the asset
          const assetPath = path.join(assetDir, "test.png");
          fs.writeFileSync(assetPath, "fake png data");
          // Calculate the SHA of the file
          const crypto = require("crypto"),
            fileContent = fs.readFileSync(assetPath),
            agentOutput = {
              items: [{ type: "upload_asset", fileName: "test.png", sha: crypto.createHash("sha256").update(fileContent).digest("hex"), size: fileContent.length, targetFileName: "test.png", url: "https://example.com/test.png" }],
            };
          setAgentOutput(agentOutput);
          // Mock git commands to succeed
          let orphanBranchCreated = !1;
          (mockExec.exec.mockImplementation(async (command, args) => {
            const fullCommand = Array.isArray(args) ? `${command} ${args.join(" ")}` : command;
            // Track if orphan branch was created
            // Mock git rev-parse to fail (branch doesn't exist yet)
            if ((fullCommand.includes("checkout --orphan") && (orphanBranchCreated = !0), fullCommand.includes("rev-parse"))) throw new Error("Branch does not exist");
            return 0;
          }),
            // Execute the script
            await executeScript(),
            // Verify orphan branch was created
            expect(orphanBranchCreated).toBe(!0),
            // Verify no error was set
            expect(mockCore.setFailed).not.toHaveBeenCalled(),
            // Cleanup
            fs.existsSync(assetPath) && fs.unlinkSync(assetPath),
            fs.existsSync("test.png") && fs.unlinkSync("test.png"));
        }),
          it("should fail when trying to create orphaned branch without 'assets/' prefix", async () => {
            // Set up environment with non-assets prefix
            ((process.env.GH_AW_ASSETS_BRANCH = "custom/branch-name"), (process.env.GH_AW_SAFE_OUTPUTS_STAGED = "false"));
            // Create a temp directory for the asset
            const assetDir = "/tmp/gh-aw/safeoutputs/assets";
            fs.existsSync(assetDir) || fs.mkdirSync(assetDir, { recursive: !0 });
            // Create a temp file for the asset
            const assetPath = path.join(assetDir, "test.png");
            fs.writeFileSync(assetPath, "fake png data");
            // Calculate the SHA of the file
            const crypto = require("crypto"),
              fileContent = fs.readFileSync(assetPath),
              agentOutput = {
                items: [{ type: "upload_asset", fileName: "test.png", sha: crypto.createHash("sha256").update(fileContent).digest("hex"), size: fileContent.length, targetFileName: "test.png", url: "https://example.com/test.png" }],
              };
            setAgentOutput(agentOutput);
            // Mock git commands
            let orphanBranchCreated = !1;
            (mockExec.exec.mockImplementation(async (command, args) => {
              const fullCommand = Array.isArray(args) ? `${command} ${args.join(" ")}` : command;
              // Track if orphan branch was attempted
              // Mock git rev-parse to fail (branch doesn't exist)
              if ((fullCommand.includes("checkout --orphan") && (orphanBranchCreated = !0), fullCommand.includes("rev-parse"))) throw new Error("Branch does not exist");
              return 0;
            }),
              // Execute the script
              await executeScript(),
              // Verify orphan branch was NOT created
              expect(orphanBranchCreated).toBe(!1),
              // Verify error was set with appropriate message
              expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("does not start with the required 'assets/' prefix")),
              expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("custom/branch-name")),
              // Cleanup
              fs.existsSync(assetPath) && fs.unlinkSync(assetPath));
          }),
          it("should allow using existing branch regardless of prefix", async () => {
            (fs.existsSync("test.png") && fs.unlinkSync("test.png"),
              // Set up environment with non-assets prefix but existing branch
              (process.env.GH_AW_ASSETS_BRANCH = "custom/existing-branch"),
              (process.env.GH_AW_SAFE_OUTPUTS_STAGED = "false"));
            // Create a temp directory for the asset
            const assetDir = "/tmp/gh-aw/safeoutputs/assets";
            fs.existsSync(assetDir) || fs.mkdirSync(assetDir, { recursive: !0 });
            // Create a temp file for the asset
            const assetPath = path.join(assetDir, "test.png");
            fs.writeFileSync(assetPath, "fake png data");
            // Calculate the SHA of the file
            const crypto = require("crypto"),
              fileContent = fs.readFileSync(assetPath),
              agentOutput = {
                items: [{ type: "upload_asset", fileName: "test.png", sha: crypto.createHash("sha256").update(fileContent).digest("hex"), size: fileContent.length, targetFileName: "test.png", url: "https://example.com/test.png" }],
              };
            setAgentOutput(agentOutput);
            // Mock git commands to succeed (branch exists)
            let orphanBranchCreated = !1,
              existingBranchCheckedOut = !1;
            (mockExec.exec.mockImplementation(async (command, args) => {
              const fullCommand = Array.isArray(args) ? `${command} ${args.join(" ")}` : command;
              // Track if orphan branch was attempted
              // Mock git rev-parse to succeed (branch exists)
              return (
                fullCommand.includes("checkout --orphan") && (orphanBranchCreated = !0),
                // Track if existing branch was checked out
                fullCommand.includes("checkout -B") && (existingBranchCheckedOut = !0),
                fullCommand.includes("rev-parse"),
                0
              );
            }),
              // Execute the script
              await executeScript(),
              // Verify orphan branch was NOT created
              expect(orphanBranchCreated).toBe(!1),
              // Verify existing branch was checked out
              expect(existingBranchCheckedOut).toBe(!0),
              // Verify no error was set
              expect(mockCore.setFailed).not.toHaveBeenCalled(),
              // Cleanup
              fs.existsSync(assetPath) && fs.unlinkSync(assetPath),
              fs.existsSync("test.png") && fs.unlinkSync("test.png"));
          }));
      }));
  }));
