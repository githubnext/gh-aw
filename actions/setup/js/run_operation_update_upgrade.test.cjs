// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

/** Environment variables managed by tests */
const TEST_ENV_VARS = ["GH_AW_OPERATION", "GH_AW_CMD_PREFIX", "GH_TOKEN", "GITHUB_TOKEN"];

describe("run_operation_update_upgrade", () => {
  let mockCore;
  let mockGithub;
  let mockContext;
  let mockExec;
  let originalGlobals;
  let originalEnv;

  beforeEach(() => {
    originalEnv = { ...process.env };

    // Save original globals
    originalGlobals = {
      core: global.core,
      github: global.github,
      context: global.context,
      exec: global.exec,
    };

    // Setup mock core module
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      notice: vi.fn(),
      summary: {
        addHeading: vi.fn().mockReturnThis(),
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };

    // Setup mock github
    mockGithub = {};

    // Setup mock context
    mockContext = {
      repo: {
        owner: "testowner",
        repo: "testrepo",
      },
    };

    // Setup mock exec module
    mockExec = {
      exec: vi.fn().mockResolvedValue(0),
      getExecOutput: vi.fn(),
    };

    // Set globals for the module
    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
    global.exec = mockExec;
  });

  afterEach(() => {
    // Restore environment variables
    for (const key of TEST_ENV_VARS) {
      if (originalEnv[key] !== undefined) {
        process.env[key] = originalEnv[key];
      } else {
        delete process.env[key];
      }
    }

    // Restore original globals
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
    global.exec = originalGlobals.exec;

    vi.clearAllMocks();
  });

  describe("formatTimestamp", () => {
    it("formats a date as YYYY-MM-DD-HH-MM-SS", async () => {
      const { formatTimestamp } = await import("./run_operation_update_upgrade.cjs");
      const date = new Date("2026-03-03T03:17:06.000Z");
      expect(formatTimestamp(date)).toBe("2026-03-03-03-17-06");
    });

    it("pads single-digit values with zeros", async () => {
      const { formatTimestamp } = await import("./run_operation_update_upgrade.cjs");
      const date = new Date("2026-01-05T09:05:03.000Z");
      expect(formatTimestamp(date)).toBe("2026-01-05-09-05-03");
    });
  });

  describe("main - skips non-update/upgrade operations", () => {
    it("skips when operation is not set", async () => {
      delete process.env.GH_AW_OPERATION;
      const { main } = await import("./run_operation_update_upgrade.cjs");
      await main();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping"));
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("skips when operation is unknown", async () => {
      process.env.GH_AW_OPERATION = "unknown-operation";
      const { main } = await import("./run_operation_update_upgrade.cjs");
      await main();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping"));
      expect(mockExec.exec).not.toHaveBeenCalled();
    });
  });

  describe("main - disable/enable operations", () => {
    it("runs gh aw disable and finishes without PR", async () => {
      process.env.GH_AW_OPERATION = "disable";
      process.env.GH_AW_CMD_PREFIX = "gh aw";

      const { main } = await import("./run_operation_update_upgrade.cjs");
      await main();

      expect(mockExec.exec).toHaveBeenCalledWith("gh", ["aw", "disable"]);
      expect(mockExec.exec).toHaveBeenCalledTimes(1);
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("disabled"));
      expect(mockExec.getExecOutput).not.toHaveBeenCalled();
    });

    it("runs gh aw enable and finishes without PR", async () => {
      process.env.GH_AW_OPERATION = "enable";
      process.env.GH_AW_CMD_PREFIX = "gh aw";

      const { main } = await import("./run_operation_update_upgrade.cjs");
      await main();

      expect(mockExec.exec).toHaveBeenCalledWith("gh", ["aw", "enable"]);
      expect(mockExec.exec).toHaveBeenCalledTimes(1);
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("enabled"));
      expect(mockExec.getExecOutput).not.toHaveBeenCalled();
    });

    it("runs ./gh-aw disable in dev mode", async () => {
      process.env.GH_AW_OPERATION = "disable";
      process.env.GH_AW_CMD_PREFIX = "./gh-aw";

      const { main } = await import("./run_operation_update_upgrade.cjs");
      await main();

      expect(mockExec.exec).toHaveBeenCalledWith("./gh-aw", ["disable"]);
      expect(mockExec.exec).toHaveBeenCalledTimes(1);
    });

    it("propagates error when disable command fails", async () => {
      process.env.GH_AW_OPERATION = "disable";
      process.env.GH_AW_CMD_PREFIX = "gh aw";

      mockExec.exec = vi.fn().mockRejectedValue(new Error("Command failed"));

      const { main } = await import("./run_operation_update_upgrade.cjs");
      await expect(main()).rejects.toThrow("Command failed");
    });

    it("throws when disable exits with non-zero code", async () => {
      process.env.GH_AW_OPERATION = "disable";
      process.env.GH_AW_CMD_PREFIX = "gh aw";

      mockExec.exec = vi.fn().mockResolvedValue(1);

      const { main } = await import("./run_operation_update_upgrade.cjs");
      await expect(main()).rejects.toThrow("exit code 1");
    });
  });

  describe("main - no changes after command", () => {
    it("finishes without creating PR when no files changed", async () => {
      process.env.GH_AW_OPERATION = "update";
      process.env.GH_AW_CMD_PREFIX = "gh aw";
      process.env.GH_TOKEN = "test-token";

      // git status shows no changes
      mockExec.getExecOutput = vi.fn().mockResolvedValue({ stdout: "", stderr: "", exitCode: 0 });

      const { main } = await import("./run_operation_update_upgrade.cjs");
      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No changes detected"));
      expect(mockExec.exec).toHaveBeenCalledWith("gh", ["aw", "update", "--no-compile"]);
    });

    it("finishes without PR when only workflow yml files changed", async () => {
      process.env.GH_AW_OPERATION = "update";
      process.env.GH_AW_CMD_PREFIX = "gh aw";
      process.env.GH_TOKEN = "test-token";

      mockExec.getExecOutput = vi.fn().mockResolvedValueOnce({
        stdout: " M .github/workflows/agentics-maintenance.yml\n",
        stderr: "",
        exitCode: 0,
      });

      const { main } = await import("./run_operation_update_upgrade.cjs");
      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No non-workflow files changed"));
      expect(mockExec.exec).not.toHaveBeenCalledWith("git", expect.arrayContaining(["add"]));
    });
  });

  describe("main - creates PR when files changed", () => {
    it("creates PR for update operation with changes", async () => {
      process.env.GH_AW_OPERATION = "update";
      process.env.GH_AW_CMD_PREFIX = "gh aw";
      process.env.GH_TOKEN = "test-token";

      const getExecOutputMock = vi.fn();
      // git status
      getExecOutputMock.mockResolvedValueOnce({
        stdout: " M .github/workflows/my-workflow.md\n",
        stderr: "",
        exitCode: 0,
      });
      // git diff --cached --name-only
      getExecOutputMock.mockResolvedValueOnce({
        stdout: ".github/workflows/my-workflow.md\n",
        stderr: "",
        exitCode: 0,
      });
      // gh pr create
      getExecOutputMock.mockResolvedValueOnce({
        stdout: "https://github.com/testowner/testrepo/pull/1\n",
        stderr: "",
        exitCode: 0,
      });
      mockExec.getExecOutput = getExecOutputMock;

      const { main } = await import("./run_operation_update_upgrade.cjs");
      await main();

      // Verify gh aw update --no-compile was run
      expect(mockExec.exec).toHaveBeenCalledWith("gh", ["aw", "update", "--no-compile"]);
      // Verify branch was created
      expect(mockExec.exec).toHaveBeenCalledWith("git", expect.arrayContaining(["checkout", "-b", expect.stringContaining("aw/update-")]));
      // Verify file was staged
      expect(mockExec.exec).toHaveBeenCalledWith("git", ["add", "--", ".github/workflows/my-workflow.md"]);
      // Verify commit was made
      expect(mockExec.exec).toHaveBeenCalledWith("git", ["commit", "-m", "chore: update agentic workflows"]);
      // Verify PR title
      expect(getExecOutputMock).toHaveBeenCalledWith("gh", expect.arrayContaining(["pr", "create", "--title", "[aw] Updates available", "--label", "agentic-workflows"]), expect.anything());
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Created PR"));
    });

    it("creates PR for upgrade operation with correct title", async () => {
      process.env.GH_AW_OPERATION = "upgrade";
      process.env.GH_AW_CMD_PREFIX = "gh aw";
      process.env.GH_TOKEN = "test-token";

      const getExecOutputMock = vi.fn();
      // git status
      getExecOutputMock.mockResolvedValueOnce({
        stdout: " M .github/agents/agentic-workflows.agent.md\n M .github/workflows/agentics-maintenance.yml\n",
        stderr: "",
        exitCode: 0,
      });
      // git diff --cached --name-only
      getExecOutputMock.mockResolvedValueOnce({
        stdout: ".github/agents/agentic-workflows.agent.md\n",
        stderr: "",
        exitCode: 0,
      });
      // gh pr create
      getExecOutputMock.mockResolvedValueOnce({
        stdout: "https://github.com/testowner/testrepo/pull/2\n",
        stderr: "",
        exitCode: 0,
      });
      mockExec.getExecOutput = getExecOutputMock;

      const { main } = await import("./run_operation_update_upgrade.cjs");
      await main();

      // Verify gh aw upgrade --no-compile was run
      expect(mockExec.exec).toHaveBeenCalledWith("gh", ["aw", "upgrade", "--no-compile"]);
      // Verify correct commit message
      expect(mockExec.exec).toHaveBeenCalledWith("git", ["commit", "-m", "chore: upgrade agentic workflows"]);
      // Verify PR title is "[aw] Upgrade available"
      expect(getExecOutputMock).toHaveBeenCalledWith("gh", expect.arrayContaining(["pr", "create", "--title", "[aw] Upgrade available", "--label", "agentic-workflows"]), expect.anything());
      // Verify workflow yml was NOT staged
      expect(mockExec.exec).not.toHaveBeenCalledWith("git", ["add", "--", ".github/workflows/agentics-maintenance.yml"]);
    });

    it("uses ./gh-aw as binary in dev mode", async () => {
      process.env.GH_AW_OPERATION = "update";
      process.env.GH_AW_CMD_PREFIX = "./gh-aw";
      process.env.GH_TOKEN = "test-token";

      const getExecOutputMock = vi.fn();
      getExecOutputMock
        .mockResolvedValueOnce({ stdout: " M .github/workflows/my-workflow.md\n", stderr: "", exitCode: 0 })
        .mockResolvedValueOnce({ stdout: ".github/workflows/my-workflow.md\n", stderr: "", exitCode: 0 })
        .mockResolvedValueOnce({ stdout: "https://github.com/testowner/testrepo/pull/3\n", stderr: "", exitCode: 0 });
      mockExec.getExecOutput = getExecOutputMock;

      const { main } = await import("./run_operation_update_upgrade.cjs");
      await main();

      // Verify binary is ./gh-aw (no prefix args) with --no-compile
      expect(mockExec.exec).toHaveBeenCalledWith("./gh-aw", ["update", "--no-compile"]);
    });
  });

  describe("main - handles errors", () => {
    it("propagates error when command fails", async () => {
      process.env.GH_AW_OPERATION = "update";
      process.env.GH_AW_CMD_PREFIX = "gh aw";
      process.env.GH_TOKEN = "test-token";

      mockExec.exec = vi.fn().mockRejectedValue(new Error("Command failed"));

      const { main } = await import("./run_operation_update_upgrade.cjs");
      await expect(main()).rejects.toThrow("Command failed");
    });

    it("throws when update exits with non-zero code", async () => {
      process.env.GH_AW_OPERATION = "update";
      process.env.GH_AW_CMD_PREFIX = "gh aw";
      process.env.GH_TOKEN = "test-token";

      mockExec.exec = vi.fn().mockResolvedValue(1);

      const { main } = await import("./run_operation_update_upgrade.cjs");
      await expect(main()).rejects.toThrow("exit code 1");
    });

    it("warns and continues when staging a file fails", async () => {
      process.env.GH_AW_OPERATION = "update";
      process.env.GH_AW_CMD_PREFIX = "gh aw";
      process.env.GH_TOKEN = "test-token";

      const getExecOutputMock = vi.fn();
      getExecOutputMock
        .mockResolvedValueOnce({
          stdout: " M .github/workflows/my-workflow.md\n?? .github/aw/actions-lock.json\n",
          stderr: "",
          exitCode: 0,
        })
        .mockResolvedValueOnce({ stdout: ".github/aw/actions-lock.json\n", stderr: "", exitCode: 0 })
        .mockResolvedValueOnce({ stdout: "https://github.com/testowner/testrepo/pull/4\n", stderr: "", exitCode: 0 });
      mockExec.getExecOutput = getExecOutputMock;

      // git add fails for the first file, succeeds for others
      mockExec.exec = vi.fn().mockImplementation(async (cmd, args) => {
        if (cmd === "git" && args[0] === "add" && args[2] === ".github/workflows/my-workflow.md") {
          throw new Error("git add failed");
        }
        return 0;
      });

      const { main } = await import("./run_operation_update_upgrade.cjs");
      await main();

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to stage"));
    });
  });
});
