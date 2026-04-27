// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

// ---- Globals ----
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),
  getCancelled: vi.fn(),
  setCancelled: vi.fn(),
  setError: vi.fn(),
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),
  summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue(undefined) },
};
global.core = mockCore;
global.github = {};
global.context = { repo: { owner: "test-owner", repo: "test-repo" }, runId: 123456 };

// Helper: relative timestamps
const minutesAgoISO = m => new Date(Date.now() - m * 60 * 1000).toISOString();

describe("check_circuit_breaker.cjs", () => {
  let tmpDir;
  let originalEnv;

  beforeEach(() => {
    vi.clearAllMocks();
    // Restore summary stubs after clearAllMocks
    mockCore.summary.addRaw.mockReturnThis();
    mockCore.summary.write.mockResolvedValue(undefined);

    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "cb-check-test-"));

    originalEnv = {
      GH_AW_CB_MAX_FAILURES: process.env.GH_AW_CB_MAX_FAILURES,
      GH_AW_CB_TIME_WINDOW_MINUTES: process.env.GH_AW_CB_TIME_WINDOW_MINUTES,
      GH_AW_CB_COOLDOWN_MINUTES: process.env.GH_AW_CB_COOLDOWN_MINUTES,
      GH_AW_CB_NOTIFY: process.env.GH_AW_CB_NOTIFY,
      GH_AW_WORKFLOW_NAME: process.env.GH_AW_WORKFLOW_NAME,
      GH_AW_CB_STATE_DIR: process.env.GH_AW_CB_STATE_DIR,
    };
    process.env.GH_AW_CB_MAX_FAILURES = "5";
    process.env.GH_AW_CB_TIME_WINDOW_MINUTES = "1440";
    process.env.GH_AW_CB_COOLDOWN_MINUTES = "60";
    process.env.GH_AW_CB_NOTIFY = "true";
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_CB_STATE_DIR = tmpDir;
  });

  afterEach(() => {
    for (const [key, val] of Object.entries(originalEnv)) {
      if (val === undefined) {
        delete process.env[key];
      } else {
        process.env[key] = val;
      }
    }
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  /** Write a state file to the temp dir. */
  function writeState(state) {
    fs.writeFileSync(path.join(tmpDir, "circuit-breaker-state.json"), JSON.stringify(state), "utf8");
  }

  async function runCheck() {
    vi.resetModules();
    const mod = await import("./check_circuit_breaker.cjs");
    await mod.main();
  }

  it("CLOSED — no previous state file: allows execution", async () => {
    await runCheck();

    expect(mockCore.setOutput).toHaveBeenCalledWith("circuit_breaker_ok", "true");
    expect(mockCore.setOutput).toHaveBeenCalledWith("consecutive_failures", "0");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("CLOSED"));
  });

  it("CLOSED — failures below threshold: allows execution", async () => {
    writeState({ consecutive_failures: 3, last_failure: minutesAgoISO(10) });

    await runCheck();

    expect(mockCore.setOutput).toHaveBeenCalledWith("circuit_breaker_ok", "true");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("CLOSED"));
  });

  it("CLOSED — failures at threshold but outside time window: allows execution", async () => {
    // 2 days ago — outside the 24h (1440 min) window
    writeState({ consecutive_failures: 5, last_failure: minutesAgoISO(2 * 24 * 60) });

    await runCheck();

    expect(mockCore.setOutput).toHaveBeenCalledWith("circuit_breaker_ok", "true");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("CLOSED"));
  });

  it("OPEN — failures at threshold within window: blocks execution", async () => {
    writeState({ consecutive_failures: 5, last_failure: minutesAgoISO(5) });

    await runCheck();

    expect(mockCore.setOutput).toHaveBeenCalledWith("circuit_breaker_ok", "false");
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("OPEN"));
  });

  it("OPEN — notify=true posts an error annotation", async () => {
    process.env.GH_AW_CB_NOTIFY = "true";
    writeState({ consecutive_failures: 5, last_failure: minutesAgoISO(5) });

    await runCheck();

    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Circuit breaker OPEN"));
  });

  it("OPEN — notify=false skips the error annotation", async () => {
    process.env.GH_AW_CB_NOTIFY = "false";
    writeState({ consecutive_failures: 5, last_failure: minutesAgoISO(5) });

    await runCheck();

    expect(mockCore.setOutput).toHaveBeenCalledWith("circuit_breaker_ok", "false");
    expect(mockCore.error).not.toHaveBeenCalled();
  });

  it("HALF-OPEN — cooldown elapsed: allows one retry", async () => {
    process.env.GH_AW_CB_COOLDOWN_MINUTES = "60";
    // 90 min ago — cooldown (60 min) elapsed, still within 24h window
    writeState({ consecutive_failures: 5, last_failure: minutesAgoISO(90) });

    await runCheck();

    expect(mockCore.setOutput).toHaveBeenCalledWith("circuit_breaker_ok", "true");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("HALF-OPEN"));
  });

  it("OPEN — cooldown not yet elapsed: blocks execution", async () => {
    process.env.GH_AW_CB_COOLDOWN_MINUTES = "60";
    // 30 min ago — cooldown (60 min) not yet elapsed
    writeState({ consecutive_failures: 5, last_failure: minutesAgoISO(30) });

    await runCheck();

    expect(mockCore.setOutput).toHaveBeenCalledWith("circuit_breaker_ok", "false");
  });

  it("CLOSED — custom lower threshold respected", async () => {
    process.env.GH_AW_CB_MAX_FAILURES = "3";
    writeState({ consecutive_failures: 2, last_failure: minutesAgoISO(5) });

    await runCheck();

    expect(mockCore.setOutput).toHaveBeenCalledWith("circuit_breaker_ok", "true");
  });

  it("handles corrupt state file gracefully — circuit CLOSED (fail-open)", async () => {
    fs.writeFileSync(path.join(tmpDir, "circuit-breaker-state.json"), "NOT VALID JSON", "utf8");

    await runCheck();

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not read"));
    // Fail-open: allow execution
    expect(mockCore.setOutput).toHaveBeenCalledWith("circuit_breaker_ok", "true");
  });
});
