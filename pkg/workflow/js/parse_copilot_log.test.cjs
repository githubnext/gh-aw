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

      // Should use 6 backticks (not 5) for code blocks
      expect(result).toContain("``````json");
      expect(result).toMatch(/``````\n#/); // Response content should start after 6 backticks

      // Should have Parameters and Response sections
      expect(result).toContain("**Parameters:**");
      expect(result).toContain("**Response:**");

      // Parameters should be formatted as JSON
      expect(result).toContain("``````json");

      // Verify the structure contains both parameter and response sections
      const detailsMatch = result.match(/<details>[\s\S]*?<\/details>/);
      expect(detailsMatch).toBeDefined();
      const detailsContent = detailsMatch[0];
      expect(detailsContent).toContain("**Parameters:**");
      expect(detailsContent).toContain("**Response:**");
      expect(detailsContent).toContain('"command": "cat README.md"');
      expect(detailsContent).toContain("Project description here");
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

    it("should parse debug log format with tool calls and mark them as successful", () => {
      // Simulating the actual debug log format from Copilot CLI
      const debugLogFormat = `2025-09-26T11:13:11.798Z [DEBUG] Using model: claude-sonnet-4
2025-09-26T11:13:12.575Z [START-GROUP] Sending request to the AI model
2025-09-26T11:13:17.989Z [DEBUG] response (Request-ID test-123):
2025-09-26T11:13:17.989Z [DEBUG] data:
{
  "id": "chatcmpl-test",
  "object": "chat.completion",
  "model": "claude-sonnet-4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "I'll help you with this task.",
        "tool_calls": [
          {
            "id": "call_abc123",
            "type": "function",
            "function": {
              "name": "bash",
              "arguments": "{\\"command\\":\\"echo 'Hello World'\\",\\"description\\":\\"Print greeting\\",\\"sessionId\\":\\"main\\",\\"async\\":false}"
            }
          },
          {
            "id": "call_def456",
            "type": "function",
            "function": {
              "name": "github-search_issues",
              "arguments": "{\\"query\\":\\"is:open label:bug\\"}"
            }
          }
        ]
      },
      "finish_reason": "tool_calls"
    }
  ],
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 50,
    "total_tokens": 150
  }
}
2025-09-26T11:13:18.000Z [END-GROUP]`;

      const result = parseCopilotLog(debugLogFormat);

      // Should successfully parse the debug log format
      expect(result).toContain("🤖 Commands and Tools");
      expect(result).toContain("echo 'Hello World'");
      expect(result).toContain("github::search_issues");

      // CRITICAL: Tools should be marked as successful (✅) not unknown (❓)
      // This is the fix for the issue - parseDebugLogFormat now creates tool_result entries
      expect(result).toContain("✅");
      expect(result).not.toContain("❓ `echo");
      expect(result).not.toContain("❓ `github::search_issues");

      // Check that the tool calls are in the Commands and Tools section with success icon
      const commandsSection = result.split("📊 Information")[0];
      expect(commandsSection).toContain("✅ `echo 'Hello World'`");
      expect(commandsSection).toContain("✅ `github::search_issues(...)`");
    });

    it("should extract and display premium model information from debug logs", () => {
      // Simulating the actual debug log format from Copilot CLI with model info
      const debugLogWithModelInfo = `2025-09-26T11:13:11.798Z [DEBUG] Using model: claude-sonnet-4
2025-09-26T11:13:11.944Z [DEBUG] Got model info: {
  "billing": {
    "is_premium": true,
    "multiplier": 1,
    "restricted_to": [
      "pro",
      "pro_plus",
      "max",
      "business",
      "enterprise"
    ]
  },
  "capabilities": {
    "family": "claude-sonnet-4",
    "limits": {
      "max_context_window_tokens": 200000,
      "max_output_tokens": 16000
    }
  },
  "id": "claude-sonnet-4",
  "name": "Claude Sonnet 4",
  "vendor": "Anthropic",
  "version": "claude-sonnet-4"
}
2025-09-26T11:13:12.575Z [START-GROUP] Sending request to the AI model
2025-09-26T11:13:17.989Z [DEBUG] response (Request-ID test-123):
2025-09-26T11:13:17.989Z [DEBUG] data:
{
  "id": "chatcmpl-test",
  "object": "chat.completion",
  "model": "claude-sonnet-4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "I'll help you with this task."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 50,
    "total_tokens": 150
  }
}
2025-09-26T11:13:18.000Z [END-GROUP]`;

      const result = parseCopilotLog(debugLogWithModelInfo);

      // Should successfully parse and display premium model information
      expect(result).toContain("🚀 Initialization");
      expect(result).toContain("**Model Name:** Claude Sonnet 4 (Anthropic)");
      expect(result).toContain("**Premium Model:** Yes");
      expect(result).toContain("**Required Plans:** pro, pro_plus, max, business, enterprise");
    });

    it("should handle non-premium models in debug logs", () => {
      // Simulating debug log with non-premium model
      const debugLogNonPremium = `2025-09-26T11:13:11.798Z [DEBUG] Using model: gpt-4o
2025-09-26T11:13:11.944Z [DEBUG] Got model info: {
  "billing": {
    "is_premium": false,
    "multiplier": 1
  },
  "id": "gpt-4o",
  "name": "GPT-4o",
  "vendor": "OpenAI"
}
2025-09-26T11:13:12.575Z [DEBUG] response (Request-ID test-123):
2025-09-26T11:13:12.575Z [DEBUG] data:
{
  "id": "chatcmpl-test",
  "model": "gpt-4o",
  "choices": [{"index": 0, "message": {"role": "assistant", "content": "Hello"}, "finish_reason": "stop"}],
  "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
}`;

      const result = parseCopilotLog(debugLogNonPremium);

      // Should display non-premium model information
      expect(result).toContain("**Model Name:** GPT-4o (OpenAI)");
      expect(result).toContain("**Premium Model:** No");
    });

    it("should handle model info with cost multiplier", () => {
      // Simulating debug log with cost multiplier
      const debugLogWithMultiplier = `2025-09-26T11:13:11.798Z [DEBUG] Using model: claude-opus
2025-09-26T11:13:11.944Z [DEBUG] Got model info: {
  "billing": {
    "is_premium": true,
    "multiplier": 2.5,
    "restricted_to": ["enterprise"]
  },
  "id": "claude-opus",
  "name": "Claude Opus",
  "vendor": "Anthropic"
}
2025-09-26T11:13:12.575Z [DEBUG] response (Request-ID test-123):
2025-09-26T11:13:12.575Z [DEBUG] data:
{
  "id": "chatcmpl-test",
  "model": "claude-opus",
  "choices": [{"index": 0, "message": {"role": "assistant", "content": "Hello"}, "finish_reason": "stop"}],
  "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
}`;

      const result = parseCopilotLog(debugLogWithMultiplier);

      // Should display cost multiplier
      expect(result).toContain("**Premium Model:** Yes (2.5x cost multiplier)");
      expect(result).toContain("**Required Plans:** enterprise");
    });

    it("should display premium requests consumed for premium models", () => {
      // Simulating log with premium model and multiple turns
      const debugLogWithPremiumRequests = `2025-09-26T11:13:11.798Z [DEBUG] Using model: claude-sonnet-4
2025-09-26T11:13:11.944Z [DEBUG] Got model info: {
  "billing": {
    "is_premium": true,
    "multiplier": 1,
    "restricted_to": ["pro", "pro_plus", "max"]
  },
  "id": "claude-sonnet-4",
  "name": "Claude Sonnet 4",
  "vendor": "Anthropic"
}
2025-09-26T11:13:12.575Z [DEBUG] response (Request-ID test-123):
2025-09-26T11:13:12.575Z [DEBUG] data:
{
  "id": "chatcmpl-test",
  "model": "claude-sonnet-4",
  "choices": [{"index": 0, "message": {"role": "assistant", "content": "Hello"}, "finish_reason": "stop"}],
  "usage": {"prompt_tokens": 1000, "completion_tokens": 250, "total_tokens": 1250}
}`;

      // Parse as structured format to get proper result entry
      const structuredLog = JSON.stringify([
        {
          type: "system",
          subtype: "init",
          session_id: "test-premium",
          model: "claude-sonnet-4",
          tools: [],
          model_info: {
            billing: {
              is_premium: true,
              multiplier: 1,
              restricted_to: ["pro", "pro_plus", "max"],
            },
            id: "claude-sonnet-4",
            name: "Claude Sonnet 4",
            vendor: "Anthropic",
          },
        },
        {
          type: "assistant",
          message: {
            content: [{ type: "text", text: "Hello" }],
          },
        },
        {
          type: "result",
          num_turns: 5,
          usage: {
            input_tokens: 1000,
            output_tokens: 250,
          },
        },
      ]);

      const result = parseCopilotLog(structuredLog);

      // Should display premium requests consumed (always 1 per workflow run, regardless of num_turns)
      expect(result).toContain("**Premium Requests Consumed:** 1");
      expect(result).toContain("**Turns:** 5");
      expect(result).toContain("**Token Usage:**");
      expect(result).toContain("- Input: 1,000");
      expect(result).toContain("- Output: 250");
    });

    it("should not display premium requests for non-premium models", () => {
      // Simulating log with non-premium model
      const structuredLog = JSON.stringify([
        {
          type: "system",
          subtype: "init",
          session_id: "test-non-premium",
          model: "gpt-4o",
          tools: [],
          model_info: {
            billing: {
              is_premium: false,
              multiplier: 1,
            },
            id: "gpt-4o",
            name: "GPT-4o",
            vendor: "OpenAI",
          },
        },
        {
          type: "result",
          num_turns: 3,
          usage: {
            input_tokens: 500,
            output_tokens: 100,
          },
        },
      ]);

      const result = parseCopilotLog(structuredLog);

      // Should NOT display premium requests consumed
      expect(result).not.toContain("Premium Requests Consumed");
      expect(result).toContain("**Turns:** 3");
      expect(result).toContain("**Token Usage:**");
    });

    it("should display 1 premium request consumed regardless of number of turns", () => {
      // Test the fix: with 17 turns, should show 1 premium request (not 17)
      // This tests the bug fix for https://github.com/githubnext/gh-aw/issues/XXX
      const structuredLog = JSON.stringify([
        {
          type: "system",
          subtype: "init",
          session_id: "test-multiple-turns",
          model: "claude-sonnet-4",
          tools: [],
          model_info: {
            billing: {
              is_premium: true,
              multiplier: 1,
              restricted_to: ["pro", "pro_plus", "max"],
            },
            id: "claude-sonnet-4",
            name: "Claude Sonnet 4",
            vendor: "Anthropic",
          },
        },
        {
          type: "assistant",
          message: {
            content: [{ type: "text", text: "Response 1" }],
          },
        },
        {
          type: "assistant",
          message: {
            content: [{ type: "text", text: "Response 2" }],
          },
        },
        {
          type: "result",
          num_turns: 17, // Many turns, but should still show only 1 premium request
          usage: {
            input_tokens: 5000,
            output_tokens: 1000,
          },
        },
      ]);

      const result = parseCopilotLog(structuredLog);

      // Should display 1 premium request consumed (not 17)
      expect(result).toContain("**Premium Requests Consumed:** 1");
      expect(result).toContain("**Turns:** 17");
      expect(result).toContain("**Token Usage:**");
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

      expect(mockCore.info).toHaveBeenCalledWith("Log path not found: /nonexistent/file.log");
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
});
