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

describe("update_circuit_breaker.cjs", () => {
  let tmpDir;
  let stateFile;
  let originalEnv;

  beforeEach(() => {
    vi.clearAllMocks();
    // Restore summary stubs after clearAllMocks
    mockCore.summary.addRaw.mockReturnThis();
    mockCore.summary.write.mockResolvedValue(undefined);

    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "cb-update-test-"));
    stateFile = path.join(tmpDir, "circuit-breaker-state.json");

    originalEnv = {
      GH_AW_CB_JOB_STATUS: process.env.GH_AW_CB_JOB_STATUS,
      GH_AW_CB_MAX_FAILURES: process.env.GH_AW_CB_MAX_FAILURES,
      GH_AW_CB_TIME_WINDOW_MINUTES: process.env.GH_AW_CB_TIME_WINDOW_MINUTES,
      GH_AW_WORKFLOW_NAME: process.env.GH_AW_WORKFLOW_NAME,
      GH_AW_CB_STATE_DIR: process.env.GH_AW_CB_STATE_DIR,
    };
    process.env.GH_AW_CB_MAX_FAILURES = "5";
    process.env.GH_AW_CB_TIME_WINDOW_MINUTES = "1440";
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

  /** Write a previous state JSON to the temp dir. */
  function writePreviousState(state) {
    fs.writeFileSync(stateFile, JSON.stringify(state), "utf8");
  }

  /** Read the state that was written by the script. */
  function readWrittenState() {
    expect(fs.existsSync(stateFile)).toBe(true);
    return JSON.parse(fs.readFileSync(stateFile, "utf8"));
  }

  async function runUpdate() {
    vi.resetModules();
    const mod = await import("./update_circuit_breaker.cjs");
    await mod.main();
  }

  it("SUCCESS — resets consecutive_failures to 0", async () => {
    process.env.GH_AW_CB_JOB_STATUS = "success";
    writePreviousState({ consecutive_failures: 3, last_failure: minutesAgoISO(10) });

    await runUpdate();

    const state = readWrittenState();
    expect(state.consecutive_failures).toBe(0);
    expect(state.last_success).toBeTruthy();
    expect(state.circuit_opened_at).toBeNull();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("resetting circuit breaker"));
  });

  it("SUCCESS — preserves last_failure timestamp after reset", async () => {
    process.env.GH_AW_CB_JOB_STATUS = "success";
    const pastFailure = minutesAgoISO(60);
    writePreviousState({ consecutive_failures: 2, last_failure: pastFailure });

    await runUpdate();

    const state = readWrittenState();
    expect(state.consecutive_failures).toBe(0);
    expect(state.last_failure).toBe(pastFailure);
  });

  it("FAILURE — increments consecutive_failures from 0 (no prior state)", async () => {
    process.env.GH_AW_CB_JOB_STATUS = "failure";
    // No previous state file written

    await runUpdate();

    const state = readWrittenState();
    expect(state.consecutive_failures).toBe(1);
    expect(state.last_failure).toBeTruthy();
  });

  it("FAILURE — increments consecutive_failures from existing in-window count", async () => {
    process.env.GH_AW_CB_JOB_STATUS = "failure";
    writePreviousState({ consecutive_failures: 3, last_failure: minutesAgoISO(5) });

    await runUpdate();

    const state = readWrittenState();
    expect(state.consecutive_failures).toBe(4);
  });

  it("FAILURE — sets circuit_opened_at when threshold is first reached", async () => {
    process.env.GH_AW_CB_JOB_STATUS = "failure";
    process.env.GH_AW_CB_MAX_FAILURES = "5";
    // 4 existing failures → this run makes 5, hitting the threshold
    writePreviousState({ consecutive_failures: 4, last_failure: minutesAgoISO(5) });

    await runUpdate();

    const state = readWrittenState();
    expect(state.consecutive_failures).toBe(5);
    expect(state.circuit_opened_at).toBeTruthy();
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("threshold reached"));
  });

  it("FAILURE — preserves original circuit_opened_at on subsequent failures", async () => {
    process.env.GH_AW_CB_JOB_STATUS = "failure";
    process.env.GH_AW_CB_MAX_FAILURES = "5";
    const originalOpenedAt = minutesAgoISO(30);
    writePreviousState({
      consecutive_failures: 6,
      last_failure: minutesAgoISO(5),
      circuit_opened_at: originalOpenedAt,
    });

    await runUpdate();

    const state = readWrittenState();
    expect(state.consecutive_failures).toBe(7);
    // Original timestamp must be preserved, not overwritten
    expect(state.circuit_opened_at).toBe(originalOpenedAt);
  });

  it("FAILURE — resets counter when last_failure is outside time window", async () => {
    process.env.GH_AW_CB_JOB_STATUS = "failure";
    process.env.GH_AW_CB_TIME_WINDOW_MINUTES = "60"; // 1-hour window
    // Last failure was 2 hours ago — outside window
    writePreviousState({ consecutive_failures: 4, last_failure: minutesAgoISO(120) });

    await runUpdate();

    // Old out-of-window failures are ignored; counter starts fresh at 1
    const state = readWrittenState();
    expect(state.consecutive_failures).toBe(1);
  });

  it("CANCELLED — treated as failure, increments counter", async () => {
    process.env.GH_AW_CB_JOB_STATUS = "cancelled";
    writePreviousState({ consecutive_failures: 2, last_failure: minutesAgoISO(5) });

    await runUpdate();

    const state = readWrittenState();
    expect(state.consecutive_failures).toBe(3);
  });

  it("corrupt state file — starts fresh rather than crashing", async () => {
    process.env.GH_AW_CB_JOB_STATUS = "failure";
    fs.writeFileSync(stateFile, "NOT VALID JSON{{{", "utf8");

    await runUpdate();

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not load"));
    const state = readWrittenState();
    // Starts from 0 + 1 failure = 1
    expect(state.consecutive_failures).toBe(1);
  });
});
