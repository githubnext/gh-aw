import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

describe("safe_inputs_mcp_server.cjs", () => {
  let tempDir;

  beforeEach(() => {
    vi.resetModules();
    // Suppress stderr output during tests
    vi.spyOn(process.stderr, "write").mockImplementation(() => true);

    // Create a temporary directory for test files
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "safe-inputs-test-"));
  });

  afterEach(() => {
    // Clean up temporary directory
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true });
    }
  });

  describe("loadConfig", () => {
    it("should load configuration from a valid JSON file", async () => {
      const { loadConfig } = await import("./safe_inputs_mcp_server.cjs");

      // Create a test configuration file
      const configPath = path.join(tempDir, "config.json");
      const config = {
        serverName: "test-server",
        version: "1.0.0",
        tools: [
          {
            name: "test_tool",
            description: "A test tool",
            inputSchema: { type: "object", properties: {} },
          },
        ],
      };
      fs.writeFileSync(configPath, JSON.stringify(config));

      const loadedConfig = loadConfig(configPath);

      expect(loadedConfig.serverName).toBe("test-server");
      expect(loadedConfig.version).toBe("1.0.0");
      expect(loadedConfig.tools).toHaveLength(1);
      expect(loadedConfig.tools[0].name).toBe("test_tool");
    });

    it("should throw error for non-existent file", async () => {
      const { loadConfig } = await import("./safe_inputs_mcp_server.cjs");

      expect(() => loadConfig("/non/existent/config.json")).toThrow("Configuration file not found");
    });

    it("should throw error for invalid JSON", async () => {
      const { loadConfig } = await import("./safe_inputs_mcp_server.cjs");

      const configPath = path.join(tempDir, "invalid.json");
      fs.writeFileSync(configPath, "not valid json");

      expect(() => loadConfig(configPath)).toThrow();
    });

    it("should throw error for missing tools array", async () => {
      const { loadConfig } = await import("./safe_inputs_mcp_server.cjs");

      const configPath = path.join(tempDir, "no-tools.json");
      fs.writeFileSync(configPath, JSON.stringify({ serverName: "test" }));

      expect(() => loadConfig(configPath)).toThrow("Configuration must contain a 'tools' array");
    });

    it("should throw error for tools that is not an array", async () => {
      const { loadConfig } = await import("./safe_inputs_mcp_server.cjs");

      const configPath = path.join(tempDir, "tools-not-array.json");
      fs.writeFileSync(configPath, JSON.stringify({ tools: "not an array" }));

      expect(() => loadConfig(configPath)).toThrow("Configuration must contain a 'tools' array");
    });
  });

  describe("createJsToolConfig", () => {
    it("should create a valid tool configuration for JavaScript handler", async () => {
      const { createJsToolConfig } = await import("./safe_inputs_mcp_server.cjs");

      const config = createJsToolConfig(
        "my_tool",
        "My tool description",
        { type: "object", properties: { input: { type: "string" } } },
        "my_tool.cjs"
      );

      expect(config.name).toBe("my_tool");
      expect(config.description).toBe("My tool description");
      expect(config.inputSchema).toEqual({ type: "object", properties: { input: { type: "string" } } });
      expect(config.handler).toBe("my_tool.cjs");
    });
  });

  describe("createShellToolConfig", () => {
    it("should create a valid tool configuration for shell script handler", async () => {
      const { createShellToolConfig } = await import("./safe_inputs_mcp_server.cjs");

      const config = createShellToolConfig("my_shell_tool", "My shell tool", { type: "object", properties: {} }, "my_tool.sh");

      expect(config.name).toBe("my_shell_tool");
      expect(config.description).toBe("My shell tool");
      expect(config.handler).toBe("my_tool.sh");
    });
  });

  describe("startSafeInputsServer integration", () => {
    it("should start server with JavaScript handler", async () => {
      // Create a handler file
      const handlerPath = path.join(tempDir, "test_handler.cjs");
      fs.writeFileSync(
        handlerPath,
        `module.exports = function(args) {
          return { result: "hello " + args.name };
        };`
      );

      // Create config file
      const configPath = path.join(tempDir, "config.json");
      const config = {
        serverName: "test-safe-inputs",
        version: "1.0.0",
        tools: [
          {
            name: "greet",
            description: "Greet someone",
            inputSchema: { type: "object", properties: { name: { type: "string" } } },
            handler: "test_handler.cjs",
          },
        ],
      };
      fs.writeFileSync(configPath, JSON.stringify(config));

      // Import and load config
      const { loadConfig } = await import("./safe_inputs_mcp_server.cjs");
      const { createServer, registerTool, loadToolHandlers, handleMessage } = await import("./mcp_server_core.cjs");

      const loadedConfig = loadConfig(configPath);
      const server = createServer({ name: loadedConfig.serverName || "safeinputs", version: loadedConfig.version || "1.0.0" });

      // Load handlers
      const tools = loadToolHandlers(server, loadedConfig.tools, tempDir);

      // Register tools
      for (const tool of tools) {
        registerTool(server, tool);
      }

      // Test tool call
      const results = [];
      server.writeMessage = msg => results.push(msg);
      server.replyResult = (id, result) => results.push({ jsonrpc: "2.0", id, result });
      server.replyError = (id, code, message) => results.push({ jsonrpc: "2.0", id, error: { code, message } });

      await handleMessage(server, {
        jsonrpc: "2.0",
        id: 1,
        method: "tools/call",
        params: { name: "greet", arguments: { name: "world" } },
      });

      expect(results).toHaveLength(1);
      expect(results[0].result.content[0].text).toContain("hello world");
    });

    it("should start server with shell script handler", async () => {
      // Create a shell script handler
      const handlerPath = path.join(tempDir, "test_handler.sh");
      fs.writeFileSync(
        handlerPath,
        `#!/bin/bash
echo "Shell says: $INPUT_NAME"
echo "greeting=Hello from shell" >> $GITHUB_OUTPUT
`,
        { mode: 0o755 }
      );

      // Create config file
      const configPath = path.join(tempDir, "config.json");
      const config = {
        tools: [
          {
            name: "shell_greet",
            description: "Greet from shell",
            inputSchema: { type: "object", properties: { name: { type: "string" } } },
            handler: "test_handler.sh",
          },
        ],
      };
      fs.writeFileSync(configPath, JSON.stringify(config));

      // Import and load config
      const { loadConfig } = await import("./safe_inputs_mcp_server.cjs");
      const { createServer, registerTool, loadToolHandlers, handleMessage } = await import("./mcp_server_core.cjs");

      const loadedConfig = loadConfig(configPath);
      const server = createServer({ name: "safeinputs", version: "1.0.0" });

      // Load handlers
      const tools = loadToolHandlers(server, loadedConfig.tools, tempDir);

      // Register tools
      for (const tool of tools) {
        registerTool(server, tool);
      }

      // Test tool call
      const results = [];
      server.writeMessage = msg => results.push(msg);
      server.replyResult = (id, result) => results.push({ jsonrpc: "2.0", id, result });
      server.replyError = (id, code, message) => results.push({ jsonrpc: "2.0", id, error: { code, message } });

      await handleMessage(server, {
        jsonrpc: "2.0",
        id: 1,
        method: "tools/call",
        params: { name: "shell_greet", arguments: { name: "tester" } },
      });

      expect(results).toHaveLength(1);
      const resultContent = JSON.parse(results[0].result.content[0].text);
      expect(resultContent.stdout).toContain("Shell says: tester");
      expect(resultContent.outputs.greeting).toBe("Hello from shell");
    });

    it("should handle tools/list request", async () => {
      // Create config file with multiple tools
      const configPath = path.join(tempDir, "config.json");
      const config = {
        serverName: "test-server",
        version: "2.0.0",
        tools: [
          {
            name: "tool_one",
            description: "First tool",
            inputSchema: { type: "object", properties: { a: { type: "string" } } },
          },
          {
            name: "tool_two",
            description: "Second tool",
            inputSchema: { type: "object", properties: { b: { type: "number" } } },
          },
        ],
      };
      fs.writeFileSync(configPath, JSON.stringify(config));

      const { loadConfig } = await import("./safe_inputs_mcp_server.cjs");
      const { createServer, registerTool, handleMessage } = await import("./mcp_server_core.cjs");

      const loadedConfig = loadConfig(configPath);
      const server = createServer({ name: loadedConfig.serverName, version: loadedConfig.version });

      // Register tools (no handlers, just for listing)
      for (const tool of loadedConfig.tools) {
        registerTool(server, tool);
      }

      // Test tools/list
      const results = [];
      server.writeMessage = msg => results.push(msg);
      server.replyResult = (id, result) => results.push({ jsonrpc: "2.0", id, result });
      server.replyError = (id, code, message) => results.push({ jsonrpc: "2.0", id, error: { code, message } });

      await handleMessage(server, {
        jsonrpc: "2.0",
        id: 1,
        method: "tools/list",
      });

      expect(results).toHaveLength(1);
      expect(results[0].result.tools).toHaveLength(2);

      const toolNames = results[0].result.tools.map(t => t.name);
      expect(toolNames).toContain("tool_one");
      expect(toolNames).toContain("tool_two");
    });

    it("should handle initialize request", async () => {
      const configPath = path.join(tempDir, "config.json");
      fs.writeFileSync(configPath, JSON.stringify({ tools: [{ name: "dummy", description: "dummy", inputSchema: {} }] }));

      const { loadConfig } = await import("./safe_inputs_mcp_server.cjs");
      const { createServer, registerTool, handleMessage } = await import("./mcp_server_core.cjs");

      const loadedConfig = loadConfig(configPath);
      const server = createServer({ name: "safeinputs", version: "1.0.0" });

      for (const tool of loadedConfig.tools) {
        registerTool(server, tool);
      }

      const results = [];
      server.writeMessage = msg => results.push(msg);
      server.replyResult = (id, result) => results.push({ jsonrpc: "2.0", id, result });
      server.replyError = (id, code, message) => results.push({ jsonrpc: "2.0", id, error: { code, message } });

      await handleMessage(server, {
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: { protocolVersion: "2024-11-05" },
      });

      expect(results).toHaveLength(1);
      expect(results[0].result.serverInfo).toEqual({ name: "safeinputs", version: "1.0.0" });
      expect(results[0].result.protocolVersion).toBe("2024-11-05");
      expect(results[0].result.capabilities).toEqual({ tools: {} });
    });

    it("should use default server name and version if not provided", async () => {
      const configPath = path.join(tempDir, "config.json");
      fs.writeFileSync(
        configPath,
        JSON.stringify({
          tools: [{ name: "test", description: "test", inputSchema: {} }],
        })
      );

      const { loadConfig } = await import("./safe_inputs_mcp_server.cjs");
      const { createServer, registerTool, handleMessage } = await import("./mcp_server_core.cjs");

      const loadedConfig = loadConfig(configPath);
      const serverName = loadedConfig.serverName || "safeinputs";
      const version = loadedConfig.version || "1.0.0";
      const server = createServer({ name: serverName, version });

      for (const tool of loadedConfig.tools) {
        registerTool(server, tool);
      }

      const results = [];
      server.replyResult = (id, result) => results.push({ jsonrpc: "2.0", id, result });

      await handleMessage(server, {
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: {},
      });

      expect(results[0].result.serverInfo.name).toBe("safeinputs");
      expect(results[0].result.serverInfo.version).toBe("1.0.0");
    });
  });

  describe("error handling", () => {
    it("should return error for unknown tool", async () => {
      const configPath = path.join(tempDir, "config.json");
      fs.writeFileSync(configPath, JSON.stringify({ tools: [{ name: "known_tool", description: "test", inputSchema: {} }] }));

      const { loadConfig } = await import("./safe_inputs_mcp_server.cjs");
      const { createServer, registerTool, handleMessage } = await import("./mcp_server_core.cjs");

      const loadedConfig = loadConfig(configPath);
      const server = createServer({ name: "safeinputs", version: "1.0.0" });

      for (const tool of loadedConfig.tools) {
        registerTool(server, tool);
      }

      const results = [];
      server.replyResult = (id, result) => results.push({ jsonrpc: "2.0", id, result });
      server.replyError = (id, code, message) => results.push({ jsonrpc: "2.0", id, error: { code, message } });

      await handleMessage(server, {
        jsonrpc: "2.0",
        id: 1,
        method: "tools/call",
        params: { name: "unknown_tool", arguments: {} },
      });

      expect(results).toHaveLength(1);
      expect(results[0].error.code).toBe(-32601);
      expect(results[0].error.message).toContain("Tool not found");
    });
  });
});
