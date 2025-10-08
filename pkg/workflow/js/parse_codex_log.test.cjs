const { describe, it, expect } = require("vitest");
const { parseCodexLog, formatBashCommand, truncateString } = require("./parse_codex_log.cjs");

describe("parseCodexLog", () => {
  it("should parse minimal Codex log with tool calls", () => {
    const logContent = `[2025-08-31T12:37:08] OpenAI Codex v0.27.0
--------
workdir: /home/runner/work/gh-aw/gh-aw
model: o4-mini
--------
[2025-08-31T12:37:33] tool time.get_current_time({"timezone":"UTC"})
[2025-08-31T12:37:33] time.get_current_time success in 2ms
[2025-08-31T12:37:55] tokens used: 465`;

    const result = parseCodexLog(logContent);

    expect(result).toContain("## 🤖 Commands and Tools");
    expect(result).toContain("✅ `time::get_current_time(...)`");
    expect(result).toContain("## 📊 Information");
    expect(result).toContain("**Total Tokens Used:** 465");
    expect(result).toContain("**Tool Calls:** 1");
    expect(result).toContain("## 🤖 Reasoning");
  });

  it("should parse Codex log with exec commands", () => {
    const logContent = `[2025-08-31T12:37:49] exec bash -lc 'git remote -v' in /tmp
[2025-08-31T12:37:55] bash succeeded in 162ms`;

    const result = parseCodexLog(logContent);

    expect(result).toContain("✅ `bash -lc 'git remote -v'`");
    expect(result).toContain("**Commands Executed:** 1");
  });

  it("should parse thinking sections", () => {
    const logContent = `[2025-08-31T12:37:33] thinking

**Planning diff analysis**

I'm analyzing the code to understand the structure.
[2025-08-31T12:37:49] tool github.list_pull_requests({})`;

    const result = parseCodexLog(logContent);

    expect(result).toContain("## 🤖 Reasoning");
    expect(result).toContain("**Planning diff analysis**");
    expect(result).toContain("I'm analyzing the code to understand the structure.");
  });

  it("should sum multiple token usage entries", () => {
    const logContent = `[2025-08-31T12:37:33] tokens used: 14582
[2025-08-31T12:37:55] tokens used: 465
[2025-08-31T12:38:06] tokens used: 558`;

    const result = parseCodexLog(logContent);

    expect(result).toContain("**Total Tokens Used:** 15,605");
  });

  it("should handle failed tool calls", () => {
    const logContent = `[2025-08-31T12:37:33] tool github.get_issue({"issue_number":123})
[2025-08-31T12:37:33] github.get_issue failure in 100ms`;

    const result = parseCodexLog(logContent);

    expect(result).toContain("❌ `github::get_issue(...)`");
  });

  it("should handle empty log", () => {
    const logContent = "";

    const result = parseCodexLog(logContent);

    expect(result).toContain("No commands or tools used.");
    expect(result).toContain("## 📊 Information");
    expect(result).toContain("## 🤖 Reasoning");
  });

  it("should skip metadata lines in reasoning section", () => {
    const logContent = `[2025-08-31T12:37:08] OpenAI Codex v0.27.0
--------
workdir: /home/runner/work/gh-aw/gh-aw
model: o4-mini
provider: openai
--------
[2025-08-31T12:37:33] thinking

This is actual thinking content.`;

    const result = parseCodexLog(logContent);

    expect(result).not.toContain("OpenAI Codex");
    expect(result).not.toContain("workdir:");
    expect(result).not.toContain("model:");
    expect(result).toContain("This is actual thinking content.");
  });

  it("should handle long bash commands with truncation", () => {
    const longCommand = "a".repeat(100);
    const logContent = `[2025-08-31T12:37:49] exec ${longCommand} in /tmp
[2025-08-31T12:37:55] bash succeeded in 162ms`;

    const result = parseCodexLog(logContent);

    // Command should be truncated to 80 characters + "..."
    expect(result).toContain("...");
    expect(result).toMatch(/`a{80}\.\.\./);
  });
});

describe("formatBashCommand", () => {
  it("should format simple bash command", () => {
    const result = formatBashCommand("echo hello");
    expect(result).toBe("echo hello");
  });

  it("should remove newlines and collapse spaces", () => {
    const result = formatBashCommand("echo   hello\n  world\n");
    expect(result).toBe("echo hello world");
  });

  it("should escape backticks", () => {
    const result = formatBashCommand("echo `whoami`");
    expect(result).toBe("echo \\`whoami\\`");
  });

  it("should truncate long commands", () => {
    const longCommand = "a".repeat(100);
    const result = formatBashCommand(longCommand);
    expect(result).toHaveLength(83); // 80 chars + "..."
    expect(result).toEndWith("...");
  });

  it("should handle empty command", () => {
    const result = formatBashCommand("");
    expect(result).toBe("");
  });
});

describe("truncateString", () => {
  it("should not truncate short strings", () => {
    const result = truncateString("hello", 10);
    expect(result).toBe("hello");
  });

  it("should truncate long strings", () => {
    const result = truncateString("hello world", 5);
    expect(result).toBe("hello...");
  });

  it("should handle empty string", () => {
    const result = truncateString("", 10);
    expect(result).toBe("");
  });

  it("should handle null/undefined", () => {
    const result1 = truncateString(null, 10);
    const result2 = truncateString(undefined, 10);
    expect(result1).toBe("");
    expect(result2).toBe("");
  });
});
