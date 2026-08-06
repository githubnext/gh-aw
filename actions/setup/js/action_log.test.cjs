import { describe, expect, it, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);
const { captureProcessOutput, createActionLogGroup, escapeWorkflowCommandMessage, logCapturedProcessOutput, logProcessOutputFiles, readFileTail, DEFAULT_CAPTURE_MAX_BYTES, DEFAULT_MAX_LOG_FILE_BYTES } = require("./action_log.cjs");

describe("action_log.cjs", () => {
  it("escapes workflow command messages", () => {
    expect(escapeWorkflowCommandMessage("title%\r\nnext")).toBe("title%25%0D%0Anext");
  });

  it("groups output with workflow commands disabled", () => {
    const writes = [];
    const output = {
      write: vi.fn(chunk => {
        writes.push(Buffer.isBuffer(chunk) ? chunk.toString() : chunk);
        return true;
      }),
    };

    const group = createActionLogGroup("Gateway logs", output);
    group.write("::error::not a workflow command");
    group.end();

    expect(writes[0]).toBe("::group::Gateway logs\n");
    expect(writes[1]).toMatch(/^::stop-commands::gh-aw-[0-9a-f-]+\n$/);
    expect(writes[2]).toBe("::error::not a workflow command");
    expect(writes[3]).toBe("\n");
    const token = writes[1].trim().replace("::stop-commands::", "");
    expect(writes[4]).toBe(`::${token}::\n`);
    expect(writes[5]).toBe("::endgroup::\n");
  });

  it("captures stdout and stderr in write order", () => {
    const capture = captureProcessOutput();
    process.stdout.write("stdout");
    process.stderr.write("stderr");
    capture.restore();

    expect(capture.entries.map(entry => [entry.stream, entry.chunk.toString()])).toEqual([
      ["stdout", "stdout"],
      ["stderr", "stderr"],
    ]);
  });

  it("does not truncate output under the configured cap", () => {
    const capture = captureProcessOutput({ maxBytes: 100 });
    process.stdout.write("hello");
    capture.restore();

    expect(capture.truncated).toBe(false);
    expect(capture.capturedBytes).toBe(5);
    expect(capture.totalBytes).toBe(5);
  });

  it("truncates captured output once the byte cap is exceeded", () => {
    const capture = captureProcessOutput({ maxBytes: 10 });
    process.stdout.write("12345678901234567890"); // 20 bytes
    process.stdout.write("more after truncation");
    capture.restore();

    expect(capture.truncated).toBe(true);
    expect(capture.capturedBytes).toBe(10);
    expect(capture.totalBytes).toBe(20 + "more after truncation".length);
    const capturedText = capture.entries.map(entry => entry.chunk.toString()).join("");
    expect(capturedText).toBe("1234567890");
    expect(capturedText.length).toBe(10);
  });

  it("uses a reasonable default cap when maxBytes is not specified", () => {
    const capture = captureProcessOutput();
    expect(capture.maxBytes).toBeGreaterThan(1024 * 1024);
    capture.restore();
  });

  it("logs captured entries after restoring process streams", () => {
    const stdoutWrite = vi.spyOn(process.stdout, "write").mockImplementation(() => true);
    try {
      logCapturedProcessOutput("Safe output logs", [
        { stream: "stdout", chunk: Buffer.from("hello\n") },
        { stream: "stderr", chunk: Buffer.from("::warning::inert\n") },
      ]);
      const output = stdoutWrite.mock.calls.map(call => String(call[0])).join("");
      expect(output).toContain("::group::Safe output logs\n");
      expect(output).toContain("hello\n::warning::inert\n");
      expect(output).toContain("::endgroup::\n");
    } finally {
      stdoutWrite.mockRestore();
    }
  });

  it("logs a truncated capture object with a safe truncation notice inside the protected group", () => {
    const stdoutWrite = vi.spyOn(process.stdout, "write").mockImplementation(() => true);
    try {
      logCapturedProcessOutput("Safe output logs", {
        entries: [{ stream: "stdout", chunk: Buffer.from("partial output\n") }],
        truncated: true,
        capturedBytes: 15,
        totalBytes: 5000,
        maxBytes: DEFAULT_CAPTURE_MAX_BYTES,
      });
      const calls = stdoutWrite.mock.calls.map(call => String(call[0]));
      const output = calls.join("");
      expect(output).toContain("::group::Safe output logs\n");
      const stopCommandsIndex = calls.findIndex(chunk => chunk.startsWith("::stop-commands::"));
      const endGroupIndex = calls.findIndex(chunk => chunk === "::endgroup::\n");
      const noticeIndex = calls.findIndex(chunk => chunk.includes("Captured output truncated"));
      expect(stopCommandsIndex).toBeGreaterThanOrEqual(0);
      expect(noticeIndex).toBeGreaterThan(stopCommandsIndex);
      expect(noticeIndex).toBeLessThan(endGroupIndex);
      expect(output).toContain("kept 15 of 5000 bytes");
      expect(output).toContain(`limit ${DEFAULT_CAPTURE_MAX_BYTES} bytes`);
    } finally {
      stdoutWrite.mockRestore();
    }
  });

  it("emits nothing when entries are empty and capture was not truncated", () => {
    const stdoutWrite = vi.spyOn(process.stdout, "write").mockImplementation(() => true);
    try {
      logCapturedProcessOutput("Safe output logs", { entries: [], truncated: false });
      expect(stdoutWrite).not.toHaveBeenCalled();
    } finally {
      stdoutWrite.mockRestore();
    }
  });

  it("logs files with explicit secret redaction", () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "action-log-"));
    const logPath = path.join(tempDir, "gateway-stderr.log");
    fs.writeFileSync(logPath, "token=secret-value\n");
    const stdoutWrite = vi.spyOn(process.stdout, "write").mockImplementation(() => true);
    try {
      logProcessOutputFiles("Gateway logs", [logPath], ["secret-value"]);
      const output = stdoutWrite.mock.calls.map(call => String(call[0])).join("");
      expect(output).toContain("--- gateway-stderr.log ---\ntoken=***\n");
      expect(output).not.toContain("secret-value");
    } finally {
      stdoutWrite.mockRestore();
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  it("readFileTail returns the whole file unmodified when under the byte cap", () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "action-log-"));
    const logPath = path.join(tempDir, "small.log");
    fs.writeFileSync(logPath, "hello world\n");
    try {
      const result = readFileTail(logPath, DEFAULT_MAX_LOG_FILE_BYTES);
      expect(result.truncated).toBe(false);
      expect(result.contents).toBe("hello world\n");
      expect(result.totalBytes).toBe(Buffer.byteLength("hello world\n"));
    } finally {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  it("readFileTail truncates to the trailing maxBytes when the file exceeds the cap", () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "action-log-"));
    const logPath = path.join(tempDir, "big.log");
    const head = "A".repeat(100);
    const tail = "B".repeat(50);
    fs.writeFileSync(logPath, head + tail);
    try {
      const result = readFileTail(logPath, 50);
      expect(result.truncated).toBe(true);
      expect(result.totalBytes).toBe(150);
      expect(result.contents).toBe(tail);
    } finally {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  it("logProcessOutputFiles emits a truncation marker and only the trailing bytes for oversized files", () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "action-log-"));
    const logPath = path.join(tempDir, "gateway-launch-stderr.log");
    const head = "old-line\n".repeat(20);
    const tail = "recent-error secret-value\n";
    fs.writeFileSync(logPath, head + tail);
    const stdoutWrite = vi.spyOn(process.stdout, "write").mockImplementation(() => true);
    try {
      logProcessOutputFiles("Gateway logs", [logPath], ["secret-value"], tail.length);
      const output = stdoutWrite.mock.calls.map(call => String(call[0])).join("");
      expect(output).toContain(`[truncated: showing last ${tail.length} of ${head.length + tail.length} bytes]`);
      expect(output).toContain("recent-error ***\n");
      expect(output).not.toContain("old-line");
      expect(output).not.toContain("secret-value");
    } finally {
      stdoutWrite.mockRestore();
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  it("logProcessOutputFiles does not truncate files under the default cap", () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "action-log-"));
    const logPath = path.join(tempDir, "gateway-launch-stderr.log");
    fs.writeFileSync(logPath, "short diagnostic line\n");
    const stdoutWrite = vi.spyOn(process.stdout, "write").mockImplementation(() => true);
    try {
      logProcessOutputFiles("Gateway logs", [logPath]);
      const output = stdoutWrite.mock.calls.map(call => String(call[0])).join("");
      expect(output).not.toContain("[truncated:");
      expect(output).toContain("short diagnostic line\n");
    } finally {
      stdoutWrite.mockRestore();
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });
});
