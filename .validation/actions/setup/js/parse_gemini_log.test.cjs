import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("parse_gemini_log.cjs", () => {
  let mockCore;
  let parseGeminiLog, transformGeminiEntries;

  beforeEach(async () => {
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

    const module = await import("./parse_gemini_log.cjs?" + Date.now());
    parseGeminiLog = module.parseGeminiLog;
    transformGeminiEntries = module.transformGeminiEntries;
  });

  afterEach(() => {
    delete global.core;
  });

  describe("parseGeminiLog function", () => {
    it("should return a default message for empty input", () => {
      const result = parseGeminiLog("");

      expect(result.markdown).toContain("No log content provided");
      expect(result.logEntries).toEqual([]);
      expect(result.mcpFailures).toEqual([]);
      expect(result.maxTurnsHit).toBe(false);
    });

    it("should return error message for null input", () => {
      const result = parseGeminiLog(null);

      expect(result.markdown).toContain("No log content provided");
    });

    it("should return unrecognized format message for non-JSON content", () => {
      const result = parseGeminiLog("plain text log content\nnot json at all");

      expect(result.markdown).toContain("Log format not recognized as Gemini JSONL");
    });

    it("should parse init entry and show model in initialization section", () => {
      const logContent = [JSON.stringify({ type: "init", timestamp: "2026-01-01T00:00:00Z", session_id: "sess-123", model: "gemini-2.0-flash" })].join("\n");

      const result = parseGeminiLog(logContent);

      expect(result.markdown).toContain("## 🚀 Initialization");
      expect(result.markdown).toContain("gemini-2.0-flash");
      expect(result.markdown).toContain("sess-123");
    });

    it("should merge consecutive assistant delta messages into one reasoning block", () => {
      const logContent = [JSON.stringify({ type: "message", role: "assistant", content: "I will analyze", delta: true }), JSON.stringify({ type: "message", role: "assistant", content: " the repository.", delta: true })].join("\n");

      const result = parseGeminiLog(logContent);

      expect(result.markdown).toContain("## 🤖 Reasoning");
      expect(result.markdown).toContain("I will analyze the repository.");
    });

    it("should render tool use with success status", () => {
      const logContent = [
        JSON.stringify({ type: "tool_use", tool_name: "list_pull_requests", tool_id: "tool_001", parameters: { owner: "github", repo: "gh-aw" } }),
        JSON.stringify({ type: "tool_result", tool_id: "tool_001", status: "success", output: '{"items":[]}' }),
      ].join("\n");

      const result = parseGeminiLog(logContent);

      expect(result.markdown).toContain("✅");
      expect(result.markdown).toContain("list_pull_requests");
    });

    it("should render tool use with error status", () => {
      const logContent = [
        JSON.stringify({ type: "tool_use", tool_name: "create_issue", tool_id: "tool_002", parameters: { title: "Test" } }),
        JSON.stringify({ type: "tool_result", tool_id: "tool_002", status: "error", output: "Permission denied" }),
      ].join("\n");

      const result = parseGeminiLog(logContent);

      expect(result.markdown).toContain("❌");
      expect(result.markdown).toContain("create_issue");
    });

    it("should extract token stats from result entry", () => {
      const logContent = [
        JSON.stringify({
          type: "result",
          status: "success",
          stats: {
            total_tokens: 1000,
            input_tokens: 900,
            output_tokens: 100,
            cached: 200,
            duration_ms: 5000,
            tool_calls: 3,
          },
        }),
      ].join("\n");

      const result = parseGeminiLog(logContent);

      expect(result.markdown).toContain("## 📊 Information");
      expect(result.markdown).toContain("900");
      expect(result.markdown).toContain("100");
    });

    it("should parse a complete conversation flow", () => {
      const logContent = [
        JSON.stringify({ type: "init", timestamp: "2026-01-01T00:00:00Z", session_id: "sess-abc", model: "auto-gemini-3" }),
        JSON.stringify({ type: "message", role: "user", content: "Please list PRs." }),
        JSON.stringify({ type: "message", role: "assistant", content: "I will list the PRs.", delta: true }),
        JSON.stringify({ type: "tool_use", tool_name: "list_pull_requests", tool_id: "tool_003", parameters: { owner: "github", repo: "gh-aw" } }),
        JSON.stringify({ type: "tool_result", tool_id: "tool_003", status: "success", output: '{"items":[{"number":1}]}' }),
        JSON.stringify({ type: "message", role: "assistant", content: "Found 1 PR.", delta: true }),
        JSON.stringify({ type: "result", status: "success", stats: { total_tokens: 500, input_tokens: 400, output_tokens: 100, cached: 50, duration_ms: 3000, tool_calls: 1 } }),
      ].join("\n");

      const result = parseGeminiLog(logContent);

      expect(result.markdown).toContain("## 🚀 Initialization");
      expect(result.markdown).toContain("auto-gemini-3");
      expect(result.markdown).toContain("## 🤖 Reasoning");
      expect(result.markdown).toContain("I will list the PRs.");
      expect(result.markdown).toContain("## 🤖 Commands and Tools");
      expect(result.markdown).toContain("list_pull_requests");
      expect(result.markdown).toContain("## 📊 Information");
      expect(result.logEntries.length).toBeGreaterThan(0);
      expect(result.mcpFailures).toEqual([]);
      expect(result.maxTurnsHit).toBe(false);
    });

    it("should skip non-JSON lines in the log", () => {
      const logContent = ["[INFO] Starting agent", JSON.stringify({ type: "init", session_id: "sess-xyz", model: "gemini-pro" }), "[INFO] Agent complete"].join("\n");

      const result = parseGeminiLog(logContent);

      expect(result.markdown).toContain("gemini-pro");
      expect(result.markdown).not.toContain("[INFO]");
    });
  });

  describe("transformGeminiEntries function", () => {
    it("should transform init entry to system init format", () => {
      const raw = [{ type: "init", session_id: "sess-1", model: "gemini-flash" }];

      const entries = transformGeminiEntries(raw);

      expect(entries).toHaveLength(1);
      expect(entries[0].type).toBe("system");
      expect(entries[0].subtype).toBe("init");
      expect(entries[0].model).toBe("gemini-flash");
      expect(entries[0].session_id).toBe("sess-1");
    });

    it("should merge consecutive delta assistant messages", () => {
      const raw = [
        { type: "message", role: "assistant", content: "Hello", delta: true },
        { type: "message", role: "assistant", content: " world", delta: true },
        { type: "message", role: "assistant", content: "!", delta: true },
      ];

      const entries = transformGeminiEntries(raw);

      expect(entries).toHaveLength(1);
      expect(entries[0].type).toBe("assistant");
      expect(entries[0].message.content[0].text).toBe("Hello world!");
    });

    it("should not merge non-consecutive delta messages", () => {
      const raw = [
        { type: "message", role: "assistant", content: "First message.", delta: true },
        { type: "tool_use", tool_name: "bash", tool_id: "t1", parameters: {} },
        { type: "message", role: "assistant", content: "Second message.", delta: true },
      ];

      const entries = transformGeminiEntries(raw);

      const assistantEntries = entries.filter(e => e.type === "assistant" && e.message?.content?.[0]?.type === "text");
      expect(assistantEntries).toHaveLength(2);
      expect(assistantEntries[0].message.content[0].text).toBe("First message.");
      expect(assistantEntries[1].message.content[0].text).toBe("Second message.");
    });

    it("should transform tool_use to assistant entry", () => {
      const raw = [{ type: "tool_use", tool_name: "search_code", tool_id: "tool_abc", parameters: { query: "test" } }];

      const entries = transformGeminiEntries(raw);

      expect(entries).toHaveLength(1);
      expect(entries[0].type).toBe("assistant");
      expect(entries[0].message.content[0].type).toBe("tool_use");
      expect(entries[0].message.content[0].id).toBe("tool_abc");
      expect(entries[0].message.content[0].name).toBe("search_code");
      expect(entries[0].message.content[0].input).toEqual({ query: "test" });
    });

    it("should transform tool_result to user entry with success status", () => {
      const raw = [{ type: "tool_result", tool_id: "tool_abc", status: "success", output: "result data" }];

      const entries = transformGeminiEntries(raw);

      expect(entries).toHaveLength(1);
      expect(entries[0].type).toBe("user");
      expect(entries[0].message.content[0].type).toBe("tool_result");
      expect(entries[0].message.content[0].tool_use_id).toBe("tool_abc");
      expect(entries[0].message.content[0].content).toBe("result data");
      expect(entries[0].message.content[0].is_error).toBe(false);
    });

    it("should transform tool_result to user entry with error status", () => {
      const raw = [{ type: "tool_result", tool_id: "tool_xyz", status: "error", output: "Something went wrong" }];

      const entries = transformGeminiEntries(raw);

      expect(entries[0].message.content[0].is_error).toBe(true);
    });

    it("should skip user messages and result entries", () => {
      const raw = [
        { type: "message", role: "user", content: "User prompt" },
        { type: "result", status: "success", stats: {} },
      ];

      const entries = transformGeminiEntries(raw);

      expect(entries).toHaveLength(0);
    });

    it("should skip empty assistant delta messages", () => {
      const raw = [
        { type: "message", role: "assistant", content: "", delta: true },
        { type: "message", role: "assistant", content: "   ", delta: true },
        { type: "message", role: "assistant", content: "Valid content", delta: true },
      ];

      const entries = transformGeminiEntries(raw);

      expect(entries).toHaveLength(1);
      expect(entries[0].message.content[0].text).toBe("Valid content");
    });

    it("should serialize non-string tool_result output as JSON", () => {
      const raw = [{ type: "tool_result", tool_id: "t1", status: "success", output: { items: [1, 2] } }];

      const entries = transformGeminiEntries(raw);

      expect(entries[0].message.content[0].content).toBe('{"items":[1,2]}');
    });
  });
});
