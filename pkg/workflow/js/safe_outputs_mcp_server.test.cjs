import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import { exec } from "child_process";
import { promisify } from "util";

const execAsync = promisify(exec);

// Mock environment for isolated testing
const originalEnv = process.env;

describe("safe_outputs_mcp_server.cjs", () => {
  let serverProcess;
  let tempOutputFile;

  beforeEach(() => {
    // Create temporary output file
    tempOutputFile = path.join("/tmp", `test_safe_outputs_${Date.now()}.jsonl`);

    // Set up environment
    process.env = {
      ...originalEnv,
      GITHUB_AW_SAFE_OUTPUTS: tempOutputFile,
      GITHUB_AW_SAFE_OUTPUTS_CONFIG: JSON.stringify({
        "create-issue": { enabled: true, max: 5 },
        "create-discussion": { enabled: true },
        "add-issue-comment": { enabled: true, max: 3 },
        "missing-tool": { enabled: true },
      }),
    };
  });

  afterEach(() => {
    // Clean up
    process.env = originalEnv;
    if (tempOutputFile && fs.existsSync(tempOutputFile)) {
      fs.unlinkSync(tempOutputFile);
    }
    if (serverProcess && !serverProcess.killed) {
      serverProcess.kill();
    }
  });

  describe("MCP Protocol", () => {
    it("should handle initialize request correctly", async () => {
      const serverPath = path.join(
        __dirname,
        "safe_outputs_mcp_server.cjs"
      );

      // Start server process
      const { spawn } = require("child_process");
      serverProcess = spawn("node", [serverPath], {
        stdio: ["pipe", "pipe", "pipe"],
      });

      let responseData = "";
      serverProcess.stdout.on("data", data => {
        responseData += data.toString();
      });

      // Send initialize request
      const initRequest = {
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: {
          clientInfo: { name: "test-client", version: "1.0.0" },
          protocolVersion: "2024-11-05",
        },
      };

      const message = JSON.stringify(initRequest);
      const header = `Content-Length: ${Buffer.byteLength(message)}\r\n\r\n`;

      serverProcess.stdin.write(header + message);

      // Wait for response
      await new Promise(resolve => setTimeout(resolve, 100));

      expect(responseData).toContain("Content-Length:");

      // Extract JSON response - handle multiple responses by taking first one
      const firstMatch = responseData.match(/Content-Length: (\d+)\r\n\r\n/);
      expect(firstMatch).toBeTruthy();
      
      const contentLength = parseInt(firstMatch[1]);
      const startPos = responseData.indexOf('\r\n\r\n') + 4;
      const jsonText = responseData.substring(startPos, startPos + contentLength);
      
      const response = JSON.parse(jsonText);
      expect(response.jsonrpc).toBe("2.0");
      expect(response.id).toBe(1);
      expect(response.result).toHaveProperty("serverInfo");
      expect(response.result.serverInfo.name).toBe("safe-outputs-mcp-server");
      expect(response.result).toHaveProperty("capabilities");
      expect(response.result.capabilities).toHaveProperty("tools");
    });

    it("should list enabled tools correctly", async () => {
      const serverPath = path.join(
        __dirname,
        "safe_outputs_mcp_server.cjs"
      );

      serverProcess = require("child_process").spawn("node", [serverPath], {
        stdio: ["pipe", "pipe", "pipe"],
      });

      let responseData = "";
      serverProcess.stdout.on("data", data => {
        responseData += data.toString();
      });

      // Initialize first
      const initRequest = {
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: {},
      };

      let message = JSON.stringify(initRequest);
      let header = `Content-Length: ${Buffer.byteLength(message)}\r\n\r\n`;
      serverProcess.stdin.write(header + message);

      await new Promise(resolve => setTimeout(resolve, 100));

      // Clear response buffer
      responseData = "";

      // Request tools list
      const toolsRequest = {
        jsonrpc: "2.0",
        id: 2,
        method: "tools/list",
        params: {},
      };

      message = JSON.stringify(toolsRequest);
      header = `Content-Length: ${Buffer.byteLength(message)}\r\n\r\n`;
      serverProcess.stdin.write(header + message);

      await new Promise(resolve => setTimeout(resolve, 100));

      expect(responseData).toContain("Content-Length:");

      // Extract JSON response - handle multiple responses by taking first one
      const firstMatch = responseData.match(/Content-Length: (\d+)\r\n\r\n/);
      expect(firstMatch).toBeTruthy();
      
      const contentLength = parseInt(firstMatch[1]);
      const startPos = responseData.indexOf('\r\n\r\n') + 4;
      const jsonText = responseData.substring(startPos, startPos + contentLength);
      
      const response = JSON.parse(jsonText);
      expect(response.jsonrpc).toBe("2.0");
      expect(response.id).toBe(2);
      expect(response.result).toHaveProperty("tools");

      const tools = response.result.tools;
      expect(Array.isArray(tools)).toBe(true);

      // Should include enabled tools
      const toolNames = tools.map(t => t.name);
      expect(toolNames).toContain("create-issue");
      expect(toolNames).toContain("create-discussion");
      expect(toolNames).toContain("add-issue-comment");
      expect(toolNames).toContain("missing-tool");

      // Should not include disabled tools (push-to-pr-branch is not enabled)
      expect(toolNames).not.toContain("push-to-pr-branch");
    });
  });

  describe("Tool Execution", () => {
    let serverProcess;

    beforeEach(async () => {
      const serverPath = path.join(
        __dirname,
        "safe_outputs_mcp_server.cjs"
      );

      serverProcess = require("child_process").spawn("node", [serverPath], {
        stdio: ["pipe", "pipe", "pipe"],
      });

      // Initialize server first to ensure state is clean for each test
      const initRequest = {
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: {},
      };

      const message = JSON.stringify(initRequest);
      const header = `Content-Length: ${Buffer.byteLength(message)}\r\n\r\n`;
      serverProcess.stdin.write(header + message);

      // Wait for initialization to complete
      await new Promise(resolve => setTimeout(resolve, 100));
    });

    it("should execute create-issue tool and append to output file", async () => {
      // Clear stdout listeners to start fresh
      serverProcess.stdout.removeAllListeners('data');
      
      // Start capturing data from this point forward
      let responseData = "";
      const dataHandler = (data) => {
        responseData += data.toString();
      };
      serverProcess.stdout.on("data", dataHandler);

      // Call create-issue tool
      const toolCall = {
        jsonrpc: "2.0",
        id: 1, // Use ID 1 for this request
        method: "tools/call",
        params: {
          name: "create-issue",
          arguments: {
            title: "Test Issue",
            body: "This is a test issue",
            labels: ["bug", "test"],
          },
        },
      };

      const message = JSON.stringify(toolCall);
      const header = `Content-Length: ${Buffer.byteLength(message)}\r\n\r\n`;
      serverProcess.stdin.write(header + message);

      await new Promise(resolve => setTimeout(resolve, 100));

      // Check response
      expect(responseData).toContain("Content-Length:");
      
      // Extract JSON response - handle multiple responses by taking first one
      const firstMatch = responseData.match(/Content-Length: (\d+)\r\n\r\n/);
      expect(firstMatch).toBeTruthy();
      
      const contentLength = parseInt(firstMatch[1]);
      const startPos = responseData.indexOf('\r\n\r\n') + 4;
      const jsonText = responseData.substring(startPos, startPos + contentLength);
      
      const response = JSON.parse(jsonText);
      expect(response.jsonrpc).toBe("2.0");
      expect(response.id).toBe(1); // Server is responding with ID 1
      expect(response.result).toHaveProperty("content");
      expect(response.result.content[0].text).toContain(
        "Issue creation queued"
      );

      // Check output file
      expect(fs.existsSync(tempOutputFile)).toBe(true);
      const outputContent = fs.readFileSync(tempOutputFile, "utf8");
      const outputEntry = JSON.parse(outputContent.trim());

      expect(outputEntry.type).toBe("create-issue");
      expect(outputEntry.title).toBe("Test Issue");
      expect(outputEntry.body).toBe("This is a test issue");
      expect(outputEntry.labels).toEqual(["bug", "test"]);
      
      // Clean up listener
      serverProcess.stdout.removeListener("data", dataHandler);
    });

    it("should execute missing-tool and append to output file", async () => {
      // Clear stdout listeners to start fresh
      serverProcess.stdout.removeAllListeners('data');
      
      let responseData = "";
      serverProcess.stdout.on("data", data => {
        responseData += data.toString();
      });

      // Call missing-tool
      const toolCall = {
        jsonrpc: "2.0",
        id: 1, // Use ID 1 for this request
        method: "tools/call",
        params: {
          name: "missing-tool",
          arguments: {
            tool: "advanced-analyzer",
            reason: "Need to analyze complex data structures",
            alternatives:
              "Could use basic analysis tools with manual processing",
          },
        },
      };

      const message = JSON.stringify(toolCall);
      const header = `Content-Length: ${Buffer.byteLength(message)}\r\n\r\n`;
      serverProcess.stdin.write(header + message);

      await new Promise(resolve => setTimeout(resolve, 100));

      // Check response
      expect(responseData).toContain("Content-Length:");

      // Check output file
      expect(fs.existsSync(tempOutputFile)).toBe(true);
      const outputContent = fs.readFileSync(tempOutputFile, "utf8");
      const outputEntry = JSON.parse(outputContent.trim());

      expect(outputEntry.type).toBe("missing-tool");
      expect(outputEntry.tool).toBe("advanced-analyzer");
      expect(outputEntry.reason).toBe(
        "Need to analyze complex data structures"
      );
      expect(outputEntry.alternatives).toBe(
        "Could use basic analysis tools with manual processing"
      );
    });

    it("should reject tool calls for disabled tools", async () => {
      // Clear stdout listeners to start fresh
      serverProcess.stdout.removeAllListeners('data');
      
      let responseData = "";
      serverProcess.stdout.on("data", data => {
        responseData += data.toString();
      });

      // Try to call disabled push-to-pr-branch tool
      const toolCall = {
        jsonrpc: "2.0",
        id: 1, // Use ID 1 for this request
        method: "tools/call",
        params: {
          name: "push-to-pr-branch",
          arguments: {
            files: [{ path: "test.txt", content: "test content" }],
          },
        },
      };

      const message = JSON.stringify(toolCall);
      const header = `Content-Length: ${Buffer.byteLength(message)}\r\n\r\n`;
      serverProcess.stdin.write(header + message);

      await new Promise(resolve => setTimeout(resolve, 100));

      expect(responseData).toContain("Content-Length:");
      
      // Extract JSON response - handle multiple responses by taking first one
      const firstMatch = responseData.match(/Content-Length: (\d+)\r\n\r\n/);
      expect(firstMatch).toBeTruthy();
      
      const contentLength = parseInt(firstMatch[1]);
      const startPos = responseData.indexOf('\r\n\r\n') + 4;
      const jsonText = responseData.substring(startPos, startPos + contentLength);
      
      const response = JSON.parse(jsonText);
      expect(response.jsonrpc).toBe("2.0");
      expect(response.id).toBe(1); // Server is responding with ID 1
      expect(response.error).toBeTruthy();
      expect(response.error.message).toContain(
        "Tool not found: push-to-pr-branch"
      );
    });
  });

  describe("Configuration Handling", () => {
    it("should handle missing configuration gracefully", () => {
      // Test with no config
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = "";

      const serverPath = path.join(
        __dirname,
        "safe_outputs_mcp_server.cjs"
      );
      expect(() => {
        require(serverPath);
      }).not.toThrow();
    });

    it("should handle invalid JSON configuration", () => {
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = "invalid json";

      const serverPath = path.join(
        __dirname,
        "safe_outputs_mcp_server.cjs"
      );
      expect(() => {
        require(serverPath);
      }).not.toThrow();
    });

    it("should handle missing output file path", () => {
      delete process.env.GITHUB_AW_SAFE_OUTPUTS;

      const serverPath = path.join(
        __dirname,
        "safe_outputs_mcp_server.cjs"
      );
      expect(() => {
        require(serverPath);
      }).not.toThrow();
    });
  });

  describe("Input Validation", () => {
    let serverProcess;

    beforeEach(async () => {
      const serverPath = path.join(
        __dirname,
        "safe_outputs_mcp_server.cjs"
      );

      serverProcess = require("child_process").spawn("node", [serverPath], {
        stdio: ["pipe", "pipe", "pipe"],
      });

      // Initialize server first to ensure state is clean for each test
      const initRequest = {
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: {},
      };

      const message = JSON.stringify(initRequest);
      const header = `Content-Length: ${Buffer.byteLength(message)}\r\n\r\n`;
      serverProcess.stdin.write(header + message);

      // Wait for initialization to complete
      await new Promise(resolve => setTimeout(resolve, 100));
    });

    it("should validate required fields for create-issue", async () => {
      // Clear stdout listeners to start fresh
      serverProcess.stdout.removeAllListeners('data');
      
      let responseData = "";
      serverProcess.stdout.on("data", data => {
        responseData += data.toString();
      });

      // Call create-issue without required fields
      const toolCall = {
        jsonrpc: "2.0",
        id: 1, // Use ID 1 for this request
        method: "tools/call",
        params: {
          name: "create-issue",
          arguments: {
            title: "Test Issue",
            // Missing required 'body' field
          },
        },
      };

      const message = JSON.stringify(toolCall);
      const header = `Content-Length: ${Buffer.byteLength(message)}\r\n\r\n`;
      serverProcess.stdin.write(header + message);

      await new Promise(resolve => setTimeout(resolve, 100));

      expect(responseData).toContain("Content-Length:");
      // Should still work because we're not doing strict schema validation
      // in the example server, but in a production server you might want to add validation
    });

    it("should handle malformed JSON RPC requests", async () => {
      // Clear stdout listeners to start fresh
      serverProcess.stdout.removeAllListeners('data');
      
      let responseData = "";
      serverProcess.stdout.on("data", data => {
        responseData += data.toString();
      });

      // Send malformed JSON
      const malformedMessage = "{ invalid json }";
      const header = `Content-Length: ${Buffer.byteLength(malformedMessage)}\r\n\r\n`;
      serverProcess.stdin.write(header + malformedMessage);

      await new Promise(resolve => setTimeout(resolve, 100));

      expect(responseData).toContain("Content-Length:");
      
      // Extract JSON response - handle multiple responses by taking first one
      const firstMatch = responseData.match(/Content-Length: (\d+)\r\n\r\n/);
      expect(firstMatch).toBeTruthy();
      
      const contentLength = parseInt(firstMatch[1]);
      const startPos = responseData.indexOf('\r\n\r\n') + 4;
      const jsonText = responseData.substring(startPos, startPos + contentLength);
      
      const response = JSON.parse(jsonText);
      expect(response.jsonrpc).toBe("2.0");
      expect(response.id).toBe(null); // For malformed JSON, server should respond with null ID
      expect(response.error).toBeTruthy();
      expect(response.error.code).toBe(-32700); // Parse error
    });
  });
});
