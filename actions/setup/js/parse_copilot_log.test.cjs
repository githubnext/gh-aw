import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

describe("parse_copilot_log.cjs", () => {
  let mockCore, originalConsole, originalProcess;
  let main, parseCopilotLog, extractPremiumRequestCount;

  beforeEach(async () => {
    originalConsole = global.console;
    originalProcess = { ...process };
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
    const module = await import("./parse_copilot_log.cjs?" + Date.now());
    main = module.main;
    parseCopilotLog = module.parseCopilotLog;
    extractPremiumRequestCount = module.extractPremiumRequestCount;
  });

  afterEach(() => {
    delete process.env.GH_AW_AGENT_OUTPUT;
    global.console = originalConsole;
    process.env = originalProcess.env;
    delete global.core;
  });

  describe("parseCopilotLog function", () => {
    it("should parse JSON array format", () => {
      const jsonArrayLog = JSON.stringify([
        { type: "system", subtype: "init", session_id: "copilot-test-123", tools: ["Bash", "Read", "mcp__github__create_issue"], model: "gpt-5" },
        {
          type: "assistant",
          message: {
            content: [
              { type: "text", text: "I'll help you with this task." },
              { type: "tool_use", id: "tool_123", name: "Bash", input: { command: "echo 'Hello World'", description: "Print greeting" } },
            ],
          },
        },
        { type: "user", message: { content: [{ type: "tool_result", tool_use_id: "tool_123", content: "Hello World\n" }] } },
        { type: "result", total_cost_usd: 0.0015, usage: { input_tokens: 150, output_tokens: 50 }, num_turns: 1 },
      ]);
      const result = parseCopilotLog(jsonArrayLog);

      expect(result.markdown).toContain("🚀 Initialization");
      expect(result.markdown).toContain("🤖 Commands and Tools");
      expect(result.markdown).toContain("copilot-test-123");
      expect(result.markdown).toContain("echo 'Hello World'");
      expect(result.markdown).toContain("Total Cost");
      expect(result.markdown).toContain("<details>");
      expect(result.markdown).toContain("<summary>");
    });

    it("should parse mixed format with debug logs and JSON array", () => {
      const result = parseCopilotLog(
        '[DEBUG] Starting Copilot CLI\n[ERROR] Some error occurred\n[{"type":"system","subtype":"init","session_id":"copilot-456","tools":["Bash","mcp__safe_outputs__missing-tool"],"model":"gpt-5"},{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tool_123","name":"mcp__safe_outputs__missing-tool","input":{"tool":"draw_pelican","reason":"Tool needed to draw pelican artwork"}}]}},{"type":"result","total_cost_usd":0.1789264,"usage":{"input_tokens":25,"output_tokens":832},"num_turns":10}]\n[DEBUG] Session completed'
      );

      expect(result.markdown).toContain("🚀 Initialization");
      expect(result.markdown).toContain("🤖 Commands and Tools");
      expect(result.markdown).toContain("copilot-456");
      expect(result.markdown).toContain("safe_outputs::missing-tool");
      expect(result.markdown).toContain("Total Cost");
    });

    it("should parse mixed format with individual JSON lines (JSONL)", () => {
      const result = parseCopilotLog(
        '[DEBUG] Starting Copilot CLI\n{"type":"system","subtype":"init","session_id":"copilot-789","tools":["Bash","Read"],"model":"gpt-5"}\n[DEBUG] Processing user prompt\n{"type":"assistant","message":{"content":[{"type":"text","text":"I\'ll help you."},{"type":"tool_use","id":"tool_123","name":"Bash","input":{"command":"ls -la"}}]}}\n{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool_123","content":"file1.txt\\nfile2.txt"}]}}\n{"type":"result","total_cost_usd":0.002,"usage":{"input_tokens":100,"output_tokens":25},"num_turns":2}\n[DEBUG] Workflow completed'
      );

      expect(result.markdown).toContain("🚀 Initialization");
      expect(result.markdown).toContain("🤖 Commands and Tools");
      expect(result.markdown).toContain("copilot-789");
      expect(result.markdown).toContain("ls -la");
      expect(result.markdown).toContain("Total Cost");
    });

    it("should handle tool calls with details in HTML format", () => {
      const logWithHtmlDetails = JSON.stringify([
        { type: "system", subtype: "init", session_id: "html-test", tools: ["Bash"], model: "gpt-5" },
        {
          type: "assistant",
          message: {
            content: [
              {
                type: "tool_use",
                id: "tool_1",
                name: "Bash",
                input: { command: "cat file.txt", description: "Read file contents" },
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
                content: "File contents here",
              },
            ],
          },
        },
      ]);
      const result = parseCopilotLog(logWithHtmlDetails);

      expect(result.markdown).toContain("<details>");
      expect(result.markdown).toContain("</details>");
      expect(result.markdown).toContain("File contents here");
    });

    it("should handle MCP tools", () => {
      const logWithMcpTools = JSON.stringify([
        {
          type: "system",
          subtype: "init",
          session_id: "mcp-test",
          tools: ["Bash", "mcp__github__create_issue", "mcp__github__list_pull_requests"],
          model: "gpt-5",
        },
        { type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "mcp__github__create_issue", input: { title: "Test" } }] } },
        { type: "result", total_cost_usd: 0.01, usage: { input_tokens: 100, output_tokens: 50 }, num_turns: 1 },
      ]);
      const result = parseCopilotLog(logWithMcpTools);

      expect(result.markdown).toContain("github::create_issue");
      expect(result.markdown).toContain("github::list_pull_requests");
    });

    it("should handle unrecognized log format", () => {
      const result = parseCopilotLog("This is not JSON or valid format");
      expect(result.markdown).toContain("Log format not recognized");
    });

    it("should handle empty log content", () => {
      const result = parseCopilotLog("");
      expect(result.markdown).toContain("Log format not recognized");
    });

    it("should handle model info with cost multiplier", () => {
      const structuredLog = JSON.stringify([
        { type: "system", subtype: "init", session_id: "cost-test", tools: ["Bash"], model: "gpt-4", model_info: { is_premium: true, cost_multiplier: 3 } },
        { type: "result", num_turns: 2, usage: { input_tokens: 500, output_tokens: 200 } },
      ]);
      const result = parseCopilotLog(structuredLog);

      expect(result.markdown).toContain("gpt-4");
    });

    it("should not display premium requests for non-premium models", () => {
      const structuredLog = JSON.stringify([
        { type: "system", subtype: "init", session_id: "non-premium-test", tools: ["Bash"], model: "gpt-3.5-turbo", model_info: { is_premium: false } },
        { type: "result", num_turns: 3, usage: { input_tokens: 500, output_tokens: 200 } },
      ]);
      const result = parseCopilotLog(structuredLog);

      expect(result.markdown).not.toContain("**Premium Requests:**");
    });
  });

  describe("extractPremiumRequestCount function", () => {
    it("should default to 1 if no match found", () => {
      expect(extractPremiumRequestCount("No premium info here")).toBe(1);
    });

    it("should ignore invalid numbers", () => {
      expect(extractPremiumRequestCount("Premium requests: abc")).toBe(1);
    });
  });

  describe("main function integration", () => {
    it("should handle valid log file", async () => {
      const validLog = JSON.stringify([
        { type: "system", subtype: "init", session_id: "integration-test", tools: ["Bash"], model: "gpt-5" },
        { type: "result", total_cost_usd: 0.001, usage: { input_tokens: 50, output_tokens: 25 }, num_turns: 1 },
      ]);

      const tempFile = path.join(process.cwd(), `test_log_${Date.now()}.txt`);
      fs.writeFileSync(tempFile, validLog);
      process.env.GH_AW_AGENT_OUTPUT = tempFile;

      try {
        await main();

        expect(mockCore.summary.addRaw).toHaveBeenCalled();
        expect(mockCore.summary.write).toHaveBeenCalled();
      } finally {
        if (fs.existsSync(tempFile)) {
          fs.unlinkSync(tempFile);
        }
      }
    });

    it("should handle missing log file", async () => {
      process.env.GH_AW_AGENT_OUTPUT = "/nonexistent/file.log";
      await main();
      expect(mockCore.info).toHaveBeenCalledWith("Log path not found: /nonexistent/file.log");
    });

    it("should handle missing environment variable", async () => {
      delete process.env.GH_AW_AGENT_OUTPUT;
      await main();
      expect(mockCore.info).toHaveBeenCalledWith("No agent log file specified");
    });
  });

  describe("helper function tests", () => {
    it("should format bash commands correctly", () => {
      const result = parseCopilotLog(JSON.stringify([{ type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "Bash", input: { command: "echo 'hello world'\n  && ls -la\n  && pwd" } }] } }]));
      expect(result.markdown).toContain("echo 'hello world' && ls -la && pwd");
    });

    it("should truncate long strings appropriately", () => {
      const longCommand = "a".repeat(400);
      const result = parseCopilotLog(JSON.stringify([{ type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "Bash", input: { command: longCommand } }] } }]));
      expect(result.markdown).toContain("...");
    });

    it("should format MCP tool names correctly", () => {
      const result = parseCopilotLog(JSON.stringify([{ type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "mcp__github__create_pull_request", input: { title: "Test PR" } }] } }]));
      expect(result.markdown).toContain("github::create_pull_request");
    });

    it("should display all tool types correctly", () => {
      const result = parseCopilotLog(
        JSON.stringify([
          {
            type: "system",
            subtype: "init",
            session_id: "all-tools",
            tools: ["Bash", "Read", "Write", "Edit", "LS", "Grep", "mcp__github__list_issues", "mcp__github__create_pull_request", "mcp__safe_outputs__create_issue"],
            model: "gpt-5",
          },
        ])
      );

      expect(result.markdown).toContain("Bash");
      expect(result.markdown).toContain("Read");
      expect(result.markdown).toContain("Write");
      expect(result.markdown).toContain("Edit");
      expect(result.markdown).toContain("LS");
      expect(result.markdown).toContain("Grep");
      expect(result.markdown).toContain("github::list_issues");
      expect(result.markdown).toContain("github::create_pull_request");
      expect(result.markdown).toContain("safe_outputs::create_issue");
    });
  });
});
