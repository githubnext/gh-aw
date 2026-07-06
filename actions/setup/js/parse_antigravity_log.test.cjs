import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("parse_antigravity_log.cjs", () => {
  let mockCore;
  let parseAntigravityLog;

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

    const module = await import("./parse_antigravity_log.cjs?" + Date.now());
    parseAntigravityLog = module.parseAntigravityLog;
  });

  afterEach(() => {
    delete global.core;
  });

  describe("parseAntigravityLog function", () => {
    it("should return a default message for empty string input", () => {
      const result = parseAntigravityLog("");

      expect(result.markdown).toContain("No log content provided");
      expect(result.logEntries).toEqual([]);
      expect(result.mcpFailures).toEqual([]);
      expect(result.maxTurnsHit).toBe(false);
    });

    it("should return a default message for null input", () => {
      const result = parseAntigravityLog(null);

      expect(result.markdown).toContain("No log content provided");
      expect(result.logEntries).toEqual([]);
      expect(result.mcpFailures).toEqual([]);
      expect(result.maxTurnsHit).toBe(false);
    });

    it("should return unrecognized format message for non-JSON lines only", () => {
      const logContent = "plain text line\nnot json at all\ndebug: some message";

      const result = parseAntigravityLog(logContent);

      expect(result.markdown).toContain("Log format not recognized as Antigravity stream-json");
      expect(result.logEntries).toEqual([]);
    });

    it("should skip non-JSON lines mixed with valid JSONL", () => {
      const logContent = ["DEBUG: starting antigravity", JSON.stringify({ response: "Final answer", stats: {} }), "DEBUG: done"].join("\n");

      const result = parseAntigravityLog(logContent);

      expect(result.markdown).toContain("Final answer");
      expect(result.logEntries.some(e => e.type === "assistant.message")).toBe(true);
    });

    it("should use the last valid JSON line as the final response", () => {
      const logContent = [JSON.stringify({ response: "Partial answer", stats: {} }), JSON.stringify({ response: "Complete final answer", stats: {} })].join("\n");

      const result = parseAntigravityLog(logContent);

      expect(result.markdown).toContain("Complete final answer");
      expect(result.markdown).not.toContain("Partial answer");
      const assistantMsg = result.logEntries.find(e => e.type === "assistant.message");
      expect(assistantMsg).toBeDefined();
      expect(assistantMsg.data?.content).toBe("Complete final answer");
    });

    it("should aggregate token counts across multiple models", () => {
      const logContent = JSON.stringify({
        response: "Done",
        stats: {
          models: {
            "model-a": { input_tokens: 100, output_tokens: 50 },
            "model-b": { input_tokens: 200, output_tokens: 75 },
          },
        },
      });

      const result = parseAntigravityLog(logContent);

      // Total: 300 input, 125 output
      expect(result.markdown).toContain("300");
      expect(result.markdown).toContain("125");
    });

    it("should handle single JSONL line with response and stats", () => {
      const logContent = JSON.stringify({
        response: "Hello from Antigravity",
        stats: {
          models: {
            "gemini-2.0-flash": { input_tokens: 500, output_tokens: 200 },
          },
          tools: { bash: 3 },
        },
      });

      const result = parseAntigravityLog(logContent);

      expect(result.markdown).toContain("<summary>Antigravity</summary>");
      expect(result.markdown).toContain("Hello from Antigravity");
      expect(result.markdown).toContain("500");
      expect(result.markdown).toContain("200");
      expect(result.logEntries.some(e => e.type === "assistant.message")).toBe(true);
    });

    it("should handle missing stats gracefully", () => {
      const logContent = JSON.stringify({ response: "Response without stats" });

      const result = parseAntigravityLog(logContent);

      expect(result.markdown).toContain("Response without stats");
      expect(result.logEntries.some(e => e.type === "assistant.message")).toBe(true);
    });

    it("should handle empty response in the last JSONL entry", () => {
      const logContent = JSON.stringify({ response: "", stats: { models: {} } });

      const result = parseAntigravityLog(logContent);

      expect(result.markdown).toContain("<summary>Antigravity</summary>");
      expect(result.logEntries).toHaveLength(0);
    });

    it("should skip JSON lines that do not have a response string field", () => {
      const logContent = [JSON.stringify({ type: "debug", message: "starting" }), JSON.stringify({ response: "Real response", stats: {} })].join("\n");

      const result = parseAntigravityLog(logContent);

      expect(result.markdown).toContain("Real response");
    });
  });
});
