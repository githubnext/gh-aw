import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

describe("parse_copilot_log.cjs", () => {
  let mockCore;
  let parseCopilotLogScript;
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
      if (module === "path") {
        return path;
      }
      if (module === "@actions/core") {
        return mockCore;
      }
      throw new Error(`Module not found: ${module}`);
    });

    // Read the script file
    const scriptPath = path.join(__dirname, "parse_copilot_log.cjs");
    parseCopilotLogScript = fs.readFileSync(scriptPath, "utf8");
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
      const scriptWithExports = parseCopilotLogScript.replace(
        "main();",
        "global.testParseCopilotLog = parseCopilotLog; global.testMain = main; main();"
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
    // Extract just the parseCopilotLog function for unit testing
    const scriptWithExport = parseCopilotLogScript.replace("main();", "global.testParseCopilotLog = parseCopilotLog;");
    const scriptFunction = new Function(scriptWithExport);
    scriptFunction();
    return global.testParseCopilotLog;
  };

  describe("parseCopilotLog function", () => {
    let parseCopilotLog;

    beforeEach(() => {
      parseCopilotLog = extractParseFunction();
    });

    it("should parse JSON array format", () => {
      const jsonArrayLog = JSON.stringify([
        {
          type: "system",
          subtype: "init",
          session_id: "copilot-test-123",
          tools: ["Bash", "Read", "mcp__github__create_issue"],
          model: "gpt-5",
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
                input: { command: "echo 'Hello World'", description: "Print greeting" },
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
                tool_use_id: "tool_123",
                content: "Hello World\n",
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

      const result = parseCopilotLog(jsonArrayLog);

      expect(result).toContain("🚀 Initialization");
      expect(result).toContain("🤖 Commands and Tools");
      expect(result).toContain("copilot-test-123");
      expect(result).toContain("echo 'Hello World'");
      expect(result).toContain("Total Cost");
      expect(result).toContain("<details>");
      expect(result).toContain("<summary>");
    });

    it("should parse mixed format with debug logs and JSON array", () => {
      const mixedFormatLog = `[DEBUG] Starting Copilot CLI
[ERROR] Some error occurred
[{"type":"system","subtype":"init","session_id":"copilot-456","tools":["Bash","mcp__safe_outputs__missing-tool"],"model":"gpt-5"},{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tool_123","name":"mcp__safe_outputs__missing-tool","input":{"tool":"draw_pelican","reason":"Tool needed to draw pelican artwork"}}]}},{"type":"result","total_cost_usd":0.1789264,"usage":{"input_tokens":25,"output_tokens":832},"num_turns":10}]
[DEBUG] Session completed`;

      const result = parseCopilotLog(mixedFormatLog);

      expect(result).toContain("🚀 Initialization");
      expect(result).toContain("🤖 Commands and Tools");
      expect(result).toContain("copilot-456");
      expect(result).toContain("safe_outputs::missing-tool");
      expect(result).toContain("Total Cost");
    });

    it("should parse mixed format with individual JSON lines (JSONL)", () => {
      const jsonlFormatLog = `[DEBUG] Starting Copilot CLI
{"type":"system","subtype":"init","session_id":"copilot-789","tools":["Bash","Read"],"model":"gpt-5"}
[DEBUG] Processing user prompt
{"type":"assistant","message":{"content":[{"type":"text","text":"I'll help you."},{"type":"tool_use","id":"tool_123","name":"Bash","input":{"command":"ls -la"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool_123","content":"file1.txt\\nfile2.txt"}]}}
{"type":"result","total_cost_usd":0.002,"usage":{"input_tokens":100,"output_tokens":25},"num_turns":2}
[DEBUG] Workflow completed`;

      const result = parseCopilotLog(jsonlFormatLog);

      expect(result).toContain("🚀 Initialization");
      expect(result).toContain("🤖 Commands and Tools");
      expect(result).toContain("copilot-789");
      expect(result).toContain("ls -la");
      expect(result).toContain("Total Cost");
    });

    it("should handle tool calls with details in HTML format", () => {
      const logWithToolOutput = JSON.stringify([
        {
          type: "system",
          subtype: "init",
          session_id: "test-details",
          tools: ["Bash"],
          model: "gpt-5",
        },
        {
          type: "assistant",
          message: {
            content: [
              {
                type: "tool_use",
                id: "tool_1",
                name: "Bash",
                input: { command: "cat README.md", description: "Read README" },
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
                content: "# Project Title\n\nProject description here.",
              },
            ],
          },
        },
      ]);

      const result = parseCopilotLog(logWithToolOutput);

      // Should contain HTML details tag
      expect(result).toContain("<details>");
      expect(result).toContain("<summary>");
      expect(result).toContain("</summary>");
      expect(result).toContain("</details>");

      // Summary should contain the command
      expect(result).toContain("cat README.md");

      // Details should contain the output
      expect(result).toContain("Project Title");
    });

    it("should handle MCP tools", () => {
      const logWithMcpTools = JSON.stringify([
        {
          type: "system",
          subtype: "init",
          session_id: "mcp-test",
          tools: ["mcp__github__create_issue", "mcp__safe_outputs__missing-tool"],
          model: "gpt-5",
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

      const result = parseCopilotLog(logWithMcpTools);

      expect(result).toContain("github::create_issue");
      expect(result).toContain("safe_outputs::missing-tool");
    });

    it("should handle unrecognized log format", () => {
      const invalidLog = "This is not JSON or valid format";

      const result = parseCopilotLog(invalidLog);

      expect(result).toContain("Log format not recognized");
    });

    it("should handle empty log content", () => {
      const result = parseCopilotLog("");

      expect(result).toContain("Log format not recognized");
    });

    it("should skip internal file operations in summary", () => {
      const logWithInternalTools = JSON.stringify([
        {
          type: "assistant",
          message: {
            content: [
              {
                type: "tool_use",
                id: "tool_1",
                name: "Read",
                input: { file_path: "/tmp/gh-aw/test.txt" },
              },
              {
                type: "tool_use",
                id: "tool_2",
                name: "Write",
                input: { file_path: "/tmp/gh-aw/output.txt" },
              },
              {
                type: "tool_use",
                id: "tool_3",
                name: "Bash",
                input: { command: "echo test" },
              },
            ],
          },
        },
      ]);

      const result = parseCopilotLog(logWithInternalTools);

      // Commands and Tools section should only show Bash
      expect(result).toContain("🤖 Commands and Tools");
      const commandsSection = result.split("📊 Information")[0];
      expect(commandsSection).toContain("echo test");
      // Read and Write should not be in the summary
      expect(commandsSection.split("🤖 Reasoning")[0]).not.toContain("Read");
      expect(commandsSection.split("🤖 Reasoning")[0]).not.toContain("Write");
    });

    it("should render user text messages as markdown", () => {
      const logWithTextMessage = JSON.stringify([
        {
          type: "assistant",
          message: {
            content: [
              {
                type: "text",
                text: "Let me analyze the code and provide feedback.\n\n## Analysis\n\nThe code looks good but could use some improvements.",
              },
            ],
          },
        },
      ]);

      const result = parseCopilotLog(logWithTextMessage);

      // Text should be rendered directly in the Reasoning section
      expect(result).toContain("🤖 Reasoning");
      expect(result).toContain("Let me analyze the code");
      expect(result).toContain("## Analysis");
      expect(result).toContain("could use some improvements");
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
          model: "gpt-5",
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

      // Verify that core.info was also called with the same content
      expect(mockCore.info).toHaveBeenCalled();
      const infoCall = mockCore.info.mock.calls.find(call => call[0].includes("🚀 Initialization"));
      expect(infoCall).toBeDefined();
      expect(infoCall[0]).toContain("integration-test");
    });

    it("should handle missing log file", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = "/nonexistent/file.log";

      // Extract main function and run it directly
      const scriptWithExport = parseCopilotLogScript.replace("main();", "global.testMain = main;");
      const scriptFunction = new Function(scriptWithExport);
      scriptFunction();
      await global.testMain();

      expect(mockCore.info).toHaveBeenCalledWith("Log file not found: /nonexistent/file.log");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle missing environment variable", async () => {
      delete process.env.GITHUB_AW_AGENT_OUTPUT;

      // Extract main function and run it directly
      const scriptWithExport = parseCopilotLogScript.replace("main();", "global.testMain = main;");
      const scriptFunction = new Function(scriptWithExport);
      scriptFunction();
      await global.testMain();

      expect(mockCore.info).toHaveBeenCalledWith("No agent log file specified");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });

  describe("helper function tests", () => {
    it("should format bash commands correctly", () => {
      const parseCopilotLog = extractParseFunction();

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

      const result = parseCopilotLog(logWithCommand);

      // Check that multi-line commands are normalized to single line
      expect(result).toContain("echo 'hello world' && ls -la && pwd");
    });

    it("should truncate long strings appropriately", () => {
      const parseCopilotLog = extractParseFunction();

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

      const result = parseCopilotLog(logWithLongCommand);

      // Should truncate and add ellipsis
      expect(result).toContain("...");
    });

    it("should format MCP tool names correctly", () => {
      const parseCopilotLog = extractParseFunction();

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

      const result = parseCopilotLog(logWithMcpTool);

      expect(result).toContain("github::create_pull_request");
    });
  });

  describe("declared output files", () => {
    let tempDir;

    beforeEach(() => {
      // Create a temp directory for test files
      tempDir = path.join(process.cwd(), `test_output_${Date.now()}`);
      fs.mkdirSync(tempDir, { recursive: true });
    });

    afterEach(() => {
      // Clean up environment variables
      delete process.env.GITHUB_AW_DECLARED_OUTPUT_FILES;

      // Clean up temp directory
      if (fs.existsSync(tempDir)) {
        fs.rmSync(tempDir, { recursive: true, force: true });
      }
    });

    it("should find agentic output in a file path", async () => {
      const outputContent = "# Agentic Output\n\nThis is the final output from the agent.";
      const outputFile = path.join(tempDir, "agentic-output.md");
      fs.writeFileSync(outputFile, outputContent);

      // Set the declared output files environment variable
      process.env.GITHUB_AW_DECLARED_OUTPUT_FILES = outputFile;

      // Extract main function and run it directly
      const scriptWithExport = parseCopilotLogScript.replace("main();", "global.testMain = main;");
      const scriptFunction = new Function(scriptWithExport);
      scriptFunction();
      await global.testMain();

      // Verify that the agentic output was found and used
      expect(mockCore.info).toHaveBeenCalledWith("Found agentic output in declared output files");
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(outputContent);
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should find agentic output in a directory path", async () => {
      const outputContent = "# Agent Summary\n\nTask completed successfully.";
      const outputFile = path.join(tempDir, "output.md");
      fs.writeFileSync(outputFile, outputContent);

      // Set the declared output files environment variable to the directory
      process.env.GITHUB_AW_DECLARED_OUTPUT_FILES = tempDir;

      // Extract main function and run it directly
      const scriptWithExport = parseCopilotLogScript.replace("main();", "global.testMain = main;");
      const scriptFunction = new Function(scriptWithExport);
      scriptFunction();
      await global.testMain();

      // Verify that the agentic output was found and used
      expect(mockCore.info).toHaveBeenCalledWith("Found agentic output in declared output files");
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(outputContent);
    });

    it("should search multiple declared paths and stop at first match", async () => {
      const outputContent1 = "# First Output\n\nThis is from the first path.";
      const outputContent2 = "# Second Output\n\nThis should not be used.";

      const dir1 = path.join(tempDir, "dir1");
      const dir2 = path.join(tempDir, "dir2");
      fs.mkdirSync(dir1);
      fs.mkdirSync(dir2);

      const outputFile1 = path.join(dir1, "agentic-output.md");
      const outputFile2 = path.join(dir2, "agentic-output.md");
      fs.writeFileSync(outputFile1, outputContent1);
      fs.writeFileSync(outputFile2, outputContent2);

      // Set multiple paths (newline separated)
      process.env.GITHUB_AW_DECLARED_OUTPUT_FILES = `${dir1}\n${dir2}`;

      // Extract main function and run it directly
      const scriptWithExport = parseCopilotLogScript.replace("main();", "global.testMain = main;");
      const scriptFunction = new Function(scriptWithExport);
      scriptFunction();
      await global.testMain();

      // Verify that only the first output was used
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(outputContent1);
      expect(mockCore.summary.addRaw).not.toHaveBeenCalledWith(outputContent2);
    });

    it("should fall back to log parsing when no agentic output found", async () => {
      const validLog = JSON.stringify([
        {
          type: "system",
          subtype: "init",
          session_id: "fallback-test",
          tools: ["Bash"],
          model: "gpt-5",
        },
        {
          type: "result",
          total_cost_usd: 0.001,
          usage: { input_tokens: 50, output_tokens: 25 },
          num_turns: 1,
        },
      ]);

      // Create a log file
      const tempLogFile = path.join(tempDir, "agent.log");
      fs.writeFileSync(tempLogFile, validLog);
      process.env.GITHUB_AW_AGENT_OUTPUT = tempLogFile;

      // Set declared output files to a directory without any output files
      process.env.GITHUB_AW_DECLARED_OUTPUT_FILES = tempDir;

      // Extract main function and run it directly
      const scriptWithExport = parseCopilotLogScript.replace("main();", "global.testMain = main;");
      const scriptFunction = new Function(scriptWithExport);
      scriptFunction();
      await global.testMain();

      // Verify that log parsing was used as fallback
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      const summaryCall = mockCore.summary.addRaw.mock.calls[0][0];
      expect(summaryCall).toContain("🚀 Initialization");
      expect(summaryCall).toContain("fallback-test");

      // Clean up
      if (fs.existsSync(tempLogFile)) {
        fs.unlinkSync(tempLogFile);
      }
    });

    it("should handle empty declared output files gracefully", async () => {
      const validLog = JSON.stringify([
        {
          type: "result",
          total_cost_usd: 0.001,
        },
      ]);

      const tempLogFile = path.join(tempDir, "agent.log");
      fs.writeFileSync(tempLogFile, validLog);
      process.env.GITHUB_AW_AGENT_OUTPUT = tempLogFile;

      // Set empty declared output files
      process.env.GITHUB_AW_DECLARED_OUTPUT_FILES = "";

      // Extract main function and run it directly
      const scriptWithExport = parseCopilotLogScript.replace("main();", "global.testMain = main;");
      const scriptFunction = new Function(scriptWithExport);
      scriptFunction();
      await global.testMain();

      // Should fall back to log parsing
      expect(mockCore.summary.write).toHaveBeenCalled();

      // Clean up
      if (fs.existsSync(tempLogFile)) {
        fs.unlinkSync(tempLogFile);
      }
    });
  });
});
