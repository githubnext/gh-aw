import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
const mockCore = { debug: vi.fn(), info: vi.fn(), notice: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn(), summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue(void 0) } };
((global.core = mockCore),
  describe("upload_assets.cjs", () => {
    let uploadAssetsScript, mockExec, tempFilePath;
    const setAgentOutput = data => {
        tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
        const content = "string" == typeof data ? data : JSON.stringify(data);
        (fs.writeFileSync(tempFilePath, content), (process.env.GH_AW_AGENT_OUTPUT = tempFilePath));
      },
      executeScript = async () => ((global.core = mockCore), (global.exec = mockExec), await eval(`(async () => { ${uploadAssetsScript}; await main(); })()`));
    (beforeEach(() => {
      (vi.clearAllMocks(), delete process.env.GH_AW_ASSETS_BRANCH, delete process.env.GH_AW_AGENT_OUTPUT, delete process.env.GH_AW_SAFE_OUTPUTS_STAGED);
      const scriptPath = path.join(__dirname, "upload_assets.cjs");
      ((uploadAssetsScript = fs.readFileSync(scriptPath, "utf8")), (mockExec = { exec: vi.fn().mockResolvedValue(0) }));
    }),
      afterEach(() => {
        tempFilePath && fs.existsSync(tempFilePath) && fs.unlinkSync(tempFilePath);
      }),
      describe("git commit command - vulnerability fix", () => {
        it("should not wrap commit message in extra quotes to prevent command injection", async () => {
          (fs.existsSync("test.png") && fs.unlinkSync("test.png"), (process.env.GH_AW_ASSETS_BRANCH = "assets/test-workflow"), (process.env.GH_AW_SAFE_OUTPUTS_STAGED = "false"));
          const assetDir = "/tmp/gh-aw/safeoutputs/assets";
          fs.existsSync(assetDir) || fs.mkdirSync(assetDir, { recursive: !0 });
          const assetPath = path.join(assetDir, "test.png");
          fs.writeFileSync(assetPath, "fake png data");
          const crypto = require("crypto"),
            fileContent = fs.readFileSync(assetPath),
            agentOutput = {
              items: [{ type: "upload_asset", fileName: "test.png", sha: crypto.createHash("sha256").update(fileContent).digest("hex"), size: fileContent.length, targetFileName: "test.png", url: "https://example.com/test.png" }],
            };
          setAgentOutput(agentOutput);
          let gitCheckoutCalled = !1;
          (mockExec.exec.mockImplementation(async (command, args) => {
            const fullCommand = Array.isArray(args) ? `${command} ${args.join(" ")}` : command;
            if ((fullCommand.includes("checkout") && (gitCheckoutCalled = !0), fullCommand.includes("rev-parse"))) throw new Error("Branch does not exist");
            return 0;
          }),
            await executeScript(),
            expect(gitCheckoutCalled).toBe(!0));
          const gitCommitCall = mockExec.exec.mock.calls.find(call => !!Array.isArray(call[1]) && "git" === call[0] && call[1].includes("commit"));
          if ((expect(gitCommitCall).toBeDefined(), gitCommitCall)) {
            const commitArgs = gitCommitCall[1],
              messageArgIndex = commitArgs.indexOf("-m"),
              commitMessage = commitArgs[messageArgIndex + 1];
            (expect(commitMessage).toBeDefined(),
              expect(typeof commitMessage).toBe("string"),
              expect(commitMessage).not.toMatch(/^"/),
              expect(commitMessage).not.toMatch(/"$/),
              expect(commitMessage).toContain("[skip-ci]"),
              expect(commitMessage).toContain("asset(s)"));
          }
          (fs.existsSync(assetPath) && fs.unlinkSync(assetPath), fs.existsSync("test.png") && fs.unlinkSync("test.png"));
        });
      }),
      describe("normalizeBranchName function", () => {
        it("should normalize branch names correctly", async () => {
          ((process.env.GH_AW_ASSETS_BRANCH = "assets/My Branch!@#$%"), (process.env.GH_AW_SAFE_OUTPUTS_STAGED = "false"), setAgentOutput({ items: [] }), await executeScript());
          const branchNameCall = mockCore.setOutput.mock.calls.find(call => "branch_name" === call[0]);
          (expect(branchNameCall).toBeDefined(), expect(branchNameCall[1]).toBe("assets/my-branch"));
        });
      }),
      describe("branch prefix validation", () => {
        (it("should allow creating orphaned branch with 'assets/' prefix when branch doesn't exist", async () => {
          (fs.existsSync("test.png") && fs.unlinkSync("test.png"), (process.env.GH_AW_ASSETS_BRANCH = "assets/test-workflow"), (process.env.GH_AW_SAFE_OUTPUTS_STAGED = "false"));
          const assetDir = "/tmp/gh-aw/safeoutputs/assets";
          fs.existsSync(assetDir) || fs.mkdirSync(assetDir, { recursive: !0 });
          const assetPath = path.join(assetDir, "test.png");
          fs.writeFileSync(assetPath, "fake png data");
          const crypto = require("crypto"),
            fileContent = fs.readFileSync(assetPath),
            agentOutput = {
              items: [{ type: "upload_asset", fileName: "test.png", sha: crypto.createHash("sha256").update(fileContent).digest("hex"), size: fileContent.length, targetFileName: "test.png", url: "https://example.com/test.png" }],
            };
          setAgentOutput(agentOutput);
          let orphanBranchCreated = !1;
          (mockExec.exec.mockImplementation(async (command, args) => {
            const fullCommand = Array.isArray(args) ? `${command} ${args.join(" ")}` : command;
            if ((fullCommand.includes("checkout --orphan") && (orphanBranchCreated = !0), fullCommand.includes("rev-parse"))) throw new Error("Branch does not exist");
            return 0;
          }),
            await executeScript(),
            expect(orphanBranchCreated).toBe(!0),
            expect(mockCore.setFailed).not.toHaveBeenCalled(),
            fs.existsSync(assetPath) && fs.unlinkSync(assetPath),
            fs.existsSync("test.png") && fs.unlinkSync("test.png"));
        }),
          it("should fail when trying to create orphaned branch without 'assets/' prefix", async () => {
            ((process.env.GH_AW_ASSETS_BRANCH = "custom/branch-name"), (process.env.GH_AW_SAFE_OUTPUTS_STAGED = "false"));
            const assetDir = "/tmp/gh-aw/safeoutputs/assets";
            fs.existsSync(assetDir) || fs.mkdirSync(assetDir, { recursive: !0 });
            const assetPath = path.join(assetDir, "test.png");
            fs.writeFileSync(assetPath, "fake png data");
            const crypto = require("crypto"),
              fileContent = fs.readFileSync(assetPath),
              agentOutput = {
                items: [{ type: "upload_asset", fileName: "test.png", sha: crypto.createHash("sha256").update(fileContent).digest("hex"), size: fileContent.length, targetFileName: "test.png", url: "https://example.com/test.png" }],
              };
            setAgentOutput(agentOutput);
            let orphanBranchCreated = !1;
            (mockExec.exec.mockImplementation(async (command, args) => {
              const fullCommand = Array.isArray(args) ? `${command} ${args.join(" ")}` : command;
              if ((fullCommand.includes("checkout --orphan") && (orphanBranchCreated = !0), fullCommand.includes("rev-parse"))) throw new Error("Branch does not exist");
              return 0;
            }),
              await executeScript(),
              expect(orphanBranchCreated).toBe(!1),
              expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("does not start with the required 'assets/' prefix")),
              expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("custom/branch-name")),
              fs.existsSync(assetPath) && fs.unlinkSync(assetPath));
          }),
          it("should allow using existing branch regardless of prefix", async () => {
            (fs.existsSync("test.png") && fs.unlinkSync("test.png"), (process.env.GH_AW_ASSETS_BRANCH = "custom/existing-branch"), (process.env.GH_AW_SAFE_OUTPUTS_STAGED = "false"));
            const assetDir = "/tmp/gh-aw/safeoutputs/assets";
            fs.existsSync(assetDir) || fs.mkdirSync(assetDir, { recursive: !0 });
            const assetPath = path.join(assetDir, "test.png");
            fs.writeFileSync(assetPath, "fake png data");
            const crypto = require("crypto"),
              fileContent = fs.readFileSync(assetPath),
              agentOutput = {
                items: [{ type: "upload_asset", fileName: "test.png", sha: crypto.createHash("sha256").update(fileContent).digest("hex"), size: fileContent.length, targetFileName: "test.png", url: "https://example.com/test.png" }],
              };
            setAgentOutput(agentOutput);
            let orphanBranchCreated = !1,
              existingBranchCheckedOut = !1;
            (mockExec.exec.mockImplementation(async (command, args) => {
              const fullCommand = Array.isArray(args) ? `${command} ${args.join(" ")}` : command;
              return (fullCommand.includes("checkout --orphan") && (orphanBranchCreated = !0), fullCommand.includes("checkout -B") && (existingBranchCheckedOut = !0), fullCommand.includes("rev-parse"), 0);
            }),
              await executeScript(),
              expect(orphanBranchCreated).toBe(!1),
              expect(existingBranchCheckedOut).toBe(!0),
              expect(mockCore.setFailed).not.toHaveBeenCalled(),
              fs.existsSync(assetPath) && fs.unlinkSync(assetPath),
              fs.existsSync("test.png") && fs.unlinkSync("test.png"));
          }));
      }));
  }));
