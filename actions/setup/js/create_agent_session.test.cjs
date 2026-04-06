import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockExec = {
  exec: vi.fn().mockResolvedValue(0),
  getExecOutput: vi.fn(),
};

const mockContext = {
  repo: { owner: "test-owner", repo: "test-repo" },
};

global.core = mockCore;
global.exec = mockExec;
global.context = mockContext;

describe("create_agent_session.cjs", () => {
  let createAgentSessionModule;

  beforeEach(() => {
    vi.clearAllMocks();

    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_TARGET_REPO_SLUG;
    delete process.env.GH_AW_ALLOWED_REPOS;
    delete process.env.GH_AW_AGENT_SESSION_TOKEN;
    delete process.env.GITHUB_TOKEN;

    // Clear module cache to get a fresh module for each test
    const scriptPath = path.join(process.cwd(), "create_agent_session.cjs");
    delete require.cache[require.resolve(scriptPath)];
    createAgentSessionModule = require(scriptPath);
  });

  afterEach(() => {
    // Clean up tmp files
    try {
      const files = fs.readdirSync("/tmp/gh-aw").filter(f => f.startsWith("agent-task-description-"));
      for (const file of files) {
        fs.unlinkSync(path.join("/tmp/gh-aw", file));
      }
    } catch {}
  });

  describe("handler factory", () => {
    it("should return a function when main() is called", async () => {
      const handler = await createAgentSessionModule.main({});
      expect(typeof handler).toBe("function");
    });

    it("should log configuration on initialization", async () => {
      await createAgentSessionModule.main({ base: "develop" });
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Configured base branch: develop"));
    });

    it("should log target repo on initialization", async () => {
      await createAgentSessionModule.main({ "target-repo": "owner/repo" });
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Default target repo: owner/repo"));
    });
  });

  describe("message handler - empty/invalid body", () => {
    it("should skip messages with empty body", async () => {
      const handler = await createAgentSessionModule.main({});
      const result = await handler({ type: "create_agent_session", body: "" });
      expect(result.success).toBe(false);
      expect(result.error).toContain("Empty task description");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Agent task description is empty, skipping"));
    });

    it("should skip messages with whitespace-only body", async () => {
      const handler = await createAgentSessionModule.main({});
      const result = await handler({ type: "create_agent_session", body: "  \n\t  " });
      expect(result.success).toBe(false);
    });
  });

  describe("staged mode", () => {
    it("should generate staged preview and return skipped without calling gh CLI", async () => {
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
      const handler = await createAgentSessionModule.main({ base: "main" });
      const result = await handler({ type: "create_agent_session", body: "Implement feature X" });
      expect(result.success).toBe(true);
      expect(result.skipped).toBe(true);
      expect(mockExec.getExecOutput).not.toHaveBeenCalled();
      // Should have written a staged preview summary
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("🎭 Staged Mode: Create Agent Session Preview"));
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Implement feature X"));
    });

    it("should include base branch and target repo in staged preview", async () => {
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
      const handler = await createAgentSessionModule.main({ base: "develop" });
      await handler({ type: "create_agent_session", body: "Test task" });
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("develop"));
    });

    it("should support staged mode via config flag", async () => {
      const handler = await createAgentSessionModule.main({ staged: true, base: "main" });
      const result = await handler({ type: "create_agent_session", body: "Test task" });
      expect(result.success).toBe(true);
      expect(result.skipped).toBe(true);
      expect(mockExec.getExecOutput).not.toHaveBeenCalled();
    });
  });

  describe("successful session creation", () => {
    it("should create agent session and extract task number from URL", async () => {
      mockExec.getExecOutput.mockResolvedValueOnce({
        exitCode: 0,
        stdout: "https://github.com/test-owner/test-repo/issues/123",
        stderr: "",
      });

      const handler = await createAgentSessionModule.main({ base: "main" });
      const result = await handler({ type: "create_agent_session", body: "Implement feature X" });

      expect(result.success).toBe(true);
      expect(result.number).toBe("123");
      expect(result.url).toBe("https://github.com/test-owner/test-repo/issues/123");
    });

    it("should use configured base branch when calling gh CLI", async () => {
      mockExec.getExecOutput.mockResolvedValueOnce({
        exitCode: 0,
        stdout: "https://github.com/test-owner/test-repo/issues/42",
        stderr: "",
      });

      const handler = await createAgentSessionModule.main({ base: "develop" });
      await handler({ type: "create_agent_session", body: "Test task" });

      expect(mockExec.getExecOutput).toHaveBeenCalledWith("gh", expect.arrayContaining(["--base", "develop"]), expect.any(Object));
    });

    it("should add --repo flag for cross-repo sessions", async () => {
      process.env.GH_AW_ALLOWED_REPOS = "other-owner/other-repo";
      mockExec.getExecOutput.mockResolvedValueOnce({
        exitCode: 0,
        stdout: "https://github.com/other-owner/other-repo/issues/99",
        stderr: "",
      });

      const handler = await createAgentSessionModule.main({ "target-repo": "other-owner/other-repo", base: "main" });
      await handler({ type: "create_agent_session", body: "Cross-repo task", repo: "other-owner/other-repo" });

      expect(mockExec.getExecOutput).toHaveBeenCalledWith("gh", expect.arrayContaining(["--repo", "other-owner/other-repo"]), expect.any(Object));
    });

    it("should use GH_AW_AGENT_SESSION_TOKEN as GH_TOKEN for gh CLI", async () => {
      process.env.GH_AW_AGENT_SESSION_TOKEN = "test-pat-token";
      mockExec.getExecOutput.mockResolvedValueOnce({
        exitCode: 0,
        stdout: "https://github.com/test-owner/test-repo/issues/55",
        stderr: "",
      });

      const handler = await createAgentSessionModule.main({ base: "main" });
      await handler({ type: "create_agent_session", body: "Test task" });

      expect(mockExec.getExecOutput).toHaveBeenCalledWith(
        "gh",
        expect.any(Array),
        expect.objectContaining({
          env: expect.objectContaining({ GH_TOKEN: "test-pat-token" }),
        })
      );
    });

    it("should prefer per-handler github-token over GH_AW_AGENT_SESSION_TOKEN", async () => {
      process.env.GH_AW_AGENT_SESSION_TOKEN = "step-token";
      mockExec.getExecOutput.mockResolvedValueOnce({
        exitCode: 0,
        stdout: "https://github.com/test-owner/test-repo/issues/55",
        stderr: "",
      });

      const handler = await createAgentSessionModule.main({ base: "main", "github-token": "per-handler-token" });
      await handler({ type: "create_agent_session", body: "Test task" });

      expect(mockExec.getExecOutput).toHaveBeenCalledWith(
        "gh",
        expect.any(Array),
        expect.objectContaining({
          env: expect.objectContaining({ GH_TOKEN: "per-handler-token" }),
        })
      );
    });
  });

  describe("error handling", () => {
    it("should return failure when gh CLI fails with auth error", async () => {
      mockExec.getExecOutput.mockRejectedValueOnce(new Error("permission denied (403)"));

      const handler = await createAgentSessionModule.main({ base: "main" });
      const result = await handler({ type: "create_agent_session", body: "Test task" });

      expect(result.success).toBe(false);
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("authentication/permission error"));
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("GH_AW_AGENT_SESSION_TOKEN"));
    });

    it("should return failure when gh CLI fails with generic error", async () => {
      mockExec.getExecOutput.mockRejectedValueOnce(new Error("gh: command not found"));

      const handler = await createAgentSessionModule.main({ base: "main" });
      const result = await handler({ type: "create_agent_session", body: "Test task" });

      expect(result.success).toBe(false);
      expect(mockCore.error).toHaveBeenCalled();
    });

    it("should warn when task number cannot be parsed from output", async () => {
      mockExec.getExecOutput.mockResolvedValueOnce({
        exitCode: 0,
        stdout: "Task created successfully",
        stderr: "",
      });

      const handler = await createAgentSessionModule.main({ base: "main" });
      const result = await handler({ type: "create_agent_session", body: "Test task" });

      expect(result.success).toBe(true);
      expect(result.number).toBe("");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not parse task number"));
    });

    it("should reject repositories not in allowlist", async () => {
      process.env.GH_AW_ALLOWED_REPOS = "allowed-owner/allowed-repo";

      const handler = await createAgentSessionModule.main({ "target-repo": "other-owner/other-repo", base: "main" });
      const result = await handler({ type: "create_agent_session", body: "Test task", repo: "not-allowed/other-repo" });

      expect(result.success).toBe(false);
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("E004:"));
    });
  });

  describe("module-level getters", () => {
    it("getCreateAgentSessionNumber() returns first successful session number", async () => {
      mockExec.getExecOutput.mockResolvedValue({
        exitCode: 0,
        stdout: "https://github.com/test-owner/test-repo/issues/42",
        stderr: "",
      });

      const handler = await createAgentSessionModule.main({ base: "main" });
      await handler({ type: "create_agent_session", body: "Task 1" });

      expect(createAgentSessionModule.getCreateAgentSessionNumber()).toBe("42");
    });

    it("getCreateAgentSessionUrl() returns first successful session URL", async () => {
      const expectedUrl = "https://github.com/test-owner/test-repo/issues/42";
      mockExec.getExecOutput.mockResolvedValue({
        exitCode: 0,
        stdout: expectedUrl,
        stderr: "",
      });

      const handler = await createAgentSessionModule.main({ base: "main" });
      await handler({ type: "create_agent_session", body: "Task 1" });

      expect(createAgentSessionModule.getCreateAgentSessionUrl()).toBe(expectedUrl);
    });

    it("getCreateAgentSessionNumber() returns empty string when no sessions created", async () => {
      await createAgentSessionModule.main({ base: "main" });
      expect(createAgentSessionModule.getCreateAgentSessionNumber()).toBe("");
    });

    it("getCreateAgentSessionUrl() returns empty string when no sessions created", async () => {
      await createAgentSessionModule.main({ base: "main" });
      expect(createAgentSessionModule.getCreateAgentSessionUrl()).toBe("");
    });
  });

  describe("writeCreateAgentSessionSummary()", () => {
    it("should write summary with successful sessions", async () => {
      mockExec.getExecOutput.mockResolvedValue({
        exitCode: 0,
        stdout: "https://github.com/test-owner/test-repo/issues/42",
        stderr: "",
      });

      const handler = await createAgentSessionModule.main({ base: "main" });
      await handler({ type: "create_agent_session", body: "Task 1" });
      await createAgentSessionModule.writeCreateAgentSessionSummary();

      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Agent Sessions"));
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("42"));
    });

    it("should not write summary when no results", async () => {
      await createAgentSessionModule.main({ base: "main" });
      await createAgentSessionModule.writeCreateAgentSessionSummary();

      expect(mockCore.summary.addRaw).not.toHaveBeenCalled();
    });

    it("should write summary with failed sessions", async () => {
      mockExec.getExecOutput.mockRejectedValueOnce(new Error("some error"));

      const handler = await createAgentSessionModule.main({ base: "main" });
      await handler({ type: "create_agent_session", body: "Task 1" });
      await createAgentSessionModule.writeCreateAgentSessionSummary();

      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("❌ Failed"));
    });
  });

  describe("cross-repository allowlist validation", () => {
    it("should allow target repository in allowlist", async () => {
      process.env.GH_AW_ALLOWED_REPOS = "allowed-owner/allowed-repo";
      mockExec.getExecOutput.mockResolvedValueOnce({
        exitCode: 0,
        stdout: "https://github.com/allowed-owner/allowed-repo/issues/123",
        stderr: "",
      });

      const handler = await createAgentSessionModule.main({ "target-repo": "allowed-owner/allowed-repo", base: "main" });
      const result = await handler({ type: "create_agent_session", body: "Test task", repo: "allowed-owner/allowed-repo" });

      expect(result.success).toBe(true);
      expect(result.number).toBe("123");
    });

    it("should allow default repository without allowlist", async () => {
      delete process.env.GH_AW_TARGET_REPO_SLUG;
      delete process.env.GH_AW_ALLOWED_REPOS;
      mockExec.getExecOutput.mockResolvedValueOnce({
        exitCode: 0,
        stdout: "https://github.com/test-owner/test-repo/issues/123",
        stderr: "",
      });

      const handler = await createAgentSessionModule.main({ base: "main" });
      const result = await handler({ type: "create_agent_session", body: "Test task" });

      expect(result.success).toBe(true);
      expect(result.number).toBe("123");
    });
  });
});
