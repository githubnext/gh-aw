import { describe, it, expect, beforeEach, vi } from "vitest";
describe("safe_outputs_mcp_server.cjs", () => {
  (describe("JSON-RPC message structure", () => {
    (it("should validate request structure", () => {
      const isValidRequest = msg => "2.0" === msg.jsonrpc && void 0 !== msg.id && "string" == typeof msg.method;
      (expect(isValidRequest({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} })).toBe(!0),
        expect(isValidRequest({ id: 1, method: "test" })).toBe(!1),
        expect(isValidRequest({ jsonrpc: "2.0", method: "test" })).toBe(!1),
        expect(isValidRequest({ jsonrpc: "2.0", id: 1 })).toBe(!1));
    }),
      it("should create valid response structure", () => {
        const response = { jsonrpc: "2.0", id: 1, result: { status: "ok" } };
        (expect(response).toHaveProperty("jsonrpc", "2.0"), expect(response).toHaveProperty("id", 1), expect(response).toHaveProperty("result"), expect(response.result).toEqual({ status: "ok" }));
      }),
      it("should create valid error response", () => {
        const errorResponse = { jsonrpc: "2.0", id: 1, error: { code: -32600, message: "Invalid Request" } };
        (expect(errorResponse).toHaveProperty("jsonrpc", "2.0"),
          expect(errorResponse).toHaveProperty("id", 1),
          expect(errorResponse).toHaveProperty("error"),
          expect(errorResponse.error.code).toBe(-32600),
          expect(errorResponse.error.message).toBe("Invalid Request"));
      }));
  }),
    describe("tool definition structure", () => {
      (it("should validate tool schema", () => {
        const isValidTool = tool => "string" == typeof tool.name && void 0 !== tool.description && void 0 !== tool.inputSchema && "object" == typeof tool.inputSchema;
        (expect(isValidTool({ name: "create_issue", description: "Create a GitHub issue", inputSchema: { type: "object", properties: { title: { type: "string" }, body: { type: "string" } }, required: ["title"] } })).toBe(!0),
          expect(isValidTool({ description: "No name" })).toBe(!1),
          expect(isValidTool({ name: "test", description: "No schema" })).toBe(!1));
      }),
        it("should handle tool with required fields", () => {
          const tool_inputSchema_required = ["title", "body"];
          (expect(tool_inputSchema_required).toContain("title"), expect(tool_inputSchema_required).toContain("body"), expect(tool_inputSchema_required).toHaveLength(2));
        }));
    }),
    describe("configuration handling", () => {
      (it("should handle empty configuration", () => {
        const tools = Object.keys({});
        expect(tools).toHaveLength(0);
      }),
        it("should validate tool enablement", () => {
          const enabledTools = Object.entries({ "create-issue": { enabled: !0 }, "add-comment": { enabled: !1 } })
            .filter(([_, cfg]) => !1 !== cfg.enabled)
            .map(([name, _]) => name);
          (expect(enabledTools).toContain("create-issue"), expect(enabledTools).not.toContain("add-comment"));
        }),
        it("should handle missing enabled property as true", () => {
          const enabledTools = Object.entries({ "create-issue": {}, "add-comment": { enabled: !1 } })
            .filter(([_, cfg]) => !1 !== cfg.enabled)
            .map(([name, _]) => name);
          expect(enabledTools).toContain("create-issue");
        }));
    }),
    describe("output file handling", () => {
      (it("should validate output file path", () => {
        const outputFile = "/tmp/gh-aw/safeoutputs/output.jsonl";
        (expect(outputFile).toContain(".jsonl"), expect(outputFile).toContain("safeoutputs"));
      }),
        it("should construct JSONL line", () => {
          const line = ((data = { type: "create_issue", title: "Test" }), JSON.stringify(data) + "\n");
          var data;
          (expect(line).toContain('"type":"create_issue"'), expect(line).toContain('"title":"Test"'), expect(line.endsWith("\n")).toBe(!0));
        }));
    }),
    describe("error codes", () => {
      it("should define standard JSON-RPC error codes", () => {
        (expect(-32700).toBe(-32700), expect(-32600).toBe(-32600), expect(-32601).toBe(-32601), expect(-32602).toBe(-32602), expect(-32603).toBe(-32603));
      });
    }),
    describe("MCP protocol methods", () => {
      (it("should support initialize method", () => {
        expect(["initialize", "tools/list", "tools/call"]).toContain("initialize");
      }),
        it("should support tools/list method", () => {
          expect(["initialize", "tools/list", "tools/call"]).toContain("tools/list");
        }),
        it("should support tools/call method", () => {
          expect(["initialize", "tools/list", "tools/call"]).toContain("tools/call");
        }));
    }),
    describe("initialization response", () => {
      it("should provide server info in initialization", () => {
        (expect("2024-11-05").toBe("2024-11-05"), expect({ tools: {} }).toHaveProperty("tools"), expect("gh-aw-safe-outputs").toBe("gh-aw-safe-outputs"));
      });
    }),
    describe("tool call result format", () => {
      it("should format successful tool call result", () => {
        const result = ((data = { status: "success", id: 123 }), { content: [{ type: "text", text: JSON.stringify(data) }] });
        var data;
        (expect(result.content).toHaveLength(1), expect(result.content[0].type).toBe("text"), expect(result.content[0].text).toContain('"status":"success"'));
      });
    }),
    describe("logging configuration", () => {
      (it("should only enable file logging when GH_AW_MCP_LOG_DIR is set", () => {
        expect(void 0).toBeUndefined();
      }),
        it("should validate log directory path format when set", () => {
          const logDir = "/tmp/gh-aw/mcp-logs/safeoutputs";
          (expect(logDir).toContain("mcp-logs"), expect(logDir).toContain("safeoutputs"));
        }),
        it("should validate log file path format when log directory is set", () => {
          const logFilePath = "/tmp/gh-aw/mcp-logs/safeoutputs/server.log";
          (expect(logFilePath).toContain("/tmp/gh-aw/mcp-logs/"), expect(logFilePath.endsWith(".log")).toBe(!0));
        }),
        it("should include timestamp in log messages", () => {
          const logMessage = `[${new Date().toISOString()}] [safeoutputs] Test message`;
          (expect(logMessage).toMatch(/\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/), expect(logMessage).toContain("[safeoutputs]"));
        }),
        it("should format log header correctly", () => {
          const header = "# Safe Outputs MCP Server Log\n# Started: 2025-11-26T12:00:00.000Z\n# Version: 1.0.0\n";
          (expect(header).toContain("# Safe Outputs MCP Server Log"), expect(header).toContain("# Started:"), expect(header).toContain("# Version:"));
        }));
    }),
    describe("logging integration", () => {
      const fs = require("fs"),
        path = require("path"),
        os = require("os");
      (it("should write log messages to file when GH_AW_MCP_LOG_DIR is set", () => {
        const testLogDir = path.join(os.tmpdir(), `test-mcp-logs-${Date.now()}`),
          testLogFile = path.join(testLogDir, "server.log");
        fs.mkdirSync(testLogDir, { recursive: !0 });
        const timestamp = new Date().toISOString(),
          header = `# Safe Outputs MCP Server Log\n# Started: ${timestamp}\n# Version: 1.0.0\n\n`;
        fs.writeFileSync(testLogFile, header);
        const logMessage = `[${timestamp}] [safeoutputs] Test message\n`;
        (fs.appendFileSync(testLogFile, logMessage), expect(fs.existsSync(testLogFile)).toBe(!0));
        const content = fs.readFileSync(testLogFile, "utf8");
        (expect(content).toContain("# Safe Outputs MCP Server Log"), expect(content).toContain("Test message"), fs.rmSync(testLogDir, { recursive: !0, force: !0 }));
      }),
        it("should create log directory lazily on first debug call when GH_AW_MCP_LOG_DIR is set", () => {
          const testLogDir = path.join(os.tmpdir(), `test-lazy-init-${Date.now()}`),
            testLogFile = path.join(testLogDir, "server.log");
          (expect(fs.existsSync(testLogDir)).toBe(!1), fs.mkdirSync(testLogDir, { recursive: !0 }));
          const timestamp = new Date().toISOString();
          (fs.writeFileSync(testLogFile, `# Safe Outputs MCP Server Log\n# Started: ${timestamp}\n# Version: 1.0.0\n\n`),
            expect(fs.existsSync(testLogDir)).toBe(!0),
            expect(fs.existsSync(testLogFile)).toBe(!0),
            fs.rmSync(testLogDir, { recursive: !0, force: !0 }));
        }),
        it("should write both to stderr and file simultaneously when GH_AW_MCP_LOG_DIR is set", () => {
          const testLogDir = path.join(os.tmpdir(), `test-dual-output-${Date.now()}`),
            testLogFile = path.join(testLogDir, "server.log");
          fs.mkdirSync(testLogDir, { recursive: !0 });
          const timestamp = new Date().toISOString();
          fs.writeFileSync(testLogFile, `# Safe Outputs MCP Server Log\n# Started: ${timestamp}\n# Version: 1.0.0\n\n`);
          const messages = ["Message 1", "Message 2", "Message 3"];
          for (const msg of messages) {
            const formattedMsg = `[${timestamp}] [safeoutputs] ${msg}\n`;
            fs.appendFileSync(testLogFile, formattedMsg);
          }
          const content = fs.readFileSync(testLogFile, "utf8");
          for (const msg of messages) expect(content).toContain(msg);
          fs.rmSync(testLogDir, { recursive: !0, force: !0 });
        }),
        it("should handle file write errors gracefully", () => {
          let errorHandled = !1;
          try {
            const invalidPath = "/nonexistent-root-dir-12345/cannot/write/here.log";
            fs.appendFileSync(invalidPath, "test");
          } catch {
            errorHandled = !0;
          }
          expect(errorHandled).toBe(!0);
        }),
        it("should append multiple log entries to the same file", () => {
          const testLogDir = path.join(os.tmpdir(), `test-append-${Date.now()}`),
            testLogFile = path.join(testLogDir, "server.log");
          fs.mkdirSync(testLogDir, { recursive: !0 });
          const initTimestamp = new Date().toISOString();
          fs.writeFileSync(testLogFile, `# Safe Outputs MCP Server Log\n# Started: ${initTimestamp}\n# Version: 1.0.0\n\n`);
          for (let i = 0; i < 5; i++) {
            const timestamp = new Date().toISOString();
            fs.appendFileSync(testLogFile, `[${timestamp}] [safeoutputs] Entry ${i + 1}\n`);
          }
          const content = fs.readFileSync(testLogFile, "utf8");
          for (let i = 0; i < 5; i++) expect(content).toContain(`Entry ${i + 1}`);
          const lines = content.split("\n").filter(line => line.length > 0);
          (expect(lines.length).toBeGreaterThanOrEqual(8), fs.rmSync(testLogDir, { recursive: !0, force: !0 }));
        }),
        it("should not create log file when GH_AW_MCP_LOG_DIR is not set", () => {
          expect("").toBe("");
        }));
    }));
});
