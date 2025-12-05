import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";
import http from "http";

describe("safe_inputs_mcp_server_http.cjs", () => {
  let tempDir;
  let serverProcess;
  let serverPort;

  beforeEach(() => {
    vi.resetModules();
    // Suppress stderr output during tests
    vi.spyOn(process.stderr, "write").mockImplementation(() => true);

    // Create a temporary directory for test files
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "safe-inputs-http-test-"));
    serverPort = 3000 + Math.floor(Math.random() * 1000); // Random port to avoid conflicts
  });

  afterEach(async () => {
    // Clean up server process
    if (serverProcess) {
      serverProcess.kill("SIGTERM");
      await new Promise(resolve => serverProcess.on("close", resolve));
      serverProcess = null;
    }

    // Clean up temporary directory
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true });
    }
  });

  /**
   * Helper function to make HTTP request to the MCP server
   * @param {Object} payload - JSON-RPC request payload
   * @param {Object} [headers] - Additional headers
   * @returns {Promise<Object>} Response object
   */
  async function makeHttpRequest(payload, headers = {}) {
    return new Promise((resolve, reject) => {
      const data = JSON.stringify(payload);

      const options = {
        hostname: "localhost",
        port: serverPort,
        path: "/",
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Accept": "application/json, text/event-stream",
          "Content-Length": Buffer.byteLength(data),
          ...headers,
        },
      };

      const req = http.request(options, res => {
        let responseData = "";

        res.on("data", chunk => {
          responseData += chunk;
        });

        res.on("end", () => {
          try {
            const parsed = JSON.parse(responseData);
            resolve({ status: res.statusCode, data: parsed, headers: res.headers });
          } catch (e) {
            resolve({ status: res.statusCode, data: responseData, headers: res.headers });
          }
        });
      });

      req.on("error", reject);
      req.write(data);
      req.end();
    });
  }

  /**
   * Helper function to wait for server to be ready
   * @param {number} maxAttempts - Maximum number of connection attempts
   * @returns {Promise<boolean>} True if server is ready
   */
  async function waitForServer(maxAttempts = 20) {
    for (let i = 0; i < maxAttempts; i++) {
      try {
        await makeHttpRequest({ jsonrpc: "2.0", id: 0, method: "initialize", params: {} });
        return true;
      } catch {
        await new Promise(resolve => setTimeout(resolve, 100));
      }
    }
    return false;
  }

  describe("HTTP transport initialization", () => {
    it("should start HTTP server and respond to initialize request", async () => {
      // Create a minimal config file
      const configPath = path.join(tempDir, "config.json");
      const config = {
        serverName: "test-http-server",
        version: "1.0.0",
        tools: [
          {
            name: "test_tool",
            description: "A test tool",
            inputSchema: { type: "object", properties: { input: { type: "string" } } },
          },
        ],
      };
      fs.writeFileSync(configPath, JSON.stringify(config));

      // Copy required dependencies
      const filesToCopy = [
        "mcp_server_core.cjs",
        "read_buffer.cjs",
        "safe_inputs_config_loader.cjs",
        "safe_inputs_tool_factory.cjs",
        "mcp_handler_python.cjs",
        "mcp_handler_shell.cjs",
        "safe_inputs_mcp_server_http.cjs",
      ];

      for (const file of filesToCopy) {
        const srcPath = path.join(__dirname, file);
        const destPath = path.join(tempDir, file);
        if (fs.existsSync(srcPath)) {
          fs.copyFileSync(srcPath, destPath);
        }
      }

      // Start the HTTP server
      const { spawn } = await import("child_process");
      serverProcess = spawn("node", [path.join(tempDir, "safe_inputs_mcp_server_http.cjs"), configPath, "--port", serverPort.toString()], {
        cwd: tempDir,
        stdio: ["pipe", "pipe", "pipe"],
      });

      let stderrOutput = "";
      serverProcess.stderr.on("data", chunk => {
        stderrOutput += chunk.toString();
      });

      // Wait for server to be ready
      const ready = await waitForServer();
      expect(ready).toBe(true);

      // Send initialize request
      const response = await makeHttpRequest({
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: { protocolVersion: "2024-11-05" },
      });

      expect(response.status).toBe(200);
      expect(response.data.jsonrpc).toBe("2.0");
      expect(response.data.id).toBe(1);
      expect(response.data.result).toBeDefined();
      expect(response.data.result.serverInfo.name).toBe("test-http-server");
      expect(response.data.result.serverInfo.version).toBe("1.0.0");
      expect(response.data.result.protocolVersion).toBe("2024-11-05");
      expect(response.data.result.capabilities).toEqual({ tools: {} });
    }, 15000);

    it("should handle tools/list request over HTTP", async () => {
      // Create config with multiple tools
      const configPath = path.join(tempDir, "config.json");
      const config = {
        serverName: "list-test-server",
        version: "2.0.0",
        tools: [
          {
            name: "tool_one",
            description: "First test tool",
            inputSchema: { type: "object", properties: { a: { type: "string" } } },
          },
          {
            name: "tool_two",
            description: "Second test tool",
            inputSchema: { type: "object", properties: { b: { type: "number" } } },
          },
        ],
      };
      fs.writeFileSync(configPath, JSON.stringify(config));

      // Copy required dependencies
      const filesToCopy = [
        "mcp_server_core.cjs",
        "read_buffer.cjs",
        "safe_inputs_config_loader.cjs",
        "safe_inputs_tool_factory.cjs",
        "mcp_handler_python.cjs",
        "mcp_handler_shell.cjs",
        "safe_inputs_mcp_server_http.cjs",
      ];

      for (const file of filesToCopy) {
        const srcPath = path.join(__dirname, file);
        const destPath = path.join(tempDir, file);
        if (fs.existsSync(srcPath)) {
          fs.copyFileSync(srcPath, destPath);
        }
      }

      // Start the HTTP server
      const { spawn } = await import("child_process");
      serverProcess = spawn("node", [path.join(tempDir, "safe_inputs_mcp_server_http.cjs"), configPath, "--port", serverPort.toString()], {
        cwd: tempDir,
        stdio: ["pipe", "pipe", "pipe"],
      });

      // Wait for server to be ready
      const ready = await waitForServer();
      expect(ready).toBe(true);

      // Initialize first
      await makeHttpRequest({
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: {},
      });

      // Request tools list
      const response = await makeHttpRequest({
        jsonrpc: "2.0",
        id: 2,
        method: "tools/list",
      });

      expect(response.status).toBe(200);
      expect(response.data.result.tools).toHaveLength(2);

      const toolNames = response.data.result.tools.map(t => t.name);
      expect(toolNames).toContain("tool_one");
      expect(toolNames).toContain("tool_two");

      const toolOne = response.data.result.tools.find(t => t.name === "tool_one");
      expect(toolOne.description).toBe("First test tool");
      expect(toolOne.inputSchema).toEqual({ type: "object", properties: { a: { type: "string" } } });
    }, 15000);

    it("should execute JavaScript tool over HTTP", async () => {
      // Create a handler file
      const handlerPath = path.join(tempDir, "echo_handler.cjs");
      fs.writeFileSync(
        handlerPath,
        `module.exports = function(args) {
          return { message: "Echo: " + args.message };
        };`
      );

      // Create config file
      const configPath = path.join(tempDir, "config.json");
      const config = {
        serverName: "echo-server",
        version: "1.0.0",
        tools: [
          {
            name: "echo",
            description: "Echoes the input message",
            inputSchema: {
              type: "object",
              properties: { message: { type: "string" } },
              required: ["message"],
            },
            handler: "echo_handler.cjs",
          },
        ],
      };
      fs.writeFileSync(configPath, JSON.stringify(config));

      // Copy required dependencies
      const filesToCopy = [
        "mcp_server_core.cjs",
        "read_buffer.cjs",
        "safe_inputs_config_loader.cjs",
        "safe_inputs_tool_factory.cjs",
        "mcp_handler_python.cjs",
        "mcp_handler_shell.cjs",
        "safe_inputs_mcp_server_http.cjs",
      ];

      for (const file of filesToCopy) {
        const srcPath = path.join(__dirname, file);
        const destPath = path.join(tempDir, file);
        if (fs.existsSync(srcPath)) {
          fs.copyFileSync(srcPath, destPath);
        }
      }

      // Start the HTTP server
      const { spawn } = await import("child_process");
      serverProcess = spawn("node", [path.join(tempDir, "safe_inputs_mcp_server_http.cjs"), configPath, "--port", serverPort.toString()], {
        cwd: tempDir,
        stdio: ["pipe", "pipe", "pipe"],
      });

      // Wait for server to be ready
      const ready = await waitForServer();
      expect(ready).toBe(true);

      // Initialize first
      await makeHttpRequest({
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: {},
      });

      // Call the echo tool
      const response = await makeHttpRequest({
        jsonrpc: "2.0",
        id: 2,
        method: "tools/call",
        params: {
          name: "echo",
          arguments: { message: "Hello, HTTP!" },
        },
      });

      expect(response.status).toBe(200);
      expect(response.data.jsonrpc).toBe("2.0");
      expect(response.data.id).toBe(2);
      expect(response.data.result).toBeDefined();
      expect(response.data.result.content).toBeDefined();
      expect(response.data.result.content.length).toBe(1);
      expect(response.data.result.content[0].type).toBe("text");

      const echoResult = JSON.parse(response.data.result.content[0].text);
      expect(echoResult.message).toBe("Echo: Hello, HTTP!");
    }, 15000);

    it("should handle CORS preflight requests", async () => {
      // Create minimal config
      const configPath = path.join(tempDir, "config.json");
      const config = {
        tools: [
          {
            name: "test",
            description: "test",
            inputSchema: { type: "object", properties: {} },
          },
        ],
      };
      fs.writeFileSync(configPath, JSON.stringify(config));

      // Copy required dependencies
      const filesToCopy = [
        "mcp_server_core.cjs",
        "read_buffer.cjs",
        "safe_inputs_config_loader.cjs",
        "safe_inputs_tool_factory.cjs",
        "mcp_handler_python.cjs",
        "mcp_handler_shell.cjs",
        "safe_inputs_mcp_server_http.cjs",
      ];

      for (const file of filesToCopy) {
        const srcPath = path.join(__dirname, file);
        const destPath = path.join(tempDir, file);
        if (fs.existsSync(srcPath)) {
          fs.copyFileSync(srcPath, destPath);
        }
      }

      // Start the HTTP server
      const { spawn } = await import("child_process");
      serverProcess = spawn("node", [path.join(tempDir, "safe_inputs_mcp_server_http.cjs"), configPath, "--port", serverPort.toString()], {
        cwd: tempDir,
        stdio: ["pipe", "pipe", "pipe"],
      });

      // Wait for server to be ready
      const ready = await waitForServer();
      expect(ready).toBe(true);

      // Make OPTIONS request
      const response = await new Promise((resolve, reject) => {
        const options = {
          hostname: "localhost",
          port: serverPort,
          path: "/",
          method: "OPTIONS",
        };

        const req = http.request(options, res => {
          resolve({
            status: res.statusCode,
            headers: res.headers,
          });
        });

        req.on("error", reject);
        req.end();
      });

      expect(response.status).toBe(200);
      expect(response.headers["access-control-allow-origin"]).toBe("*");
      expect(response.headers["access-control-allow-methods"]).toContain("POST");
      expect(response.headers["access-control-allow-headers"]).toContain("Content-Type");
    }, 15000);

    it("should reject invalid method with 405", async () => {
      // Create minimal config
      const configPath = path.join(tempDir, "config.json");
      const config = {
        tools: [
          {
            name: "test",
            description: "test",
            inputSchema: { type: "object", properties: {} },
          },
        ],
      };
      fs.writeFileSync(configPath, JSON.stringify(config));

      // Copy required dependencies
      const filesToCopy = [
        "mcp_server_core.cjs",
        "read_buffer.cjs",
        "safe_inputs_config_loader.cjs",
        "safe_inputs_tool_factory.cjs",
        "mcp_handler_python.cjs",
        "mcp_handler_shell.cjs",
        "safe_inputs_mcp_server_http.cjs",
      ];

      for (const file of filesToCopy) {
        const srcPath = path.join(__dirname, file);
        const destPath = path.join(tempDir, file);
        if (fs.existsSync(srcPath)) {
          fs.copyFileSync(srcPath, destPath);
        }
      }

      // Start the HTTP server
      const { spawn } = await import("child_process");
      serverProcess = spawn("node", [path.join(tempDir, "safe_inputs_mcp_server_http.cjs"), configPath, "--port", serverPort.toString()], {
        cwd: tempDir,
        stdio: ["pipe", "pipe", "pipe"],
      });

      // Wait for server to be ready
      const ready = await waitForServer();
      expect(ready).toBe(true);

      // Make PUT request (not allowed)
      const response = await new Promise((resolve, reject) => {
        const options = {
          hostname: "localhost",
          port: serverPort,
          path: "/",
          method: "PUT",
        };

        const req = http.request(options, res => {
          let data = "";
          res.on("data", chunk => {
            data += chunk;
          });
          res.on("end", () => {
            resolve({
              status: res.statusCode,
              data: JSON.parse(data),
            });
          });
        });

        req.on("error", reject);
        req.end();
      });

      expect(response.status).toBe(405);
      expect(response.data.error).toBe("Method not allowed");
    }, 15000);
  });

  describe("stateless mode", () => {
    it("should run in stateless mode without session management", async () => {
      // Create minimal config
      const configPath = path.join(tempDir, "config.json");
      const config = {
        serverName: "stateless-server",
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

      // Copy required dependencies
      const filesToCopy = [
        "mcp_server_core.cjs",
        "read_buffer.cjs",
        "safe_inputs_config_loader.cjs",
        "safe_inputs_tool_factory.cjs",
        "mcp_handler_python.cjs",
        "mcp_handler_shell.cjs",
        "safe_inputs_mcp_server_http.cjs",
      ];

      for (const file of filesToCopy) {
        const srcPath = path.join(__dirname, file);
        const destPath = path.join(tempDir, file);
        if (fs.existsSync(srcPath)) {
          fs.copyFileSync(srcPath, destPath);
        }
      }

      // Start the HTTP server in stateless mode
      const { spawn } = await import("child_process");
      serverProcess = spawn("node", [path.join(tempDir, "safe_inputs_mcp_server_http.cjs"), configPath, "--port", serverPort.toString(), "--stateless"], {
        cwd: tempDir,
        stdio: ["pipe", "pipe", "pipe"],
      });

      // Wait for server to be ready
      const ready = await waitForServer();
      expect(ready).toBe(true);

      // Send initialize request
      const response = await makeHttpRequest({
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: {},
      });

      expect(response.status).toBe(200);
      expect(response.data.result.serverInfo.name).toBe("stateless-server");

      // In stateless mode, there should be no session ID in headers
      // (This depends on the MCP SDK implementation - we're just verifying the server works)
    }, 15000);
  });
});
