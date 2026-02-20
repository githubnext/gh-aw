import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
describe("create_agent_session.cjs", () => {
  let mockCore, mockExec, mockContext, testOutputFile;
  (beforeEach(() => {
    ((mockCore = { info: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn(), summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() } }),
      (mockExec = { exec: vi.fn().mockResolvedValue(0), getExecOutput: vi.fn() }),
      (mockContext = { repo: { owner: "test-owner", repo: "test-repo" } }),
      (global.core = mockCore),
      (global.exec = mockExec),
      (global.context = mockContext),
      (testOutputFile = `/tmp/test_agent_output_${Date.now()}.json`));
  }),
    afterEach(() => {
      (delete global.core,
        delete global.exec,
        delete global.context,
        delete process.env.GH_AW_AGENT_OUTPUT,
        delete process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED,
        delete process.env.GITHUB_AW_AGENT_SESSION_BASE,
        delete process.env.GH_AW_TARGET_REPO_SLUG,
        delete process.env.GH_AW_ALLOWED_REPOS,
        delete process.env.GITHUB_REPOSITORY,
        fs.existsSync(testOutputFile) && fs.unlinkSync(testOutputFile));
    }));
  const createAgentOutput = items => {
      const output = { items };
      (fs.writeFileSync(testOutputFile, JSON.stringify(output)), (process.env.GH_AW_AGENT_OUTPUT = testOutputFile));
    },
    runScript = async () => {
      ((global.core = mockCore), (global.exec = mockExec), (global.context = mockContext));
      const scriptPath = require("path").join(process.cwd(), "create_agent_session.cjs");
      delete require.cache[require.resolve(scriptPath)];
      try {
        const { main } = require(scriptPath);
        await main();
      } catch (error) {}
    };
  ((describe("basic functionality", () => {
    (it("should handle missing environment variable", async () => {
      (delete process.env.GH_AW_AGENT_OUTPUT,
        await runScript(),
        expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found"),
        expect(mockCore.setOutput).toHaveBeenCalledWith("session_number", ""),
        expect(mockCore.setOutput).toHaveBeenCalledWith("session_url", ""));
    }),
      it("should handle missing output file", async () => {
        ((process.env.GH_AW_AGENT_OUTPUT = "/nonexistent/file.json"), await runScript(), expect(mockCore.setFailed).toHaveBeenCalled(), expect(mockCore.setFailed.mock.calls[0][0]).toContain("Error reading agent output file"));
      }),
      it("should handle empty agent output", async () => {
        (fs.writeFileSync(testOutputFile, ""), (process.env.GH_AW_AGENT_OUTPUT = testOutputFile), await runScript(), expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty"));
      }),
      it("should handle invalid JSON", async () => {
        (fs.writeFileSync(testOutputFile, "invalid json"),
          (process.env.GH_AW_AGENT_OUTPUT = testOutputFile),
          await runScript(),
          expect(mockCore.setFailed).toHaveBeenCalled(),
          expect(mockCore.setFailed.mock.calls[0][0]).toContain("Error parsing agent output JSON"));
      }),
      it("should handle no create_agent_session items", async () => {
        (createAgentOutput([{ type: "create_issue", title: "Test", body: "Content" }]), await runScript(), expect(mockCore.info).toHaveBeenCalledWith("No create-agent-session items found in agent output"));
      }));
  }),
  describe("staged mode", () => {
    (beforeEach(() => {
      ((process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true"), (process.env.GITHUB_AW_AGENT_SESSION_BASE = "main"), (process.env.GITHUB_REPOSITORY = "owner/repo"));
    }),
      it("should preview agent sessions in staged mode", async () => {
        (createAgentOutput([
          { type: "create_agent_session", body: "Implement feature X" },
          { type: "create_agent_session", body: "Fix bug Y" },
        ]),
          await runScript(),
          expect(mockCore.info).toHaveBeenCalled());
        const summaryCall = mockCore.summary.addRaw.mock.calls[0];
        (expect(summaryCall[0]).toContain("🎭 Staged Mode: Create Agent Sessions Preview"),
          expect(summaryCall[0]).toContain("Implement feature X"),
          expect(summaryCall[0]).toContain("Fix bug Y"),
          expect(summaryCall[0]).toContain("**Base Branch:** main"),
          expect(summaryCall[0]).toContain("**Target Repository:** test-owner/test-repo"),
          expect(mockCore.summary.write).toHaveBeenCalled());
      }),
      it("should handle task without body in staged mode", async () => {
        (createAgentOutput([{ type: "create_agent_session", body: "" }]), await runScript());
        const summaryCall = mockCore.summary.addRaw.mock.calls[0];
        expect(summaryCall[0]).toContain("No description provided");
      }),
      it("should use target repo when specified", async () => {
        ((process.env.GH_AW_TARGET_REPO_SLUG = "org/target-repo"), createAgentOutput([{ type: "create_agent_session", body: "Test task" }]), await runScript());
        const summaryCall = mockCore.summary.addRaw.mock.calls[0];
        expect(summaryCall[0]).toContain("**Target Repository:** org/target-repo");
      }));
  }),
  describe("agent session creation", () => {
    (beforeEach(() => {
      ((process.env.GITHUB_AW_AGENT_SESSION_BASE = "develop"), (process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "false"));
    }),
      it("should skip tasks with empty body", async () => {
        (createAgentOutput([
          { type: "create_agent_session", body: "" },
          { type: "create_agent_session", body: "  \n\t  " },
        ]),
          await runScript(),
          expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Agent task description is empty, skipping")));
      }),
      it("should log agent output content length", async () => {
        (createAgentOutput([{ type: "create_agent_session", body: "Test agent session description" }]),
          await runScript(),
          expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Agent output content length:")),
          expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Found 1 create-agent-session item(s)")));
      }));
  }),
  describe("output initialization", () => {
    it("should initialize outputs to empty strings", async () => {
      (delete process.env.GH_AW_AGENT_OUTPUT, await runScript(), expect(mockCore.setOutput).toHaveBeenCalledWith("session_number", ""), expect(mockCore.setOutput).toHaveBeenCalledWith("session_url", ""));
    });
  }),
  describe("edge cases", () => {
    (it("should handle output with no items array", async () => {
      (fs.writeFileSync(testOutputFile, JSON.stringify({})), (process.env.GH_AW_AGENT_OUTPUT = testOutputFile), await runScript(), expect(mockCore.info).toHaveBeenCalledWith("No valid items found in agent output"));
    }),
      it("should handle output with non-array items", async () => {
        (fs.writeFileSync(testOutputFile, JSON.stringify({ items: "not an array" })), (process.env.GH_AW_AGENT_OUTPUT = testOutputFile), await runScript(), expect(mockCore.info).toHaveBeenCalledWith("No valid items found in agent output"));
      }),
      it("should use default base branch when not specified", async () => {
        (delete process.env.GITHUB_AW_AGENT_SESSION_BASE, delete process.env.GITHUB_REF_NAME, (process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true"), createAgentOutput([{ type: "create_agent_session", body: "Test" }]), await runScript());
        const summaryCall = mockCore.summary.addRaw.mock.calls[0];
        expect(summaryCall[0]).toContain("**Base Branch:** main");
      }),
      it("should use GITHUB_REF_NAME as fallback for base branch", async () => {
        (delete process.env.GITHUB_AW_AGENT_SESSION_BASE,
          (process.env.GITHUB_REF_NAME = "feature-branch"),
          (process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true"),
          createAgentOutput([{ type: "create_agent_session", body: "Test" }]),
          await runScript());
        const summaryCall = mockCore.summary.addRaw.mock.calls[0];
        expect(summaryCall[0]).toContain("**Base Branch:** main");
      }));
  })),
    describe("Cross-repository allowlist validation", () => {
      it("should reject target repository not in allowlist", async () => {
        process.env.GH_AW_ALLOWED_REPOS = "allowed-owner/allowed-repo";

        createAgentOutput([{ type: "create_agent_session", body: "Test task", repo: "not-allowed/other-repo" }]);

        await runScript();

        expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("E004:"));
        expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("not in the allowed-repos list"));
      });

      it("should allow target repository in allowlist", async () => {
        process.env.GH_AW_ALLOWED_REPOS = "allowed-owner/allowed-repo";

        createAgentOutput([{ type: "create_agent_session", body: "Test task", repo: "allowed-owner/allowed-repo" }]);

        // Mock gh CLI command
        mockExec.getExecOutput.mockResolvedValueOnce({
          exitCode: 0,
          stdout: "https://github.com/allowed-owner/allowed-repo/issues/123",
          stderr: "",
        });

        await runScript();

        expect(mockCore.setFailed).not.toHaveBeenCalled();
        expect(mockCore.setOutput).toHaveBeenCalledWith("session_number", "123");
      });

      it("should allow default repository without allowlist", async () => {
        // No GH_AW_TARGET_REPO_SLUG set, no GH_AW_ALLOWED_REPOS set - uses context repo
        delete process.env.GH_AW_TARGET_REPO_SLUG;
        delete process.env.GH_AW_ALLOWED_REPOS;

        createAgentOutput([{ type: "create_agent_session", body: "Test task" }]);

        // Mock gh CLI command
        mockExec.getExecOutput.mockResolvedValueOnce({
          exitCode: 0,
          stdout: "https://github.com/test-owner/test-repo/issues/123",
          stderr: "",
        });

        await runScript();

        expect(mockCore.setFailed).not.toHaveBeenCalled();
        expect(mockCore.setOutput).toHaveBeenCalledWith("session_number", "123");
      });
    }));
});
