import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
describe("parse_copilot_log.cjs", () => {
  let mockCore, parseCopilotLogScript, originalConsole, originalProcess;
  (beforeEach(() => {
    ((originalConsole = global.console),
      (originalProcess = { ...process }),
      (global.console = { log: vi.fn(), error: vi.fn() }),
      (mockCore = {
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
        isDebug: vi.fn().mockReturnValue(!1),
        getIDToken: vi.fn(),
        toPlatformPath: vi.fn(),
        toPosixPath: vi.fn(),
        toWin32Path: vi.fn(),
        summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() },
      }),
      (global.core = mockCore),
      (global.require = vi.fn().mockImplementation(module => {
        if ("fs" === module) return fs;
        if ("path" === module) return path;
        if ("@actions/core" === module) return mockCore;
        if ("./log_parser_bootstrap.cjs" === module) return require("./log_parser_bootstrap.cjs");
        if ("./log_parser_shared.cjs" === module) return require("./log_parser_shared.cjs");
        throw new Error(`Module not found: ${module}`);
      })));
    const scriptPath = path.join(__dirname, "parse_copilot_log.cjs");
    parseCopilotLogScript = fs.readFileSync(scriptPath, "utf8");
  }),
    afterEach(() => {
      (delete process.env.GH_AW_AGENT_OUTPUT, (global.console = originalConsole), (process.env = originalProcess.env), delete global.core, delete global.require);
    }));
  const extractParseFunction = () => {
    const scriptWithExport = parseCopilotLogScript.replace("main();", "global.testParseCopilotLog = parseCopilotLog;");
    return (new Function(scriptWithExport)(), global.testParseCopilotLog);
  };
  (describe("parseCopilotLog function", () => {
    let parseCopilotLog;
    (beforeEach(() => {
      parseCopilotLog = extractParseFunction();
    }),
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
          ]),
          result = parseCopilotLog(jsonArrayLog);
        (expect(result.markdown).toContain("🚀 Initialization"),
          expect(result.markdown).toContain("🤖 Commands and Tools"),
          expect(result.markdown).toContain("copilot-test-123"),
          expect(result.markdown).toContain("echo 'Hello World'"),
          expect(result.markdown).toContain("Total Cost"),
          expect(result.markdown).toContain("<details>"),
          expect(result.markdown).toContain("<summary>"));
      }),
      it("should parse mixed format with debug logs and JSON array", () => {
        const result = parseCopilotLog(
          '[DEBUG] Starting Copilot CLI\n[ERROR] Some error occurred\n[{"type":"system","subtype":"init","session_id":"copilot-456","tools":["Bash","mcp__safe_outputs__missing-tool"],"model":"gpt-5"},{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tool_123","name":"mcp__safe_outputs__missing-tool","input":{"tool":"draw_pelican","reason":"Tool needed to draw pelican artwork"}}]}},{"type":"result","total_cost_usd":0.1789264,"usage":{"input_tokens":25,"output_tokens":832},"num_turns":10}]\n[DEBUG] Session completed'
        );
        (expect(result.markdown).toContain("🚀 Initialization"),
          expect(result.markdown).toContain("🤖 Commands and Tools"),
          expect(result.markdown).toContain("copilot-456"),
          expect(result.markdown).toContain("safe_outputs::missing-tool"),
          expect(result.markdown).toContain("Total Cost"));
      }),
      it("should parse mixed format with individual JSON lines (JSONL)", () => {
        const result = parseCopilotLog(
          '[DEBUG] Starting Copilot CLI\n{"type":"system","subtype":"init","session_id":"copilot-789","tools":["Bash","Read"],"model":"gpt-5"}\n[DEBUG] Processing user prompt\n{"type":"assistant","message":{"content":[{"type":"text","text":"I\'ll help you."},{"type":"tool_use","id":"tool_123","name":"Bash","input":{"command":"ls -la"}}]}}\n{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool_123","content":"file1.txt\\nfile2.txt"}]}}\n{"type":"result","total_cost_usd":0.002,"usage":{"input_tokens":100,"output_tokens":25},"num_turns":2}\n[DEBUG] Workflow completed'
        );
        (expect(result.markdown).toContain("🚀 Initialization"),
          expect(result.markdown).toContain("🤖 Commands and Tools"),
          expect(result.markdown).toContain("copilot-789"),
          expect(result.markdown).toContain("ls -la"),
          expect(result.markdown).toContain("Total Cost"));
      }),
      it("should handle tool calls with details in HTML format", () => {
        const logWithToolOutput = JSON.stringify([
            { type: "system", subtype: "init", session_id: "test-details", tools: ["Bash"], model: "gpt-5" },
            { type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "Bash", input: { command: "cat README.md", description: "Read README" } }] } },
            { type: "user", message: { content: [{ type: "tool_result", tool_use_id: "tool_1", content: "# Project Title\n\nProject description here." }] } },
          ]),
          result = parseCopilotLog(logWithToolOutput);
        (expect(result.markdown).toContain("<details>"),
          expect(result.markdown).toContain("<summary>"),
          expect(result.markdown).toContain("</summary>"),
          expect(result.markdown).toContain("</details>"),
          expect(result.markdown).toContain("cat README.md"),
          expect(result.markdown).toContain("Project Title"),
          expect(result.markdown).toContain("``````json"),
          expect(result.markdown).toMatch(/``````\n#/),
          expect(result.markdown).toContain("**Parameters:**"),
          expect(result.markdown).toContain("**Response:**"),
          expect(result.markdown).toContain("``````json"));
        const detailsMatch = result.markdown.match(/<details>[\s\S]*?<\/details>/);
        expect(detailsMatch).toBeDefined();
        const detailsContent = detailsMatch[0];
        (expect(detailsContent).toContain("**Parameters:**"), expect(detailsContent).toContain("**Response:**"), expect(detailsContent).toContain('"command": "cat README.md"'), expect(detailsContent).toContain("Project description here"));
      }),
      it("should handle MCP tools", () => {
        const logWithMcpTools = JSON.stringify([
            { type: "system", subtype: "init", session_id: "mcp-test", tools: ["mcp__github__create_issue", "mcp__safe_outputs__missing-tool"], model: "gpt-5" },
            {
              type: "assistant",
              message: {
                content: [
                  { type: "tool_use", id: "tool_1", name: "mcp__github__create_issue", input: { title: "Test Issue", body: "Test description" } },
                  { type: "tool_use", id: "tool_2", name: "mcp__safe_outputs__missing-tool", input: { tool: "missing_tool", reason: "Not available" } },
                ],
              },
            },
          ]),
          result = parseCopilotLog(logWithMcpTools);
        (expect(result.markdown).toContain("github::create_issue"), expect(result.markdown).toContain("safe_outputs::missing-tool"));
      }),
      it("should handle unrecognized log format", () => {
        const result = parseCopilotLog("This is not JSON or valid format");
        expect(result.markdown).toContain("Log format not recognized");
      }),
      it("should handle empty log content", () => {
        const result = parseCopilotLog("");
        expect(result.markdown).toContain("Log format not recognized");
      }),
      it("should handle empty JSON array", () => {
        const result = parseCopilotLog("[]");
        expect(result.markdown).toContain("Log format not recognized");
      }),
      it("should skip internal file operations in summary", () => {
        const logWithInternalTools = JSON.stringify([
            {
              type: "assistant",
              message: {
                content: [
                  { type: "tool_use", id: "tool_1", name: "Read", input: { file_path: "/tmp/gh-aw/test.txt" } },
                  { type: "tool_use", id: "tool_2", name: "Write", input: { file_path: "/tmp/gh-aw/output.txt" } },
                  { type: "tool_use", id: "tool_3", name: "Bash", input: { command: "echo test" } },
                ],
              },
            },
          ]),
          result = parseCopilotLog(logWithInternalTools);
        expect(result.markdown).toContain("🤖 Commands and Tools");
        const commandsSection = result.markdown.split("📊 Information")[0];
        (expect(commandsSection).toContain("echo test"), expect(commandsSection.split("🤖 Reasoning")[0]).not.toContain("Read"), expect(commandsSection.split("🤖 Reasoning")[0]).not.toContain("Write"));
      }),
      it("should render user text messages as markdown", () => {
        const logWithTextMessage = JSON.stringify([
            { type: "assistant", message: { content: [{ type: "text", text: "Let me analyze the code and provide feedback.\n\n## Analysis\n\nThe code looks good but could use some improvements." }] } },
          ]),
          result = parseCopilotLog(logWithTextMessage);
        (expect(result.markdown).toContain("🤖 Reasoning"), expect(result.markdown).toContain("Let me analyze the code"), expect(result.markdown).toContain("## Analysis"), expect(result.markdown).toContain("could use some improvements"));
      }),
      it("should parse debug log format with tool calls and mark them as successful", () => {
        const result = parseCopilotLog(
          '2025-09-26T11:13:11.798Z [DEBUG] Using model: claude-sonnet-4\n2025-09-26T11:13:12.575Z [START-GROUP] Sending request to the AI model\n2025-09-26T11:13:17.989Z [DEBUG] response (Request-ID test-123):\n2025-09-26T11:13:17.989Z [DEBUG] data:\n{\n  "id": "chatcmpl-test",\n  "object": "chat.completion",\n  "model": "claude-sonnet-4",\n  "choices": [\n    {\n      "index": 0,\n      "message": {\n        "role": "assistant",\n        "content": "I\'ll help you with this task.",\n        "tool_calls": [\n          {\n            "id": "call_abc123",\n            "type": "function",\n            "function": {\n              "name": "bash",\n              "arguments": "{\\"command\\":\\"echo \'Hello World\'\\",\\"description\\":\\"Print greeting\\",\\"sessionId\\":\\"main\\",\\"async\\":false}"\n            }\n          },\n          {\n            "id": "call_def456",\n            "type": "function",\n            "function": {\n              "name": "github-search_issues",\n              "arguments": "{\\"query\\":\\"is:open label:bug\\"}"\n            }\n          }\n        ]\n      },\n      "finish_reason": "tool_calls"\n    }\n  ],\n  "usage": {\n    "prompt_tokens": 100,\n    "completion_tokens": 50,\n    "total_tokens": 150\n  }\n}\n2025-09-26T11:13:18.000Z [END-GROUP]'
        );
        (expect(result.markdown).toContain("🤖 Commands and Tools"),
          expect(result.markdown).toContain("echo 'Hello World'"),
          expect(result.markdown).toContain("github::search_issues"),
          expect(result.markdown).toContain("✅"),
          expect(result.markdown).not.toContain("❓ `echo"),
          expect(result.markdown).not.toContain("❓ `github::search_issues"));
        const commandsSection = result.markdown.split("📊 Information")[0];
        (expect(commandsSection).toContain("✅ `echo 'Hello World'`"), expect(commandsSection).toContain("✅ `github::search_issues(...)`"));
      }),
      it("should extract and display premium model information from debug logs", () => {
        const result = parseCopilotLog(
          '2025-09-26T11:13:11.798Z [DEBUG] Using model: claude-sonnet-4\n2025-09-26T11:13:11.944Z [DEBUG] Got model info: {\n  "billing": {\n    "is_premium": true,\n    "multiplier": 1,\n    "restricted_to": [\n      "pro",\n      "pro_plus",\n      "max",\n      "business",\n      "enterprise"\n    ]\n  },\n  "capabilities": {\n    "family": "claude-sonnet-4",\n    "limits": {\n      "max_context_window_tokens": 200000,\n      "max_output_tokens": 16000\n    }\n  },\n  "id": "claude-sonnet-4",\n  "name": "Claude Sonnet 4",\n  "vendor": "Anthropic",\n  "version": "claude-sonnet-4"\n}\n2025-09-26T11:13:12.575Z [START-GROUP] Sending request to the AI model\n2025-09-26T11:13:17.989Z [DEBUG] response (Request-ID test-123):\n2025-09-26T11:13:17.989Z [DEBUG] data:\n{\n  "id": "chatcmpl-test",\n  "object": "chat.completion",\n  "model": "claude-sonnet-4",\n  "choices": [\n    {\n      "index": 0,\n      "message": {\n        "role": "assistant",\n        "content": "I\'ll help you with this task."\n      },\n      "finish_reason": "stop"\n    }\n  ],\n  "usage": {\n    "prompt_tokens": 100,\n    "completion_tokens": 50,\n    "total_tokens": 150\n  }\n}\n2025-09-26T11:13:18.000Z [END-GROUP]'
        );
        (expect(result.markdown).toContain("🚀 Initialization"),
          expect(result.markdown).toContain("**Model Name:** Claude Sonnet 4 (Anthropic)"),
          expect(result.markdown).toContain("**Premium Model:** Yes"),
          expect(result.markdown).toContain("**Required Plans:** pro, pro_plus, max, business, enterprise"));
      }),
      it("should handle non-premium models in debug logs", () => {
        const result = parseCopilotLog(
          '2025-09-26T11:13:11.798Z [DEBUG] Using model: gpt-4o\n2025-09-26T11:13:11.944Z [DEBUG] Got model info: {\n  "billing": {\n    "is_premium": false,\n    "multiplier": 1\n  },\n  "id": "gpt-4o",\n  "name": "GPT-4o",\n  "vendor": "OpenAI"\n}\n2025-09-26T11:13:12.575Z [DEBUG] response (Request-ID test-123):\n2025-09-26T11:13:12.575Z [DEBUG] data:\n{\n  "id": "chatcmpl-test",\n  "model": "gpt-4o",\n  "choices": [{"index": 0, "message": {"role": "assistant", "content": "Hello"}, "finish_reason": "stop"}],\n  "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}\n}'
        );
        (expect(result.markdown).toContain("**Model Name:** GPT-4o (OpenAI)"), expect(result.markdown).toContain("**Premium Model:** No"));
      }),
      it("should handle model info with cost multiplier", () => {
        const result = parseCopilotLog(
          '2025-09-26T11:13:11.798Z [DEBUG] Using model: claude-opus\n2025-09-26T11:13:11.944Z [DEBUG] Got model info: {\n  "billing": {\n    "is_premium": true,\n    "multiplier": 2.5,\n    "restricted_to": ["enterprise"]\n  },\n  "id": "claude-opus",\n  "name": "Claude Opus",\n  "vendor": "Anthropic"\n}\n2025-09-26T11:13:12.575Z [DEBUG] response (Request-ID test-123):\n2025-09-26T11:13:12.575Z [DEBUG] data:\n{\n  "id": "chatcmpl-test",\n  "model": "claude-opus",\n  "choices": [{"index": 0, "message": {"role": "assistant", "content": "Hello"}, "finish_reason": "stop"}],\n  "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}\n}'
        );
        (expect(result.markdown).toContain("**Premium Model:** Yes (2.5x cost multiplier)"), expect(result.markdown).toContain("**Required Plans:** enterprise"));
      }),
      it("should display premium requests consumed for premium models", () => {
        const structuredLog = JSON.stringify([
            {
              type: "system",
              subtype: "init",
              session_id: "test-premium",
              model: "claude-sonnet-4",
              tools: [],
              model_info: { billing: { is_premium: !0, multiplier: 1, restricted_to: ["pro", "pro_plus", "max"] }, id: "claude-sonnet-4", name: "Claude Sonnet 4", vendor: "Anthropic" },
            },
            { type: "assistant", message: { content: [{ type: "text", text: "Hello" }] } },
            { type: "result", num_turns: 5, usage: { input_tokens: 1e3, output_tokens: 250 } },
          ]),
          result = parseCopilotLog(structuredLog);
        (expect(result.markdown).toContain("**Premium Requests Consumed:** 1"),
          expect(result.markdown).toContain("**Turns:** 5"),
          expect(result.markdown).toContain("**Token Usage:**"),
          expect(result.markdown).toContain("- Input: 1,000"),
          expect(result.markdown).toContain("- Output: 250"));
      }),
      it("should not display premium requests for non-premium models", () => {
        const structuredLog = JSON.stringify([
            { type: "system", subtype: "init", session_id: "test-non-premium", model: "gpt-4o", tools: [], model_info: { billing: { is_premium: !1, multiplier: 1 }, id: "gpt-4o", name: "GPT-4o", vendor: "OpenAI" } },
            { type: "result", num_turns: 3, usage: { input_tokens: 500, output_tokens: 100 } },
          ]),
          result = parseCopilotLog(structuredLog);
        (expect(result.markdown).not.toContain("Premium Requests Consumed"), expect(result.markdown).toContain("**Turns:** 3"), expect(result.markdown).toContain("**Token Usage:**"));
      }),
      it("should display 1 premium request consumed regardless of number of turns", () => {
        const structuredLog = JSON.stringify([
            {
              type: "system",
              subtype: "init",
              session_id: "test-multiple-turns",
              model: "claude-sonnet-4",
              tools: [],
              model_info: { billing: { is_premium: !0, multiplier: 1, restricted_to: ["pro", "pro_plus", "max"] }, id: "claude-sonnet-4", name: "Claude Sonnet 4", vendor: "Anthropic" },
            },
            { type: "assistant", message: { content: [{ type: "text", text: "Response 1" }] } },
            { type: "assistant", message: { content: [{ type: "text", text: "Response 2" }] } },
            { type: "result", num_turns: 17, usage: { input_tokens: 5e3, output_tokens: 1e3 } },
          ]),
          result = parseCopilotLog(structuredLog);
        (expect(result.markdown).toContain("**Premium Requests Consumed:** 1"), expect(result.markdown).toContain("**Turns:** 17"), expect(result.markdown).toContain("**Token Usage:**"));
      }),
      it("should accumulate token usage across multiple API responses in debug logs", () => {
        const result = parseCopilotLog(
          '2025-10-21T01:00:00.000Z [INFO] Starting Copilot CLI: 0.0.350\n2025-10-21T01:00:01.000Z [DEBUG] response (Request-ID test-1):\n2025-10-21T01:00:01.000Z [DEBUG] data:\n{\n  "id": "chatcmpl-1",\n  "model": "claude-sonnet-4",\n  "choices": [{\n    "message": {\n      "role": "assistant",\n      "content": "I\'ll help you."\n    },\n    "finish_reason": "stop"\n  }],\n  "usage": {\n    "prompt_tokens": 100,\n    "completion_tokens": 50,\n    "total_tokens": 150\n  }\n}\n2025-10-21T01:00:02.000Z [DEBUG] response (Request-ID test-2):\n2025-10-21T01:00:02.000Z [DEBUG] data:\n{\n  "id": "chatcmpl-2",\n  "model": "claude-sonnet-4",\n  "choices": [{\n    "message": {\n      "role": "assistant",\n      "content": "Done!"\n    },\n    "finish_reason": "stop"\n  }],\n  "usage": {\n    "prompt_tokens": 200,\n    "completion_tokens": 10,\n    "total_tokens": 210\n  }\n}'
        );
        (expect(result.markdown).toContain("**Token Usage:**"), expect(result.markdown).toContain("- Input: 300"), expect(result.markdown).toContain("- Output: 60"), expect(result.markdown).toContain("**Turns:** 2"));
      }),
      it("should extract premium request count from log content using regex", () => {
        const logWithPremiumInfo =
            "\nSome log output here\n[INFO] Premium requests consumed: 3\nMore log content\n" +
            JSON.stringify([
              { type: "system", subtype: "init", session_id: "test-regex", model: "claude-sonnet-4", tools: [], model_info: { billing: { is_premium: !0 }, id: "claude-sonnet-4", name: "Claude Sonnet 4", vendor: "Anthropic" } },
              { type: "result", num_turns: 10, usage: { input_tokens: 1e3, output_tokens: 200 } },
            ]),
          result = parseCopilotLog(logWithPremiumInfo);
        (expect(result.markdown).toContain("**Premium Requests Consumed:** 3"), expect(result.markdown).toContain("**Turns:** 10"));
      }));
  }),
    describe("extractPremiumRequestCount function", () => {
      let extractPremiumRequestCount;
      (beforeEach(() => {
        const scriptWithExport = fs.readFileSync(path.join(__dirname, "parse_copilot_log.cjs"), "utf8").replace("main();", "global.testExtractPremiumRequestCount = extractPremiumRequestCount;");
        (new Function(scriptWithExport)(), (extractPremiumRequestCount = global.testExtractPremiumRequestCount));
      }),
        it("should extract premium request count from various formats", () => {
          (expect(extractPremiumRequestCount("Premium requests consumed: 5")).toBe(5),
            expect(extractPremiumRequestCount("3 premium requests consumed")).toBe(3),
            expect(extractPremiumRequestCount("Consumed 7 premium requests")).toBe(7),
            expect(extractPremiumRequestCount("[INFO] Premium request consumed: 1")).toBe(1));
        }),
        it("should default to 1 if no match found", () => {
          (expect(extractPremiumRequestCount("No premium info here")).toBe(1), expect(extractPremiumRequestCount("")).toBe(1), expect(extractPremiumRequestCount("Some random log content")).toBe(1));
        }),
        it("should handle case-insensitive matching", () => {
          (expect(extractPremiumRequestCount("PREMIUM REQUESTS CONSUMED: 4")).toBe(4), expect(extractPremiumRequestCount("premium Request Consumed: 2")).toBe(2));
        }),
        it("should ignore invalid numbers", () => {
          (expect(extractPremiumRequestCount("Premium requests consumed: 0")).toBe(1),
            expect(extractPremiumRequestCount("Premium requests consumed: -5")).toBe(1),
            expect(extractPremiumRequestCount("Premium requests consumed: abc")).toBe(1));
        }));
    }),
    describe("main function integration", () => {
      (it("should handle valid log file", async () => {
        const validLog = JSON.stringify([
          { type: "system", subtype: "init", session_id: "integration-test", tools: ["Bash"], model: "gpt-5" },
          { type: "result", total_cost_usd: 0.001, usage: { input_tokens: 50, output_tokens: 25 }, num_turns: 1 },
        ]);
        (await (async logContent => {
          const tempFile = path.join(process.cwd(), `test_log_${Date.now()}.txt`);
          (fs.writeFileSync(tempFile, logContent), (process.env.GH_AW_AGENT_OUTPUT = tempFile));
          try {
            const scriptWithExports = parseCopilotLogScript.replace("main();", "global.testParseCopilotLog = parseCopilotLog; global.testMain = main; main();"),
              scriptFunction = new Function(scriptWithExports);
            await scriptFunction();
          } finally {
            fs.existsSync(tempFile) && fs.unlinkSync(tempFile);
          }
        })(validLog),
          expect(mockCore.summary.addRaw).toHaveBeenCalled(),
          expect(mockCore.summary.write).toHaveBeenCalled(),
          expect(mockCore.setFailed).not.toHaveBeenCalled());
        const markdownCall = mockCore.summary.addRaw.mock.calls[0];
        (expect(markdownCall[0]).toContain("```"), expect(markdownCall[0]).toContain("Conversation:"), expect(markdownCall[0]).toContain("Statistics:"), expect(mockCore.info).toHaveBeenCalled());
        const infoCall = mockCore.info.mock.calls.find(call => call[0].includes("=== Copilot Execution Summary ==="));
        (expect(infoCall).toBeDefined(), expect(infoCall[0]).toContain("Model: gpt-5"));
      }),
        it("should handle missing log file", async () => {
          process.env.GH_AW_AGENT_OUTPUT = "/nonexistent/file.log";
          const scriptWithExport = parseCopilotLogScript.replace("main();", "global.testMain = main;");
          (new Function(scriptWithExport)(), await global.testMain(), expect(mockCore.info).toHaveBeenCalledWith("Log path not found: /nonexistent/file.log"), expect(mockCore.setFailed).not.toHaveBeenCalled());
        }),
        it("should handle missing environment variable", async () => {
          delete process.env.GH_AW_AGENT_OUTPUT;
          const scriptWithExport = parseCopilotLogScript.replace("main();", "global.testMain = main;");
          (new Function(scriptWithExport)(), await global.testMain(), expect(mockCore.info).toHaveBeenCalledWith("No agent log file specified"), expect(mockCore.setFailed).not.toHaveBeenCalled());
        }));
    }),
    describe("helper function tests", () => {
      (it("should format bash commands correctly", () => {
        const result = extractParseFunction()(JSON.stringify([{ type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "Bash", input: { command: "echo 'hello world'\n  && ls -la\n  && pwd" } }] } }]));
        expect(result.markdown).toContain("echo 'hello world' && ls -la && pwd");
      }),
        it("should truncate long strings appropriately", () => {
          const parseCopilotLog = extractParseFunction(),
            longCommand = "a".repeat(400),
            result = parseCopilotLog(JSON.stringify([{ type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "Bash", input: { command: longCommand } }] } }]));
          expect(result.markdown).toContain("...");
        }),
        it("should format MCP tool names correctly", () => {
          const result = extractParseFunction()(JSON.stringify([{ type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "mcp__github__create_pull_request", input: { title: "Test PR" } }] } }]));
          expect(result.markdown).toContain("github::create_pull_request");
        }),
        it("should extract tools from debug log format", () => {
          const result = extractParseFunction()(
            '2025-10-18T01:34:52.534Z [INFO] Starting Copilot CLI: 0.0.343\n2025-10-18T01:34:55.314Z [DEBUG] Got model info: {\n  "id": "claude-sonnet-4.5",\n  "name": "Claude Sonnet 4.5",\n  "vendor": "Anthropic",\n  "billing": {\n    "is_premium": true,\n    "multiplier": 1,\n    "restricted_to": ["pro", "pro_plus", "max"]\n  }\n}\n2025-10-18T01:34:55.407Z [DEBUG] Tools:\n2025-10-18T01:34:55.412Z [DEBUG] [\n  {\n    "type": "function",\n    "function": {\n      "name": "bash",\n      "description": "Runs a Bash command"\n    }\n  },\n  {\n    "type": "function",\n    "function": {\n      "name": "github-create_issue",\n      "description": "Creates a GitHub issue"\n    }\n  },\n  {\n    "type": "function",\n    "function": {\n      "name": "safe_outputs-create_issue",\n      "description": "Safe output create issue"\n    }\n  }\n2025-10-18T01:34:55.500Z [DEBUG] ]\n2025-10-18T01:35:00.739Z [DEBUG] data:\n2025-10-18T01:35:00.739Z [DEBUG] {\n  "choices": [\n    {\n      "finish_reason": "tool_calls",\n      "message": {\n        "content": "I\'ll help you with this task.",\n        "role": "assistant"\n      }\n    },\n    {\n      "finish_reason": "tool_calls",\n      "message": {\n        "role": "assistant",\n        "tool_calls": [\n          {\n            "function": {\n              "arguments": "{\\"command\\":\\"echo test\\"}",\n              "name": "bash"\n            },\n            "id": "tool_123",\n            "type": "function"\n          }\n        ]\n      }\n    }\n  ],\n  "model": "Claude Sonnet 4.5",\n  "usage": {\n    "completion_tokens": 50,\n    "prompt_tokens": 100\n  }\n2025-10-18T01:35:00.800Z [DEBUG] }'
          );
          (expect(result.markdown).toContain("**Available Tools:**"),
            expect(result.markdown).toContain("bash"),
            expect(result.markdown).toContain("github::create_issue"),
            expect(result.markdown).toContain("**Safe Outputs:**"),
            expect(result.markdown).toContain("create_issue"),
            expect(result.markdown).toContain("**Git/GitHub:**"),
            expect(result.markdown).toContain("**Builtin:**"),
            expect(result.markdown).toContain("Claude Sonnet 4.5"),
            expect(result.markdown).toContain("**Premium Model:** Yes"),
            expect(result.markdown).toContain("echo test"));
        }),
        it("should detect permission denied errors in tool calls from debug logs", () => {
          const result = extractParseFunction()(
            '2025-10-24T16:00:00.000Z [INFO] Starting Copilot CLI: 0.0.350\n2025-10-24T16:00:01.000Z [DEBUG] response (Request-ID test-1):\n2025-10-24T16:00:01.000Z [DEBUG] data:\n{\n  "id": "chatcmpl-1",\n  "model": "claude-sonnet-4",\n  "choices": [{\n    "message": {\n      "role": "assistant",\n      "content": "I\'ll create an issue for you.",\n      "tool_calls": [\n        {\n          "id": "call_create_issue_123",\n          "type": "function",\n          "function": {\n            "name": "github-create_issue",\n            "arguments": "{\\"title\\":\\"Test Issue\\",\\"body\\":\\"Test body\\"}"\n          }\n        }\n      ]\n    },\n    "finish_reason": "tool_calls"\n  }],\n  "usage": {\n    "prompt_tokens": 100,\n    "completion_tokens": 50\n  }\n}\n2025-10-24T16:00:02.000Z [ERROR] Tool execution failed: github-create_issue\n2025-10-24T16:00:02.000Z [ERROR] Permission denied: Resource not accessible by integration\n2025-10-24T16:00:02.000Z [DEBUG] response (Request-ID test-2):\n2025-10-24T16:00:02.000Z [DEBUG] data:\n{\n  "id": "chatcmpl-2",\n  "model": "claude-sonnet-4",\n  "choices": [{\n    "message": {\n      "role": "assistant",\n      "content": "I encountered a permission error."\n    },\n    "finish_reason": "stop"\n  }],\n  "usage": {\n    "prompt_tokens": 200,\n    "completion_tokens": 10\n  }\n}'
          );
          expect(result.markdown).toContain("github::create_issue");
          const commandsSection = result.markdown.split("📊 Information")[0];
          (expect(commandsSection).toContain("❌"), expect(commandsSection).toContain("❌ `github::create_issue(...)`"), expect(commandsSection).not.toContain("✅ `github::create_issue(...)`"));
        }),
        it("should display all tools even when there are many (more than 5)", () => {
          const result = extractParseFunction()(
            JSON.stringify([
              {
                type: "system",
                subtype: "init",
                session_id: "many-tools-test",
                tools: [
                  "Bash",
                  "Read",
                  "Write",
                  "Edit",
                  "LS",
                  "Grep",
                  "mcp__github__create_issue",
                  "mcp__github__list_issues",
                  "mcp__github__get_issue",
                  "mcp__github__create_pull_request",
                  "mcp__github__list_pull_requests",
                  "mcp__github__get_pull_request",
                  "mcp__github__create_discussion",
                  "mcp__github__list_discussions",
                  "safe_outputs-create_issue",
                  "safe_outputs-add-comment",
                ],
                model: "gpt-5",
              },
            ])
          );
          (expect(result.markdown).toContain("github::create_issue"),
            expect(result.markdown).toContain("github::list_issues"),
            expect(result.markdown).toContain("github::get_issue"),
            expect(result.markdown).toContain("github::create_pull_request"),
            expect(result.markdown).toContain("github::list_pull_requests"),
            expect(result.markdown).toContain("github::get_pull_request"),
            expect(result.markdown).toContain("github::create_discussion"),
            expect(result.markdown).toContain("github::list_discussions"),
            expect(result.markdown).toContain("**Safe Outputs:**"),
            expect(result.markdown).toContain("create_issue"),
            expect(result.markdown).toContain("add-comment"),
            expect(result.markdown).toContain("Read"),
            expect(result.markdown).toContain("Write"),
            expect(result.markdown).toContain("Edit"),
            expect(result.markdown).toContain("LS"),
            expect(result.markdown).toContain("Grep"),
            expect(result.markdown).toContain("Bash"));
          const toolsSection = result.markdown.split("## 🤖 Reasoning")[0];
          expect(toolsSection).not.toMatch(/and \d+ more/);
        }));
    }));
});
