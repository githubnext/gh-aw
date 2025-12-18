import { describe, it, expect, beforeEach, vi } from "vitest";
describe("safe_outputs_mcp_client.cjs", () => {
  (describe("JSONL parsing", () => {
    (it("should parse simple JSONL input", () => {
      const result = '{"key":"value1"}\n{"key":"value2"}'
        .split(/\r?\n/)
        .map(l => l.trim())
        .filter(Boolean)
        .map(line => JSON.parse(line));
      (expect(result).toHaveLength(2), expect(result[0]).toEqual({ key: "value1" }), expect(result[1]).toEqual({ key: "value2" }));
    }),
      it("should handle empty input", () => {
        const parseJsonl = input =>
          input
            ? input
                .split(/\r?\n/)
                .map(l => l.trim())
                .filter(Boolean)
                .map(line => JSON.parse(line))
            : [];
        (expect(parseJsonl("")).toEqual([]), expect(parseJsonl(null)).toEqual([]), expect(parseJsonl(void 0)).toEqual([]));
      }),
      it("should skip empty lines", () => {
        const result = '{"key":"value1"}\n\n\n{"key":"value2"}\n'
          .split(/\r?\n/)
          .map(l => l.trim())
          .filter(Boolean)
          .map(line => JSON.parse(line));
        expect(result).toHaveLength(2);
      }),
      it("should handle Windows line endings", () => {
        const result = '{"key":"value1"}\r\n{"key":"value2"}\r\n'
          .split(/\r?\n/)
          .map(l => l.trim())
          .filter(Boolean)
          .map(line => JSON.parse(line));
        expect(result).toHaveLength(2);
      }),
      it("should handle whitespace in lines", () => {
        const result = '  {"key":"value1"}  \n  {"key":"value2"}  '
          .split(/\r?\n/)
          .map(l => l.trim())
          .filter(Boolean)
          .map(line => JSON.parse(line));
        (expect(result).toHaveLength(2), expect(result[0]).toEqual({ key: "value1" }));
      }));
  }),
    describe("message structure", () => {
      (it("should create valid JSON-RPC request", () => {
        const request = { jsonrpc: "2.0", id: 1, method: "test_method", params: { arg: "value" } };
        (expect(request).toHaveProperty("jsonrpc", "2.0"), expect(request).toHaveProperty("id", 1), expect(request).toHaveProperty("method", "test_method"), expect(request).toHaveProperty("params"));
      }),
        it("should handle notification messages (no id)", () => {
          const isNotification = msg => msg.method && !msg.id;
          (expect(isNotification({ method: "notify", params: {} })).toBe(!0), expect(isNotification({ jsonrpc: "2.0", id: 1, method: "request", params: {} })).toBe(!1));
        }),
        it("should identify response messages", () => {
          const isResponse = msg => void 0 !== msg.id && (void 0 !== msg.result || void 0 !== msg.error);
          (expect(isResponse({ id: 1, result: { data: "test" } })).toBe(!0), expect(isResponse({ id: 2, error: { message: "error" } })).toBe(!0), expect(isResponse({ id: 3, method: "test" })).toBe(!1));
        }));
    }),
    describe("error handling", () => {
      (it("should handle JSON parse errors gracefully", () => {
        const parseLine = line => {
          try {
            return { success: !0, data: JSON.parse(line) };
          } catch (e) {
            return { success: !1, error: e.message };
          }
        };
        (expect(parseLine('{"key":"value"}').success).toBe(!0), expect(parseLine("{invalid json}").success).toBe(!1));
      }),
        it("should handle error responses", () => {
          const handleResponse = (msg, pending) => (msg.error ? new Error(msg.error.message || JSON.stringify(msg.error)) : msg.result),
            errorResult = handleResponse({ id: 1, error: { message: "test error" } });
          (expect(errorResult).toBeInstanceOf(Error), expect(errorResult.message).toBe("test error"));
          const successResult = handleResponse({ id: 2, result: { data: "success" } });
          expect(successResult).toEqual({ data: "success" });
        }));
    }),
    describe("server path validation", () => {
      it("should construct valid server path", () => {
        const serverPath = require("path").join("/tmp/gh-aw/safeoutputs/mcp-server.cjs");
        (expect(serverPath).toContain("mcp-server.cjs"), expect(serverPath).toContain("safeoutputs"));
      });
    }),
    describe("buffer handling", () => {
      (it("should handle line extraction from buffer", () => {
        let buffer = Buffer.from('{"key":"value"}\n{"key2":"value2"}');
        const newlineIndex = buffer.indexOf("\n");
        expect(newlineIndex).toBeGreaterThan(-1);
        const line = buffer.slice(0, newlineIndex).toString("utf8"),
          remaining = buffer.slice(newlineIndex + 1);
        (expect(line).toBe('{"key":"value"}'), expect(remaining.toString()).toBe('{"key2":"value2"}'));
      }),
        it("should handle buffer without newline", () => {
          const newlineIndex = Buffer.from('{"key":"value"}').indexOf("\n");
          expect(newlineIndex).toBe(-1);
        }),
        it("should handle empty buffer", () => {
          const buffer = Buffer.alloc(0);
          expect(buffer.length).toBe(0);
        }));
    }));
});
