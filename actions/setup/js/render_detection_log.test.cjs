import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

// Minimal mock for @actions/core used by github-script CJS modules.
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  setSecret: vi.fn(),
};

global.core = mockCore;

// Patch process.stdout.write so we can capture workflow-command output.
const originalWrite = process.stdout.write.bind(process.stdout);
let stdoutChunks = [];
const stubbedWrite = chunk => {
  stdoutChunks.push(typeof chunk === "string" ? chunk : chunk.toString("utf8"));
  return true;
};

/** Collected stdout output joined as a single string. */
function capturedStdout() {
  return stdoutChunks.join("");
}

describe("render_detection_log.cjs", () => {
  let module;
  let tempDir;
  let logPath;

  beforeEach(async () => {
    vi.clearAllMocks();
    stdoutChunks = [];
    process.stdout.write = stubbedWrite;

    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "render-detection-log-test-"));
    logPath = path.join(tempDir, "detection.log");

    // Re-import on each test so internal state is fresh.
    module = await import("./render_detection_log.cjs?t=" + Date.now());
  });

  afterEach(() => {
    process.stdout.write = originalWrite;
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  describe("main()", () => {
    it("is a no-op when the log file does not exist", async () => {
      await module.main("/nonexistent/path/detection.log");
      expect(capturedStdout()).toBe("");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Detection log not found"));
    });

    it("is a no-op when the log file is empty", async () => {
      fs.writeFileSync(logPath, "", "utf8");
      await module.main(logPath);
      expect(capturedStdout()).toBe("");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Detection log is empty"));
    });

    it("wraps content in ::group:: and ::endgroup::", async () => {
      fs.writeFileSync(logPath, "hello from detection\n", "utf8");
      await module.main(logPath);

      const out = capturedStdout();
      expect(out).toMatch(/^::group::Detection Log\n/);
      expect(out).toContain("::endgroup::\n");
    });

    it("wraps content in ::stop-commands::<token> and ::<token>::", async () => {
      fs.writeFileSync(logPath, "some log line\n", "utf8");
      await module.main(logPath);

      const out = capturedStdout();
      const stopMatch = out.match(/::stop-commands::(detection-log-[a-z0-9]+)\n/);
      expect(stopMatch).not.toBeNull();

      const token = stopMatch[1];
      expect(out).toContain("::" + token + "::\n");
    });

    it("places log content between stop-commands and endtoken lines", async () => {
      fs.writeFileSync(logPath, "the-log-content\n", "utf8");
      await module.main(logPath);

      const out = capturedStdout();
      const stopIdx = out.indexOf("::stop-commands::");
      const endTokenIdx = out.lastIndexOf("::");
      const contentIdx = out.indexOf("the-log-content");

      expect(contentIdx).toBeGreaterThan(stopIdx);
      expect(contentIdx).toBeLessThan(endTokenIdx);
    });

    it("applies redactBuiltInPatterns and replaces GitHub tokens", async () => {
      const fakeToken = "ghp_" + "A".repeat(36);
      fs.writeFileSync(logPath, "token=" + fakeToken + "\n", "utf8");
      await module.main(logPath);

      const out = capturedStdout();
      expect(out).not.toContain(fakeToken);
      expect(out).toContain("***REDACTED***");
    });

    it("appends a trailing newline when log content lacks one", async () => {
      fs.writeFileSync(logPath, "no-trailing-newline", "utf8");
      await module.main(logPath);

      const out = capturedStdout();
      // The stop-token end marker must appear at the start of its own line.
      expect(out).toMatch(/\n::detection-log-/);
    });

    it("logs the rendered byte count via core.info", async () => {
      fs.writeFileSync(logPath, "data\n", "utf8");
      await module.main(logPath);
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("bytes"));
    });

    it("uses the default DETECTION_LOG_PATH when no argument is supplied and file is absent", async () => {
      // No log at the default path in this test environment — just verify no crash.
      await expect(module.main()).resolves.toBeUndefined();
    });
  });

  describe("DETECTION_LOG_PATH constant", () => {
    it("exports the expected constant", () => {
      expect(module.DETECTION_LOG_PATH).toBe("/tmp/gh-aw/threat-detection/detection.log");
    });
  });
});
