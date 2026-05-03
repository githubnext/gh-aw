import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("parse_pi_log.cjs", () => {
  let mockCore;
  let parsePiLog, transformPiEntries;

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

    const module = await import("./parse_pi_log.cjs?" + Date.now());
    parsePiLog = module.parsePiLog;
    transformPiEntries = module.transformPiEntries;
  });

  afterEach(() => {
    delete global.core;
  });

  describe("parsePiLog function", () => {
    it("should return a default message for empty input", () => {
      const result = parsePiLog("");

      expect(result.markdown).toContain("No log content provided");
      expect(result.logEntries).toEqual([]);
      expect(result.mcpFailures).toEqual([]);
      expect(result.maxTurnsHit).toBe(false);
    });

    it("should return error message for null input", () => {
      const result = parsePiLog(null);

      expect(result.markdown).toContain("No log content provided");
    });

    it("should return unrecognized format message for non-JSON content", () => {
      const result = parsePiLog("plain text log content\nnot json at all");

      expect(result.markdown).toContain("Log format not recognized as Pi JSONL");
    });

    it("should parse init entry and show model in initialization section", () => {
      const logContent = [JSON.stringify({ type: "init", session_id: "sess-abc", model: "pi-3" })].join("\n");

      const result = parsePiLog(logContent);

      expect(result.markdown).toContain("## 🚀 Initialization");
      expect(result.markdown).toContain("pi-3");
      expect(result.markdown).toContain("sess-abc");
    });

    it("should merge consecutive assistant delta messages into one entry", () => {
      const logContent = [JSON.stringify({ type: "assistant", content: "I will analyze", delta: true }), JSON.stringify({ type: "assistant", content: " the repository.", delta: true })].join("\n");

      const result = parsePiLog(logContent);

      expect(result.markdown).toContain("## 🤖 Reasoning");
      expect(result.markdown).toContain("I will analyze the repository.");
    });

    it("should render tool use with success status", () => {
      const logContent = [
        JSON.stringify({ type: "tool_use", tool_name: "list_pull_requests", tool_id: "tool_001", parameters: { owner: "github", repo: "gh-aw" } }),
        JSON.stringify({ type: "tool_result", tool_id: "tool_001", status: "success", output: '{"items":[]}' }),
      ].join("\n");

      const result = parsePiLog(logContent);

      expect(result.markdown).toContain("✅");
      expect(result.markdown).toContain("list_pull_requests");
    });

    it("should render tool error when status is not success", () => {
      const logContent = [
        JSON.stringify({ type: "tool_use", tool_name: "read_file", tool_id: "tool_002", parameters: { path: "/nonexistent" } }),
        JSON.stringify({ type: "tool_result", tool_id: "tool_002", status: "error", output: "File not found" }),
      ].join("\n");

      const result = parsePiLog(logContent);

      expect(result.markdown).toContain("❌");
      expect(result.markdown).toContain("read_file");
    });

    it("should extract token usage from result stats", () => {
      const logContent = [JSON.stringify({ type: "result", stats: { input_tokens: 500, output_tokens: 200, duration_ms: 3000, turns: 2 } })].join("\n");

      const result = parsePiLog(logContent);

      // The Information section should include token counts
      expect(result.markdown).toContain("500");
      expect(result.markdown).toContain("200");
    });
  });

  describe("transformPiEntries function", () => {
    it("should transform init entry to system/init", () => {
      const raw = [{ type: "init", model: "pi-3", session_id: "s1" }];
      const entries = transformPiEntries(raw);

      expect(entries).toHaveLength(1);
      expect(entries[0].type).toBe("system");
      expect(entries[0].subtype).toBe("init");
      expect(entries[0].model).toBe("pi-3");
    });

    it("should transform assistant entry to canonical assistant type", () => {
      const raw = [{ type: "assistant", content: "Hello world", delta: false }];
      const entries = transformPiEntries(raw);

      expect(entries).toHaveLength(1);
      expect(entries[0].type).toBe("assistant");
      expect(entries[0].message.content[0].text).toBe("Hello world");
    });

    it("should merge consecutive delta assistant entries", () => {
      const raw = [
        { type: "assistant", content: "Part one", delta: true },
        { type: "assistant", content: " part two", delta: true },
      ];
      const entries = transformPiEntries(raw);

      expect(entries).toHaveLength(1);
      expect(entries[0].message.content[0].text).toBe("Part one part two");
    });

    it("should not merge non-delta assistant entries", () => {
      const raw = [
        { type: "assistant", content: "First message", delta: false },
        { type: "assistant", content: "Second message", delta: false },
      ];
      const entries = transformPiEntries(raw);

      expect(entries).toHaveLength(2);
    });

    it("should transform tool_use entry correctly", () => {
      const raw = [{ type: "tool_use", tool_name: "bash", tool_id: "t1", parameters: { command: "ls" } }];
      const entries = transformPiEntries(raw);

      expect(entries).toHaveLength(1);
      expect(entries[0].type).toBe("assistant");
      expect(entries[0].message.content[0].type).toBe("tool_use");
      expect(entries[0].message.content[0].name).toBe("bash");
      expect(entries[0].message.content[0].id).toBe("t1");
    });

    it("should transform tool_result entry correctly", () => {
      const raw = [{ type: "tool_result", tool_id: "t1", status: "success", output: "output text" }];
      const entries = transformPiEntries(raw);

      expect(entries).toHaveLength(1);
      expect(entries[0].type).toBe("user");
      expect(entries[0].message.content[0].type).toBe("tool_result");
      expect(entries[0].message.content[0].content).toBe("output text");
      expect(entries[0].message.content[0].is_error).toBe(false);
    });

    it("should mark tool_result as error when status is not success", () => {
      const raw = [{ type: "tool_result", tool_id: "t1", status: "error", output: "failed" }];
      const entries = transformPiEntries(raw);

      expect(entries[0].message.content[0].is_error).toBe(true);
    });

    it("should skip empty assistant content", () => {
      const raw = [
        { type: "assistant", content: "", delta: false },
        { type: "assistant", content: "   ", delta: false },
      ];
      const entries = transformPiEntries(raw);

      expect(entries).toHaveLength(0);
    });

    it("should ignore unknown event types", () => {
      const raw = [{ type: "unknown_event", data: "something" }];
      const entries = transformPiEntries(raw);

      expect(entries).toHaveLength(0);
    });
  });
});
