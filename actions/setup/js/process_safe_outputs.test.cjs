// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import path from "path";
import os from "os";

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
  /** @type {string} */
  let tempDir;
  /** @type {typeof process.stdout.write} */
  let originalStdoutWrite;
  /** @type {typeof process.stderr.write} */
  let originalStderrWrite;
  /** @type {ReturnType<typeof vi.fn>} */
  let mockHandlerMain;

  beforeEach(() => {
    vi.clearAllMocks();

    // Use a temporary directory for log files so tests are isolated.
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "pso-test-"));

    // Point the module at our temp directory by overriding the env it uses.
    // process_safe_outputs.cjs hard-codes LOGS_DIR = '/tmp/gh-aw'; to avoid
    // polluting that path we capture the write methods at OS level and verify
    // content via the real file streams by temporarily redirecting LOGS_DIR.
    // Since LOGS_DIR is a module-level constant, we instead verify behavior
    // through the captured output written to the temp files we create below.

    // Capture original stream methods before each test.
    originalStdoutWrite = process.stdout.write.bind(process.stdout);
    originalStderrWrite = process.stderr.write.bind(process.stderr);

    // Default stub: handler resolves without writing anything.
    mockHandlerMain = vi.fn().mockResolvedValue(undefined);
    handlerManagerModule.main = mockHandlerMain;
  });

  afterEach(() => {
    // Ensure stream methods are always restored even if a test fails.
    process.stdout.write = originalStdoutWrite;
    process.stderr.write = originalStderrWrite;
    handlerManagerModule.main = originalHandlerMain;

    fs.rmSync(tempDir, { recursive: true, force: true });
    // Clean up log files written to the real LOGS_DIR by the module.
    fs.rmSync("/tmp/gh-aw/process-safe-outputs.stdout.log", { force: true });
    fs.rmSync("/tmp/gh-aw/process-safe-outputs.stderr.log", { force: true });
  });

  it("calls the safe output handler manager", async () => {
    await main();
    expect(mockHandlerMain).toHaveBeenCalledOnce();
  });

  it("restores process.stdout.write after success — no tee after completion", async () => {
    await main();
    // Write after completion — should NOT appear in the log file.
    process.stdout.write("post-main stdout\n");
    const content = fs.existsSync("/tmp/gh-aw/process-safe-outputs.stdout.log") ? fs.readFileSync("/tmp/gh-aw/process-safe-outputs.stdout.log", "utf8") : "";
    expect(content).not.toContain("post-main stdout");
  });

  it("restores process.stderr.write after success — no tee after completion", async () => {
    await main();
    // Write after completion — should NOT appear in the log file.
    process.stderr.write("post-main stderr\n");
    const content = fs.existsSync("/tmp/gh-aw/process-safe-outputs.stderr.log") ? fs.readFileSync("/tmp/gh-aw/process-safe-outputs.stderr.log", "utf8") : "";
    expect(content).not.toContain("post-main stderr");
  });

  it("restores process.stdout.write after handler rejection — no tee after completion", async () => {
    mockHandlerMain.mockRejectedValue(new Error("handler failed"));
    await expect(main()).rejects.toThrow("handler failed");
    process.stdout.write("post-rejection stdout\n");
    const content = fs.existsSync("/tmp/gh-aw/process-safe-outputs.stdout.log") ? fs.readFileSync("/tmp/gh-aw/process-safe-outputs.stdout.log", "utf8") : "";
    expect(content).not.toContain("post-rejection stdout");
  });

  it("restores process.stderr.write after handler rejection — no tee after completion", async () => {
    mockHandlerMain.mockRejectedValue(new Error("handler failed"));
    await expect(main()).rejects.toThrow("handler failed");
    process.stderr.write("post-rejection stderr\n");
    const content = fs.existsSync("/tmp/gh-aw/process-safe-outputs.stderr.log") ? fs.readFileSync("/tmp/gh-aw/process-safe-outputs.stderr.log", "utf8") : "";
    expect(content).not.toContain("post-rejection stderr");
  });

  it("tees stdout writes to the log file", async () => {
    mockHandlerMain.mockImplementation(() => {
      process.stdout.write("captured stdout line\n");
      return Promise.resolve();
    });
    await main();
    const content = fs.readFileSync("/tmp/gh-aw/process-safe-outputs.stdout.log", "utf8");
    expect(content).toContain("captured stdout line");
  });

  it("tees stderr writes to the log file", async () => {
    mockHandlerMain.mockImplementation(() => {
      process.stderr.write("captured stderr line\n");
      return Promise.resolve();
    });
    await main();
    const content = fs.readFileSync("/tmp/gh-aw/process-safe-outputs.stderr.log", "utf8");
    expect(content).toContain("captured stderr line");
  });

  it("captures stdout when handler rejects", async () => {
    mockHandlerMain.mockImplementation(() => {
      process.stdout.write("stdout before failure\n");
      return Promise.reject(new Error("fail"));
    });
    await expect(main()).rejects.toThrow("fail");
    const content = fs.readFileSync("/tmp/gh-aw/process-safe-outputs.stdout.log", "utf8");
    expect(content).toContain("stdout before failure");
  });

  it("forwards stdout write to the original stream", async () => {
    const written = /** @type {Array<string|Buffer>} */ [];
    const captureWrite = /** @type {typeof process.stdout.write} */ (chunk, _enc, cb) => {
      written.push(chunk);
      if (typeof cb === "function") cb();
      return true;
    };
    process.stdout.write = captureWrite;
    originalStdoutWrite = captureWrite;

    mockHandlerMain.mockImplementation(() => {
      process.stdout.write("forwarded\n");
      return Promise.resolve();
    });
    await main();
    expect(written.some(c => String(c).includes("forwarded"))).toBe(true);
  });

  it("forwards 2-arg write (chunk, callback) correctly to original stream", async () => {
    let callbackFired = false;
    const captureWrite = /** @type {typeof process.stdout.write} */ (chunk, encOrCb, cb) => {
      if (typeof encOrCb === "function") encOrCb();
      else if (typeof cb === "function") cb();
      return true;
    };
    process.stdout.write = captureWrite;
    originalStdoutWrite = captureWrite;

    mockHandlerMain.mockImplementation(() => {
      process.stdout.write("two-arg chunk\n", () => {
        callbackFired = true;
      });
      return Promise.resolve();
    });
    await main();
    expect(callbackFired).toBe(true);
  });

  it("truncates log file on each run (flags: w, not a)", async () => {
    // First run: write something.
    mockHandlerMain.mockImplementationOnce(() => {
      process.stdout.write("first run\n");
      return Promise.resolve();
    });
    await main();

    // Second run: write something different.
    mockHandlerMain.mockImplementationOnce(() => {
      process.stdout.write("second run\n");
      return Promise.resolve();
    });
    // Re-require so the module state is fresh for the second call.
    await main();

    const content = fs.readFileSync("/tmp/gh-aw/process-safe-outputs.stdout.log", "utf8");
    // With flags: "w", only the latest run's output should be present.
    expect(content).not.toContain("first run");
    expect(content).toContain("second run");
  });
});
