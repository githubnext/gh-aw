import { describe, it, expect, beforeEach, vi } from "vitest";
import { Readable, Writable } from "stream";

// Mock the stdin/stdout for server testing
let mockStdinData = [];
let mockStdoutData = [];

describe("mcp_server_core.cjs", () => {
  beforeEach(() => {
    vi.resetModules();
    mockStdinData = [];
    mockStdoutData = [];
    delete process.env.GH_AW_MCP_LOG_DIR;
  });

  describe("createServer", () => {
    it("should create a server with the given info", async () => {
      const { createServer } = await import("./mcp_server_core.cjs");
      const server = createServer({ name: "test-server", version: "1.0.0" });

      expect(server.serverInfo).toEqual({ name: "test-server", version: "1.0.0" });
      expect(server.tools).toEqual({});
      expect(typeof server.debug).toBe("function");
      expect(typeof server.writeMessage).toBe("function");
      expect(typeof server.replyResult).toBe("function");
      expect(typeof server.replyError).toBe("function");
    });

    it("should accept log directory option", async () => {
      const { createServer } = await import("./mcp_server_core.cjs");
      const server = createServer({ name: "test-server", version: "1.0.0" }, { logDir: "/tmp/test-logs" });

      expect(server.logDir).toBe("/tmp/test-logs");
      expect(server.logFilePath).toBe("/tmp/test-logs/server.log");
    });
  });

  describe("registerTool", () => {
    it("should register a tool with the server", async () => {
      const { createServer, registerTool } = await import("./mcp_server_core.cjs");
      const server = createServer({ name: "test-server", version: "1.0.0" });

      registerTool(server, {
        name: "test_tool",
        description: "A test tool",
        inputSchema: { type: "object", properties: {} },
        handler: () => ({ content: [{ type: "text", text: "ok" }] }),
      });

      expect(server.tools["test_tool"]).toBeDefined();
      expect(server.tools["test_tool"].name).toBe("test_tool");
      expect(server.tools["test_tool"].description).toBe("A test tool");
    });

    it("should normalize tool names with dashes to underscores", async () => {
      const { createServer, registerTool } = await import("./mcp_server_core.cjs");
      const server = createServer({ name: "test-server", version: "1.0.0" });

      registerTool(server, {
        name: "test-tool",
        description: "A test tool",
        inputSchema: { type: "object", properties: {} },
      });

      expect(server.tools["test_tool"]).toBeDefined();
      expect(server.tools["test_tool"].name).toBe("test_tool");
    });

    it("should normalize tool names to lowercase", async () => {
      const { createServer, registerTool } = await import("./mcp_server_core.cjs");
      const server = createServer({ name: "test-server", version: "1.0.0" });

      registerTool(server, {
        name: "Test-Tool",
        description: "A test tool",
        inputSchema: { type: "object", properties: {} },
      });

      expect(server.tools["test_tool"]).toBeDefined();
    });
  });

  describe("normalizeTool", () => {
    it("should normalize tool names", async () => {
      const { normalizeTool } = await import("./mcp_server_core.cjs");

      expect(normalizeTool("test-tool")).toBe("test_tool");
      expect(normalizeTool("Test-Tool")).toBe("test_tool");
      expect(normalizeTool("create_issue")).toBe("create_issue");
      expect(normalizeTool("CREATE-ISSUE")).toBe("create_issue");
    });

    it("should handle empty string input", async () => {
      const { normalizeTool } = await import("./mcp_server_core.cjs");

      expect(normalizeTool("")).toBe("");
    });
  });

  describe("handleMessage", () => {
    let server;
    let results = [];

    beforeEach(async () => {
      vi.resetModules();
      results = [];

      // Suppress stderr output during tests
      vi.spyOn(process.stderr, "write").mockImplementation(() => true);

      const { createServer, registerTool } = await import("./mcp_server_core.cjs");
      server = createServer({ name: "test-server", version: "1.0.0" });

      // Override writeMessage to capture results
      server.writeMessage = msg => {
        results.push(msg);
      };
      server.replyResult = (id, result) => {
        if (id === undefined || id === null) return;
        results.push({ jsonrpc: "2.0", id, result });
      };
      server.replyError = (id, code, message) => {
        if (id === undefined || id === null) return;
        results.push({ jsonrpc: "2.0", id, error: { code, message } });
      };

      registerTool(server, {
        name: "test_tool",
        description: "A test tool",
        inputSchema: {
          type: "object",
          properties: { input: { type: "string" } },
          required: ["input"],
        },
        handler: args => ({
          content: [{ type: "text", text: `received: ${args.input}` }],
        }),
      });
    });

    it("should handle initialize method", async () => {
      const { handleMessage } = await import("./mcp_server_core.cjs");

      handleMessage(server, {
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: { protocolVersion: "2024-11-05" },
      });

      expect(results).toHaveLength(1);
      expect(results[0].result.serverInfo).toEqual({ name: "test-server", version: "1.0.0" });
      expect(results[0].result.protocolVersion).toBe("2024-11-05");
      expect(results[0].result.capabilities).toEqual({ tools: {} });
    });

    it("should handle tools/list method", async () => {
      const { handleMessage } = await import("./mcp_server_core.cjs");

      handleMessage(server, {
        jsonrpc: "2.0",
        id: 1,
        method: "tools/list",
      });

      expect(results).toHaveLength(1);
      expect(results[0].result.tools).toHaveLength(1);
      expect(results[0].result.tools[0].name).toBe("test_tool");
    });

    it("should handle tools/call method with handler", async () => {
      const { handleMessage } = await import("./mcp_server_core.cjs");

      handleMessage(server, {
        jsonrpc: "2.0",
        id: 1,
        method: "tools/call",
        params: {
          name: "test_tool",
          arguments: { input: "hello" },
        },
      });

      expect(results).toHaveLength(1);
      expect(results[0].result.content[0].text).toBe("received: hello");
      expect(results[0].result.isError).toBe(false);
    });

    it("should return error for unknown tool", async () => {
      const { handleMessage } = await import("./mcp_server_core.cjs");

      handleMessage(server, {
        jsonrpc: "2.0",
        id: 1,
        method: "tools/call",
        params: {
          name: "unknown_tool",
          arguments: {},
        },
      });

      expect(results).toHaveLength(1);
      expect(results[0].error.code).toBe(-32601);
      expect(results[0].error.message).toContain("Tool not found");
    });

    it("should return error for missing required fields", async () => {
      const { handleMessage } = await import("./mcp_server_core.cjs");

      handleMessage(server, {
        jsonrpc: "2.0",
        id: 1,
        method: "tools/call",
        params: {
          name: "test_tool",
          arguments: {}, // missing required 'input'
        },
      });

      expect(results).toHaveLength(1);
      expect(results[0].error.code).toBe(-32602);
      expect(results[0].error.message).toContain("missing or empty");
    });

    it("should return error for unknown method", async () => {
      const { handleMessage } = await import("./mcp_server_core.cjs");

      handleMessage(server, {
        jsonrpc: "2.0",
        id: 1,
        method: "unknown/method",
      });

      expect(results).toHaveLength(1);
      expect(results[0].error.code).toBe(-32601);
      expect(results[0].error.message).toContain("Method not found");
    });

    it("should ignore notifications (no response)", async () => {
      const { handleMessage } = await import("./mcp_server_core.cjs");

      handleMessage(server, {
        jsonrpc: "2.0",
        method: "notifications/initialized",
        // no id - this is a notification
      });

      expect(results).toHaveLength(0);
    });

    it("should validate JSON-RPC version", async () => {
      const { handleMessage } = await import("./mcp_server_core.cjs");

      handleMessage(server, {
        jsonrpc: "1.0", // wrong version
        id: 1,
        method: "test",
      });

      // Should not produce a response (invalid message silently ignored)
      expect(results).toHaveLength(0);
    });

    it("should use default handler when tool has no handler", async () => {
      const { handleMessage, registerTool } = await import("./mcp_server_core.cjs");

      // Register tool without handler
      registerTool(server, {
        name: "no_handler_tool",
        description: "A tool without handler",
        inputSchema: { type: "object", properties: {} },
      });

      const defaultHandler = type => args => ({
        content: [{ type: "text", text: `default handler for ${type}` }],
      });

      handleMessage(
        server,
        {
          jsonrpc: "2.0",
          id: 1,
          method: "tools/call",
          params: {
            name: "no_handler_tool",
            arguments: {},
          },
        },
        defaultHandler
      );

      expect(results).toHaveLength(1);
      expect(results[0].result.content[0].text).toBe("default handler for no_handler_tool");
    });
  });

  describe("startWithTransport", () => {
    it("should start server with custom transport", async () => {
      vi.resetModules();

      // Suppress stderr output during tests
      vi.spyOn(process.stderr, "write").mockImplementation(() => true);

      const { createServer, registerTool, startWithTransport } = await import("./mcp_server_core.cjs");
      const server = createServer({ name: "test-server", version: "1.0.0" });

      registerTool(server, {
        name: "test_tool",
        description: "A test tool",
        inputSchema: { type: "object", properties: {} },
        handler: () => ({ content: [{ type: "text", text: "ok" }] }),
      });

      const mockTransport = {
        send: vi.fn(),
        onMessage: vi.fn(),
        start: vi.fn(),
      };

      startWithTransport(server, mockTransport);

      expect(mockTransport.onMessage).toHaveBeenCalled();
      expect(mockTransport.start).toHaveBeenCalled();
      expect(server.transport).toBe(mockTransport);
    });

    it("should throw error if no tools registered", async () => {
      vi.resetModules();

      // Suppress stderr output during tests
      vi.spyOn(process.stderr, "write").mockImplementation(() => true);

      const { createServer, startWithTransport } = await import("./mcp_server_core.cjs");
      const server = createServer({ name: "test-server", version: "1.0.0" });

      const mockTransport = {
        send: vi.fn(),
        onMessage: vi.fn(),
        start: vi.fn(),
      };

      expect(() => startWithTransport(server, mockTransport)).toThrow("No tools registered");
    });

    it("should use transport.send for writeMessage", async () => {
      vi.resetModules();

      // Suppress stderr output during tests
      vi.spyOn(process.stderr, "write").mockImplementation(() => true);

      const { createServer, registerTool, startWithTransport } = await import("./mcp_server_core.cjs");
      const server = createServer({ name: "test-server", version: "1.0.0" });

      registerTool(server, {
        name: "test_tool",
        description: "A test tool",
        inputSchema: { type: "object", properties: {} },
        handler: () => ({ content: [{ type: "text", text: "ok" }] }),
      });

      const mockTransport = {
        send: vi.fn(),
        onMessage: vi.fn(),
        start: vi.fn(),
      };

      startWithTransport(server, mockTransport);

      const message = { jsonrpc: "2.0", id: 1, result: { status: "ok" } };
      server.writeMessage(message);

      expect(mockTransport.send).toHaveBeenCalledWith(message);
    });

    it("should handle messages from transport", async () => {
      vi.resetModules();

      // Suppress stderr output during tests
      vi.spyOn(process.stderr, "write").mockImplementation(() => true);

      const { createServer, registerTool, startWithTransport } = await import("./mcp_server_core.cjs");
      const server = createServer({ name: "test-server", version: "1.0.0" });

      registerTool(server, {
        name: "test_tool",
        description: "A test tool",
        inputSchema: { type: "object", properties: {} },
        handler: () => ({ content: [{ type: "text", text: "ok" }] }),
      });

      let messageHandler = null;
      const mockTransport = {
        send: vi.fn(),
        onMessage: vi.fn(handler => {
          messageHandler = handler;
        }),
        start: vi.fn(),
      };

      startWithTransport(server, mockTransport);

      // Simulate receiving an initialize message
      messageHandler({
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: {},
      });

      expect(mockTransport.send).toHaveBeenCalled();
      const sentMessage = mockTransport.send.mock.calls[0][0];
      expect(sentMessage.jsonrpc).toBe("2.0");
      expect(sentMessage.id).toBe(1);
      expect(sentMessage.result.serverInfo.name).toBe("test-server");
    });
  });
});
