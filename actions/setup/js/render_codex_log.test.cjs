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

describe("render_codex_log.cjs", () => {
  let module;
  let tempDir;
  let codexHome;
  let originalOutcome;
  let originalCodexHome;

  beforeEach(async () => {
    vi.clearAllMocks();
    stdoutChunks = [];
    process.stdout.write = stubbedWrite;

    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "render-codex-log-test-"));
    codexHome = path.join(tempDir, "codex-home");
    fs.mkdirSync(path.join(codexHome, "logs"), { recursive: true });

    originalOutcome = process.env.GH_AW_AGENTIC_EXECUTION_OUTCOME;
    originalCodexHome = process.env.CODEX_HOME;

    // Re-import on each test so internal state is fresh.
    module = await import("./render_codex_log.cjs?t=" + Date.now());
  });

  afterEach(() => {
    process.stdout.write = originalWrite;
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
    if (originalOutcome === undefined) {
      delete process.env.GH_AW_AGENTIC_EXECUTION_OUTCOME;
    } else {
      process.env.GH_AW_AGENTIC_EXECUTION_OUTCOME = originalOutcome;
    }
    if (originalCodexHome === undefined) {
      delete process.env.CODEX_HOME;
    } else {
      process.env.CODEX_HOME = originalCodexHome;
    }
  });

  describe("findMostRecentLogFile()", () => {
    it("returns undefined when the directory does not exist", () => {
      expect(module.findMostRecentLogFile(path.join(tempDir, "missing"))).toBeUndefined();
    });

    it("returns undefined when no *.log files are present", () => {
      fs.writeFileSync(path.join(codexHome, "logs", "notes.txt"), "hi", "utf8");
      expect(module.findMostRecentLogFile(path.join(codexHome, "logs"))).toBeUndefined();
    });

    it("finds a *.log file nested in subdirectories", () => {
      const nested = path.join(codexHome, "logs", "session-1");
      fs.mkdirSync(nested, { recursive: true });
      const nestedLog = path.join(nested, "codex.log");
      fs.writeFileSync(nestedLog, "nested log content\n", "utf8");

      expect(module.findMostRecentLogFile(path.join(codexHome, "logs"))).toBe(nestedLog);
    });

    it("picks the most recently modified *.log file", async () => {
      const older = path.join(codexHome, "logs", "older.log");
      const newer = path.join(codexHome, "logs", "newer.log");
      fs.writeFileSync(older, "older\n", "utf8");
      // Ensure a distinct mtime ordering regardless of filesystem timestamp resolution.
      await new Promise(resolve => setTimeout(resolve, 10));
      fs.writeFileSync(newer, "newer\n", "utf8");
      fs.utimesSync(older, new Date(Date.now() - 60_000), new Date(Date.now() - 60_000));

      expect(module.findMostRecentLogFile(path.join(codexHome, "logs"))).toBe(newer);
    });
  });

  describe("main()", () => {
    it("is a no-op when the execution outcome was not 'failure'", async () => {
      process.env.GH_AW_AGENTIC_EXECUTION_OUTCOME = "success";
      process.env.CODEX_HOME = codexHome;
      fs.writeFileSync(path.join(codexHome, "logs", "codex.log"), "should not be rendered\n", "utf8");

      await module.main();
      expect(capturedStdout()).toBe("");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("skipping internal log render"));
    });

    it("is a no-op when CODEX_HOME is not set", async () => {
      process.env.GH_AW_AGENTIC_EXECUTION_OUTCOME = "failure";
      delete process.env.CODEX_HOME;

      await module.main();
      expect(capturedStdout()).toBe("");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("CODEX_HOME not set"));
    });

    it("is a no-op when no log files exist under $CODEX_HOME/logs", async () => {
      process.env.GH_AW_AGENTIC_EXECUTION_OUTCOME = "failure";
      process.env.CODEX_HOME = codexHome;

      await module.main();
      expect(capturedStdout()).toBe("");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No Codex internal log files found"));
    });

    it("renders the most recent log file wrapped in group + stop-commands macros on failure", async () => {
      process.env.GH_AW_AGENTIC_EXECUTION_OUTCOME = "failure";
      process.env.CODEX_HOME = codexHome;
      fs.writeFileSync(path.join(codexHome, "logs", "codex.log"), "codex crashed here\n", "utf8");

      await module.main();

      const out = capturedStdout();
      expect(out).toMatch(/^::group::Codex internal logs \(/);
      expect(out).toContain("codex crashed here");
      const stopMatch = out.match(/::stop-commands::(render-[a-f0-9]+)\n/);
      expect(stopMatch).not.toBeNull();
      expect(out).toContain("::" + stopMatch[1] + "::\n");
      expect(out).toContain("::endgroup::\n");
    });
  });
});
