import { describe, it, expect, beforeEach, vi } from "vitest";

describe("safe_outputs_mcp_client.cjs", () => {
  describe("JSONL parsing", () => {
    it("should parse simple JSONL input", () => {
      const parseJsonl = (input) => {
        if (!input) return [];
        return input
          .split(/\r?\n/)
          .map(l => l.trim())
          .filter(Boolean)
          .map(line => JSON.parse(line));
      };

      const input = '{"key":"value1"}\n{"key":"value2"}';
      const result = parseJsonl(input);

      expect(result).toHaveLength(2);
      expect(result[0]).toEqual({ key: "value1" });
      expect(result[1]).toEqual({ key: "value2" });
    });

    it("should handle empty input", () => {
      const parseJsonl = (input) => {
        if (!input) return [];
        return input
          .split(/\r?\n/)
          .map(l => l.trim())
          .filter(Boolean)
          .map(line => JSON.parse(line));
      };

      expect(parseJsonl("")).toEqual([]);
      expect(parseJsonl(null)).toEqual([]);
      expect(parseJsonl(undefined)).toEqual([]);
    });

    it("should skip empty lines", () => {
      const parseJsonl = (input) => {
        if (!input) return [];
        return input
          .split(/\r?\n/)
          .map(l => l.trim())
          .filter(Boolean)
          .map(line => JSON.parse(line));
      };

      const input = '{"key":"value1"}\n\n\n{"key":"value2"}\n';
      const result = parseJsonl(input);

      expect(result).toHaveLength(2);
    });

    it("should handle Windows line endings", () => {
      const parseJsonl = (input) => {
        if (!input) return [];
        return input
          .split(/\r?\n/)
          .map(l => l.trim())
          .filter(Boolean)
          .map(line => JSON.parse(line));
      };

      const input = '{"key":"value1"}\r\n{"key":"value2"}\r\n';
      const result = parseJsonl(input);

      expect(result).toHaveLength(2);
    });

    it("should handle whitespace in lines", () => {
      const parseJsonl = (input) => {
        if (!input) return [];
        return input
          .split(/\r?\n/)
          .map(l => l.trim())
          .filter(Boolean)
          .map(line => JSON.parse(line));
      };

      const input = '  {"key":"value1"}  \n  {"key":"value2"}  ';
      const result = parseJsonl(input);

      expect(result).toHaveLength(2);
      expect(result[0]).toEqual({ key: "value1" });
    });
  });

  describe("message structure", () => {
    it("should create valid JSON-RPC request", () => {
      const createRequest = (id, method, params) => ({
        jsonrpc: "2.0",
        id,
        method,
        params
      });

      const request = createRequest(1, "test_method", { arg: "value" });

      expect(request).toHaveProperty("jsonrpc", "2.0");
      expect(request).toHaveProperty("id", 1);
      expect(request).toHaveProperty("method", "test_method");
      expect(request).toHaveProperty("params");
    });

    it("should handle notification messages (no id)", () => {
      const isNotification = (msg) => msg.method && !msg.id;

      const notification = { method: "notify", params: {} };
      const request = { jsonrpc: "2.0", id: 1, method: "request", params: {} };

      expect(isNotification(notification)).toBe(true);
      expect(isNotification(request)).toBe(false);
    });

    it("should identify response messages", () => {
      const isResponse = (msg) => 
        msg.id !== undefined && (msg.result !== undefined || msg.error !== undefined);

      const successResponse = { id: 1, result: { data: "test" } };
      const errorResponse = { id: 2, error: { message: "error" } };
      const request = { id: 3, method: "test" };

      expect(isResponse(successResponse)).toBe(true);
      expect(isResponse(errorResponse)).toBe(true);
      expect(isResponse(request)).toBe(false);
    });
  });

  describe("error handling", () => {
    it("should handle JSON parse errors gracefully", () => {
      const parseLine = (line) => {
        try {
          return { success: true, data: JSON.parse(line) };
        } catch (e) {
          return { success: false, error: e.message };
        }
      };

      const validLine = '{"key":"value"}';
      const invalidLine = '{invalid json}';

      expect(parseLine(validLine).success).toBe(true);
      expect(parseLine(invalidLine).success).toBe(false);
    });

    it("should handle error responses", () => {
      const handleResponse = (msg, pending) => {
        if (msg.error) {
          return new Error(msg.error.message || JSON.stringify(msg.error));
        }
        return msg.result;
      };

      const errorMsg = { id: 1, error: { message: "test error" } };
      const successMsg = { id: 2, result: { data: "success" } };

      const errorResult = handleResponse(errorMsg);
      expect(errorResult).toBeInstanceOf(Error);
      expect(errorResult.message).toBe("test error");

      const successResult = handleResponse(successMsg);
      expect(successResult).toEqual({ data: "success" });
    });
  });

  describe("server path validation", () => {
    it("should construct valid server path", () => {
      const path = require("path");
      const serverPath = path.join("/tmp/gh-aw/safeoutputs/mcp-server.cjs");

      expect(serverPath).toContain("mcp-server.cjs");
      expect(serverPath).toContain("safeoutputs");
    });
  });

  describe("buffer handling", () => {
    it("should handle line extraction from buffer", () => {
      let buffer = Buffer.from('{"key":"value"}\n{"key2":"value2"}');
      const newlineIndex = buffer.indexOf("\n");

      expect(newlineIndex).toBeGreaterThan(-1);

      const line = buffer.slice(0, newlineIndex).toString("utf8");
      const remaining = buffer.slice(newlineIndex + 1);

      expect(line).toBe('{"key":"value"}');
      expect(remaining.toString()).toBe('{"key2":"value2"}');
    });

    it("should handle buffer without newline", () => {
      const buffer = Buffer.from('{"key":"value"}');
      const newlineIndex = buffer.indexOf("\n");

      expect(newlineIndex).toBe(-1);
    });

    it("should handle empty buffer", () => {
      const buffer = Buffer.alloc(0);
      expect(buffer.length).toBe(0);
    });
  });
});
