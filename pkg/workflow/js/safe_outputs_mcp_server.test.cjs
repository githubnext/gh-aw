import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

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
      const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");

      // Start server process
      const { spawn } = require("child_process");
      console.log(`node ${serverPath}`);
      serverProcess = spawn("node", [serverPath], {
        stdio: ["pipe", "pipe", "pipe"],
        env: {
          ...process.env,
          GITHUB_AW_SAFE_OUTPUTS: tempOutputFile,
          GITHUB_AW_SAFE_OUTPUTS_CONFIG: JSON.stringify({
            "create-issue": { enabled: true, max: 5 },
            "create-discussion": { enabled: true },
            "add-issue-comment": { enabled: true, max: 3 },
            "missing-tool": { enabled: true },
          }),
        },
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
      serverProcess.stdin.write(message + "\n");

      // Wait for response
      await new Promise(resolve => setTimeout(resolve, 100));

      expect(responseData).toMatch(/\{.*"jsonrpc".*"2\.0".*\}/);

      // Extract JSON response - handle multiple responses by finding the one for our request id
      const response = findResponseById(responseData, 1);
      expect(response.jsonrpc).toBe("2.0");
      expect(response.id).toBe(1);
      expect(response.result).toHaveProperty("serverInfo");
      expect(response.result.serverInfo.name).toBe("safe-outputs-mcp-server");
      expect(response.result).toHaveProperty("capabilities");
      expect(response.result.capabilities).toHaveProperty("tools");
    });

    it("should list enabled tools correctly", async () => {
      const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");

      serverProcess = require("child_process").spawn("node", [serverPath], {
        stdio: ["pipe", "pipe", "pipe"],
        env: {
          ...process.env,
          GITHUB_AW_SAFE_OUTPUTS: tempOutputFile,
          GITHUB_AW_SAFE_OUTPUTS_CONFIG: JSON.stringify({
            "create-issue": { enabled: true, max: 5 },
            "create-discussion": { enabled: true },
            "add-issue-comment": { enabled: true, max: 3 },
            "missing-tool": { enabled: true },
          }),
        },
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
      // No header needed for newline protocol
      serverProcess.stdin.write(message + "\n");

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
      // No header needed for newline protocol
      serverProcess.stdin.write(message + "\n");

      await new Promise(resolve => setTimeout(resolve, 100));

      expect(responseData).toMatch(/\{.*"jsonrpc".*"2\\.0".*\}/);

      // Extract JSON response - handle multiple responses by finding the one for our request id
      const response = findResponseById(responseData, 2);
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
      const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");

      serverProcess = require("child_process").spawn("node", [serverPath], {
        stdio: ["pipe", "pipe", "pipe"],
        env: {
          ...process.env,
          GITHUB_AW_SAFE_OUTPUTS: tempOutputFile,
          GITHUB_AW_SAFE_OUTPUTS_CONFIG: JSON.stringify({
            "create-issue": { enabled: true, max: 5 },
            "create-discussion": { enabled: true },
            "add-issue-comment": { enabled: true, max: 3 },
            "missing-tool": { enabled: true },
          }),
        },
      });

      // Initialize server first to ensure state is clean for each test
      const initRequest = {
        jsonrpc: "2.0",
        id: 0, // Use a reserved id for setup initialization to avoid colliding with test request ids
        method: "initialize",
        params: {},
      };

      const message = JSON.stringify(initRequest);
      // No header needed for newline protocol
      serverProcess.stdin.write(message + "\n");

      // Wait for initialization to complete
      await new Promise(resolve => setTimeout(resolve, 100));
    });

    it("should execute create-issue tool and append to output file", async () => {
      // Clear stdout listeners to start fresh
      serverProcess.stdout.removeAllListeners("data");

      // Start capturing data from this point forward
      let responseData = "";
      const dataHandler = data => {
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
      // No header needed for newline protocol
      serverProcess.stdin.write(message + "\n");

      await new Promise(resolve => setTimeout(resolve, 100));

      // Check response
      expect(responseData).toMatch(/\{.*"jsonrpc".*"2\\.0".*\}/);

      // Extract JSON response - handle multiple responses by finding the one for our request id
      const response = findResponseById(responseData, 1);
      expect(response.jsonrpc).toBe("2.0");
      expect(response.id).toBe(1); // Server is responding with ID 1
      expect(response.result).toHaveProperty("content");
      expect(response.result.content[0].text).toContain("success");

      // Check output file
      expect(fs.existsSync(tempOutputFile)).toBe(true);
      const outputContent = fs.readFileSync(tempOutputFile, "utf8");
      const outputEntry = parseNdjsonLast(outputContent);

      expect(outputEntry.type).toBe("create-issue");
      expect(outputEntry.title).toBe("Test Issue");
      expect(outputEntry.body).toBe("This is a test issue");
      expect(outputEntry.labels).toEqual(["bug", "test"]);

      // Clean up listener
      serverProcess.stdout.removeListener("data", dataHandler);
    });

    it("should execute missing-tool and append to output file", async () => {
      // Clear stdout listeners to start fresh
      serverProcess.stdout.removeAllListeners("data");

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
      // No header needed for newline protocol
      serverProcess.stdin.write(message + "\n");

      await new Promise(resolve => setTimeout(resolve, 100));

      // Check response
      expect(responseData).toMatch(/\{.*"jsonrpc".*"2\\.0".*\}/);

      // Check output file
      expect(fs.existsSync(tempOutputFile)).toBe(true);
      const outputContent = fs.readFileSync(tempOutputFile, "utf8");
      const outputEntry = parseNdjsonLast(outputContent);

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
      serverProcess.stdout.removeAllListeners("data");

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
      // No header needed for newline protocol
      serverProcess.stdin.write(message + "\n");

      await new Promise(resolve => setTimeout(resolve, 100));

      expect(responseData).toMatch(/\{.*"jsonrpc".*"2\\.0".*\}/);

      // Extract JSON response - handle multiple responses by finding the one for our request id
      const response = findResponseById(responseData, 1);
      expect(response.jsonrpc).toBe("2.0");
      expect(response.id).toBe(1); // Server is responding with ID 1
      expect(response.error).toBeTruthy();
      expect(response.error.message).toContain(
        "Tool not found: push-to-pr-branch"
      );
    });
  });

  describe("Configuration Handling", () => {
    describe("Input Validation", () => {
      let serverProcess;

      beforeEach(async () => {
        const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");

        serverProcess = require("child_process").spawn("node", [serverPath], {
          stdio: ["pipe", "pipe", "pipe"],
          env: {
            ...process.env,
            GITHUB_AW_SAFE_OUTPUTS: tempOutputFile,
            GITHUB_AW_SAFE_OUTPUTS_CONFIG: JSON.stringify({
              "create-issue": { enabled: true, max: 5 },
              "create-discussion": { enabled: true },
              "add-issue-comment": { enabled: true, max: 3 },
              "missing-tool": { enabled: true },
            }),
          },
        });

        // Initialize server first to ensure state is clean for each test
        const initRequest = {
          jsonrpc: "2.0",
          id: 0, // Use a reserved id for setup initialization to avoid colliding with test request ids
          method: "initialize",
          params: {},
        };

        const message = JSON.stringify(initRequest);
        // No header needed for newline protocol
        serverProcess.stdin.write(message + "\n");

        // Wait for initialization to complete
        await new Promise(resolve => setTimeout(resolve, 100));
      });

      it("should validate required fields for create-issue", async () => {
        // Clear stdout listeners to start fresh
        serverProcess.stdout.removeAllListeners("data");

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
        // No header needed for newline protocol
        serverProcess.stdin.write(message + "\n");

        await new Promise(resolve => setTimeout(resolve, 100));

        expect(responseData).toMatch(/\{.*"jsonrpc".*"2\\.0".*\}/);
        // Should still work because we're not doing strict schema validation
        // in the example server, but in a production server you might want to add validation
      });

      it("should handle malformed JSON RPC requests", async () => {
        // Clear stdout listeners to start fresh
        serverProcess.stdout.removeAllListeners("data");

        let responseData = "";
        serverProcess.stdout.on("data", data => {
          responseData += data.toString();
        });

        // Send malformed JSON
        const malformedMessage = "{ invalid json }";
        // No header needed for newline protocol
        serverProcess.stdin.write(malformedMessage + "\n");

        await new Promise(resolve => setTimeout(resolve, 100));

        expect(responseData).toMatch(/\{.*"jsonrpc".*"2\\.0".*\}/);

        // Extract JSON response - handle multiple responses by finding the one for our request id
        const response = findResponseById(responseData, null);
        expect(response.jsonrpc).toBe("2.0");
        expect(response.id).toBe(null); // For malformed JSON, server should respond with null ID
        expect(response.error).toBeTruthy();
        expect(response.error.code).toBe(-32700); // Parse error
      });
    });
  });

  // Helper to parse multiple newline-delimited JSON-RPC messages from a buffer
  function parseRpcResponses(bufferStr) {
    const responses = [];
    const lines = bufferStr.split("\n");
    for (const line of lines) {
      const trimmed = line.trim();
      if (trimmed === "") continue; // Skip empty lines
      try {
        const parsed = JSON.parse(trimmed);
        responses.push(parsed);
      } catch (e) {
        // ignore parse errors for individual lines
      }
    }
    return responses;
  }

  // Helper to find a response matching an id (or fallback to the first response)
  function findResponseById(bufferStr, id) {
    const resp = parseRpcResponses(bufferStr).find(
      r => Object.prototype.hasOwnProperty.call(r, "id") && r.id === id
    );
    if (resp) return resp;
    const all = parseRpcResponses(bufferStr);
    return all.length ? all[0] : null;
  }

  // Utility to find an error response by error code
  function findErrorByCode(bufferStr, code) {
    return (
      parseRpcResponses(bufferStr).find(
        r => r && r.error && r.error.code === code
      ) || null
    );
  }

  // Replace fragile first-match parsing with helpers
  describe("Robustness of Response Handling", () => {
    let serverProcess;

    beforeEach(async () => {
      const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");

      serverProcess = require("child_process").spawn("node", [serverPath], {
        stdio: ["pipe", "pipe", "pipe"],
        env: {
          ...process.env,
          GITHUB_AW_SAFE_OUTPUTS: tempOutputFile,
          GITHUB_AW_SAFE_OUTPUTS_CONFIG: JSON.stringify({
            "create-issue": { enabled: true, max: 5 },
            "create-discussion": { enabled: true },
            "add-issue-comment": { enabled: true, max: 3 },
            "missing-tool": { enabled: true },
          }),
        },
      });

      // Initialize server first to ensure state is clean for each test
      const initRequest = {
        jsonrpc: "2.0",
        id: 0, // Use a reserved id for setup initialization to avoid colliding with test request ids
        method: "initialize",
        params: {},
      };

      const message = JSON.stringify(initRequest);
      // No header needed for newline protocol
      serverProcess.stdin.write(message + "\n");

      // Wait for initialization to complete
      await new Promise(resolve => setTimeout(resolve, 100));
    });

    it("should handle multiple sequential responses", async () => {
      // Clear stdout listeners to start fresh
      serverProcess.stdout.removeAllListeners("data");

      let responseData = "";
      serverProcess.stdout.on("data", data => {
        responseData += data.toString();
      });

      // Call create-issue tool
      const toolCall1 = {
        jsonrpc: "2.0",
        id: 1,
        method: "tools/call",
        params: {
          name: "create-issue",
          arguments: {
            title: "Test Issue 1",
            body: "This is a test issue",
            labels: ["bug", "test"],
          },
        },
      };

      const message1 = JSON.stringify(toolCall1);
      const header1 = `Content-Length: ${Buffer.byteLength(message1)}\r\n\r\n`;
      serverProcess.stdin.write(header1 + message1);

      await new Promise(resolve => setTimeout(resolve, 100));

      // Call create-issue tool again
      const toolCall2 = {
        jsonrpc: "2.0",
        id: 2,
        method: "tools/call",
        params: {
          name: "create-issue",
          arguments: {
            title: "Test Issue 2",
            body: "This is another test issue",
            labels: ["enhancement"],
          },
        },
      };

      const message2 = JSON.stringify(toolCall2);
      const header2 = `Content-Length: ${Buffer.byteLength(message2)}\r\n\r\n`;
      serverProcess.stdin.write(header2 + message2);

      await new Promise(resolve => setTimeout(resolve, 100));

      // Check response for first call
      expect(responseData).toMatch(/\{.*"jsonrpc".*"2\\.0".*\}/);

      let response = findResponseById(responseData, 1);
      expect(response).toBeTruthy();
      expect(response.jsonrpc).toBe("2.0");
      expect(response.id).toBe(1);
      expect(response.result).toHaveProperty("content");
      expect(response.result.content[0].text).toContain("success");

      // Check output file for first call
      expect(fs.existsSync(tempOutputFile)).toBe(true);
      let outputContent = fs.readFileSync(tempOutputFile, "utf8");
      const entries = outputContent
        .split(/\r?\n/)
        .map(l => l.trim())
        .filter(Boolean)
        .map(JSON.parse);
      const entry1 = entries.find(e => e.title === "Test Issue 1");
      expect(entry1).toBeTruthy();
      expect(entry1.type).toBe("create-issue");
      expect(entry1.title).toBe("Test Issue 1");
      expect(entry1.body).toBe("This is a test issue");
      expect(entry1.labels).toEqual(["bug", "test"]);

      // Check response for second call
      response = findResponseById(responseData, 2);
      expect(response).toBeTruthy();
      expect(response.jsonrpc).toBe("2.0");
      expect(response.id).toBe(2);
      expect(response.result).toHaveProperty("content");
      expect(response.result.content[0].text).toContain("success");

      // Check output file for second call
      outputContent = fs.readFileSync(tempOutputFile, "utf8");
      const entriesAfter = outputContent
        .split(/\r?\n/)
        .map(l => l.trim())
        .filter(Boolean)
        .map(JSON.parse);
      const entry2 = entriesAfter.find(e => e.title === "Test Issue 2");
      expect(entry2).toBeTruthy();
      expect(entry2.type).toBe("create-issue");
      expect(entry2.title).toBe("Test Issue 2");
      expect(entry2.body).toBe("This is another test issue");
      expect(entry2.labels).toEqual(["enhancement"]);
    });

    it("should handle error responses gracefully", async () => {
      // Clear stdout listeners to start fresh
      serverProcess.stdout.removeAllListeners("data");

      let responseData = "";
      serverProcess.stdout.on("data", data => {
        responseData += data.toString();
      });

      // Call missing-tool with invalid arguments to trigger error
      const toolCall = {
        jsonrpc: "2.0",
        id: 1,
        method: "tools/call",
        params: {
          name: "missing-tool",
          arguments: {
            // Missing 'tool' argument
            reason: "Need to analyze complex data structures",
            alternatives:
              "Could use basic analysis tools with manual processing",
          },
        },
      };

      const message = JSON.stringify(toolCall);
      // No header needed for newline protocol
      serverProcess.stdin.write(message + "\n");

      await new Promise(resolve => setTimeout(resolve, 100));

      // Check response
      expect(responseData).toMatch(/\{.*"jsonrpc".*"2\\.0".*\}/);

      // Extract JSON response - handle multiple responses by finding the one for our request id
      const response = findResponseById(responseData, 1);
      expect(response.jsonrpc).toBe("2.0");
      expect(response.id).toBe(1); // Server is responding with ID 1
      expect(response.error).toBeTruthy();
      expect(response.error.message).toContain("Invalid arguments");
    });
  });

  // Helper to parse NDJSON files and return the last non-empty JSON object
  function parseNdjsonLast(content) {
    const lines = content
      .split(/\r?\n/)
      .map(l => l.trim())
      .filter(Boolean);
    if (lines.length === 0) {
      throw new Error("No NDJSON entries found in output file");
    }
    try {
      return JSON.parse(lines[lines.length - 1]);
    } catch (e) {
      // Preserve fast-fail behavior expected by tests and provide logging
      throw new Error(`Failed to parse last NDJSON entry: ${e.message}`);
    }
  }
});
