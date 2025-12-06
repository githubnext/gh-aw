import { describe, it, expect } from "vitest";
import { MCPServer } from "./mcp_server.cjs";

/**
 * Unit tests for mcp_server.cjs
 *
 * These tests validate the MCPServer class functionality independently
 * of any transport implementation.
 */
describe("mcp_server.cjs", () => {
  describe("MCPServer", () => {
    it("should create a server with basic info", () => {
      const server = new MCPServer({ name: "test-server", version: "1.0.0" }, { capabilities: { tools: {} } });

      expect(server.serverInfo.name).toBe("test-server");
      expect(server.serverInfo.version).toBe("1.0.0");
      expect(server.capabilities.tools).toBeDefined();
    });

    it("should create a server with default capabilities", () => {
      const server = new MCPServer({ name: "test-server", version: "1.0.0" });

      expect(server.capabilities.tools).toBeDefined();
      expect(server.initialized).toBe(false);
    });

    it("should register a tool", () => {
      const server = new MCPServer({ name: "test-server", version: "1.0.0" });

      server.tool(
        "test_tool",
        "A test tool",
        {
          type: "object",
          properties: {
            input: { type: "string" },
          },
        },
        async args => {
          return { content: [{ type: "text", text: "result" }] };
        }
      );

      expect(server.tools.size).toBe(1);
      expect(server.tools.has("test_tool")).toBe(true);
      const tool = server.tools.get("test_tool");
      expect(tool.name).toBe("test_tool");
      expect(tool.description).toBe("A test tool");
    });

    it("should handle initialize request", async () => {
      const server = new MCPServer({ name: "test-server", version: "1.0.0" });

      const response = await server.handleRequest({
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: { protocolVersion: "2024-11-05" },
      });

      expect(response.jsonrpc).toBe("2.0");
      expect(response.id).toBe(1);
      expect(response.result.protocolVersion).toBe("2024-11-05");
      expect(response.result.serverInfo.name).toBe("test-server");
      expect(server.initialized).toBe(true);
    });

    it("should handle tools/list request", async () => {
      const server = new MCPServer({ name: "test-server", version: "1.0.0" });

      server.tool("tool1", "First tool", {}, async () => ({ content: [] }));
      server.tool("tool2", "Second tool", {}, async () => ({ content: [] }));

      const response = await server.handleRequest({
        jsonrpc: "2.0",
        id: 2,
        method: "tools/list",
      });

      expect(response.result.tools).toHaveLength(2);
      expect(response.result.tools[0].name).toBe("tool1");
      expect(response.result.tools[1].name).toBe("tool2");
    });

    it("should handle tools/call request", async () => {
      const server = new MCPServer({ name: "test-server", version: "1.0.0" });

      const mockHandler = async args => {
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify({ echo: args.message }),
            },
          ],
        };
      };

      server.tool("echo", "Echo tool", { type: "object" }, mockHandler);

      const response = await server.handleRequest({
        jsonrpc: "2.0",
        id: 3,
        method: "tools/call",
        params: {
          name: "echo",
          arguments: { message: "hello" },
        },
      });

      expect(response.result.content).toHaveLength(1);
      expect(response.result.content[0].type).toBe("text");
      const result = JSON.parse(response.result.content[0].text);
      expect(result.echo).toBe("hello");
    });

    it("should return error for unknown tool", async () => {
      const server = new MCPServer({ name: "test-server", version: "1.0.0" });

      const response = await server.handleRequest({
        jsonrpc: "2.0",
        id: 4,
        method: "tools/call",
        params: {
          name: "unknown_tool",
          arguments: {},
        },
      });

      expect(response.error).toBeDefined();
      expect(response.error.code).toBe(-32602);
      expect(response.error.message).toContain("not found");
    });

    it("should return error for unknown method", async () => {
      const server = new MCPServer({ name: "test-server", version: "1.0.0" });

      const response = await server.handleRequest({
        jsonrpc: "2.0",
        id: 5,
        method: "unknown/method",
      });

      expect(response.error).toBeDefined();
      expect(response.error.code).toBe(-32601);
      expect(response.error.message).toContain("not found");
    });

    it("should handle tool handler errors", async () => {
      const server = new MCPServer({ name: "test-server", version: "1.0.0" });

      server.tool("error_tool", "Tool that throws", {}, async () => {
        throw new Error("Test error");
      });

      const response = await server.handleRequest({
        jsonrpc: "2.0",
        id: 6,
        method: "tools/call",
        params: {
          name: "error_tool",
          arguments: {},
        },
      });

      expect(response.error).toBeDefined();
      expect(response.error.code).toBe(-32603);
      expect(response.error.message).toBe("Test error");
    });

    it("should handle ping request", async () => {
      const server = new MCPServer({ name: "test-server", version: "1.0.0" });

      const response = await server.handleRequest({
        jsonrpc: "2.0",
        id: 7,
        method: "ping",
      });

      expect(response.jsonrpc).toBe("2.0");
      expect(response.id).toBe(7);
      expect(response.result).toEqual({});
    });

    it("should handle notifications/initialized without response", async () => {
      const server = new MCPServer({ name: "test-server", version: "1.0.0" });

      const response = await server.handleRequest({
        jsonrpc: "2.0",
        method: "notifications/initialized",
        // no id - this is a notification per JSON-RPC 2.0 spec
      });

      expect(response).toBeNull();
    });

    it("should handle any notification without response (no id field)", async () => {
      const server = new MCPServer({ name: "test-server", version: "1.0.0" });

      const response = await server.handleRequest({
        jsonrpc: "2.0",
        method: "some/custom/notification",
        params: { data: "test" },
        // no id - this is a notification per JSON-RPC 2.0 spec
      });

      expect(response).toBeNull();
    });
  });
});
