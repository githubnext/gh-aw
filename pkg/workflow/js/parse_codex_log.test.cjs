import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

describe("parse_codex_log.cjs", () => {
  let mockCore;
  let parseCodexLogScript;
  let originalConsole;
  let originalProcess;

  beforeEach(() => {
    // Save originals before mocking
    originalConsole = global.console;
    originalProcess = { ...process };

    // Mock console methods
    global.console = {
      log: vi.fn(),
      error: vi.fn(),
    };

    // Mock core actions methods
    mockCore = {
      // Core logging functions
      debug: vi.fn(),
      info: vi.fn(),
      notice: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),

      // Core workflow functions
      setFailed: vi.fn(),
      setOutput: vi.fn(),
      exportVariable: vi.fn(),
      setSecret: vi.fn(),

      // Input/state functions
      getInput: vi.fn(),
      getBooleanInput: vi.fn(),
      getMultilineInput: vi.fn(),
      getState: vi.fn(),
      saveState: vi.fn(),

      // Group functions
      startGroup: vi.fn(),
      endGroup: vi.fn(),
      group: vi.fn(),

      // Other utility functions
      addPath: vi.fn(),
      setCommandEcho: vi.fn(),
      isDebug: vi.fn().mockReturnValue(false),
      getIDToken: vi.fn(),
      toPlatformPath: vi.fn(),
      toPosixPath: vi.fn(),
      toWin32Path: vi.fn(),

      // Summary object with chainable methods
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(),
      },
    };
    global.core = mockCore;

    // Mock require
    global.require = vi.fn().mockImplementation(module => {
      if (module === "fs") {
        return fs;
      }
      if (module === "@actions/core") {
        return mockCore;
      }
      throw new Error(`Module not found: ${module}`);
    });

    // Read the script file
    const scriptPath = path.join(__dirname, "parse_codex_log.cjs");
    parseCodexLogScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Clean up environment variables
    delete process.env.GITHUB_AW_AGENT_OUTPUT;

    // Restore originals
    global.console = originalConsole;
    process.env = originalProcess.env;

    // Clean up globals
    delete global.core;
    delete global.require;
  });

  const runScript = async logContent => {
    // Create a temporary log file
    const tempFile = path.join(process.cwd(), `test_log_${Date.now()}.txt`);
    fs.writeFileSync(tempFile, logContent);
    process.env.GITHUB_AW_AGENT_OUTPUT = tempFile;

    try {
      // Evaluate the script in a function context
      const scriptFunc = new Function("require", "core", parseCodexLogScript);
      scriptFunc(global.require, mockCore);

      // Get the summary content
      const summaryContent = mockCore.summary.addRaw.mock.calls[0]?.[0] || "";
      return summaryContent;
    } finally {
      // Clean up temp file
      if (fs.existsSync(tempFile)) {
        fs.unlinkSync(tempFile);
      }
    }
  };

  it("should parse minimal Codex log with tool calls", async () => {
    const logContent = `[2025-08-31T12:37:08] OpenAI Codex v0.27.0
--------
workdir: /home/runner/work/gh-aw/gh-aw
model: o4-mini
--------
[2025-08-31T12:37:33] tool time.get_current_time({"timezone":"UTC"})
[2025-08-31T12:37:33] time.get_current_time success in 2ms
[2025-08-31T12:37:55] tokens used: 465`;

    const result = await runScript(logContent);

    expect(result).toContain("## 🤖 Commands and Tools");
    expect(result).toContain("✅ `time::get_current_time(...)`");
    expect(result).toContain("## 📊 Information");
    expect(result).toContain("**Total Tokens Used:** 465");
    expect(result).toContain("**Tool Calls:** 1");
    expect(result).toContain("## 🤖 Reasoning");
  });

  it("should parse Codex log with exec commands", async () => {
    const logContent = `[2025-08-31T12:37:49] exec bash -lc 'git remote -v' in /tmp
[2025-08-31T12:37:55] bash succeeded in 162ms`;

    const result = await runScript(logContent);

    expect(result).toContain("✅ `bash -lc 'git remote -v'`");
    expect(result).toContain("**Commands Executed:** 1");
  });

  it("should parse thinking sections", async () => {
    const logContent = `[2025-08-31T12:37:33] thinking

**Planning diff analysis**

I'm analyzing the code to understand the structure.
[2025-08-31T12:37:49] tool github.list_pull_requests({})`;

    const result = await runScript(logContent);

    expect(result).toContain("## 🤖 Reasoning");
    expect(result).toContain("**Planning diff analysis**");
    expect(result).toContain("I'm analyzing the code to understand the structure.");
  });

  it("should sum multiple token usage entries", async () => {
    const logContent = `[2025-08-31T12:37:33] tokens used: 14582
[2025-08-31T12:37:55] tokens used: 465
[2025-08-31T12:38:06] tokens used: 558`;

    const result = await runScript(logContent);

    expect(result).toContain("**Total Tokens Used:** 15,605");
  });

  it("should handle failed tool calls", async () => {
    const logContent = `[2025-08-31T12:37:33] tool github.get_issue({"issue_number":123})
[2025-08-31T12:37:33] github.get_issue failure in 100ms`;

    const result = await runScript(logContent);

    expect(result).toContain("❌ `github::get_issue(...)`");
  });

  it("should handle empty log", async () => {
    const logContent = "";

    const result = await runScript(logContent);

    expect(result).toContain("No commands or tools used.");
    expect(result).toContain("## 📊 Information");
    expect(result).toContain("## 🤖 Reasoning");
  });

  it("should skip metadata lines in reasoning section", async () => {
    const logContent = `[2025-08-31T12:37:08] OpenAI Codex v0.27.0
--------
workdir: /home/runner/work/gh-aw/gh-aw
model: o4-mini
provider: openai
--------
[2025-08-31T12:37:33] thinking

This is actual thinking content.`;

    const result = await runScript(logContent);

    expect(result).not.toContain("OpenAI Codex");
    expect(result).not.toContain("workdir:");
    expect(result).not.toContain("model:");
    expect(result).toContain("This is actual thinking content.");
  });
});
