import { describe, it, expect } from "vitest";
import { MCPServer, MCPHTTPTransport } from "./mcp_http_transport.cjs";
describe("mcp_http_transport.cjs", () => {
  (describe("MCPServer", () => {
    (it("should create a server with basic info", () => {
      const server = new MCPServer({ name: "test-server", version: "1.0.0" }, { capabilities: { tools: {} } });
      (expect(server.serverInfo.name).toBe("test-server"), expect(server.serverInfo.version).toBe("1.0.0"), expect(server.capabilities.tools).toBeDefined());
    }),
      it("should register a tool", () => {
        const server = new MCPServer({ name: "test-server", version: "1.0.0" });
        (server.tool("test_tool", "A test tool", { type: "object", properties: { input: { type: "string" } } }, async args => ({ content: [{ type: "text", text: "result" }] })),
          expect(server.tools.size).toBe(1),
          expect(server.tools.has("test_tool")).toBe(!0));
        const tool = server.tools.get("test_tool");
        (expect(tool.name).toBe("test_tool"), expect(tool.description).toBe("A test tool"));
      }),
      it("should handle initialize request", async () => {
        const server = new MCPServer({ name: "test-server", version: "1.0.0" }),
          response = await server.handleRequest({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2024-11-05" } });
        (expect(response.jsonrpc).toBe("2.0"),
          expect(response.id).toBe(1),
          expect(response.result.protocolVersion).toBe("2024-11-05"),
          expect(response.result.serverInfo.name).toBe("test-server"),
          expect(response.initialized).toBeUndefined(),
          expect(server.initialized).toBe(!0));
      }),
      it("should handle tools/list request", async () => {
        const server = new MCPServer({ name: "test-server", version: "1.0.0" });
        (server.tool("tool1", "First tool", {}, async () => ({ content: [] })), server.tool("tool2", "Second tool", {}, async () => ({ content: [] })));
        const response = await server.handleRequest({ jsonrpc: "2.0", id: 2, method: "tools/list" });
        (expect(response.result.tools).toHaveLength(2), expect(response.result.tools[0].name).toBe("tool1"), expect(response.result.tools[1].name).toBe("tool2"));
      }),
      it("should handle tools/call request", async () => {
        const server = new MCPServer({ name: "test-server", version: "1.0.0" });
        server.tool("echo", "Echo tool", { type: "object" }, async args => ({ content: [{ type: "text", text: JSON.stringify({ echo: args.message }) }] }));
        const response = await server.handleRequest({ jsonrpc: "2.0", id: 3, method: "tools/call", params: { name: "echo", arguments: { message: "hello" } } });
        (expect(response.result.content).toHaveLength(1), expect(response.result.content[0].type).toBe("text"));
        const result = JSON.parse(response.result.content[0].text);
        expect(result.echo).toBe("hello");
      }),
      it("should return error for unknown tool", async () => {
        const server = new MCPServer({ name: "test-server", version: "1.0.0" }),
          response = await server.handleRequest({ jsonrpc: "2.0", id: 4, method: "tools/call", params: { name: "unknown_tool", arguments: {} } });
        (expect(response.error).toBeDefined(), expect(response.error.code).toBe(-32602), expect(response.error.message).toContain("not found"));
      }),
      it("should return error for unknown method", async () => {
        const server = new MCPServer({ name: "test-server", version: "1.0.0" }),
          response = await server.handleRequest({ jsonrpc: "2.0", id: 5, method: "unknown/method" });
        (expect(response.error).toBeDefined(), expect(response.error.code).toBe(-32601), expect(response.error.message).toContain("not found"));
      }),
      it("should handle ping request", async () => {
        const server = new MCPServer({ name: "test-server", version: "1.0.0" }),
          response = await server.handleRequest({ jsonrpc: "2.0", id: 6, method: "ping" });
        (expect(response.jsonrpc).toBe("2.0"), expect(response.id).toBe(6), expect(response.result).toEqual({}));
      }),
      it("should handle notifications/initialized without response", async () => {
        const server = new MCPServer({ name: "test-server", version: "1.0.0" }),
          response = await server.handleRequest({ jsonrpc: "2.0", method: "notifications/initialized" });
        expect(response).toBeNull();
      }),
      it("should handle any notification without response (no id field)", async () => {
        const server = new MCPServer({ name: "test-server", version: "1.0.0" }),
          response = await server.handleRequest({ jsonrpc: "2.0", method: "some/custom/notification", params: { data: "test" } });
        expect(response).toBeNull();
      }),
      it("should handle request with id: null as a valid request", async () => {
        const server = new MCPServer({ name: "test-server", version: "1.0.0" }),
          response = await server.handleRequest({ jsonrpc: "2.0", id: null, method: "ping" });
        (expect(response).not.toBeNull(), expect(response.jsonrpc).toBe("2.0"), expect(response.id).toBeNull(), expect(response.result).toEqual({}));
      }));
  }),
    describe("MCPHTTPTransport", () => {
      (it("should create a transport with session ID generator", () => {
        const transport = new MCPHTTPTransport({ sessionIdGenerator: () => "test-session-id", enableJsonResponse: !0 });
        (expect(transport).toBeDefined(), expect(transport.sessionIdGenerator).toBeDefined(), expect(transport.enableJsonResponse).toBe(!0));
      }),
        it("should create a stateless transport", () => {
          const transport = new MCPHTTPTransport({ sessionIdGenerator: void 0, enableJsonResponse: !0 });
          (expect(transport).toBeDefined(), expect(transport.sessionIdGenerator).toBeUndefined());
        }),
        it("should start successfully", async () => {
          const transport = new MCPHTTPTransport({ sessionIdGenerator: () => "test-session-id" });
          (await expect(transport.start()).resolves.toBeUndefined(), expect(transport.started).toBe(!0));
        }),
        it("should not allow starting twice", async () => {
          const transport = new MCPHTTPTransport({ sessionIdGenerator: () => "test-session-id" });
          (await transport.start(), await expect(transport.start()).rejects.toThrow("already started"));
        }));
    }),
    describe("Integration", () => {
      it("should connect server to transport", async () => {
        const server = new MCPServer({ name: "test-server", version: "1.0.0" }),
          transport = new MCPHTTPTransport({ sessionIdGenerator: () => "test-session-id" });
        (await server.connect(transport), expect(server.transport).toBe(transport), expect(transport.server).toBe(server), expect(transport.started).toBe(!0));
      });
    }));
});
