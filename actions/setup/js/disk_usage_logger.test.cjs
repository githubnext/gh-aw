import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import childProcess from "child_process";
import { logDiskUsage, startDiskUsageMonitor, stopDiskUsageMonitor, runDiskUsageMonitor, collectDiskUsageArtifact, SAMPLE_INTERVAL_MS } from "./disk_usage_logger.cjs";
import { normalizeDiskUsage, sanitizeDiskDiagnostics } from "./disk_usage.cjs";

const collectorScript = fs.readFileSync(new URL("../sh/collect_usage_artifact_files.sh", import.meta.url), "utf8");

describe("runner disk usage monitoring", () => {
  const stats = { bsize: 4096, bavail: 1234567, files: 9876543, ffree: 12345 };
  const privateError = code => Object.assign(new Error("secret /private/device/path 123456789 bytes"), { code });
  let originalEnv;
  let writes;
  let warning;
  let watchCallback;
  let watchError;
  let watcher;
  let descriptors;
  let nextFd;

  beforeEach(() => {
    originalEnv = { ...process.env };
    process.env.RUNNER_TEMP = "/runner/temp";
    process.env.GITHUB_WORKSPACE = "/runner/work";
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-10-14T16:00:00Z"));
    descriptors = new Map();
    nextFd = 40;
    vi.spyOn(fs, "statfsSync").mockReturnValue(stats);
    vi.spyOn(fs, "mkdirSync").mockImplementation(() => undefined);
    writes = vi.spyOn(fs, "writeFileSync").mockImplementation(() => undefined);
    warning = vi.spyOn(process.stdout, "write").mockImplementation(() => true);
    vi.spyOn(fs, "openSync").mockImplementation((file, flags) => {
      const fd = nextFd++;
      descriptors.set(fd, { file, flags, nlink: 1 });
      return fd;
    });
    vi.spyOn(fs, "fstatSync").mockImplementation(fd => ({
      isDirectory: () => true,
      isFile: () => true,
      size: 0,
      nlink: descriptors.get(fd)?.nlink ?? 1,
    }));
    vi.spyOn(fs, "statSync").mockReturnValue({ isDirectory: () => true });
    vi.spyOn(fs, "ftruncateSync").mockImplementation(() => undefined);
    vi.spyOn(fs, "closeSync").mockImplementation(() => undefined);
    vi.spyOn(fs, "readSync").mockReturnValue(0);
    vi.spyOn(fs, "readFileSync").mockReturnValue("");
    vi.spyOn(fs, "writeSync").mockImplementation(() => 0);
    vi.spyOn(fs, "unlinkSync").mockImplementation(file => {
      for (const value of descriptors.values()) if (value.file === file) value.nlink = 0;
    });
    watcher = {
      close: vi.fn(),
      on: vi.fn((name, callback) => {
        watchError = callback;
      }),
    };
    vi.spyOn(fs, "watch").mockImplementation((file, callback) => {
      watchCallback = callback;
      return watcher;
    });
    vi.spyOn(childProcess, "spawn").mockReturnValue({ on: vi.fn(), unref: vi.fn() });
  });

  afterEach(() => {
    vi.clearAllTimers();
    vi.useRealTimers();
    vi.restoreAllMocks();
    process.env = originalEnv;
  });

  function entries() {
    return writes.mock.calls.flatMap(([, content]) => (content === "stop" || !content ? [] : content.trim().split("\n").map(JSON.parse)));
  }

  it("persists only fixed resource labels and coarse headroom bands", () => {
    expect(logDiskUsage("setup")).toBe(true);
    expect(entries().map(entry => entry.resource)).toEqual(["root", "runtime", "workspace", "runner_temp"]);
    expect(entries()[0]).toEqual({
      timestamp: "2026-10-14T16:00:00.000Z",
      source: "statfs",
      operation: "setup",
      resource: "root",
      availableSpaceBand: "low",
      inodeHeadroom: "low",
    });
    const output = JSON.stringify(entries());
    for (const value of ["1234567", "9876543", "12345", "/runner", "totalBytes", "availableBytes", "freeInodes"]) expect(output).not.toContain(value);
  });

  it.each([
    [0, "exhausted"],
    [1, "critical"],
    [1024 ** 3 - 1, "critical"],
    [1024 ** 3, "low"],
    [5 * 1024 ** 3 - 1, "low"],
    [5 * 1024 ** 3, "healthy"],
  ])("uses fixed byte thresholds in memory only: %s", (available, band) => {
    fs.statfsSync.mockReturnValue({ ...stats, bsize: 1, bavail: available });
    logDiskUsage("interval");
    expect(entries()[0].availableSpaceBand).toBe(band);
    expect(entries()[0]).not.toHaveProperty("availableBytes");
  });

  it.each([
    [0, 0, "unavailable"],
    [1000, 0, "exhausted"],
    [1000, 9, "low"],
    [1000, 10, "healthy"],
  ])("reports coarse inode headroom: %s/%s", (files, ffree, band) => {
    fs.statfsSync.mockReturnValue({ ...stats, files, ffree });
    logDiskUsage("interval");
    expect(entries()[0].inodeHeadroom).toBe(band);
  });

  it.each(["ENOENT", "EACCES", "ELOOP"])("broken, inaccessible, or looping measurement symlinks warn without revealing targets: %s", code => {
    fs.statfsSync.mockImplementation(location => {
      if (location === "/runner/work") throw privateError(code);
      return stats;
    });
    expect(logDiskUsage("setup")).toBe(true);
    expect(entries().find(entry => entry.resource === "workspace").availableSpaceBand).toBe("unavailable");
    expect(warning).toHaveBeenCalledWith(`::warning::disk_usage_logger: failed to sample workspace (${code})\n`);
    expect(JSON.stringify(warning.mock.calls)).not.toContain("/private");
  });

  it("uses the resolved target filesystem capacity for legitimate symlinked resources", () => {
    fs.statfsSync.mockImplementation(location => ({ ...stats, bsize: 1, bavail: location === "/runner/work" ? 0 : 10 * 1024 ** 3 }));
    logDiskUsage("interval");
    expect(entries().find(entry => entry.resource === "workspace").availableSpaceBand).toBe("exhausted");
    expect(entries().find(entry => entry.resource === "runner_temp").availableSpaceBand).toBe("healthy");
    expect(fs.statfsSync).toHaveBeenCalledWith("/runner/work");
  });

  it.each([
    [-1, 1],
    [Number.MAX_SAFE_INTEGER, 4096],
  ])("invalid statistics become warning-only unavailable telemetry", (bavail, bsize) => {
    fs.statfsSync.mockReturnValue({ ...stats, bavail, bsize });
    expect(logDiskUsage("interval")).toBe(true);
    expect(entries().every(entry => entry.availableSpaceBand === "unavailable")).toBe(true);
    expect(warning).toHaveBeenCalled();
  });

  it("isolates disk-full writes and never prints raw diagnostics", () => {
    writes.mockImplementation(() => {
      throw privateError("ENOSPC");
    });
    expect(logDiskUsage("interval")).toBe(false);
    expect(warning).toHaveBeenCalledWith("::warning::disk_usage_logger: failed to write disk usage (ENOSPC)\n");
    expect(JSON.stringify(warning.mock.calls)).not.toContain("123456789");
  });

  it("spawns one sampler without retaining credentials", () => {
    process.env.GH_TOKEN = "private-token";
    startDiskUsageMonitor();
    expect(childProcess.spawn).toHaveBeenCalledOnce();
    const [, args, options] = childProcess.spawn.mock.calls[0];
    expect(args[1]).toBe("monitor");
    expect(options.env).toEqual({ RUNNER_TEMP: "/runner/temp", GITHUB_WORKSPACE: "/runner/work" });
    expect(options.detached).toBe(true);
  });

  it.each(["open", "write", "spawn", "close"])("isolates %s failures during startup", failure => {
    if (failure === "open")
      fs.openSync.mockImplementation(() => {
        throw privateError("EACCES");
      });
    if (failure === "write")
      writes.mockImplementation(() => {
        throw privateError("ENOSPC");
      });
    if (failure === "spawn")
      childProcess.spawn.mockImplementation(() => {
        throw privateError("EIO");
      });
    if (failure === "close")
      fs.closeSync.mockImplementation(() => {
        throw privateError("EIO");
      });
    expect(() => startDiskUsageMonitor()).not.toThrow();
    expect(warning).toHaveBeenCalled();
    expect(JSON.stringify(warning.mock.calls)).not.toContain("/private");
  });

  it("isolates asynchronous spawn errors even when control removal also fails", () => {
    let spawnError;
    childProcess.spawn.mockReturnValue({
      on: vi.fn((name, callback) => {
        if (name === "error") spawnError = callback;
      }),
      unref: vi.fn(),
    });
    startDiskUsageMonitor();
    fs.unlinkSync.mockImplementation(() => {
      throw privateError("EPERM");
    });
    expect(() => spawnError(privateError("EIO"))).not.toThrow();
    expect(warning.mock.calls).toHaveLength(2);
  });

  it("does no 1Hz polling or expensive enumeration: four capacity calls and one compact write per 30 seconds", async () => {
    const readdir = vi.spyOn(fs, "readdirSync");
    const exec = vi.spyOn(childProcess, "execSync");
    const execFile = vi.spyOn(childProcess, "execFileSync");
    runDiskUsageMonitor();
    fs.readSync.mockClear();
    await vi.advanceTimersByTimeAsync(SAMPLE_INTERVAL_MS - 1);
    expect(fs.readSync).not.toHaveBeenCalled();
    expect(fs.statfsSync).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(1);
    expect(fs.readSync).toHaveBeenCalledOnce();
    expect(fs.statfsSync).toHaveBeenCalledTimes(4);
    expect(writes).toHaveBeenCalledOnce();
    expect(Buffer.byteLength(writes.mock.calls[0][1])).toBeLessThan(1024);
    expect(readdir).not.toHaveBeenCalled();
    expect(exec).not.toHaveBeenCalled();
    expect(execFile).not.toHaveBeenCalled();
    expect(childProcess.spawn).not.toHaveBeenCalled();
  });

  it("stops on filesystem notification before redaction, without waiting for the next sample", () => {
    runDiskUsageMonitor();
    fs.readSync.mockImplementation((fd, buffer) => {
      buffer.write("stop");
      return 4;
    });
    watchCallback();
    expect(entries()).toEqual([]);
    expect(watcher.close).toHaveBeenCalledOnce();
    expect(vi.getTimerCount()).toBe(0);
  });

  it.each(["watch", "read", "unlink", "watch-close"])("clears timers before warning-only %s cleanup failures", failure => {
    if (failure === "watch")
      fs.watch.mockImplementation(() => {
        throw privateError("ENOSYS");
      });
    runDiskUsageMonitor();
    if (failure === "read")
      fs.readSync.mockImplementation(() => {
        throw privateError("EIO");
      });
    if (failure === "unlink")
      fs.unlinkSync.mockImplementation(() => {
        throw privateError("EPERM");
      });
    if (failure === "watch-close")
      watcher.close.mockImplementation(() => {
        throw privateError("EIO");
      });
    if (failure !== "watch") expect(() => watchError(privateError("EIO"))).not.toThrow();
    expect(vi.getTimerCount()).toBe(0);
    expect(warning).toHaveBeenCalled();
  });

  it("uses the 30-second timer as a missed-notification fallback", async () => {
    runDiskUsageMonitor();
    fs.readSync.mockImplementation((fd, buffer) => {
      buffer.write("stop");
      return 4;
    });
    await vi.advanceTimersByTimeAsync(SAMPLE_INTERVAL_MS);
    expect(entries()).toEqual([]);
    expect(vi.getTimerCount()).toBe(0);
  });

  it("waits at most five seconds for acknowledgement", async () => {
    const stopped = stopDiskUsageMonitor();
    await vi.advanceTimersByTimeAsync(5000);
    await expect(stopped).resolves.toBeUndefined();
    expect(warning).toHaveBeenCalledWith(expect.stringContaining("did not acknowledge shutdown"));
    expect(vi.getTimerCount()).toBe(0);
  });

  it("acknowledges a stopped worker promptly and sanitizes telemetry before returning", async () => {
    const stopped = stopDiskUsageMonitor();
    for (const value of descriptors.values()) if (String(value.file).endsWith("disk-usage-monitor")) value.nlink = 0;
    await vi.advanceTimersByTimeAsync(100);
    await expect(stopped).resolves.toBeUndefined();
    expect(warning).not.toHaveBeenCalled();
    expect(entries().every(entry => entry.operation === "shutdown")).toBe(true);
    expect(writes.mock.calls.at(-1)[1]).toBe("stop");
    expect(fs.writeSync).toHaveBeenCalled();
    expect(vi.getTimerCount()).toBe(0);
  });

  it("stops rather than retrying after a sample write failure", async () => {
    runDiskUsageMonitor();
    writes.mockImplementation(() => {
      throw privateError("ENOSPC");
    });
    await vi.advanceTimersByTimeAsync(SAMPLE_INTERVAL_MS);
    expect(vi.getTimerCount()).toBe(0);
    const calls = writes.mock.calls.length;
    await vi.advanceTimersByTimeAsync(SAMPLE_INTERVAL_MS * 10);
    expect(writes.mock.calls).toHaveLength(calls);
    expect(warning).toHaveBeenCalled();
  });

  it("bounds telemetry size without growing or repeatedly retrying the file", async () => {
    runDiskUsageMonitor();
    const original = fs.fstatSync.getMockImplementation();
    fs.fstatSync.mockImplementation(fd => ({ ...original(fd), size: String(descriptors.get(fd)?.file).endsWith("disk_usage.jsonl") ? 16 * 1024 * 1024 : 0 }));
    await vi.advanceTimersByTimeAsync(SAMPLE_INTERVAL_MS);
    expect(writes).not.toHaveBeenCalled();
    expect(vi.getTimerCount()).toBe(0);
    expect(warning).toHaveBeenCalledWith(expect.stringContaining("telemetry size limit reached"));
  });

  it.each(["EACCES", "ENOSPC"])("stop failures resolve with sanitized warnings: %s", code => {
    writes.mockImplementation(() => {
      throw privateError(code);
    });
    return expect(stopDiskUsageMonitor()).resolves.toBeUndefined();
  });

  it("bounds abandoned sampler lifetime", async () => {
    runDiskUsageMonitor();
    vi.setSystemTime(Date.now() + 72 * 60 * 60 * 1000);
    await vi.advanceTimersByTimeAsync(SAMPLE_INTERVAL_MS);
    expect(entries()[0].operation).toBe("shutdown");
    expect(vi.getTimerCount()).toBe(0);
  });

  it.each(["disk_usage.jsonl", "disk_usage_monitor.log", "disk-usage-monitor"])("refuses malicious %s symlinks without opening their targets", name => {
    const original = fs.openSync.getMockImplementation();
    fs.openSync.mockImplementation((file, flags) => {
      if (String(file).endsWith(name)) {
        expect(flags & fs.constants.O_NOFOLLOW).not.toBe(0);
        throw privateError("ELOOP");
      }
      return original(file, flags);
    });
    expect(() => startDiskUsageMonitor()).not.toThrow();
    expect(childProcess.spawn).not.toHaveBeenCalled();
    expect(warning).toHaveBeenCalled();
    expect(fs.unlinkSync.mock.calls.every(([file]) => !String(file).includes("/private"))).toBe(true);
  });

  it("refuses non-regular files and hard links before truncation", () => {
    fs.fstatSync.mockReturnValue({ isDirectory: () => true, isFile: () => false, nlink: 2, size: 0 });
    expect(() => startDiskUsageMonitor()).not.toThrow();
    expect(fs.ftruncateSync).not.toHaveBeenCalled();
    expect(warning).toHaveBeenCalled();
  });

  it("fails closed when descriptor-anchored paths are unavailable", () => {
    fs.statSync.mockImplementation(() => {
      throw privateError("ENOENT");
    });
    expect(logDiskUsage("setup")).toBe(false);
    expect(() => startDiskUsageMonitor()).not.toThrow();
    expect(writes).not.toHaveBeenCalled();
    expect(warning).toHaveBeenCalled();
  });

  it("never traverses malicious runtime or runner-side directory symlinks", () => {
    fs.openSync.mockImplementation((file, flags) => {
      expect(flags & fs.constants.O_NOFOLLOW).not.toBe(0);
      throw privateError("ELOOP");
    });
    expect(logDiskUsage("setup")).toBe(false);
    expect(() => startDiskUsageMonitor()).not.toThrow();
    expect(writes).not.toHaveBeenCalled();
    expect(warning).toHaveBeenCalled();
  });

  it("rejects malicious control symlinks during stop and worker reads", async () => {
    const original = fs.openSync.getMockImplementation();
    fs.openSync.mockImplementation((file, flags) => {
      if (String(file).endsWith("disk-usage-monitor")) {
        expect(flags & fs.constants.O_NOFOLLOW).not.toBe(0);
        throw privateError("ELOOP");
      }
      return original(file, flags);
    });
    await expect(stopDiskUsageMonitor()).resolves.toBeUndefined();
    expect(() => runDiskUsageMonitor()).not.toThrow();
    expect(fs.readSync).not.toHaveBeenCalled();
    expect(warning).toHaveBeenCalled();
  });

  it("strips legacy exact fields and opaque paths from both primary/fallback and usage artifacts", () => {
    fs.readFileSync.mockReturnValue(
      JSON.stringify({
        timestamp: "2026-10-14T16:00:00Z",
        source: "/private/device",
        operation: "interval",
        resource: "runtime",
        availableSpaceBand: "critical",
        inodeHeadroom: "low",
        availableBytes: 123456789,
        totalBytes: 987654321,
        freeInodes: 555555,
        path: "/private/target",
        deviceId: "PRIVATE",
      }) + "\ninvalid /private/target\n"
    );
    collectDiskUsageArtifact();
    const primary = fs.writeSync.mock.calls[0][1];
    const usage = writes.mock.calls[0][1];
    expect(primary).toBe(usage);
    expect(primary).toContain('"availableSpaceBand":"critical"');
    for (const privateValue of ["123456789", "987654321", "555555", "/private", "PRIVATE", "Bytes", "Inodes"]) expect(primary).not.toContain(privateValue);
    expect(warning).toHaveBeenCalledWith(expect.stringContaining("discarded invalid disk record"));
  });

  it("rejects opaque labels and band values rather than copying them", () => {
    expect(normalizeDiskUsage({ operation: "interval", resource: "/private/target" })).toBeUndefined();
    expect(normalizeDiskUsage({ operation: "/private/target", resource: "runtime" })).toBeUndefined();
    expect(normalizeDiskUsage({ operation: "interval", resource: "runtime", availableSpaceBand: "/private/target" }).availableSpaceBand).toBe("unavailable");
  });

  it("strips raw legacy filesystem diagnostics and precise capacities from diagnostic artifacts", () => {
    const output = sanitizeDiskDiagnostics("EACCES /private/device 123456789 bytes\n::warning::disk_usage_logger: failed to sample runtime (ENOSPC)\n");
    expect(output).toContain("monitor diagnostic (ENOSPC)");
    expect(output).toContain("monitor diagnostic (UNAVAILABLE)");
    expect(output).not.toContain("/private");
    expect(output).not.toContain("123456789");
    expect(warning).toHaveBeenCalledWith(expect.stringContaining("discarded unsafe monitor diagnostics"));
  });

  it("artifact copy failures warn without affecting other collection", () => {
    fs.readFileSync.mockReturnValue('{"operation":"interval","resource":"runtime","availableSpaceBand":"healthy"}\n');
    writes.mockImplementation(() => {
      throw privateError("ENOSPC");
    });
    expect(() => collectDiskUsageArtifact()).not.toThrow();
    expect(warning).toHaveBeenCalledWith("::warning::disk_usage_logger: failed to collect disk usage artifact (ENOSPC)\n");
  });
});

it("the usage shell collector survives a missing or failing monitor runtime under errexit", () => {
  const block = collectorScript.match(/if \[ -f \/tmp\/gh-aw\/disk_usage.jsonl \]; then[\s\S]*?\nfi/)[0];
  const script = "set -euo pipefail\nRUNNER_TEMP=unused\nnode() { return 1; }\n" + block.replace("[ -f /tmp/gh-aw/disk_usage.jsonl ]", "true") + "\necho continued";
  const output = childProcess.execFileSync("/bin/bash", ["-c", script], { encoding: "utf8" });
  expect(output).toContain("::warning::Failed to collect disk capacity telemetry");
  expect(output).toContain("continued");
});

it("monitor startup is a warning-only successful CLI operation when runner control is unavailable", () => {
  const output = childProcess.spawnSync(process.execPath, [new URL("./disk_usage_logger.cjs", import.meta.url).pathname, "start"], { env: { RUNNER_TEMP: "" }, encoding: "utf8" });
  expect(output.status).toBe(0);
  expect(output.stdout).toContain("::warning::disk_usage_logger: failed to start disk monitor (UNAVAILABLE)");
  expect(output.stderr).not.toContain("Error:");
});
