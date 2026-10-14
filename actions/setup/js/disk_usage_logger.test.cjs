import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import childProcess from "child_process";
import { logDiskUsage, startDiskUsageMonitor, stopDiskUsageMonitor, runDiskUsageMonitor, SAMPLE_INTERVAL_MS } from "./disk_usage_logger.cjs";
import { DISK_USAGE_JSONL_PATH } from "./constants.cjs";

describe("runner disk usage monitoring", () => {
  const stats = { bsize: 4096, blocks: 100, bfree: 25, bavail: 20, files: 1000, ffree: 250 };
  let originalEnv;
  let append;
  let warning;

  beforeEach(() => {
    originalEnv = { ...process.env };
    process.env.RUNNER_TEMP = "/runner/temp";
    process.env.GITHUB_WORKSPACE = "/runner/work";
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-10-14T16:00:00Z"));
    vi.spyOn(fs, "statfsSync").mockReturnValue(stats);
    vi.spyOn(fs, "mkdirSync").mockImplementation(() => undefined);
    append = vi.spyOn(fs, "appendFileSync").mockImplementation(() => undefined);
    warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    vi.spyOn(fs, "existsSync").mockReturnValue(true);
    vi.spyOn(fs, "readFileSync").mockReturnValue("");
    vi.spyOn(fs, "writeFileSync").mockImplementation(() => undefined);
    vi.spyOn(fs, "rmSync").mockImplementation(() => undefined);
    vi.spyOn(fs, "openSync").mockReturnValue(42);
    vi.spyOn(fs, "closeSync").mockImplementation(() => undefined);
    vi.spyOn(childProcess, "spawn").mockReturnValue({ on: vi.fn(), unref: vi.fn() });
  });

  afterEach(() => {
    vi.clearAllTimers();
    vi.useRealTimers();
    vi.restoreAllMocks();
    process.env = originalEnv;
  });

  function entries() {
    return append.mock.calls.flatMap(([, lines]) => lines.trim().split("\n").map(JSON.parse));
  }

  it("records available bytes separately from free bytes, with inode capacity and timestamp", () => {
    expect(logDiskUsage("setup")).toBe(true);
    expect(append.mock.calls[0][0]).toBe(DISK_USAGE_JSONL_PATH);
    expect(entries().map(entry => entry.resource)).toEqual(["root", "runtime", "workspace", "runner_temp"]);
    expect(entries()[0]).toEqual({
      timestamp: "2026-10-14T16:00:00.000Z",
      source: "statfs",
      operation: "setup",
      resource: "root",
      totalBytes: 409600,
      freeBytes: 102400,
      availableBytes: 81920,
      usedBytes: 307200,
      totalInodes: 1000,
      freeInodes: 250,
    });
    expect(append.mock.calls[0][1]).toMatch(/\n$/);
  });

  it("retains zero available bytes and zero free inodes as real exhaustion evidence", () => {
    fs.statfsSync.mockReturnValue({ ...stats, bavail: 0, ffree: 0 });
    logDiskUsage("interval");
    expect(entries()[0]).toMatchObject({ availableBytes: 0, freeInodes: 0 });
  });

  it("reports unsupported inode capacity as null, not as exhausted", () => {
    fs.statfsSync.mockReturnValue({ ...stats, files: 0, ffree: 0 });
    logDiskUsage("interval");
    expect(entries()[0]).toMatchObject({ totalInodes: null, freeInodes: null });
  });

  it("warns on an unavailable filesystem without losing other measurements", () => {
    fs.statfsSync.mockImplementation(directory => {
      if (directory === "/runner/work") throw new Error("filesystem unavailable");
      return stats;
    });
    expect(logDiskUsage("setup")).toBe(true);
    expect(entries().map(entry => entry.resource)).not.toContain("workspace");
    expect(warning).toHaveBeenCalledWith(expect.stringContaining("failed to sample workspace"));
  });

  it("warns without writing invalid, negative, or imprecise statistics", () => {
    fs.statfsSync.mockReturnValue({ ...stats, bavail: -1 });
    expect(logDiskUsage("interval")).toBe(false);
    fs.statfsSync.mockReturnValue({ ...stats, blocks: Number.MAX_SAFE_INTEGER });
    expect(logDiskUsage("interval")).toBe(false);
    expect(append).not.toHaveBeenCalled();
    expect(warning).toHaveBeenCalled();
  });

  it("surfaces disk-full write errors without failing the workflow", () => {
    append.mockImplementation(() => {
      throw new Error("ENOSPC");
    });
    expect(logDiskUsage("interval")).toBe(false);
    expect(warning).toHaveBeenCalledWith(expect.stringContaining("ENOSPC"));
  });

  it("starts before agent execution without passing ambient credentials to the worker", () => {
    fs.existsSync.mockReturnValue(false);
    process.env.GH_TOKEN = "private-token";
    process.env.RUNNER_TRACKING_ID = "tracking-id";
    startDiskUsageMonitor();
    expect(entries()[0].operation).toBe("setup");
    const [, args, options] = childProcess.spawn.mock.calls[0];
    expect(args[1]).toBe("monitor");
    expect(options.detached).toBe(true);
    expect(options.env).toEqual({ RUNNER_TEMP: "/runner/temp", GITHUB_WORKSPACE: "/runner/work", RUNNER_TRACKING_ID: "tracking-id" });
    expect(JSON.stringify(entries())).not.toContain("private-token");
  });

  it("does not start a second sampler or start one when logging fails", () => {
    startDiskUsageMonitor();
    expect(childProcess.spawn).not.toHaveBeenCalled();
    fs.existsSync.mockReturnValue(false);
    append.mockImplementation(() => {
      throw new Error("ENOSPC");
    });
    startDiskUsageMonitor();
    expect(childProcess.spawn).not.toHaveBeenCalled();
  });

  it("samples every 30 seconds and acknowledges shutdown before producing more records", async () => {
    runDiskUsageMonitor();
    await vi.advanceTimersByTimeAsync(SAMPLE_INTERVAL_MS - 1000);
    expect(append).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(1000);
    expect(entries()).toHaveLength(4);
    expect(entries()[0].operation).toBe("interval");
    fs.readFileSync.mockReturnValue("stop");
    await vi.advanceTimersByTimeAsync(1000);
    expect(entries()[4].operation).toBe("shutdown");
    expect(fs.rmSync).toHaveBeenCalledWith("/runner/temp/gh-aw/disk-usage-monitor", { force: true });
    await vi.advanceTimersByTimeAsync(60_000);
    expect(entries()).toHaveLength(8);
    expect(vi.getTimerCount()).toBe(0);
  });

  it("waits for the worker to acknowledge a stop request", async () => {
    runDiskUsageMonitor();
    fs.writeFileSync.mockImplementation(() => fs.readFileSync.mockReturnValue("stop"));
    fs.rmSync.mockImplementation(() => fs.existsSync.mockReturnValue(false));
    const stopped = stopDiskUsageMonitor();
    await vi.advanceTimersByTimeAsync(1000);
    await stopped;
    expect(fs.writeFileSync).toHaveBeenCalledWith("/runner/temp/gh-aw/disk-usage-monitor", "stop");
    expect(warning).not.toHaveBeenCalled();
  });

  it("warns on an unresponsive worker and leaves unrelated processes alone", async () => {
    const stopped = stopDiskUsageMonitor();
    await vi.advanceTimersByTimeAsync(5000);
    await stopped;
    expect(warning).toHaveBeenCalledWith(expect.stringContaining("did not acknowledge shutdown"));
    expect(fs.rmSync).not.toHaveBeenCalled();
  });

  it("bounds an abandoned monitor's lifetime", async () => {
    runDiskUsageMonitor();
    vi.setSystemTime(Date.now() + 72 * 60 * 60 * 1000);
    await vi.advanceTimersByTimeAsync(1000);
    expect(entries()[0].operation).toBe("shutdown");
    expect(vi.getTimerCount()).toBe(0);
  });
});
