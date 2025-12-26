import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
describe("parse_claude_log.cjs", () => {
  let mockCore, parseClaudeLogScript, originalConsole, originalProcess;
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
        if ("@actions/core" === module) return mockCore;
        if ("./log_parser_bootstrap.cjs" === module) return require("./log_parser_bootstrap.cjs");
        if ("./log_parser_shared.cjs" === module) return require("./log_parser_shared.cjs");
        throw new Error(`Module not found: ${module}`);
      })));
    const scriptPath = path.join(__dirname, "parse_claude_log.cjs");
    parseClaudeLogScript = fs.readFileSync(scriptPath, "utf8");
  }),
    afterEach(() => {
      (delete process.env.GH_AW_AGENT_OUTPUT, (global.console = originalConsole), (process.env = originalProcess.env), delete global.core, delete global.require);
    }));
  const runScript = async logContent => {
      const tempFile = path.join(process.cwd(), `test_log_${Date.now()}.txt`);
      (fs.writeFileSync(tempFile, logContent), (process.env.GH_AW_AGENT_OUTPUT = tempFile));
      try {
        const scriptWithExports = parseClaudeLogScript.replace("main();", "global.testParseClaudeLog = parseClaudeLog; global.testMain = main; main();"),
          scriptFunction = new Function(scriptWithExports);
        await scriptFunction();
      } finally {
        fs.existsSync(tempFile) && fs.unlinkSync(tempFile);
      }
    },
    extractParseFunction = () => {
      const scriptWithExport = parseClaudeLogScript.replace("main();", "global.testParseClaudeLog = parseClaudeLog;");
      return (new Function(scriptWithExport)(), global.testParseClaudeLog);
    };
  (describe("parseClaudeLog function", () => {
    let parseClaudeLog;
    (beforeEach(() => {
      parseClaudeLog = extractParseFunction();
    }),
      it("should parse old JSON array format", () => {
        const jsonArrayLog = JSON.stringify([
            { type: "system", subtype: "init", session_id: "test-123", tools: ["Bash", "Read"], model: "claude-sonnet-4-20250514" },
            {
              type: "assistant",
              message: {
                content: [
                  { type: "text", text: "I'll help you with this task." },
                  { type: "tool_use", id: "tool_123", name: "Bash", input: { command: "echo 'Hello World'" } },
                ],
              },
            },
            { type: "result", total_cost_usd: 0.0015, usage: { input_tokens: 150, output_tokens: 50 }, num_turns: 1 },
          ]),
          result = parseClaudeLog(jsonArrayLog);
        (expect(result.markdown).toContain("🚀 Initialization"),
          expect(result.markdown).toContain("🤖 Commands and Tools"),
          expect(result.markdown).toContain("test-123"),
          expect(result.markdown).toContain("echo 'Hello World'"),
          expect(result.markdown).toContain("Total Cost"),
          expect(result.mcpFailures).toEqual([]));
      }),
      it("should parse new mixed format with debug logs and JSON array", () => {
        const result = parseClaudeLog(
          '[DEBUG] Starting Claude Code CLI\n[ERROR] Some error occurred\nnpm warn exec The following package was not found\n[{"type":"system","subtype":"init","session_id":"29d324d8-1a92-43c6-8740-babc2875a1d6","tools":["Task","Bash","mcp__safe_outputs__missing-tool"],"model":"claude-sonnet-4-20250514"},{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tool_123","name":"mcp__safe_outputs__missing-tool","input":{"tool":"draw_pelican","reason":"Tool needed to draw pelican artwork"}}]}},{"type":"result","total_cost_usd":0.1789264,"usage":{"input_tokens":25,"output_tokens":832},"num_turns":10}]\n[DEBUG] Session completed'
        );
        (expect(result.markdown).toContain("🚀 Initialization"),
          expect(result.markdown).toContain("🤖 Commands and Tools"),
          expect(result.markdown).toContain("29d324d8-1a92-43c6-8740-babc2875a1d6"),
          expect(result.markdown).toContain("safe_outputs::missing-tool"),
          expect(result.markdown).toContain("Total Cost"),
          expect(result.mcpFailures).toEqual([]));
      }),
      it("should parse mixed format with individual JSON lines", () => {
        const result = parseClaudeLog(
          '[DEBUG] Starting Claude Code CLI\n{"type":"system","subtype":"init","session_id":"test-456","tools":["Bash","Read"],"model":"claude-sonnet-4-20250514"}\n[DEBUG] Processing user prompt\n{"type":"assistant","message":{"content":[{"type":"text","text":"I\'ll help you."},{"type":"tool_use","id":"tool_123","name":"Bash","input":{"command":"ls -la"}}]}}\n{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool_123","content":"file1.txt\\nfile2.txt"}]}}\n{"type":"result","total_cost_usd":0.002,"usage":{"input_tokens":100,"output_tokens":25},"num_turns":2}\n[DEBUG] Workflow completed'
        );
        (expect(result.markdown).toContain("🚀 Initialization"),
          expect(result.markdown).toContain("🤖 Commands and Tools"),
          expect(result.markdown).toContain("test-456"),
          expect(result.markdown).toContain("ls -la"),
          expect(result.markdown).toContain("Total Cost"),
          expect(result.mcpFailures).toEqual([]));
      }),
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
          ]),
          result = parseClaudeLog(logWithFailures);
        (expect(result.markdown).toContain("🚀 Initialization"), expect(result.markdown).toContain("failed_server (failed)"), expect(result.mcpFailures).toEqual(["failed_server"]));
      }),
      it("should display detailed error information for failed MCP servers", () => {
        const logWithDetailedErrors = JSON.stringify([
            {
              type: "system",
              subtype: "init",
              session_id: "test-detailed-errors",
              tools: ["Bash"],
              mcp_servers: [
                { name: "working_server", status: "connected" },
                {
                  name: "failed_with_error",
                  status: "failed",
                  error: "Connection timeout after 30s",
                  stderr: "Error: ECONNREFUSED connect ECONNREFUSED 127.0.0.1:3000\n    at TCPConnectWrap.afterConnect",
                  exitCode: 1,
                  command: "npx @github/github-mcp-server",
                },
              ],
              model: "claude-sonnet-4-20250514",
            },
          ]),
          result = parseClaudeLog(logWithDetailedErrors);
        (expect(result.markdown).toContain("🚀 Initialization"),
          expect(result.markdown).toContain("failed_with_error (failed)"),
          expect(result.markdown).toContain("**Error:** Connection timeout after 30s"),
          expect(result.markdown).toContain("**Stderr:** `Error: ECONNREFUSED"),
          expect(result.markdown).toContain("**Exit Code:** 1"),
          expect(result.markdown).toContain("**Command:** `npx @github/github-mcp-server`"),
          expect(result.mcpFailures).toEqual(["failed_with_error"]));
      }),
      it("should handle MCP server failures with message and reason fields", () => {
        const logWithMessageAndReason = JSON.stringify([
            {
              type: "system",
              subtype: "init",
              session_id: "test-message-reason",
              tools: ["Bash"],
              mcp_servers: [{ name: "failed_server", status: "failed", message: "Failed to initialize MCP server", reason: "Server binary not found in PATH" }],
              model: "claude-sonnet-4-20250514",
            },
          ]),
          result = parseClaudeLog(logWithMessageAndReason);
        (expect(result.markdown).toContain("failed_server (failed)"),
          expect(result.markdown).toContain("**Message:** Failed to initialize MCP server"),
          expect(result.markdown).toContain("**Reason:** Server binary not found in PATH"),
          expect(result.mcpFailures).toEqual(["failed_server"]));
      }),
      it("should truncate long stderr output", () => {
        const longStderr = "x".repeat(1e3),
          logWithLongStderr = JSON.stringify([
            { type: "system", subtype: "init", session_id: "test-long-stderr", tools: ["Bash"], mcp_servers: [{ name: "verbose_failure", status: "failed", stderr: longStderr }], model: "claude-sonnet-4-20250514" },
          ]),
          result = parseClaudeLog(logWithLongStderr);
        (expect(result.markdown).toContain("verbose_failure (failed)"), expect(result.markdown).toContain("**Stderr:**"), expect(result.markdown).toMatch(/Stderr:.*x{500}\.\.\./), expect(result.mcpFailures).toEqual(["verbose_failure"]));
      }),
      it("should handle MCP server failures with partial error information", () => {
        const logWithPartialInfo = JSON.stringify([
            {
              type: "system",
              subtype: "init",
              session_id: "test-partial",
              tools: ["Bash"],
              mcp_servers: [
                { name: "partial_error_1", status: "failed", error: "Connection refused" },
                { name: "partial_error_2", status: "failed", exitCode: 127 },
                { name: "partial_error_3", status: "failed", stderr: "Command not found" },
              ],
              model: "claude-sonnet-4-20250514",
            },
          ]),
          result = parseClaudeLog(logWithPartialInfo);
        (expect(result.markdown).toContain("partial_error_1 (failed)"),
          expect(result.markdown).toContain("**Error:** Connection refused"),
          expect(result.markdown).toContain("partial_error_2 (failed)"),
          expect(result.markdown).toContain("**Exit Code:** 127"),
          expect(result.markdown).toContain("partial_error_3 (failed)"),
          expect(result.markdown).toContain("**Stderr:** `Command not found`"),
          expect(result.mcpFailures).toEqual(["partial_error_1", "partial_error_2", "partial_error_3"]));
      }),
      it("should handle exitCode zero for failed servers", () => {
        const logWithExitCodeZero = JSON.stringify([
            {
              type: "system",
              subtype: "init",
              session_id: "test-exitcode-zero",
              tools: ["Bash"],
              mcp_servers: [{ name: "failed_but_exit_zero", status: "failed", error: "Server exited unexpectedly", exitCode: 0 }],
              model: "claude-sonnet-4-20250514",
            },
          ]),
          result = parseClaudeLog(logWithExitCodeZero);
        (expect(result.markdown).toContain("failed_but_exit_zero (failed)"),
          expect(result.markdown).toContain("**Error:** Server exited unexpectedly"),
          expect(result.markdown).toContain("**Exit Code:** 0"),
          expect(result.mcpFailures).toEqual(["failed_but_exit_zero"]));
      }),
      it("should handle unrecognized log format", () => {
        const result = parseClaudeLog("This is not JSON or valid format");
        (expect(result.markdown).toContain("Log format not recognized"), expect(result.mcpFailures).toEqual([]));
      }),
      it("should handle empty log content", () => {
        const result = parseClaudeLog("");
        (expect(result.markdown).toContain("Log format not recognized"), expect(result.mcpFailures).toEqual([]));
      }),
      it("should skip debug lines that look like arrays but aren't JSON", () => {
        const result = parseClaudeLog(
          '[DEBUG] Starting process\n[ERROR] Failed with error\n[INFO] Some information\n[{"type":"system","subtype":"init","session_id":"test-999","tools":["Bash"],"model":"claude-sonnet-4-20250514"}]\n[DEBUG] Process completed'
        );
        (expect(result.markdown).toContain("🚀 Initialization"), expect(result.markdown).toContain("test-999"), expect(result.mcpFailures).toEqual([]));
      }),
      it("should handle tool use with MCP tools", () => {
        const logWithMcpTools = JSON.stringify([
            { type: "system", subtype: "init", session_id: "mcp-test", tools: ["mcp__github__create_issue", "mcp__safe_outputs__missing-tool"], model: "claude-sonnet-4-20250514" },
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
          result = parseClaudeLog(logWithMcpTools);
        (expect(result.markdown).toContain("github::create_issue"), expect(result.markdown).toContain("safe_outputs::missing-tool"), expect(result.mcpFailures).toEqual([]));
      }),
      it("should detect when max-turns limit is hit", () => {
        process.env.GH_AW_MAX_TURNS = "5";
        const logWithMaxTurns = JSON.stringify([
            { type: "system", subtype: "init", session_id: "test-789", tools: ["Bash"], model: "claude-sonnet-4-20250514" },
            { type: "assistant", message: { content: [{ type: "text", text: "Task in progress" }] } },
            { type: "result", total_cost_usd: 0.05, usage: { input_tokens: 500, output_tokens: 200 }, num_turns: 5 },
          ]),
          result = parseClaudeLog(logWithMaxTurns);
        (expect(result.markdown).toContain("**Turns:** 5"), expect(result.maxTurnsHit).toBe(!0), delete process.env.GH_AW_MAX_TURNS);
      }),
      it("should not flag max-turns when turns is less than limit", () => {
        process.env.GH_AW_MAX_TURNS = "10";
        const logBelowMaxTurns = JSON.stringify([
            { type: "system", subtype: "init", session_id: "test-890", tools: ["Bash"], model: "claude-sonnet-4-20250514" },
            { type: "result", total_cost_usd: 0.01, usage: { input_tokens: 100, output_tokens: 50 }, num_turns: 3 },
          ]),
          result = parseClaudeLog(logBelowMaxTurns);
        (expect(result.markdown).toContain("**Turns:** 3"), expect(result.maxTurnsHit).toBe(!1), delete process.env.GH_AW_MAX_TURNS);
      }),
      it("should not flag max-turns when environment variable is not set", () => {
        const logWithoutMaxTurnsEnv = JSON.stringify([
            { type: "system", subtype: "init", session_id: "test-901", tools: ["Bash"], model: "claude-sonnet-4-20250514" },
            { type: "result", total_cost_usd: 0.01, usage: { input_tokens: 100, output_tokens: 50 }, num_turns: 10 },
          ]),
          result = parseClaudeLog(logWithoutMaxTurnsEnv);
        (expect(result.markdown).toContain("**Turns:** 10"), expect(result.maxTurnsHit).toBe(!1));
      }));
  }),
    describe("main function integration", () => {
      (it("should handle valid log file", async () => {
        const validLog = JSON.stringify([
          { type: "system", subtype: "init", session_id: "integration-test", tools: ["Bash"], model: "claude-sonnet-4-20250514" },
          { type: "result", total_cost_usd: 0.001, usage: { input_tokens: 50, output_tokens: 25 }, num_turns: 1 },
        ]);
        (await runScript(validLog), expect(mockCore.summary.addRaw).toHaveBeenCalled(), expect(mockCore.summary.write).toHaveBeenCalled(), expect(mockCore.setFailed).not.toHaveBeenCalled());
        const markdownCall = mockCore.summary.addRaw.mock.calls[0];
        (expect(markdownCall[0]).toContain("```"), expect(markdownCall[0]).toContain("Conversation:"), expect(markdownCall[0]).toContain("Statistics:"), expect(mockCore.info).toHaveBeenCalled());
        const infoCall = mockCore.info.mock.calls.find(call => call[0].includes("=== Claude Execution Summary ==="));
        (expect(infoCall).toBeDefined(), expect(infoCall[0]).toContain("Model: claude-sonnet-4-20250514"));
      }),
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
          (await runScript(logWithFailures),
            expect(mockCore.summary.addRaw).toHaveBeenCalled(),
            expect(mockCore.summary.write).toHaveBeenCalled(),
            expect(mockCore.setFailed).toHaveBeenCalledWith("MCP server(s) failed to launch: broken_server"));
        }),
        it("should call setFailed when max-turns limit is hit", async () => {
          process.env.GH_AW_MAX_TURNS = "3";
          const logHittingMaxTurns = JSON.stringify([
            { type: "system", subtype: "init", session_id: "max-turns-test", tools: ["Bash"], model: "claude-sonnet-4-20250514" },
            { type: "result", total_cost_usd: 0.02, usage: { input_tokens: 200, output_tokens: 100 }, num_turns: 3 },
          ]);
          (await runScript(logHittingMaxTurns),
            expect(mockCore.summary.addRaw).toHaveBeenCalled(),
            expect(mockCore.summary.write).toHaveBeenCalled(),
            expect(mockCore.setFailed).toHaveBeenCalledWith("Agent execution stopped: max-turns limit reached. The agent did not complete its task successfully."),
            delete process.env.GH_AW_MAX_TURNS);
        }),
        it("should handle missing log file", async () => {
          process.env.GH_AW_AGENT_OUTPUT = "/nonexistent/file.log";
          const scriptWithExport = parseClaudeLogScript.replace("main();", "global.testMain = main;");
          (new Function(scriptWithExport)(), await global.testMain(), expect(mockCore.info).toHaveBeenCalledWith("Log path not found: /nonexistent/file.log"), expect(mockCore.setFailed).not.toHaveBeenCalled());
        }),
        it("should handle missing environment variable", async () => {
          delete process.env.GH_AW_AGENT_OUTPUT;
          const scriptWithExport = parseClaudeLogScript.replace("main();", "global.testMain = main;");
          (new Function(scriptWithExport)(), await global.testMain(), expect(mockCore.info).toHaveBeenCalledWith("No agent log file specified"), expect(mockCore.setFailed).not.toHaveBeenCalled());
        }));
    }),
    describe("helper function tests", () => {
      (it("should format bash commands correctly", () => {
        const result = extractParseFunction()(JSON.stringify([{ type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "Bash", input: { command: "echo 'hello world'\n  && ls -la\n  && pwd" } }] } }]));
        expect(result.markdown).toContain("echo 'hello world' && ls -la && pwd");
      }),
        it("should truncate long strings appropriately", () => {
          const parseClaudeLog = extractParseFunction(),
            longCommand = "a".repeat(400),
            result = parseClaudeLog(JSON.stringify([{ type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "Bash", input: { command: longCommand } }] } }]));
          expect(result.markdown).toContain("...");
        }),
        it("should format MCP tool names correctly", () => {
          const result = extractParseFunction()(JSON.stringify([{ type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "mcp__github__create_pull_request", input: { title: "Test PR" } }] } }]));
          expect(result.markdown).toContain("github::create_pull_request");
        }),
        it("should render tool outputs in collapsible HTML details elements", () => {
          const result = extractParseFunction()(
            JSON.stringify([
              { type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "Bash", input: { command: "ls -la", description: "List files" } }] } },
              {
                type: "user",
                message: {
                  content: [
                    {
                      type: "tool_result",
                      tool_use_id: "tool_1",
                      content: "total 48\ndrwxr-xr-x 5 user user 4096 Jan 1 00:00 .\ndrwxr-xr-x 3 user user 4096 Jan 1 00:00 ..\n-rw-r--r-- 1 user user  123 Jan 1 00:00 file1.txt\n-rw-r--r-- 1 user user  456 Jan 1 00:00 file2.txt",
                      is_error: !1,
                    },
                  ],
                },
              },
            ])
          );
          (expect(result.markdown).toContain("<details>"),
            expect(result.markdown).toContain("<summary>"),
            expect(result.markdown).toContain("</summary>"),
            expect(result.markdown).toContain("</details>"),
            expect(result.markdown).toContain("List files: <code>ls -la</code>"),
            expect(result.markdown).toMatch(/~\d+t/),
            expect(result.markdown).toContain("```"),
            expect(result.markdown).toContain("total 48"),
            expect(result.markdown).toContain("file1.txt"));
        }),
        it("should include token estimates in tool call rendering", () => {
          const result = extractParseFunction()(
            JSON.stringify([
              { type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "mcp__github__create_issue", input: { title: "Test Issue", body: "Test description that is long enough to generate some tokens" } }] } },
              { type: "user", message: { content: [{ type: "tool_result", tool_use_id: "tool_1", content: "Issue created successfully with number 123", is_error: !1 }] } },
            ])
          );
          (expect(result.markdown).toMatch(/~\d+t/), expect(result.markdown).toContain("github::create_issue"));
        }),
        it("should include duration when available in tool_result", () => {
          const result = extractParseFunction()(
            JSON.stringify([
              { type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "Bash", input: { command: "sleep 2" } }] } },
              { type: "user", message: { content: [{ type: "tool_result", tool_use_id: "tool_1", content: "", is_error: !1, duration_ms: 2500 }] } },
            ])
          );
          (expect(result.markdown).toMatch(/<code>\d+s<\/code>/), expect(result.markdown).toMatch(/~\d+t/));
        }),
        it("should truncate long tool outputs", () => {
          const parseClaudeLog = extractParseFunction(),
            longOutput = "x".repeat(600),
            result = parseClaudeLog(
              JSON.stringify([
                { type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "Bash", input: { command: "cat large_file.txt" } }] } },
                { type: "user", message: { content: [{ type: "tool_result", tool_use_id: "tool_1", content: longOutput, is_error: !1 }] } },
              ])
            );
          (expect(result.markdown).toContain("..."), expect(result.markdown).not.toContain("x".repeat(600)));
        }),
        it("should show summary only when no tool output", () => {
          const result = extractParseFunction()(
            JSON.stringify([
              { type: "assistant", message: { content: [{ type: "tool_use", id: "tool_1", name: "Bash", input: { command: "mkdir test_dir" } }] } },
              { type: "user", message: { content: [{ type: "tool_result", tool_use_id: "tool_1", content: "", is_error: !1 }] } },
            ])
          );
          (expect(result.markdown).not.toContain("<details>"), expect(result.markdown).toContain("mkdir test_dir"));
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
                model: "claude-sonnet-4",
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
