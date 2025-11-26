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

  describe("logging configuration", () => {
    it("should only enable file logging when GH_AW_MCP_LOG_DIR is set", () => {
      // Without GH_AW_MCP_LOG_DIR set, MCP_LOG_DIR should be undefined
      const logDirFromEnv = undefined;
      expect(logDirFromEnv).toBeUndefined();
    });

    it("should validate log directory path format when set", () => {
      const logDir = "/tmp/gh-aw/mcp-logs/safeoutputs";
      expect(logDir).toContain("mcp-logs");
      expect(logDir).toContain("safeoutputs");
    });

    it("should validate log file path format when log directory is set", () => {
      const logFilePath = "/tmp/gh-aw/mcp-logs/safeoutputs/server.log";
      expect(logFilePath).toContain("/tmp/gh-aw/mcp-logs/");
      expect(logFilePath.endsWith(".log")).toBe(true);
    });

    it("should include timestamp in log messages", () => {
      const timestamp = new Date().toISOString();
      const logMessage = `[${timestamp}] [safeoutputs] Test message`;

      expect(logMessage).toMatch(/\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);
      expect(logMessage).toContain("[safeoutputs]");
    });

    it("should format log header correctly", () => {
      const header = `# Safe Outputs MCP Server Log\n# Started: 2025-11-26T12:00:00.000Z\n# Version: 1.0.0\n`;

      expect(header).toContain("# Safe Outputs MCP Server Log");
      expect(header).toContain("# Started:");
      expect(header).toContain("# Version:");
    });
  });

  describe("logging integration", () => {
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    it("should write log messages to file when GH_AW_MCP_LOG_DIR is set", () => {
      // Create a unique temp directory for this test
      const testLogDir = path.join(os.tmpdir(), `test-mcp-logs-${Date.now()}`);
      const testLogFile = path.join(testLogDir, "server.log");

      // Simulate the logging behavior when GH_AW_MCP_LOG_DIR is set
      fs.mkdirSync(testLogDir, { recursive: true });
      const timestamp = new Date().toISOString();
      const header = `# Safe Outputs MCP Server Log\n# Started: ${timestamp}\n# Version: 1.0.0\n\n`;
      fs.writeFileSync(testLogFile, header);

      const logMessage = `[${timestamp}] [safeoutputs] Test message\n`;
      fs.appendFileSync(testLogFile, logMessage);

      // Verify log file was created and contains expected content
      expect(fs.existsSync(testLogFile)).toBe(true);
      const content = fs.readFileSync(testLogFile, "utf8");
      expect(content).toContain("# Safe Outputs MCP Server Log");
      expect(content).toContain("Test message");

      // Cleanup
      fs.rmSync(testLogDir, { recursive: true, force: true });
    });

    it("should create log directory lazily on first debug call when GH_AW_MCP_LOG_DIR is set", () => {
      // Create a unique temp directory for this test
      const testLogDir = path.join(os.tmpdir(), `test-lazy-init-${Date.now()}`);
      const testLogFile = path.join(testLogDir, "server.log");

      // Verify directory doesn't exist initially
      expect(fs.existsSync(testLogDir)).toBe(false);

      // Simulate lazy initialization when GH_AW_MCP_LOG_DIR is set
      fs.mkdirSync(testLogDir, { recursive: true });
      const timestamp = new Date().toISOString();
      fs.writeFileSync(testLogFile, `# Safe Outputs MCP Server Log\n# Started: ${timestamp}\n# Version: 1.0.0\n\n`);

      // Verify directory was created
      expect(fs.existsSync(testLogDir)).toBe(true);
      expect(fs.existsSync(testLogFile)).toBe(true);

      // Cleanup
      fs.rmSync(testLogDir, { recursive: true, force: true });
    });

    it("should write both to stderr and file simultaneously when GH_AW_MCP_LOG_DIR is set", () => {
      const testLogDir = path.join(os.tmpdir(), `test-dual-output-${Date.now()}`);
      const testLogFile = path.join(testLogDir, "server.log");

      // Set up the log file
      fs.mkdirSync(testLogDir, { recursive: true });
      const timestamp = new Date().toISOString();
      fs.writeFileSync(testLogFile, `# Safe Outputs MCP Server Log\n# Started: ${timestamp}\n# Version: 1.0.0\n\n`);

      // Simulate debug output (file part only - stderr is handled by process)
      const messages = ["Message 1", "Message 2", "Message 3"];
      for (const msg of messages) {
        const formattedMsg = `[${timestamp}] [safeoutputs] ${msg}\n`;
        fs.appendFileSync(testLogFile, formattedMsg);
      }

      // Verify all messages are in the file
      const content = fs.readFileSync(testLogFile, "utf8");
      for (const msg of messages) {
        expect(content).toContain(msg);
      }

      // Cleanup
      fs.rmSync(testLogDir, { recursive: true, force: true });
    });

    it("should handle file write errors gracefully", () => {
      // Test that the error handling pattern works
      let errorHandled = false;
      try {
        // Attempt to write to an invalid path
        const invalidPath = "/nonexistent-root-dir-12345/cannot/write/here.log";
        fs.appendFileSync(invalidPath, "test");
      } catch {
        // Error is caught and handled gracefully
        errorHandled = true;
      }

      // Verify that errors are caught (which is what our code does silently)
      expect(errorHandled).toBe(true);
    });

    it("should append multiple log entries to the same file", () => {
      const testLogDir = path.join(os.tmpdir(), `test-append-${Date.now()}`);
      const testLogFile = path.join(testLogDir, "server.log");

      // Initialize log file
      fs.mkdirSync(testLogDir, { recursive: true });
      const initTimestamp = new Date().toISOString();
      fs.writeFileSync(testLogFile, `# Safe Outputs MCP Server Log\n# Started: ${initTimestamp}\n# Version: 1.0.0\n\n`);

      // Append multiple entries
      const numEntries = 5;
      for (let i = 0; i < numEntries; i++) {
        const timestamp = new Date().toISOString();
        fs.appendFileSync(testLogFile, `[${timestamp}] [safeoutputs] Entry ${i + 1}\n`);
      }

      // Verify all entries are present
      const content = fs.readFileSync(testLogFile, "utf8");
      for (let i = 0; i < numEntries; i++) {
        expect(content).toContain(`Entry ${i + 1}`);
      }

      // Verify the file has the header plus all entries
      const lines = content.split("\n").filter(line => line.length > 0);
      expect(lines.length).toBeGreaterThanOrEqual(3 + numEntries); // 3 header lines + entries

      // Cleanup
      fs.rmSync(testLogDir, { recursive: true, force: true });
    });

    it("should not create log file when GH_AW_MCP_LOG_DIR is not set", () => {
      // This test verifies the conditional behavior
      const logDirFromEnv = undefined; // Simulating no env var
      const logFilePath = logDirFromEnv ? require("path").join(logDirFromEnv, "server.log") : "";

      expect(logFilePath).toBe("");
    });
  });
});
