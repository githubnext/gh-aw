// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";

const req = createRequire(import.meta.url);

// Set up global.core mock before loading any module so shim.cjs does not
// overwrite it with its stderr-based fallback.
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
};
global.core = mockCore;

// Load safe_output_handler_manager first so we can patch its exports through
// the shared CJS module cache before process_safe_outputs.cjs is loaded.
const handlerManagerModule = req("./safe_output_handler_manager.cjs");
const originalHandlerMain = handlerManagerModule.main;

// Load the module under test after the dependency is in the cache.
const { main } = req("./process_safe_outputs.cjs");

describe("process_safe_outputs.cjs", () => {
  /** @type {ReturnType<typeof vi.fn>} */
  let mockHandlerMain;

  beforeEach(() => {
    vi.clearAllMocks();
    mockHandlerMain = vi.fn().mockResolvedValue(undefined);
    handlerManagerModule.main = mockHandlerMain;
  });

  afterEach(() => {
    handlerManagerModule.main = originalHandlerMain;
  });

  it("calls the safe output handler manager", async () => {
    await main();
    expect(mockHandlerMain).toHaveBeenCalledOnce();
  });

  it("propagates handler rejection", async () => {
    mockHandlerMain.mockRejectedValue(new Error("handler failed"));
    await expect(main()).rejects.toThrow("handler failed");
  });

  it("does not create stdout or stderr log files", async () => {
    await main();
    expect(fs.existsSync("/tmp/gh-aw/process-safe-outputs.stdout.log")).toBe(false);
    expect(fs.existsSync("/tmp/gh-aw/process-safe-outputs.stderr.log")).toBe(false);
  });

  it("logs captured process output through a protected action log group", async () => {
    mockHandlerMain.mockImplementation(async () => {
      process.stdout.write("safe stdout\n");
      process.stderr.write("::error::safe stderr\n");
    });
    const stdoutWrite = vi.spyOn(process.stdout, "write").mockImplementation(() => true);
    try {
      await main();
      const output = stdoutWrite.mock.calls.map(call => String(call[0])).join("");
      expect(output).toContain("::group::Safe output processing logs\n");
      expect(output).toContain("safe stdout\n::error::safe stderr\n");
      expect(output).toContain("::endgroup::\n");
    } finally {
      stdoutWrite.mockRestore();
    }
  });

  it("defers core.setFailed calls made during capture and replays them for real after restore", async () => {
    const originalWrite = process.stdout.write;
    /** @type {boolean | null} */
    let calledAfterStdoutRestored = null;
    mockCore.setFailed.mockImplementation(() => {
      calledAfterStdoutRestored = process.stdout.write === originalWrite;
    });
    mockHandlerMain.mockImplementation(async () => {
      // Simulate safe_output_handler_manager.cjs's real behavior: it swallows
      // its own errors and calls core.setFailed directly instead of
      // rethrowing, while process output capture is still active.
      process.stdout.write("working...\n");
      global.core.setFailed("handler manager failed: boom");
    });

    await main();

    expect(mockCore.setFailed).toHaveBeenCalledTimes(1);
    expect(mockCore.setFailed).toHaveBeenCalledWith("handler manager failed: boom");
    // The critical assertion: core.setFailed must not fire until AFTER
    // process.stdout.write has been restored to the real writer, otherwise
    // the annotation would be silently buffered as inert captured text
    // inside the stop-commands-guarded group instead of being a real,
    // live annotation.
    expect(calledAfterStdoutRestored).toBe(true);
  });

  it("does not call core.setFailed when the handler manager does not signal a failure", async () => {
    await main();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("restores core.setFailed to the original function after replay", async () => {
    const originalSetFailed = mockCore.setFailed;
    mockHandlerMain.mockImplementation(async () => {
      global.core.setFailed("boom");
    });
    await main();
    expect(global.core.setFailed).toBe(originalSetFailed);
  });

  it("restores the stdout writer and still flushes partial output when the handler rejects", async () => {
    mockHandlerMain.mockImplementation(async () => {
      process.stdout.write("partial output before failure\n");
      throw new Error("handler failed");
    });
    /** @type {string[]} */
    const calls = [];
    const originalWrite = process.stdout.write;
    // A plain reassignment (not vi.spyOn) so this test observes exactly what
    // process_safe_outputs.cjs's finally block restores process.stdout.write
    // to, without any spy-framework indirection.
    const capturingWrite = /** @type {any} */ chunk => {
      calls.push(String(chunk));
      return true;
    };
    process.stdout.write = capturingWrite;
    try {
      await expect(main()).rejects.toThrow("handler failed");
      // Even though the handler rejected, process_safe_outputs.cjs must restore
      // stdout.write to the writer that was active before it started capturing
      // (not leave its internal capture stub installed), so later writers in
      // the process are never silently swallowed.
      expect(process.stdout.write).toBe(capturingWrite);
    } finally {
      process.stdout.write = originalWrite;
    }
    const output = calls.join("");
    expect(output).toContain("::group::Safe output processing logs\n");
    expect(output).toContain("partial output before failure\n");
    expect(output).toContain("::endgroup::\n");
  });
});
