import { describe, it, expect, afterAll } from "vitest";
import http from "http";
import { spawn } from "child_process";
import fs from "fs";
import path from "path";
import os from "os";

/**
 * Integration tests for safe_inputs_mcp_server_http.cjs
 *
 * These tests validate that the HTTP transport layer works correctly with the MCP protocol.
 * They spawn actual server processes and make real HTTP requests to test the integration.
 */
describe("safe_inputs_mcp_server_http.cjs integration", () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "http-mcp-test-"));
  const serverPort = 4100;
  let serverProcess = null;
  let sessionId = null;

  // Create test files before all tests
  const configPath = path.join(tempDir, "test-config.json");
  const handlerPath = path.join(tempDir, "echo-handler.cjs");

  fs.writeFileSync(
    handlerPath,
    `module.exports = function(args) {
      return { echo: args.message || "empty", timestamp: Date.now() };
    };`
  );

  fs.writeFileSync(
    configPath,
    JSON.stringify({
      serverName: "http-integration-test-server",
      version: "1.0.0",
      tools: [
        {
          name: "echo_tool",
          description: "Echoes the input message",
          inputSchema: {
            type: "object",
            properties: {
              message: { type: "string", description: "Message to echo" },
            },
            required: ["message"],
          },
          handler: "echo-handler.cjs",
        },
      ],
    })
  );

  /**
   * Helper to make HTTP requests to the MCP server
   */
  async function makeRequest(payload, additionalHeaders = {}) {
    return new Promise((resolve, reject) => {
      const data = JSON.stringify(payload);
      const headers = {
        "Content-Type": "application/json",
        Accept: "application/json, text/event-stream",
        "Content-Length": Buffer.byteLength(data),
        ...additionalHeaders,
      };

      const req = http.request(
        {
          hostname: "localhost",
          port: serverPort,
          path: "/",
          method: "POST",
          headers,
        },
        res => {
          let responseData = "";
          res.on("data", chunk => {
            responseData += chunk;
          });
          res.on("end", () => {
            try {
              resolve({
                status: res.statusCode,
                data: JSON.parse(responseData),
                headers: res.headers,
              });
            } catch (e) {
              reject(new Error(`Failed to parse response: ${responseData}`));
            }
          });
        }
      );

      req.on("error", reject);
      req.write(data);
      req.end();
    });
  }

  /**
   * Wait for server to be ready
   */
  async function waitForServer(maxAttempts = 30) {
    for (let i = 0; i < maxAttempts; i++) {
      try {
        const response = await makeRequest({
          jsonrpc: "2.0",
          id: 0,
          method: "initialize",
          params: { protocolVersion: "2024-11-05" },
        });
        if (response.status === 200 && response.data.result) {
          // Store session ID for later use
          sessionId = response.headers["mcp-session-id"];
          return true;
        }
      } catch {
        // Server not ready yet
      }
      await new Promise(resolve => setTimeout(resolve, 200));
    }
    return false;
  }

  /**
   * Start the HTTP MCP server before running tests
   */
  it("should start HTTP MCP server successfully", { timeout: 15000 }, async () => {
    serverProcess = spawn("node", ["safe_inputs_mcp_server_http.cjs", configPath, "--port", serverPort.toString()], {
      cwd: process.cwd(),
      stdio: ["pipe", "pipe", "pipe"],
    });

    let serverOutput = "";
    serverProcess.stderr.on("data", chunk => {
      serverOutput += chunk.toString();
    });

    serverProcess.on("error", error => {
      console.error("Server process error:", error);
    });

    // Wait for server to be ready
    const ready = await waitForServer();
    expect(ready).toBe(true);
    expect(serverOutput).toContain("HTTP server listening");
  });

  it("should respond to GET health check requests", async () => {
    return new Promise((resolve, reject) => {
      const req = http.request(
        {
          hostname: "localhost",
          port: serverPort,
          path: "/",
          method: "GET",
        },
        res => {
          let responseData = "";
          res.on("data", chunk => {
            responseData += chunk;
          });
          res.on("end", () => {
            try {
              expect(res.statusCode).toBe(200);
              const data = JSON.parse(responseData);
              expect(data.status).toBe("ok");
              expect(data.server).toBe("http-integration-test-server");
              expect(data.version).toBe("1.0.0");
              expect(data.tools).toBe(1);
              resolve();
            } catch (e) {
              reject(e);
            }
          });
        }
      );

      req.on("error", reject);
      req.end();
    });
  });

  it("should initialize with proper MCP protocol response", async () => {
    const response = await makeRequest({
      jsonrpc: "2.0",
      id: 1,
      method: "initialize",
      params: {
        protocolVersion: "2024-11-05",
        clientInfo: { name: "test-client", version: "1.0.0" },
        capabilities: {},
      },
    });

    expect(response.status).toBe(200);
    expect(response.data.jsonrpc).toBe("2.0");
    expect(response.data.id).toBe(1);
    expect(response.data.result).toBeDefined();
    expect(response.data.result.protocolVersion).toBe("2024-11-05");
    expect(response.data.result.serverInfo.name).toBe("http-integration-test-server");
    expect(response.data.result.serverInfo.version).toBe("1.0.0");
    expect(response.data.result.capabilities.tools).toBeDefined();
    expect(response.headers["mcp-session-id"]).toBeDefined();

    // Store session ID for subsequent requests
    sessionId = response.headers["mcp-session-id"];
  });

  it("should list tools via HTTP", async () => {
    const headers = sessionId ? { "Mcp-Session-Id": sessionId } : {};

    const response = await makeRequest(
      {
        jsonrpc: "2.0",
        id: 2,
        method: "tools/list",
      },
      headers
    );

    expect(response.status).toBe(200);
    expect(response.data.result).toBeDefined();
    expect(response.data.result.tools).toBeInstanceOf(Array);
    expect(response.data.result.tools.length).toBe(1);

    const tool = response.data.result.tools[0];
    expect(tool.name).toBe("echo_tool");
    expect(tool.description).toBe("Echoes the input message");
    expect(tool.inputSchema).toBeDefined();
    expect(tool.inputSchema.properties.message).toBeDefined();
  });

  it("should execute tool via HTTP", async () => {
    const headers = sessionId ? { "Mcp-Session-Id": sessionId } : {};

    const response = await makeRequest(
      {
        jsonrpc: "2.0",
        id: 3,
        method: "tools/call",
        params: {
          name: "echo_tool",
          arguments: {
            message: "Hello from HTTP transport!",
          },
        },
      },
      headers
    );

    expect(response.status).toBe(200);
    expect(response.data.result).toBeDefined();
    expect(response.data.result.content).toBeInstanceOf(Array);
    expect(response.data.result.content.length).toBe(1);
    expect(response.data.result.content[0].type).toBe("text");

    const result = JSON.parse(response.data.result.content[0].text);
    expect(result.echo).toBe("Hello from HTTP transport!");
    expect(result.timestamp).toBeDefined();
  });

  it("should handle CORS preflight requests", async () => {
    return new Promise((resolve, reject) => {
      const req = http.request(
        {
          hostname: "localhost",
          port: serverPort,
          path: "/",
          method: "OPTIONS",
        },
        res => {
          expect(res.statusCode).toBe(200);
          expect(res.headers["access-control-allow-origin"]).toBe("*");
          expect(res.headers["access-control-allow-methods"]).toContain("POST");
          expect(res.headers["access-control-allow-headers"]).toContain("Content-Type");
          resolve();
        }
      );

      req.on("error", reject);
      req.end();
    });
  });

  it("should reject invalid HTTP methods", async () => {
    return new Promise((resolve, reject) => {
      const req = http.request(
        {
          hostname: "localhost",
          port: serverPort,
          path: "/",
          method: "PUT",
        },
        res => {
          let data = "";
          res.on("data", chunk => {
            data += chunk;
          });
          res.on("end", () => {
            expect(res.statusCode).toBe(405);
            const parsed = JSON.parse(data);
            expect(parsed.error).toBe("Method not allowed");
            resolve();
          });
        }
      );

      req.on("error", reject);
      req.end();
    });
  });

  it("should handle missing required arguments", async () => {
    const headers = sessionId ? { "Mcp-Session-Id": sessionId } : {};

    const response = await makeRequest(
      {
        jsonrpc: "2.0",
        id: 4,
        method: "tools/call",
        params: {
          name: "echo_tool",
          arguments: {},
        },
      },
      headers
    );

    expect(response.status).toBe(200);
    expect(response.data.error).toBeDefined();
    expect(response.data.error.message).toContain("missing");
  });

  afterAll(async () => {
    // Clean up server process
    if (serverProcess) {
      serverProcess.kill("SIGTERM");
      await new Promise(resolve => {
        serverProcess.on("close", () => {
          resolve();
        });
        // Force kill after timeout
        setTimeout(() => {
          if (serverProcess && !serverProcess.killed) {
            serverProcess.kill("SIGKILL");
            resolve();
          }
        }, 2000);
      });
    }

    // Clean up temporary directory
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  }, 10000);
});
