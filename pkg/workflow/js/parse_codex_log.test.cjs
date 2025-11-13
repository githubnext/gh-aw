import { describe, it, expect, beforeEach, vi } from "vitest";

describe("parse_codex_log.cjs", () => {
  let mockCore;
  let parseCodexLog;
  let formatCodexToolCall;
  let formatCodexBashCall;
  let truncateString;
  let estimateTokens;
  let formatDuration;

  beforeEach(async () => {
    // Mock core actions methods
    mockCore = {
      debug: vi.fn(),
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
      setOutput: vi.fn(),
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(),
      },
    };
    global.core = mockCore;

    // Import the module
    const module = await import("./parse_codex_log.cjs");
    parseCodexLog = module.parseCodexLog;
    formatCodexToolCall = module.formatCodexToolCall;
    formatCodexBashCall = module.formatCodexBashCall;
    truncateString = module.truncateString;
    estimateTokens = module.estimateTokens;
    formatDuration = module.formatDuration;
  });

  describe("parseCodexLog function", () => {
    it("should parse basic tool call with success", () => {
      const logContent = `tool github.list_pull_requests({"state":"open"})
github.list_pull_requests(...) success in 123ms:
{"items": [{"number": 1}]}`;

      const result = parseCodexLog(logContent);

      expect(result).toContain("## 🤖 Reasoning");
      expect(result).toContain("## 🤖 Commands and Tools");
      expect(result).toContain("github::list_pull_requests");
      expect(result).toContain("✅");
    });

    it("should parse tool call with failure", () => {
      const logContent = `tool github.create_issue({"title":"Test"})
github.create_issue(...) failed in 456ms:
{"error": "permission denied"}`;

      const result = parseCodexLog(logContent);

      expect(result).toContain("github::create_issue");
      expect(result).toContain("❌");
    });

    it("should parse thinking sections", () => {
      const logContent = `thinking
I need to analyze the repository structure to understand the codebase
Let me start by listing the files in the root directory`;

      const result = parseCodexLog(logContent);

      expect(result).toContain("## 🤖 Reasoning");
      expect(result).toContain("I need to analyze the repository structure");
      expect(result).toContain("Let me start by listing the files");
    });

    it("should skip metadata lines", () => {
      const logContent = `OpenAI Codex v1.0
--------
workdir: /tmp/test
model: gpt-4
provider: openai
thinking
This is actual thinking content`;

      const result = parseCodexLog(logContent);

      expect(result).not.toContain("OpenAI Codex");
      expect(result).not.toContain("workdir");
      expect(result).not.toContain("model:");
      expect(result).toContain("This is actual thinking content");
    });

    it("should skip debug and timestamp lines", () => {
      const logContent = `DEBUG codex: starting session
2024-01-15T12:30:00.000Z DEBUG processing request
INFO codex: tool call completed
thinking
Actual thinking content that is long enough to be included`;

      const result = parseCodexLog(logContent);

      expect(result).not.toContain("DEBUG codex");
      expect(result).not.toContain("INFO codex");
      expect(result).toContain("Actual thinking content");
    });

    it("should parse bash commands", () => {
      const logContent = `[2024-01-15T12:30:00.000Z] exec bash -lc 'ls -la'
bash -lc 'ls -la' succeeded in 50ms:
total 8
-rw-r--r-- 1 user user 100 Jan 15 12:30 file.txt`;

      const result = parseCodexLog(logContent);

      expect(result).toContain("bash: ls -la");
      expect(result).toContain("✅");
    });

    it("should extract total tokens from log", () => {
      const logContent = `tool github.list_issues({})
total_tokens: 1500
tokens used
1,500`;

      const result = parseCodexLog(logContent);

      expect(result).toContain("📊 Information");
      expect(result).toContain("Total Tokens Used");
      expect(result).toContain("1,500");
    });

    it("should count tool calls", () => {
      const logContent = `ToolCall: github__list_issues {}
ToolCall: github__create_comment {}
ToolCall: github__add_labels {}`;

      const result = parseCodexLog(logContent);

      expect(result).toContain("**Tool Calls:** 3");
    });

    it("should handle empty log content", () => {
      const result = parseCodexLog("");

      expect(result).toContain("## 🤖 Reasoning");
      expect(result).toContain("## 🤖 Commands and Tools");
    });

    it("should handle log with errors gracefully", () => {
      // Mock core.error to prevent actual error logging in test output
      mockCore.error.mockImplementation(() => {});

      const malformedLog = null;
      const result = parseCodexLog(malformedLog);

      expect(result).toContain("Error parsing log content");
      expect(mockCore.error).toHaveBeenCalled();
    });

    it("should handle tool calls without responses", () => {
      const logContent = `tool github.list_issues({})`;

      const result = parseCodexLog(logContent);

      expect(result).toContain("github::list_issues");
      expect(result).toContain("❓"); // Unknown status
    });

    it("should filter out short lines in thinking sections", () => {
      const logContent = `thinking
Short
This is a long enough line to be included in the thinking section
x`;

      const result = parseCodexLog(logContent);

      expect(result).toContain("This is a long enough line");
      expect(result).not.toContain("Short\n\n");
      expect(result).not.toContain("x\n\n");
    });

    it("should handle ToolCall format", () => {
      const logContent = `ToolCall: github__create_issue {"title":"Test"}`;

      const result = parseCodexLog(logContent);

      expect(result).toContain("📊 Information");
      expect(result).toContain("**Tool Calls:** 1");
    });

    it("should handle tokens with commas in final count", () => {
      const logContent = `tokens used
12,345`;

      const result = parseCodexLog(logContent);

      expect(result).toContain("12,345");
    });
  });

  describe("formatCodexToolCall function", () => {
    it("should format tool call with response", () => {
      const result = formatCodexToolCall("github", "list_issues", '{"state":"open"}', '{"items":[]}', "✅");

      expect(result).toContain("<details>");
      expect(result).toContain("<summary>");
      expect(result).toContain("github::list_issues");
      expect(result).toContain("✅");
      expect(result).toContain("Parameters:");
      expect(result).toContain("Response:");
      expect(result).toContain("```json");
    });

    it("should format tool call without response", () => {
      const result = formatCodexToolCall("github", "create_issue", '{"title":"Test"}', "", "❌");

      expect(result).not.toContain("<details>");
      expect(result).toContain("github::create_issue");
      expect(result).toContain("❌");
    });

    it("should include token estimate", () => {
      const result = formatCodexToolCall("github", "get_issue", '{"number":123}', '{"title":"Test issue"}', "✅");

      expect(result).toMatch(/~\d+t/);
    });
  });

  describe("formatCodexBashCall function", () => {
    it("should format bash call with output", () => {
      const result = formatCodexBashCall("ls -la", "file1.txt\nfile2.txt", "✅");

      expect(result).toContain("<details>");
      expect(result).toContain("bash: ls -la");
      expect(result).toContain("✅");
      expect(result).toContain("Command:");
      expect(result).toContain("Output:");
    });

    it("should format bash call without output", () => {
      const result = formatCodexBashCall("mkdir test_dir", "", "✅");

      expect(result).not.toContain("<details>");
      expect(result).toContain("bash: mkdir test_dir");
      expect(result).toContain("✅");
    });

    it("should truncate long commands", () => {
      const longCommand = "echo " + "x".repeat(100);
      const result = formatCodexBashCall(longCommand, "output", "✅");

      expect(result).toContain("...");
      expect(result.split("...")[0].length).toBeLessThan(longCommand.length);
    });
  });

  describe("truncateString function", () => {
    it("should not truncate short strings", () => {
      expect(truncateString("hello", 10)).toBe("hello");
    });

    it("should truncate long strings", () => {
      expect(truncateString("hello world this is a long string", 10)).toBe("hello worl...");
    });

    it("should handle empty strings", () => {
      expect(truncateString("", 10)).toBe("");
    });

    it("should handle null/undefined", () => {
      expect(truncateString(null, 10)).toBe("");
      expect(truncateString(undefined, 10)).toBe("");
    });
  });

  describe("estimateTokens function", () => {
    it("should estimate tokens using 4 chars per token", () => {
      expect(estimateTokens("1234")).toBe(1);
      expect(estimateTokens("12345678")).toBe(2);
    });

    it("should handle empty strings", () => {
      expect(estimateTokens("")).toBe(0);
    });

    it("should handle null/undefined", () => {
      expect(estimateTokens(null)).toBe(0);
      expect(estimateTokens(undefined)).toBe(0);
    });

    it("should round up", () => {
      expect(estimateTokens("123")).toBe(1); // 3/4 = 0.75, rounds up to 1
      expect(estimateTokens("12345")).toBe(2); // 5/4 = 1.25, rounds up to 2
    });
  });

  describe("formatDuration function", () => {
    it("should format seconds", () => {
      expect(formatDuration(1000)).toBe("1s");
      expect(formatDuration(5000)).toBe("5s");
      expect(formatDuration(59000)).toBe("59s");
    });

    it("should format minutes", () => {
      expect(formatDuration(60000)).toBe("1m");
      expect(formatDuration(120000)).toBe("2m");
    });

    it("should format minutes and seconds", () => {
      expect(formatDuration(90000)).toBe("1m 30s");
      expect(formatDuration(125000)).toBe("2m 5s");
    });

    it("should handle zero or negative values", () => {
      expect(formatDuration(0)).toBe("");
      expect(formatDuration(-1000)).toBe("");
    });

    it("should handle null/undefined", () => {
      expect(formatDuration(null)).toBe("");
      expect(formatDuration(undefined)).toBe("");
    });

    it("should round to nearest second", () => {
      expect(formatDuration(1499)).toBe("1s");
      expect(formatDuration(1500)).toBe("2s");
    });
  });
});
