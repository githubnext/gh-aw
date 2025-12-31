import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

describe("parse_safe_inputs_logs.cjs", () => {
  let mockCore, originalConsole;
  let main, parseSafeInputsLogLine, generateSafeInputsSummary, generatePlainTextSummary;

  beforeEach(async () => {
    originalConsole = global.console;
    global.console = { log: vi.fn(), error: vi.fn() };

    mockCore = {
      debug: vi.fn(),
      info: vi.fn(),
      notice: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
      setOutput: vi.fn(),
      exportVariable: vi.fn(),
      setSecret: vi.fn(),
      getInput: vi.fn(),
      getBooleanInput: vi.fn(),
      getMultilineInput: vi.fn(),
      getState: vi.fn(),
      saveState: vi.fn(),
      startGroup: vi.fn(),
      endGroup: vi.fn(),
      group: vi.fn(),
      addPath: vi.fn(),
      setCommandEcho: vi.fn(),
      isDebug: vi.fn().mockReturnValue(false),
      getIDToken: vi.fn(),
      toPlatformPath: vi.fn(),
      toPosixPath: vi.fn(),
      toWin32Path: vi.fn(),
      summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() },
    };

    global.core = mockCore;

    // Import the module to get the exported functions
    const module = await import("./parse_safe_inputs_logs.cjs?" + Date.now());
    main = module.main;
    parseSafeInputsLogLine = module.parseSafeInputsLogLine;
    generateSafeInputsSummary = module.generateSafeInputsSummary;
    generatePlainTextSummary = module.generatePlainTextSummary;
  });

  afterEach(() => {
    global.console = originalConsole;
    delete global.core;
  });

  describe("parseSafeInputsLogLine", () => {
    it("should parse valid safe-inputs log line with standard format", () => {
      const line = "[2025-12-31T15:43:54.123Z] [safe-inputs-server] Starting Safe Inputs MCP Server";
      const result = parseSafeInputsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBe("2025-12-31T15:43:54.123Z");
      expect(result.serverName).toBe("safe-inputs-server");
      expect(result.message).toBe("Starting Safe Inputs MCP Server");
      expect(result.raw).toBe(false);
    });

    it("should parse log line with tool registration message", () => {
      const line = "[2025-12-31T15:43:55.456Z] [safe-inputs] Registering tool: create_issue";
      const result = parseSafeInputsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBe("2025-12-31T15:43:55.456Z");
      expect(result.serverName).toBe("safe-inputs");
      expect(result.message).toBe("Registering tool: create_issue");
      expect(result.raw).toBe(false);
    });

    it("should parse log line with tool execution message", () => {
      const line = "[2025-12-31T15:44:10.789Z] [mcp-server] Calling handler for tool: create_pull_request";
      const result = parseSafeInputsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBe("2025-12-31T15:44:10.789Z");
      expect(result.serverName).toBe("mcp-server");
      expect(result.message).toBe("Calling handler for tool: create_pull_request");
      expect(result.raw).toBe(false);
    });

    it("should parse log line with error message", () => {
      const line = "[2025-12-31T15:44:20.000Z] [safe-inputs] Error: Failed to process request";
      const result = parseSafeInputsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBe("2025-12-31T15:44:20.000Z");
      expect(result.serverName).toBe("safe-inputs");
      expect(result.message).toBe("Error: Failed to process request");
      expect(result.raw).toBe(false);
    });

    it("should handle log line with extra whitespace", () => {
      const line = "[2025-12-31T15:43:54.123Z]   [safe-inputs-server]   Server started successfully  ";
      const result = parseSafeInputsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBe("2025-12-31T15:43:54.123Z");
      expect(result.serverName).toBe("safe-inputs-server");
      expect(result.message).toBe("Server started successfully");
      expect(result.raw).toBe(false);
    });

    it("should return raw entry for unparseable log line", () => {
      const line = "This is an unparseable log line without the expected format";
      const result = parseSafeInputsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBeNull();
      expect(result.serverName).toBeNull();
      expect(result.message).toBe("This is an unparseable log line without the expected format");
      expect(result.raw).toBe(true);
    });

    it("should return raw entry for empty line", () => {
      const line = "";
      const result = parseSafeInputsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBeNull();
      expect(result.serverName).toBeNull();
      expect(result.message).toBe("");
      expect(result.raw).toBe(true);
    });

    it("should parse log line with brackets in message", () => {
      const line = "[2025-12-31T15:43:54.123Z] [safe-inputs] Tool returned: {status: [200]}";
      const result = parseSafeInputsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBe("2025-12-31T15:43:54.123Z");
      expect(result.serverName).toBe("safe-inputs");
      expect(result.message).toBe("Tool returned: {status: [200]}");
      expect(result.raw).toBe(false);
    });
  });

  describe("generatePlainTextSummary", () => {
    it("should generate summary with startup events", () => {
      const logEntries = [
        { timestamp: "2025-12-31T15:43:54.123Z", serverName: "safe-inputs", message: "Starting Safe Inputs MCP Server", raw: false },
        { timestamp: "2025-12-31T15:43:54.456Z", serverName: "safe-inputs", message: "Server started successfully", raw: false },
      ];

      const summary = generatePlainTextSummary(logEntries);

      expect(summary).toContain("=== Safe Inputs MCP Server Logs ===");
      expect(summary).toContain("Total entries: 2");
      expect(summary).toContain("Startup events: 2");
    });

    it("should generate summary with tool registration events", () => {
      const logEntries = [
        { timestamp: "2025-12-31T15:43:55.000Z", serverName: "safe-inputs", message: "Registering tool: create_issue", raw: false },
        { timestamp: "2025-12-31T15:43:55.100Z", serverName: "safe-inputs", message: "Tool registration complete", raw: false },
      ];

      const summary = generatePlainTextSummary(logEntries);

      expect(summary).toContain("Total entries: 2");
      expect(summary).toContain("Tool registrations: 2");
    });

    it("should generate summary with tool execution events", () => {
      const logEntries = [
        { timestamp: "2025-12-31T15:44:10.000Z", serverName: "safe-inputs", message: "Calling handler for tool: create_issue", raw: false },
        { timestamp: "2025-12-31T15:44:10.500Z", serverName: "safe-inputs", message: "Handler returned successfully", raw: false },
      ];

      const summary = generatePlainTextSummary(logEntries);

      expect(summary).toContain("Total entries: 2");
      expect(summary).toContain("Tool executions: 2");
      expect(summary).toContain("Tool Executions:");
      expect(summary).toContain("create_issue");
    });

    it("should generate summary with error events", () => {
      const logEntries = [
        { timestamp: "2025-12-31T15:44:20.000Z", serverName: "safe-inputs", message: "Error: Failed to process request", raw: false },
        { timestamp: "2025-12-31T15:44:20.100Z", serverName: "safe-inputs", message: "Request failed with status 500", raw: false },
      ];

      const summary = generatePlainTextSummary(logEntries);

      expect(summary).toContain("Total entries: 2");
      expect(summary).toContain("Errors: 2");
      expect(summary).toContain("Errors:");
      expect(summary).toContain("Failed to process request");
      expect(summary).toContain("Request failed with status 500");
    });

    it("should generate summary with mixed event types", () => {
      const logEntries = [
        { timestamp: "2025-12-31T15:43:54.000Z", serverName: "safe-inputs", message: "Starting Safe Inputs MCP Server", raw: false },
        { timestamp: "2025-12-31T15:43:55.000Z", serverName: "safe-inputs", message: "Registering tool: create_issue", raw: false },
        { timestamp: "2025-12-31T15:44:10.000Z", serverName: "safe-inputs", message: "Calling handler for tool: create_issue", raw: false },
        { timestamp: "2025-12-31T15:44:10.500Z", serverName: "safe-inputs", message: "Handler returned successfully", raw: false },
        { timestamp: "2025-12-31T15:44:20.000Z", serverName: "safe-inputs", message: "Some other log message", raw: false },
      ];

      const summary = generatePlainTextSummary(logEntries);

      expect(summary).toContain("Total entries: 5");
      expect(summary).toContain("Startup events: 1");
      expect(summary).toContain("Tool registrations: 1");
      expect(summary).toContain("Tool executions: 2");
      expect(summary).toContain("Tool Executions:");
      expect(summary).toContain("create_issue");
    });

    it("should handle empty log entries", () => {
      const logEntries = [];
      const summary = generatePlainTextSummary(logEntries);

      expect(summary).toContain("=== Safe Inputs MCP Server Logs ===");
      expect(summary).toContain("Total entries: 0");
    });

    it("should include full logs in plain text summary", () => {
      const logEntries = [
        { timestamp: "2025-12-31T15:43:54.000Z", serverName: "safe-inputs", message: "Starting Safe Inputs MCP Server", raw: false },
        { timestamp: "2025-12-31T15:43:55.000Z", serverName: "safe-inputs", message: "Server started successfully", raw: false },
        { timestamp: null, serverName: null, message: "Unparsed log line", raw: true },
      ];

      const summary = generatePlainTextSummary(logEntries);

      expect(summary).toContain("Full Logs (first 5000 lines):");
      expect(summary).toContain("[safe-inputs] Starting Safe Inputs MCP Server");
      expect(summary).toContain("[safe-inputs] Server started successfully");
      expect(summary).toContain("Unparsed log line");
    });

    it("should limit full logs to 5000 lines", () => {
      // Create 5500 log entries
      const logEntries = [];
      for (let i = 0; i < 5500; i++) {
        logEntries.push({
          timestamp: "2025-12-31T15:43:54.000Z",
          serverName: "safe-inputs",
          message: `Log entry ${i}`,
          raw: false,
        });
      }

      const summary = generatePlainTextSummary(logEntries);

      expect(summary).toContain("Full Logs (first 5000 lines):");
      expect(summary).toContain("Log entry 0");
      expect(summary).toContain("Log entry 4999");
      expect(summary).toContain("... (truncated, showing first 5000 lines of 5500 total entries)");
      expect(summary).not.toContain("Log entry 5000");
      expect(summary).not.toContain("Log entry 5499");
    });
  });

  describe("generateSafeInputsSummary", () => {
    it("should generate markdown summary with details/summary structure", () => {
      const logEntries = [
        { timestamp: "2025-12-31T15:43:54.000Z", serverName: "safe-inputs", message: "Starting Safe Inputs MCP Server", raw: false },
        { timestamp: "2025-12-31T15:43:55.000Z", serverName: "safe-inputs", message: "Registering tool: create_issue", raw: false },
        { timestamp: "2025-12-31T15:44:10.000Z", serverName: "safe-inputs", message: "Calling handler for tool: create_issue", raw: false },
      ];

      const summary = generateSafeInputsSummary(logEntries);

      expect(summary).toContain("<details>");
      expect(summary).toContain("</details>");
      expect(summary).toContain("<summary>Safe Inputs MCP Server Logs</summary>");
      expect(summary).toContain("**Statistics**");
      expect(summary).toContain("| Metric | Count |");
      expect(summary).toContain("| Total Log Entries | 3 |");
      expect(summary).toContain("| Startup Events | 1 |");
      expect(summary).toContain("| Tool Registrations | 1 |");
      expect(summary).toContain("| Tool Executions | 1 |");
    });

    it("should generate markdown summary with tool execution details", () => {
      const logEntries = [
        { timestamp: "2025-12-31T15:44:10.000Z", serverName: "safe-inputs", message: "Calling handler for tool: create_issue", raw: false },
        { timestamp: "2025-12-31T15:44:15.000Z", serverName: "safe-inputs", message: "Calling handler for tool: create_pull_request", raw: false },
      ];

      const summary = generateSafeInputsSummary(logEntries);

      expect(summary).toContain("**Tool Executions**");
      expect(summary).toContain("<summary>View tool execution details</summary>");
      expect(summary).toContain("| Time | Tool Name |");
      expect(summary).toContain("`create_issue`");
      expect(summary).toContain("`create_pull_request`");
    });

    it("should generate markdown summary with error details", () => {
      const logEntries = [
        { timestamp: "2025-12-31T15:44:20.000Z", serverName: "safe-inputs", message: "Error: Failed to process request", raw: false },
        { timestamp: "2025-12-31T15:44:21.000Z", serverName: "safe-inputs", message: "Request failed with status 500", raw: false },
      ];

      const summary = generateSafeInputsSummary(logEntries);

      expect(summary).toContain("**Errors**");
      expect(summary).toContain("<summary>View error details</summary>");
      expect(summary).toContain("```");
      expect(summary).toContain("Failed to process request");
      expect(summary).toContain("Request failed with status 500");
    });

    it("should generate markdown summary with full logs section", () => {
      const logEntries = [
        { timestamp: "2025-12-31T15:43:54.000Z", serverName: "safe-inputs", message: "Starting Safe Inputs MCP Server", raw: false },
        { timestamp: null, serverName: null, message: "Unparsed log line", raw: true },
      ];

      const summary = generateSafeInputsSummary(logEntries);

      expect(summary).toContain("**Full Logs**");
      expect(summary).toContain("<summary>View full safe-inputs logs</summary>");
      expect(summary).toContain("```");
      expect(summary).toContain("[2025-12-31T15:43:54.000Z] [safe-inputs] Starting Safe Inputs MCP Server");
      expect(summary).toContain("Unparsed log line");
    });

    it("should handle empty log entries", () => {
      const logEntries = [];
      const summary = generateSafeInputsSummary(logEntries);

      expect(summary).toContain("<details>");
      expect(summary).toContain("</details>");
      expect(summary).toContain("| Total Log Entries | 0 |");
      expect(summary).toContain("| Startup Events | 0 |");
      expect(summary).toContain("| Tool Registrations | 0 |");
      expect(summary).toContain("| Tool Executions | 0 |");
      expect(summary).toContain("| Errors | 0 |");
    });

    it("should not show tool executions section when no tools called", () => {
      const logEntries = [{ timestamp: "2025-12-31T15:43:54.000Z", serverName: "safe-inputs", message: "Starting Safe Inputs MCP Server", raw: false }];

      const summary = generateSafeInputsSummary(logEntries);

      expect(summary).not.toContain("**Tool Executions**");
    });

    it("should not show errors section when no errors", () => {
      const logEntries = [{ timestamp: "2025-12-31T15:43:54.000Z", serverName: "safe-inputs", message: "Starting Safe Inputs MCP Server", raw: false }];

      const summary = generateSafeInputsSummary(logEntries);

      expect(summary).not.toContain("**Errors**");
    });
  });

  describe("main function", () => {
    let tempDir;

    beforeEach(() => {
      tempDir = "/tmp/gh-aw/safe-inputs/logs/";
    });

    it("should handle missing logs directory", async () => {
      // Mock fs.existsSync to return false
      vi.spyOn(fs, "existsSync").mockReturnValue(false);

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No safe-inputs logs directory found"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle empty logs directory", async () => {
      // Mock fs to return empty directory
      vi.spyOn(fs, "existsSync").mockReturnValue(true);
      vi.spyOn(fs, "readdirSync").mockReturnValue([]);

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No safe-inputs log files found"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should process log files and generate summaries", async () => {
      const logContent = "[2025-12-31T15:43:54.000Z] [safe-inputs] Starting Safe Inputs MCP Server\n[2025-12-31T15:43:55.000Z] [safe-inputs] Registering tool: create_issue\n";

      // Mock fs functions
      vi.spyOn(fs, "existsSync").mockReturnValue(true);
      vi.spyOn(fs, "readdirSync").mockReturnValue(["server.log"]);
      vi.spyOn(fs, "readFileSync").mockReturnValue(logContent);

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Found 1 safe-inputs log file"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Parsing safe-inputs log: server.log"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("=== Safe Inputs MCP Server Logs ==="));
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle errors gracefully", async () => {
      const error = new Error("Test error");

      // Mock fs to throw error
      vi.spyOn(fs, "existsSync").mockImplementation(() => {
        throw error;
      });

      await main();

      expect(mockCore.setFailed).toHaveBeenCalledWith("Test error");
    });

    it("should process multiple log files", async () => {
      const logContent = "[2025-12-31T15:43:54.000Z] [safe-inputs] Test message\n";

      // Mock fs functions
      vi.spyOn(fs, "existsSync").mockReturnValue(true);
      vi.spyOn(fs, "readdirSync").mockReturnValue(["server1.log", "server2.log"]);
      vi.spyOn(fs, "readFileSync").mockReturnValue(logContent);

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Found 2 safe-inputs log file"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Parsing safe-inputs log: server1.log"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Parsing safe-inputs log: server2.log"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle empty log entries", async () => {
      const logContent = "\n\n\n";

      // Mock fs functions
      vi.spyOn(fs, "existsSync").mockReturnValue(true);
      vi.spyOn(fs, "readdirSync").mockReturnValue(["empty.log"]);
      vi.spyOn(fs, "readFileSync").mockReturnValue(logContent);

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No parseable log entries found"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });
});
