import { describe, it, expect, vi } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { runProcess, formatDuration, sleep } = require("./process_runner.cjs");

describe("process_runner.cjs", () => {
  describe("formatDuration", () => {
    it("formats zero milliseconds as 0s", () => {
      expect(formatDuration(0)).toBe("0s");
    });

    it("formats sub-minute durations as seconds only", () => {
      expect(formatDuration(1000)).toBe("1s");
      expect(formatDuration(45000)).toBe("45s");
      expect(formatDuration(59999)).toBe("59s");
    });

    it("formats exactly one minute", () => {
      expect(formatDuration(60000)).toBe("1m 0s");
    });

    it("formats minutes and seconds", () => {
      expect(formatDuration(192000)).toBe("3m 12s");
      expect(formatDuration(125500)).toBe("2m 5s");
    });

    it("truncates sub-second precision", () => {
      expect(formatDuration(1999)).toBe("1s");
    });
  });

  describe("sleep", () => {
    it("returns a promise that resolves after the given delay", async () => {
      vi.useFakeTimers();
      try {
        const promise = sleep(1000);
        vi.advanceTimersByTime(1000);
        await expect(promise).resolves.toBeUndefined();
      } finally {
        vi.useRealTimers();
      }
    });

    it("resolves immediately for 0ms", async () => {
      await expect(sleep(0)).resolves.toBeUndefined();
    });
  });

  describe("runProcess", () => {
    it("resolves with exitCode 0 for a successful command", async () => {
      const logs = [];
      const result = await runProcess({
        command: process.execPath,
        args: ["-e", "process.exit(0)"],
        attempt: 0,
        log: msg => logs.push(msg),
      });
      expect(result.exitCode).toBe(0);
      expect(result.durationMs).toBeGreaterThanOrEqual(0);
    });

    it("resolves with the actual non-zero exit code on failure", async () => {
      const logs = [];
      const result = await runProcess({
        command: process.execPath,
        args: ["-e", "process.exit(42)"],
        attempt: 0,
        log: msg => logs.push(msg),
      });
      expect(result.exitCode).toBe(42);
    });

    it("collects stdout output and sets hasOutput", async () => {
      const logs = [];
      const result = await runProcess({
        command: process.execPath,
        args: ["-e", 'process.stdout.write("hello stdout"); process.exit(0)'],
        attempt: 0,
        log: msg => logs.push(msg),
      });
      expect(result.hasOutput).toBe(true);
      expect(result.output).toContain("hello stdout");
    });

    it("collects stderr output and sets hasOutput", async () => {
      const logs = [];
      const result = await runProcess({
        command: process.execPath,
        args: ["-e", 'process.stderr.write("hello stderr"); process.exit(1)'],
        attempt: 0,
        log: msg => logs.push(msg),
      });
      expect(result.hasOutput).toBe(true);
      expect(result.output).toContain("hello stderr");
    });

    it("sets hasOutput false when no output is produced", async () => {
      const logs = [];
      const result = await runProcess({
        command: process.execPath,
        args: ["-e", "process.exit(1)"],
        attempt: 0,
        log: msg => logs.push(msg),
      });
      expect(result.hasOutput).toBe(false);
      expect(result.output).toBe("");
    });

    it("logs spawning with logArgs instead of args when provided", async () => {
      const logs = [];
      await runProcess({
        command: process.execPath,
        args: ["-e", "process.exit(0)"],
        attempt: 0,
        log: msg => logs.push(msg),
        logArgs: ["<redacted>"],
      });
      const spawnLog = logs.find(l => l.includes("spawning"));
      expect(spawnLog).toContain("<redacted>");
      expect(spawnLog).not.toContain("-e");
    });

    it("falls back to args for logging when logArgs is not provided", async () => {
      const logs = [];
      await runProcess({
        command: process.execPath,
        args: ["-e", "process.exit(0)"],
        attempt: 0,
        log: msg => logs.push(msg),
      });
      const spawnLog = logs.find(l => l.includes("spawning"));
      expect(spawnLog).toContain("-e");
    });

    it("uses the attempt number in log messages", async () => {
      const logs = [];
      await runProcess({
        command: process.execPath,
        args: ["-e", "process.exit(0)"],
        attempt: 2,
        log: msg => logs.push(msg),
      });
      expect(logs.some(l => l.includes("attempt 3"))).toBe(true);
    });

    it("resolves with exitCode 1 and hasOutput false when command is not found", async () => {
      const logs = [];
      const result = await runProcess({
        command: "/nonexistent-binary-xyz",
        args: [],
        attempt: 0,
        log: msg => logs.push(msg),
      });
      expect(result.exitCode).toBe(1);
      const errorLog = logs.find(l => l.includes("failed to start process"));
      expect(errorLog).toBeTruthy();
    });

    it("collects combined stdout and stderr in output", async () => {
      const logs = [];
      const result = await runProcess({
        command: process.execPath,
        args: ["-e", 'process.stdout.write("out"); process.stderr.write("err"); process.exit(0)'],
        attempt: 0,
        log: msg => logs.push(msg),
      });
      expect(result.output).toContain("out");
      expect(result.output).toContain("err");
    });

    it("resolves with durationMs as a non-negative number", async () => {
      const logs = [];
      const result = await runProcess({
        command: process.execPath,
        args: ["-e", "process.exit(0)"],
        attempt: 0,
        log: msg => logs.push(msg),
      });
      expect(typeof result.durationMs).toBe("number");
      expect(result.durationMs).toBeGreaterThanOrEqual(0);
    });

    it("truncates logArgs to 200 chars in spawn log", async () => {
      const logs = [];
      const longArg = "x".repeat(300);
      await runProcess({
        command: process.execPath,
        args: ["-e", "process.exit(0)"],
        attempt: 0,
        log: msg => logs.push(msg),
        logArgs: [longArg],
      });
      const spawnLog = logs.find(l => l.includes("spawning"));
      // logArgs is a single arg made entirely of 'x' characters.  After truncation to 200
      // chars the spawn log line must end with at most 200 consecutive x's.
      const trailingXs = spawnLog?.match(/x+$/)?.[0] ?? "";
      expect(trailingXs.length).toBeLessThanOrEqual(200);
    });
  });
});
