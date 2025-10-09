import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

describe("parse_claude_log.cjs", () => {
  let mockCore;
  let parseClaudeLogScript;
  let originalConsole;
  let originalProcess;

  beforeEach(() => {
    // Save originals before mocking
    originalConsole = global.console;
    originalProcess = { ...process };

    // Mock console methods
    global.console = {
      log: vi.fn(),
      error: vi.fn(),
    };

    // Mock core actions methods
    mockCore = {
      // Core logging functions
      debug: vi.fn(),
      info: vi.fn(),
      notice: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),

      // Core workflow functions
      setFailed: vi.fn(),
      setOutput: vi.fn(),
      exportVariable: vi.fn(),
      setSecret: vi.fn(),

      // Input/state functions
      getInput: vi.fn(),
      getBooleanInput: vi.fn(),
      getMultilineInput: vi.fn(),
      getState: vi.fn(),
      saveState: vi.fn(),

      // Group functions
      startGroup: vi.fn(),
      endGroup: vi.fn(),
      group: vi.fn(),

      // Other utility functions
      addPath: vi.fn(),
      setCommandEcho: vi.fn(),
      isDebug: vi.fn().mockReturnValue(false),
      getIDToken: vi.fn(),
      toPlatformPath: vi.fn(),
      toPosixPath: vi.fn(),
      toWin32Path: vi.fn(),

      // Summary object with chainable methods
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(),
      },
    };
    global.core = mockCore;

    // Mock require
    global.require = vi.fn().mockImplementation(module => {
      if (module === "fs") {
        return fs;
      }
      if (module === "@actions/core") {
        return mockCore;
      }
      throw new Error(`Module not found: ${module}`);
    });

    // Read the script file
    const scriptPath = path.join(__dirname, "parse_claude_log.cjs");
    parseClaudeLogScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Clean up environment variables
    delete process.env.GITHUB_AW_AGENT_OUTPUT;

    // Restore originals
    global.console = originalConsole;
    process.env = originalProcess.env;

    // Clean up globals
    delete global.core;
    delete global.require;
  });

  const runScript = async logContent => {
    // Create a temporary log file
    const tempFile = path.join(process.cwd(), `test_log_${Date.now()}.txt`);
    fs.writeFileSync(tempFile, logContent);
    process.env.GITHUB_AW_AGENT_OUTPUT = tempFile;

    try {
      // Create a new function context to execute the script
      const scriptWithExports = parseClaudeLogScript.replace(
        "main();",
        "global.testParseClaudeLog = parseClaudeLog; global.testMain = main; main();"
      );
      const scriptFunction = new Function(scriptWithExports);
      await scriptFunction();
    } finally {
      // Clean up temp file
      if (fs.existsSync(tempFile)) {
        fs.unlinkSync(tempFile);
      }
    }
  };

  const extractParseFunction = () => {
    // Extract just the parseClaudeLog function for unit testing
    const scriptWithExport = parseClaudeLogScript.replace("main();", "global.testParseClaudeLog = parseClaudeLog;");
    const scriptFunction = new Function(scriptWithExport);
    scriptFunction();
    return global.testParseClaudeLog;
  };

  describe("parseClaudeLog function", () => {
    let parseClaudeLog;

    beforeEach(() => {
      parseClaudeLog = extractParseFunction();
    });

    it("should parse old JSON array format", () => {
      const jsonArrayLog = JSON.stringify([
        {
          type: "system",
          subtype: "init",
          session_id: "test-123",
          tools: ["Bash", "Read"],
          model: "claude-sonnet-4-20250514",
        },
        {
          type: "assistant",
          message: {
            content: [
              { type: "text", text: "I'll help you with this task." },
              {
                type: "tool_use",
                id: "tool_123",
                name: "Bash",
                input: { command: "echo 'Hello World'" },
              },
            ],
          },
        },
        {
          type: "result",
          total_cost_usd: 0.0015,
          usage: { input_tokens: 150, output_tokens: 50 },
          num_turns: 1,
        },
      ]);

      const result = parseClaudeLog(jsonArrayLog);

      expect(result.markdown).toContain("🚀 Initialization");
      expect(result.markdown).toContain("🤖 Commands and Tools");
      expect(result.markdown).toContain("test-123");
      expect(result.markdown).toContain("echo 'Hello World'");
      expect(result.markdown).toContain("Total Cost");
      expect(result.mcpFailures).toEqual([]);
    });

    it("should parse new mixed format with debug logs and JSON array", () => {
      const mixedFormatLog = `[DEBUG] Starting Claude Code CLI
[ERROR] Some error occurred
npm warn exec The following package was not found
[{"type":"system","subtype":"init","session_id":"29d324d8-1a92-43c6-8740-babc2875a1d6","tools":["Task","Bash","mcp__safe_outputs__missing-tool"],"model":"claude-sonnet-4-20250514"},{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tool_123","name":"mcp__safe_outputs__missing-tool","input":{"tool":"draw_pelican","reason":"Tool needed to draw pelican artwork"}}]}},{"type":"result","total_cost_usd":0.1789264,"usage":{"input_tokens":25,"output_tokens":832},"num_turns":10}]
[DEBUG] Session completed`;

      const result = parseClaudeLog(mixedFormatLog);

      expect(result.markdown).toContain("🚀 Initialization");
      expect(result.markdown).toContain("🤖 Commands and Tools");
      expect(result.markdown).toContain("29d324d8-1a92-43c6-8740-babc2875a1d6");
      expect(result.markdown).toContain("safe_outputs::missing-tool");
      expect(result.markdown).toContain("Total Cost");
      expect(result.mcpFailures).toEqual([]);
    });

    it("should parse mixed format with individual JSON lines", () => {
      const jsonlFormatLog = `[DEBUG] Starting Claude Code CLI
{"type":"system","subtype":"init","session_id":"test-456","tools":["Bash","Read"],"model":"claude-sonnet-4-20250514"}
[DEBUG] Processing user prompt
{"type":"assistant","message":{"content":[{"type":"text","text":"I'll help you."},{"type":"tool_use","id":"tool_123","name":"Bash","input":{"command":"ls -la"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool_123","content":"file1.txt\\nfile2.txt"}]}}
{"type":"result","total_cost_usd":0.002,"usage":{"input_tokens":100,"output_tokens":25},"num_turns":2}
[DEBUG] Workflow completed`;

      const result = parseClaudeLog(jsonlFormatLog);

      expect(result.markdown).toContain("🚀 Initialization");
      expect(result.markdown).toContain("🤖 Commands and Tools");
      expect(result.markdown).toContain("test-456");
      expect(result.markdown).toContain("ls -la");
      expect(result.markdown).toContain("Total Cost");
      expect(result.mcpFailures).toEqual([]);
    });

    it("should handle MCP server failures", () => {
      const logWithFailures = JSON.stringify([
        {
          type: "system",
          subtype: "init",
          session_id: "test-789",
          tools: ["Bash"],
          mcp_servers: [
            { name: "github", status: "connected" },
            { name: "failed_server", status: "failed" },
          ],
          model: "claude-sonnet-4-20250514",
        },
      ]);

      const result = parseClaudeLog(logWithFailures);

      expect(result.markdown).toContain("🚀 Initialization");
      expect(result.markdown).toContain("failed_server (failed)");
      expect(result.mcpFailures).toEqual(["failed_server"]);
    });

    it("should handle unrecognized log format", () => {
      const invalidLog = "This is not JSON or valid format";

      const result = parseClaudeLog(invalidLog);

      expect(result.markdown).toContain("Log format not recognized");
      expect(result.mcpFailures).toEqual([]);
    });

    it("should handle empty log content", () => {
      const result = parseClaudeLog("");

      expect(result.markdown).toContain("Log format not recognized");
      expect(result.mcpFailures).toEqual([]);
    });

    it("should skip debug lines that look like arrays but aren't JSON", () => {
      const logWithFakeArrays = `[DEBUG] Starting process
[ERROR] Failed with error
[INFO] Some information
[{"type":"system","subtype":"init","session_id":"test-999","tools":["Bash"],"model":"claude-sonnet-4-20250514"}]
[DEBUG] Process completed`;

      const result = parseClaudeLog(logWithFakeArrays);

      expect(result.markdown).toContain("🚀 Initialization");
      expect(result.markdown).toContain("test-999");
      expect(result.mcpFailures).toEqual([]);
    });

    it("should handle tool use with MCP tools", () => {
      const logWithMcpTools = JSON.stringify([
        {
          type: "system",
          subtype: "init",
          session_id: "mcp-test",
          tools: ["mcp__github__create_issue", "mcp__safe_outputs__missing-tool"],
          model: "claude-sonnet-4-20250514",
        },
        {
          type: "assistant",
          message: {
            content: [
              {
                type: "tool_use",
                id: "tool_1",
                name: "mcp__github__create_issue",
                input: { title: "Test Issue", body: "Test description" },
              },
              {
                type: "tool_use",
                id: "tool_2",
                name: "mcp__safe_outputs__missing-tool",
                input: { tool: "missing_tool", reason: "Not available" },
              },
            ],
          },
        },
      ]);

      const result = parseClaudeLog(logWithMcpTools);

      expect(result.markdown).toContain("github::create_issue");
      expect(result.markdown).toContain("safe_outputs::missing-tool");
      expect(result.mcpFailures).toEqual([]);
    });
  });

  describe("main function integration", () => {
    it("should handle valid log file", async () => {
      const validLog = JSON.stringify([
        {
          type: "system",
          subtype: "init",
          session_id: "integration-test",
          tools: ["Bash"],
          model: "claude-sonnet-4-20250514",
        },
        {
          type: "result",
          total_cost_usd: 0.001,
          usage: { input_tokens: 50, output_tokens: 25 },
          num_turns: 1,
        },
      ]);

      await runScript(validLog);

      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
      expect(mockCore.setFailed).not.toHaveBeenCalled();

      // Check that markdown was added to summary
      const markdownCall = mockCore.summary.addRaw.mock.calls[0];
      expect(markdownCall[0]).toContain("🚀 Initialization");
      expect(markdownCall[0]).toContain("integration-test");

      // Verify that core.info was also called with the same content (via write helper)
      expect(mockCore.info).toHaveBeenCalled();
      const infoCall = mockCore.info.mock.calls.find(call => call[0].includes("🚀 Initialization"));
      expect(infoCall).toBeDefined();
      expect(infoCall[0]).toContain("integration-test");
    });

    it("should handle log with MCP failures", async () => {
      const logWithFailures = JSON.stringify([
        {
          type: "system",
          subtype: "init",
          session_id: "failure-test",
          mcp_servers: [
            { name: "working_server", status: "connected" },
            { name: "broken_server", status: "failed" },
          ],
          tools: ["Bash"],
          model: "claude-sonnet-4-20250514",
        },
      ]);

      await runScript(logWithFailures);

      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
      expect(mockCore.setFailed).toHaveBeenCalledWith("MCP server(s) failed to launch: broken_server");
    });

    it("should handle missing log file", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = "/nonexistent/file.log";

      // Extract main function and run it directly
      const scriptWithExport = parseClaudeLogScript.replace("main();", "global.testMain = main;");
      const scriptFunction = new Function(scriptWithExport);
      scriptFunction();
      await global.testMain();

      expect(mockCore.info).toHaveBeenCalledWith("Log file not found: /nonexistent/file.log");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle missing environment variable", async () => {
      delete process.env.GITHUB_AW_AGENT_OUTPUT;

      // Extract main function and run it directly
      const scriptWithExport = parseClaudeLogScript.replace("main();", "global.testMain = main;");
      const scriptFunction = new Function(scriptWithExport);
      scriptFunction();
      await global.testMain();

      expect(mockCore.info).toHaveBeenCalledWith("No agent log file specified");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });

  describe("helper function tests", () => {
    it("should format bash commands correctly", () => {
      const parseClaudeLog = extractParseFunction();

      // Test with the parseClaudeLog function to access formatBashCommand indirectly
      const logWithCommand = JSON.stringify([
        {
          type: "assistant",
          message: {
            content: [
              {
                type: "tool_use",
                id: "tool_1",
                name: "Bash",
                input: { command: "echo 'hello world'\n  && ls -la\n  && pwd" },
              },
            ],
          },
        },
      ]);

      const result = parseClaudeLog(logWithCommand);

      // Check that multi-line commands are normalized to single line
      expect(result.markdown).toContain("echo 'hello world' && ls -la && pwd");
    });

    it("should truncate long strings appropriately", () => {
      const parseClaudeLog = extractParseFunction();

      const longCommand = "a".repeat(100);
      const logWithLongCommand = JSON.stringify([
        {
          type: "assistant",
          message: {
            content: [
              {
                type: "tool_use",
                id: "tool_1",
                name: "Bash",
                input: { command: longCommand },
              },
            ],
          },
        },
      ]);

      const result = parseClaudeLog(logWithLongCommand);

      // Should truncate and add ellipsis
      expect(result.markdown).toContain("...");
    });

    it("should format MCP tool names correctly", () => {
      const parseClaudeLog = extractParseFunction();

      const logWithMcpTool = JSON.stringify([
        {
          type: "assistant",
          message: {
            content: [
              {
                type: "tool_use",
                id: "tool_1",
                name: "mcp__github__create_pull_request",
                input: { title: "Test PR" },
              },
            ],
          },
        },
      ]);

      const result = parseClaudeLog(logWithMcpTool);

      expect(result.markdown).toContain("github::create_pull_request");
    });

    it("should render tool outputs in collapsible HTML details elements", () => {
      const parseClaudeLog = extractParseFunction();

      const logWithToolOutput = JSON.stringify([
        {
          type: "assistant",
          message: {
            content: [
              {
                type: "tool_use",
                id: "tool_1",
                name: "Bash",
                input: { command: "ls -la", description: "List files" },
              },
            ],
          },
        },
        {
          type: "user",
          message: {
            content: [
              {
                type: "tool_result",
                tool_use_id: "tool_1",
                content:
                  "total 48\ndrwxr-xr-x 5 user user 4096 Jan 1 00:00 .\ndrwxr-xr-x 3 user user 4096 Jan 1 00:00 ..\n-rw-r--r-- 1 user user  123 Jan 1 00:00 file1.txt\n-rw-r--r-- 1 user user  456 Jan 1 00:00 file2.txt",
                is_error: false,
              },
            ],
          },
        },
      ]);

      const result = parseClaudeLog(logWithToolOutput);

      // Should contain HTML details tag
      expect(result.markdown).toContain("<details>");
      expect(result.markdown).toContain("<summary>");
      expect(result.markdown).toContain("</summary>");
      expect(result.markdown).toContain("</details>");

      // Summary should contain the tool description and command
      expect(result.markdown).toContain("List files: <code>ls -la</code>");

      // Details should contain the output in a code block
      expect(result.markdown).toContain("```");
      expect(result.markdown).toContain("total 48");
      expect(result.markdown).toContain("file1.txt");
    });

    it("should truncate long tool outputs", () => {
      const parseClaudeLog = extractParseFunction();

      const longOutput = "x".repeat(600);
      const logWithLongOutput = JSON.stringify([
        {
          type: "assistant",
          message: {
            content: [
              {
                type: "tool_use",
                id: "tool_1",
                name: "Bash",
                input: { command: "cat large_file.txt" },
              },
            ],
          },
        },
        {
          type: "user",
          message: {
            content: [
              {
                type: "tool_result",
                tool_use_id: "tool_1",
                content: longOutput,
                is_error: false,
              },
            ],
          },
        },
      ]);

      const result = parseClaudeLog(logWithLongOutput);

      // Should truncate with ellipsis
      expect(result.markdown).toContain("...");
      // Should not contain the full output
      expect(result.markdown).not.toContain("x".repeat(600));
    });

    it("should show summary only when no tool output", () => {
      const parseClaudeLog = extractParseFunction();

      const logWithoutOutput = JSON.stringify([
        {
          type: "assistant",
          message: {
            content: [
              {
                type: "tool_use",
                id: "tool_1",
                name: "Bash",
                input: { command: "mkdir test_dir" },
              },
            ],
          },
        },
        {
          type: "user",
          message: {
            content: [
              {
                type: "tool_result",
                tool_use_id: "tool_1",
                content: "",
                is_error: false,
              },
            ],
          },
        },
      ]);

      const result = parseClaudeLog(logWithoutOutput);

      // Should not contain details tag when there's no output
      expect(result.markdown).not.toContain("<details>");
      // Should still contain the summary line
      expect(result.markdown).toContain("mkdir test_dir");
    });
  });
});
