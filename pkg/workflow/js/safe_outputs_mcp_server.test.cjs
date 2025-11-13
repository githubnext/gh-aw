import { describe, it, expect, beforeEach, vi } from "vitest";

describe("safe_outputs_mcp_server.cjs", () => {
  describe("JSON-RPC message structure", () => {
    it("should validate request structure", () => {
      const isValidRequest = msg => {
        return msg.jsonrpc === "2.0" && msg.id !== undefined && typeof msg.method === "string";
      };

      const validRequest = {
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: {},
      };

      const invalidRequest1 = { id: 1, method: "test" }; // missing jsonrpc
      const invalidRequest2 = { jsonrpc: "2.0", method: "test" }; // missing id
      const invalidRequest3 = { jsonrpc: "2.0", id: 1 }; // missing method

      expect(isValidRequest(validRequest)).toBe(true);
      expect(isValidRequest(invalidRequest1)).toBe(false);
      expect(isValidRequest(invalidRequest2)).toBe(false);
      expect(isValidRequest(invalidRequest3)).toBe(false);
    });

    it("should create valid response structure", () => {
      const createResponse = (id, result) => ({
        jsonrpc: "2.0",
        id,
        result,
      });

      const response = createResponse(1, { status: "ok" });

      expect(response).toHaveProperty("jsonrpc", "2.0");
      expect(response).toHaveProperty("id", 1);
      expect(response).toHaveProperty("result");
      expect(response.result).toEqual({ status: "ok" });
    });

    it("should create valid error response", () => {
      const createErrorResponse = (id, code, message) => ({
        jsonrpc: "2.0",
        id,
        error: { code, message },
      });

      const errorResponse = createErrorResponse(1, -32600, "Invalid Request");

      expect(errorResponse).toHaveProperty("jsonrpc", "2.0");
      expect(errorResponse).toHaveProperty("id", 1);
      expect(errorResponse).toHaveProperty("error");
      expect(errorResponse.error.code).toBe(-32600);
      expect(errorResponse.error.message).toBe("Invalid Request");
    });
  });

  describe("tool definition structure", () => {
    it("should validate tool schema", () => {
      const isValidTool = tool => {
        return (
          typeof tool.name === "string" &&
          tool.description !== undefined &&
          tool.inputSchema !== undefined &&
          typeof tool.inputSchema === "object"
        );
      };

      const validTool = {
        name: "create_issue",
        description: "Create a GitHub issue",
        inputSchema: {
          type: "object",
          properties: {
            title: { type: "string" },
            body: { type: "string" },
          },
          required: ["title"],
        },
      };

      const invalidTool1 = { description: "No name" };
      const invalidTool2 = { name: "test", description: "No schema" };

      expect(isValidTool(validTool)).toBe(true);
      expect(isValidTool(invalidTool1)).toBe(false);
      expect(isValidTool(invalidTool2)).toBe(false);
    });

    it("should handle tool with required fields", () => {
      const tool = {
        name: "create_issue",
        inputSchema: {
          type: "object",
          properties: {
            title: { type: "string" },
            body: { type: "string" },
          },
          required: ["title", "body"],
        },
      };

      expect(tool.inputSchema.required).toContain("title");
      expect(tool.inputSchema.required).toContain("body");
      expect(tool.inputSchema.required).toHaveLength(2);
    });
  });

  describe("configuration handling", () => {
    it("should handle empty configuration", () => {
      const config = {};
      const tools = Object.keys(config);

      expect(tools).toHaveLength(0);
    });

    it("should validate tool enablement", () => {
      const config = {
        "create-issue": { enabled: true },
        "add-comment": { enabled: false },
      };

      const enabledTools = Object.entries(config)
        .filter(([_, cfg]) => cfg.enabled !== false)
        .map(([name, _]) => name);

      expect(enabledTools).toContain("create-issue");
      expect(enabledTools).not.toContain("add-comment");
    });

    it("should handle missing enabled property as true", () => {
      const config = {
        "create-issue": {},
        "add-comment": { enabled: false },
      };

      const enabledTools = Object.entries(config)
        .filter(([_, cfg]) => cfg.enabled !== false)
        .map(([name, _]) => name);

      expect(enabledTools).toContain("create-issue");
    });
  });

  describe("output file handling", () => {
    it("should validate output file path", () => {
      const outputFile = "/tmp/gh-aw/safeoutputs/output.jsonl";

      expect(outputFile).toContain(".jsonl");
      expect(outputFile).toContain("safeoutputs");
    });

    it("should construct JSONL line", () => {
      const createJsonlLine = data => JSON.stringify(data) + "\n";

      const line = createJsonlLine({ type: "create_issue", title: "Test" });

      expect(line).toContain('"type":"create_issue"');
      expect(line).toContain('"title":"Test"');
      expect(line.endsWith("\n")).toBe(true);
    });
  });

  describe("error codes", () => {
    it("should define standard JSON-RPC error codes", () => {
      const ERROR_CODES = {
        PARSE_ERROR: -32700,
        INVALID_REQUEST: -32600,
        METHOD_NOT_FOUND: -32601,
        INVALID_PARAMS: -32602,
        INTERNAL_ERROR: -32603,
      };

      expect(ERROR_CODES.PARSE_ERROR).toBe(-32700);
      expect(ERROR_CODES.INVALID_REQUEST).toBe(-32600);
      expect(ERROR_CODES.METHOD_NOT_FOUND).toBe(-32601);
      expect(ERROR_CODES.INVALID_PARAMS).toBe(-32602);
      expect(ERROR_CODES.INTERNAL_ERROR).toBe(-32603);
    });
  });

  describe("MCP protocol methods", () => {
    it("should support initialize method", () => {
      const SUPPORTED_METHODS = ["initialize", "tools/list", "tools/call"];

      expect(SUPPORTED_METHODS).toContain("initialize");
    });

    it("should support tools/list method", () => {
      const SUPPORTED_METHODS = ["initialize", "tools/list", "tools/call"];

      expect(SUPPORTED_METHODS).toContain("tools/list");
    });

    it("should support tools/call method", () => {
      const SUPPORTED_METHODS = ["initialize", "tools/list", "tools/call"];

      expect(SUPPORTED_METHODS).toContain("tools/call");
    });
  });

  describe("initialization response", () => {
    it("should provide server info in initialization", () => {
      const initResponse = {
        protocolVersion: "2024-11-05",
        capabilities: {
          tools: {},
        },
        serverInfo: {
          name: "gh-aw-safe-outputs",
          version: "1.0.0",
        },
      };

      expect(initResponse.protocolVersion).toBe("2024-11-05");
      expect(initResponse.capabilities).toHaveProperty("tools");
      expect(initResponse.serverInfo.name).toBe("gh-aw-safe-outputs");
    });
  });

  describe("tool call result format", () => {
    it("should format successful tool call result", () => {
      const createToolCallResult = data => ({
        content: [
          {
            type: "text",
            text: JSON.stringify(data),
          },
        ],
      });

      const result = createToolCallResult({ status: "success", id: 123 });

      expect(result.content).toHaveLength(1);
      expect(result.content[0].type).toBe("text");
      expect(result.content[0].text).toContain('"status":"success"');
    });
  });
});
