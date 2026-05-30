import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
const { ERR_PARSE } = require("./error_codes.cjs");

describe("parse_mcp_scripts_logs.cjs", () => {
  let mockCore, originalConsole;
  let main, parseMCPScriptsLogLine, generateMCPScriptsSummary, generatePlainTextSummary;

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
    const module = await import("./parse_mcp_scripts_logs.cjs?" + Date.now());
    main = module.main;
    parseMCPScriptsLogLine = module.parseMCPScriptsLogLine;
    generateMCPScriptsSummary = module.generateMCPScriptsSummary;
    generatePlainTextSummary = module.generatePlainTextSummary;
  });

  afterEach(() => {
    global.console = originalConsole;
    delete global.core;
  });

  describe("parseMCPScriptsLogLine", () => {
    it("should parse valid mcp-scripts log line with standard format", () => {
      const line = "[2025-12-31T15:43:54.123Z] [mcp-scripts-server] Starting mcp-scripts MCP Server";
      const result = parseMCPScriptsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBe("2025-12-31T15:43:54.123Z");
      expect(result.serverName).toBe("mcp-scripts-server");
      expect(result.message).toBe("Starting mcp-scripts MCP Server");
      expect(result.raw).toBe(false);
    });

    it("should parse log line with tool registration message", () => {
      const line = "[2025-12-31T15:43:55.456Z] [mcp-scripts] Registering tool: create_issue";
      const result = parseMCPScriptsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBe("2025-12-31T15:43:55.456Z");
      expect(result.serverName).toBe("mcp-scripts");
      expect(result.message).toBe("Registering tool: create_issue");
      expect(result.raw).toBe(false);
    });

    it("should parse log line with tool execution message", () => {
      const line = "[2025-12-31T15:44:10.789Z] [mcp-server] Calling handler for tool: create_pull_request";
      const result = parseMCPScriptsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBe("2025-12-31T15:44:10.789Z");
      expect(result.serverName).toBe("mcp-server");
      expect(result.message).toBe("Calling handler for tool: create_pull_request");
      expect(result.raw).toBe(false);
    });

    it("should parse log line with error message", () => {
      const line = "[2025-12-31T15:44:20.000Z] [mcp-scripts] Error: Failed to process request";
      const result = parseMCPScriptsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBe("2025-12-31T15:44:20.000Z");
      expect(result.serverName).toBe("mcp-scripts");
      expect(result.message).toBe("Error: Failed to process request");
      expect(result.raw).toBe(false);
    });

    it("should handle log line with extra whitespace", () => {
      const line = "[2025-12-31T15:43:54.123Z]   [mcp-scripts-server]   Server started successfully  ";
      const result = parseMCPScriptsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBe("2025-12-31T15:43:54.123Z");
      expect(result.serverName).toBe("mcp-scripts-server");
      expect(result.message).toBe("Server started successfully");
      expect(result.raw).toBe(false);
    });

    it("should return raw entry for unparseable log line", () => {
      const line = "This is an unparseable log line without the expected format";
      const result = parseMCPScriptsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBeNull();
      expect(result.serverName).toBeNull();
      expect(result.message).toBe("This is an unparseable log line without the expected format");
      expect(result.raw).toBe(true);
    });

    it("should return raw entry for empty line", () => {
      const line = "";
      const result = parseMCPScriptsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBeNull();
      expect(result.serverName).toBeNull();
      expect(result.message).toBe("");
      expect(result.raw).toBe(true);
    });

    it("should parse log line with brackets in message", () => {
      const line = "[2025-12-31T15:43:54.123Z] [mcp-scripts] Tool returned: {status: [200]}";
      const result = parseMCPScriptsLogLine(line);

      expect(result).not.toBeNull();
      expect(result.timestamp).toBe("2025-12-31T15:43:54.123Z");
      expect(result.serverName).toBe("mcp-scripts");
      expect(result.message).toBe("Tool returned: {status: [200]}");
      expect(result.raw).toBe(false);
    });
  });

  describe("generatePlainTextSummary", () => {
    it("should render Copilot-style MCP tool lines with args and result preview", () => {
      const logEntries = [
        {
          timestamp: "2025-12-31T15:44:10.000Z",
          serverName: "mcpscripts",
          message: '[2025-12-31T15:44:10.000Z] [mcpscripts]   [gh] Invoking handler with args: {"args":"pr view 26467 --repo github/gh-aw --json number"}',
          raw: false,
        },
        {
          timestamp: "2025-12-31T15:44:10.100Z",
          serverName: "mcpscripts",
          message: '[2025-12-31T15:44:10.100Z] [mcpscripts]   [gh] Serialized result: {"stdout":"gh pr view 26467 --repo github/gh-aw --json number","stderr":""}',
          raw: false,
        },
      ];

      const summary = generatePlainTextSummary(logEntries);

      expect(summary).toContain('● gh (MCP: mcpscripts) · args: "pr view 26467 --repo github/gh-aw --json number"');
      expect(summary).toContain('└ {"stdout":"gh pr view 26467 --repo github/gh-aw --json number","stderr":""}');
    });

    it("should render backend tool call format in Copilot-style", () => {
      const logEntries = [
        {
          timestamp: "2025-12-31T15:44:10.000Z",
          serverName: "mcp-gateway",
          message: "server:unified callBackendTool: serverID=mcpscripts, toolName=gh, args=map[args:pr view 26450 --repo github/gh-aw --json number]",
          raw: false,
        },
      ];

      const summary = generatePlainTextSummary(logEntries);

      expect(summary).toContain('● gh (MCP: mcp-gateway) · args: "pr view 26450 --repo github/gh-aw --json number"');
    });

    it("should include multiple tool calls", () => {
      const logEntries = [
        {
          timestamp: "2025-12-31T15:44:10.000Z",
          serverName: "mcpscripts",
          message: '[2025-12-31T15:44:10.000Z] [mcpscripts]   [gh] Invoking handler with args: {"args":"pr view 26444 --repo github/gh-aw --json number"}',
          raw: false,
        },
        {
          timestamp: "2025-12-31T15:44:11.000Z",
          serverName: "mcpscripts",
          message: '[2025-12-31T15:44:11.000Z] [mcpscripts]   [gh] Invoking handler with args: {"args":"pr view 26380 --repo github/gh-aw --json number"}',
          raw: false,
        },
      ];

      const summary = generatePlainTextSummary(logEntries);

      expect(summary).toContain('● gh (MCP: mcpscripts) · args: "pr view 26444 --repo github/gh-aw --json number"');
      expect(summary).toContain('● gh (MCP: mcpscripts) · args: "pr view 26380 --repo github/gh-aw --json number"');
    });

    it("should include diagnostics for error lines", () => {
      const logEntries = [
        {
          timestamp: "2025-12-31T15:44:10.000Z",
          serverName: "mcpscripts",
          message: '[2025-12-31T15:44:10.000Z] [mcpscripts]   [gh] Invoking handler with args: {"args":"pr view 25928 --repo github/gh-aw --json number"}',
          raw: false,
        },
        { timestamp: "2025-12-31T15:44:20.000Z", serverName: "mcp-scripts", message: "Error: Failed to process request", raw: false },
      ];

      const summary = generatePlainTextSummary(logEntries);

      expect(summary).toContain("Additional MCP diagnostics:");
      expect(summary).toContain("Failed to process request");
    });

    it("should fall back to raw logs when no tool calls are present", () => {
      const logEntries = [
        { timestamp: "2025-12-31T15:43:54.000Z", serverName: "mcp-scripts", message: "Starting mcp-scripts MCP Server", raw: false },
        { timestamp: "2025-12-31T15:43:55.000Z", serverName: "mcp-scripts", message: "Server started successfully", raw: false },
      ];

      const summary = generatePlainTextSummary(logEntries);

      expect(summary).toContain("[mcp-scripts] Starting mcp-scripts MCP Server");
      expect(summary).toContain("[mcp-scripts] Server started successfully");
    });

    it("should handle empty log entries", () => {
      const logEntries = [];
      const summary = generatePlainTextSummary(logEntries);

      expect(summary).toBe("");
    });

    it("should limit fallback raw logs to 200 lines", () => {
      const logEntries = [];
      for (let i = 0; i < 250; i++) {
        logEntries.push({
          timestamp: "2025-12-31T15:43:54.000Z",
          serverName: "mcp-scripts",
          message: `Log entry ${i}`,
          raw: false,
        });
      }

      const summary = generatePlainTextSummary(logEntries);

      expect(summary).toContain("Log entry 0");
      expect(summary).toContain("Log entry 199");
      expect(summary).toContain("... (truncated, showing first 200 lines of 250 total entries)");
      expect(summary).not.toContain("Log entry 200");
      expect(summary).not.toContain("Log entry 249");
    });
  });

  describe("generateMCPScriptsSummary", () => {
    it("should generate markdown summary with details/summary structure", () => {
      const logEntries = [
        { timestamp: "2025-12-31T15:43:54.000Z", serverName: "mcp-scripts", message: "Starting mcp-scripts MCP Server", raw: false },
        { timestamp: "2025-12-31T15:43:55.000Z", serverName: "mcp-scripts", message: "Registering tool: create_issue", raw: false },
        { timestamp: "2025-12-31T15:44:10.000Z", serverName: "mcp-scripts", message: "Calling handler for tool: create_issue", raw: false },
      ];

      const summary = generateMCPScriptsSummary(logEntries);

      expect(summary).toContain("<details>");
      expect(summary).toContain("</details>");
      expect(summary).toContain("<summary>MCP Scripts Server Logs</summary>");
      expect(summary).toContain("**Statistics**");
      expect(summary).toContain("| Metric | Count |");
      expect(summary).toContain("| Total Log Entries | 3 |");
      expect(summary).toContain("| Startup Events | 1 |");
      expect(summary).toContain("| Tool Registrations | 1 |");
      expect(summary).toContain("| Tool Executions | 1 |");
    });

    it("should generate markdown summary with tool execution details", () => {
      const logEntries = [
        { timestamp: "2025-12-31T15:44:10.000Z", serverName: "mcp-scripts", message: "Calling handler for tool: create_issue", raw: false },
        { timestamp: "2025-12-31T15:44:15.000Z", serverName: "mcp-scripts", message: "Calling handler for tool: create_pull_request", raw: false },
      ];

      const summary = generateMCPScriptsSummary(logEntries);

      expect(summary).toContain("**Tool Executions**");
      expect(summary).toContain("<summary>View tool execution details</summary>");
      expect(summary).toContain("| Time | Tool Name |");
      expect(summary).toContain("`create_issue`");
      expect(summary).toContain("`create_pull_request`");
    });

    it("should generate markdown summary with error details", () => {
      const logEntries = [
        { timestamp: "2025-12-31T15:44:20.000Z", serverName: "mcp-scripts", message: "Error: Failed to process request", raw: false },
        { timestamp: "2025-12-31T15:44:21.000Z", serverName: "mcp-scripts", message: "Request failed with status 500", raw: false },
      ];

      const summary = generateMCPScriptsSummary(logEntries);

      expect(summary).toContain("**Errors**");
      expect(summary).toContain("<summary>View error details</summary>");
      expect(summary).toContain("```");
      expect(summary).toContain("Failed to process request");
      expect(summary).toContain("Request failed with status 500");
    });

    it("should generate markdown summary with full logs section", () => {
      const logEntries = [
        { timestamp: "2025-12-31T15:43:54.000Z", serverName: "mcp-scripts", message: "Starting mcp-scripts MCP Server", raw: false },
        { timestamp: null, serverName: null, message: "Unparsed log line", raw: true },
      ];

      const summary = generateMCPScriptsSummary(logEntries);

      expect(summary).toContain("**Full Logs**");
      expect(summary).toContain("<summary>View full mcp-scripts logs</summary>");
      expect(summary).toContain("```");
      expect(summary).toContain("[2025-12-31T15:43:54.000Z] [mcp-scripts] Starting mcp-scripts MCP Server");
      expect(summary).toContain("Unparsed log line");
    });

    it("should handle empty log entries", () => {
      const logEntries = [];
      const summary = generateMCPScriptsSummary(logEntries);

      expect(summary).toContain("<details>");
      expect(summary).toContain("</details>");
      expect(summary).toContain("| Total Log Entries | 0 |");
      expect(summary).toContain("| Startup Events | 0 |");
      expect(summary).toContain("| Tool Registrations | 0 |");
      expect(summary).toContain("| Tool Executions | 0 |");
      expect(summary).toContain("| Errors | 0 |");
    });

    it("should not show tool executions section when no tools called", () => {
      const logEntries = [{ timestamp: "2025-12-31T15:43:54.000Z", serverName: "mcp-scripts", message: "Starting mcp-scripts MCP Server", raw: false }];

      const summary = generateMCPScriptsSummary(logEntries);

      expect(summary).not.toContain("**Tool Executions**");
    });

    it("should not show errors section when no errors", () => {
      const logEntries = [{ timestamp: "2025-12-31T15:43:54.000Z", serverName: "mcp-scripts", message: "Starting mcp-scripts MCP Server", raw: false }];

      const summary = generateMCPScriptsSummary(logEntries);

      expect(summary).not.toContain("**Errors**");
    });
  });

  describe("main function", () => {
    let tempDir;

    beforeEach(() => {
      tempDir = "/tmp/gh-aw/mcp-scripts/logs/";
    });

    it("should handle missing logs directory", async () => {
      // Mock fs.existsSync to return false
      vi.spyOn(fs, "existsSync").mockReturnValue(false);

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No mcp-scripts logs directory found"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle empty logs directory", async () => {
      // Mock fs to return empty directory
      vi.spyOn(fs, "existsSync").mockReturnValue(true);
      vi.spyOn(fs, "readdirSync").mockReturnValue([]);

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No mcp-scripts log files found"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should process log files and generate summaries", async () => {
      const logContent = "[2025-12-31T15:43:54.000Z] [mcp-scripts] Starting mcp-scripts MCP Server\n[2025-12-31T15:43:55.000Z] [mcp-scripts] Registering tool: create_issue\n";

      // Mock fs functions
      vi.spyOn(fs, "existsSync").mockReturnValue(true);
      vi.spyOn(fs, "readdirSync").mockReturnValue(["server.log"]);
      vi.spyOn(fs, "readFileSync").mockReturnValue(logContent);

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Found 1 mcp-scripts log file"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Parsing mcp-scripts log: server.log"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Starting mcp-scripts MCP Server"));
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

      expect(mockCore.setFailed).toHaveBeenCalledWith(`${ERR_PARSE}: Test error`);
    });

    it("should process multiple log files", async () => {
      const logContent = "[2025-12-31T15:43:54.000Z] [mcp-scripts] Test message\n";

      // Mock fs functions
      vi.spyOn(fs, "existsSync").mockReturnValue(true);
      vi.spyOn(fs, "readdirSync").mockReturnValue(["server1.log", "server2.log"]);
      vi.spyOn(fs, "readFileSync").mockReturnValue(logContent);

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Found 2 mcp-scripts log file"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Parsing mcp-scripts log: server1.log"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Parsing mcp-scripts log: server2.log"));
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
