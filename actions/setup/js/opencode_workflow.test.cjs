import { describe, expect, it, vi } from "vitest";
import fs from "node:fs";
import vm from "node:vm";

const source = fs.readFileSync(new URL("../../../.github/workflows/shared/opencode.md", import.meta.url), "utf8");
const req = require;
const reflect = req("./awf_reflect.cjs");
const gateway = req("./convert_gateway_config_shared.cjs");

function engineScript(key) {
  const block = source.split(`    ${key}: |\n`)[1];
  if (!block) throw new Error(`Missing OpenCode ${key}`);
  return block.split(/\n(?= {0,5}\S)/)[0].replace(/^ {6}/gm, "");
}

async function harness({
  env = {},
  endpoints,
  processResult = {},
  events = [{ type: "text", sessionID: "ses", part: { text: "Finished." } }],
  mcp,
  stderrLines = [],
  sessionExport = { info: { id: "ses" }, messages: [{ type: "assistant", id: "default", finish: "stop", time: { completed: 1000 } }] },
  exportError,
  exportOutput,
} = {}) {
  const stdout = [];
  const stderr = [];
  const errors = [];
  class HarnessError extends Error {
    constructor(message, options) {
      super(message, options);
      errors.push(this);
    }
  }
  const process = {
    argv: ["node", "harness", "opencode", "run", "--standalone", "--format", "json", "--agent", "build", "--log-level", "info"],
    env: { OPENCODE_MODEL: "copilot/auto", GH_AW_LLM_PROVIDER: "github", AWF_REFLECT_ENABLED: "1", GH_AW_PROMPT: "/prompt.txt", GITHUB_WORKSPACE: "/workspace", ...env },
    stdout: { write: value => stdout.push(value) },
    stderr: { write: value => stderr.push(value) },
    exitCode: undefined,
  };
  const runProcess = vi.fn(async options => {
    for (const event of events) options.onStdoutLine(JSON.stringify(event));
    for (const line of stderrLines) options.onStderrLine(line);
    return { exitCode: 0, stderr: "", ...processResult };
  });
  const readFileSync = vi.fn(file => {
    if (file === "/prompt.txt") return "Secret prompt; $(do-not-execute) --flag\n";
    if (file === "/mcp.json") return JSON.stringify(mcp ?? { mcp: { servers: {} } });
    throw new Error(`Unexpected file ${file}`);
  });
  const execFile = vi.fn((command, args, options, callback) => callback(exportError, exportOutput ?? JSON.stringify(sessionExport)));
  await vm.runInNewContext(engineScript("harness-script"), {
    process,
    Error: HarnessError,
    require: name => {
      if (name === "fs") return { readFileSync };
      if (name === "child_process") return { execFile };
      if (name === "./process_runner.cjs") return { runProcess };
      if (name === "./parse_opencode_log.cjs") return req(name);
      if (name === "./awf_reflect.cjs")
        return {
          ...reflect,
          fetchAWFReflect: async () => ({
            ok: true,
            reflectData: { endpoints: endpoints ?? [{ configured: true, provider: process.env.GH_AW_LLM_PROVIDER, models_url: "http://127.0.0.1:10002/v1/models" }] },
          }),
        };
      throw new Error(`Unexpected dependency ${name}`);
    },
  });
  const options = runProcess.mock.calls[0]?.[0];
  return { process, options, config: options ? JSON.parse(options.env.OPENCODE_CONFIG_CONTENT) : undefined, stdout: stdout.join(""), stderr: stderr.join(""), readFileSync, execFile, errors };
}

describe("OpenCode shared workflow harness", () => {
  it.each([
    ["copilot/auto", "github", "@opencode/ai/providers/openai-compatible"],
    ["anthropic/claude-sonnet-4-5", "anthropic", "@opencode/ai/providers/anthropic"],
    ["openai/gpt-5", "openai", "@opencode/ai/providers/openai-compatible"],
    ["codex/gpt-5", "openai", "@opencode/ai/providers/openai-compatible"],
  ])("routes %s through its selected provider with an explicit model", async (model, provider, sdk) => {
    const result = await harness({ env: { OPENCODE_MODEL: model, GH_AW_LLM_PROVIDER: provider } });
    expect(result.process.exitCode).toBeUndefined();
    expect(result.config.providers["awf-proxy"]).toMatchObject({ package: sdk, settings: { baseURL: "http://127.0.0.1:10002/v1", apiKey: "awf-proxy" } });
    expect(result.config.experimental.policies).toEqual([
      { action: "provider.use", resource: "*", effect: "deny" },
      { action: "provider.use", resource: "awf-proxy", effect: "allow" },
    ]);
    expect(result.config.update).toBe("disable");
    expect(result.config.permissions).toEqual([{ action: "*", resource: "*", effect: "allow" }]);
    for (const legacy of ["provider", "agent", "permission", "enabled_providers", "disabled_providers", "autoupdate"]) expect(result.config).not.toHaveProperty(legacy);
    expect(result.options.args).toContain("--standalone");
    expect(result.options.args.slice(-2)).toEqual(["--model", `awf-proxy/${model.slice(model.indexOf("/") + 1)}`]);
    expect(result.options.stdin).toContain("Secret prompt");
    expect(result.options.args.join(" ")).not.toContain("Secret prompt");
    expect(result.options.logArgs.join(" ")).not.toContain("Secret prompt");
    expect(result.stderr).not.toContain("Secret prompt");
    expect(result.readFileSync.mock.calls.map(call => call[0])).toEqual(["/prompt.txt"]);
  });

  it("does not select a different configured provider when the requested one is missing", async () => {
    const result = await harness({ endpoints: [{ configured: true, provider: "anthropic", models_url: "http://127.0.0.1:10001/v1/models" }] });
    expect(result.process.exitCode).toBe(1);
    expect(result.options).toBeUndefined();
    expect(result.stderr).toContain("No configured /reflect models endpoint");
  });

  it("preserves remote MCP credentials in the runtime overlay without modifying repository config", async () => {
    const mcp = { github: { type: "remote", url: "http://gateway/mcp/github", headers: { Authorization: "Bearer fixture" }, oauth: false } };
    const result = await harness({ env: { GH_AW_MCP_CONFIG: "/mcp.json" }, mcp: { mcp: { servers: mcp } } });
    expect(result.config.mcp.servers).toEqual(mcp);
    expect(result.readFileSync.mock.calls.map(call => call[0])).toEqual(["/mcp.json", "/prompt.txt"]);
    expect(result.stderr).not.toContain("Bearer fixture");
  });

  it.each([{ mcpServers: {} }, { mcp: {} }, { mcp: { servers: [] } }, { mcp: { servers: null } }, { mcp: null }])("fails explicitly on malformed MCP configuration: %j", async mcp => {
    const result = await harness({ env: { GH_AW_MCP_CONFIG: "/mcp.json" }, mcp });
    expect(result.process.exitCode).toBe(1);
    expect(result.stderr).toContain("must contain an mcp.servers object");
  });

  it("converts configured MCP startup and tool timeout units", async () => {
    const result = await harness({
      env: { GH_AW_MCP_CONFIG: "/mcp.json", GH_AW_STARTUP_TIMEOUT: "240", GH_AW_TOOL_TIMEOUT: "90" },
      mcp: { mcp: { servers: { github: { type: "remote", url: "http://gateway/mcp/github", oauth: false } } } },
    });
    expect(result.config.mcp.timeout).toEqual({ startup: 240000, catalog: 240000, execution: 90000 });
    expect(result.config.mcp.servers.github).not.toHaveProperty("timeout");
    expect(result.config.experimental).not.toHaveProperty("mcp_timeout");
    const nativeOnly = await harness({ env: { GH_AW_STARTUP_TIMEOUT: "240", GH_AW_TOOL_TIMEOUT: "90" } });
    expect(nativeOnly.config.mcp.timeout).toEqual(result.config.mcp.timeout);
    const invalid = await harness({ env: { GH_AW_TOOL_TIMEOUT: "not-a-number" } });
    expect(invalid.process.exitCode).toBe(1);
    expect(invalid.stderr).toContain("positive timeout in seconds");
  });

  it("removes inherited PWD so OpenCode uses the configured engine working directory", async () => {
    const result = await harness({ env: { PWD: "/wrong/inherited", GH_AW_ENGINE_CWD: "/workspace/project" } });
    expect(result.options.env).not.toHaveProperty("PWD");
    expect(result.execFile.mock.calls[0][2]).toMatchObject({ cwd: "/workspace/project", env: result.options.env });
  });

  it.each(["", "copilot", "/auto", "copilot/"])("rejects malformed model %s", async model => {
    const result = await harness({ env: { OPENCODE_MODEL: model } });
    expect(result.process.exitCode).toBe(1);
    expect(result.options).toBeUndefined();
    expect(result.stderr).toContain("provider/model format");
  });

  it.each(["0", "-1", "bad", "1.5", "Infinity"])("rejects invalid max-turns %s", async maxTurns => {
    const result = await harness({ env: { GH_AW_MAX_TURNS: maxTurns } });
    expect(result.process.exitCode).toBe(1);
    expect(result.stderr).toContain("positive integer");
  });

  it("enforces max-turns only when the limit requires another tool-calling turn", async () => {
    const event = { type: "step_finish", sessionID: "ses", part: { id: "finish", reason: "tool-calls" } };
    const result = await harness({ env: { GH_AW_MAX_TURNS: "1" }, events: [event, event] });
    expect(result.config.agents.build.steps).toBe(1);
    expect(result.options.runtimeGuard.shouldTerminate()).toMatchObject({ terminate: true });
    expect(result.process.exitCode).toBe(1);
    expect(JSON.parse(result.stdout)).toMatchObject({ type: "opencode.max_turns", data: { maxTurns: 1 } });
    const completed = await harness({ env: { GH_AW_MAX_TURNS: "1" }, events: [{ ...event, part: { ...event.part, reason: "stop" } }] });
    expect(completed.options.runtimeGuard.shouldTerminate()).toMatchObject({ terminate: false });
    expect(completed.process.exitCode).toBeUndefined();
  });

  it("does not count replayed step completions with a fresh part ID as another turn", async () => {
    const first = { type: "step_finish", sessionID: "ses", part: { id: "projection-one", messageID: "stable-message", reason: "tool-calls" } };
    const replay = { ...first, part: { ...first.part, id: "projection-two" } };
    const result = await harness({ env: { GH_AW_MAX_TURNS: "2" }, events: [first, replay] });
    expect(result.options.runtimeGuard.shouldTerminate()).toMatchObject({ terminate: false });
    expect(result.process.exitCode).toBeUndefined();
  });

  it("preserves a nonzero exit code and rejects JSON errors with zero exit status", async () => {
    expect((await harness({ processResult: { exitCode: 7 } })).process.exitCode).toBe(7);
    const failure = await harness({ events: [{ type: "error", sessionID: "ses", error: { name: "APIError" } }] });
    expect(failure.process.exitCode).toBe(1);
    expect(failure.stderr).toContain("reported an error despite exiting with code 0");
  });

  it("does not accept a successful process status when an MCP server failed to start", async () => {
    const result = await harness({ processResult: { stderr: 'timestamp=2026-10-03T17:28:22.606Z level=WARN run=fixture message="server unavailable" key=github type=remote status=failed\n' } });
    expect(result.process.exitCode).toBe(1);
    expect(result.stderr).toContain("MCP server(s) failed to start: github");
    expect(JSON.parse(result.stdout)).toMatchObject({ type: "opencode.mcp_failure", data: { serverName: "github" } });
  });

  it("retains MCP startup failures even after they leave the bounded classifier tail", async () => {
    const result = await harness({
      stderrLines: ['timestamp=2026-10-03T17:28:22.606Z level=WARN run=fixture message="server unavailable" key=github type=remote status=failed'],
      processResult: { stderr: "only recent unrelated diagnostics remain" },
    });
    expect(result.process.exitCode).toBe(1);
    expect(result.stderr).toContain("MCP server(s) failed to start: github");
  });

  it("supports direct BYOK without pretending Copilot can use a GitHub token as an API key", async () => {
    const result = await harness({ env: { AWF_REFLECT_ENABLED: "0", GH_AW_LLM_PROVIDER: "openai", OPENCODE_MODEL: "openai/gpt-5", OPENAI_API_KEY: "secret-fixture" } });
    expect(result.process.exitCode).toBeUndefined();
    expect(result.config.providers["awf-proxy"].settings).toEqual({ baseURL: "https://api.openai.com/v1", apiKey: "secret-fixture" });
    expect(result.stderr).not.toContain("secret-fixture");
    const github = await harness({ env: { AWF_REFLECT_ENABLED: "0" } });
    expect(github.process.exitCode).toBe(1);
    expect(github.stderr).toContain("Copilot routing requires the AWF sandbox");
  });

  it("rejects v2 MCP startup failures without relying on a nonzero CLI exit", async () => {
    const result = await harness({
      stderrLines: ['timestamp=2026-10-18T15:00:00.000Z level=WARN fiber=#1 message="mcp connect failed" role=server server=github status="{\\"status\\":\\"failed\\",\\"error\\":\\"Connection refused\\"}"'],
    });
    expect(result.process.exitCode).toBe(1);
    expect(result.stderr).toContain("MCP server(s) failed to start: github");
    expect(JSON.parse(result.stdout)).toMatchObject({ type: "opencode.mcp_failure", data: { serverName: "github" } });
  });

  it("does not fail a successful run on a retryable MCP connect warning", async () => {
    const result = await harness({
      stderrLines: [
        'timestamp=2026-10-18T15:00:00.000Z level=WARN fiber=#1 message="mcp connect failed" role=server server=github status="{\\"status\\":\\"connecting\\"}"',
        'timestamp=2026-10-18T15:00:01.000Z level=INFO fiber=#1 message="mcp connected" role=server server=github',
      ],
    });
    expect(result.process.exitCode).toBeUndefined();
    expect(result.stdout).not.toContain("opencode.mcp_failure");
  });

  it("reconciles missing completed steps without publishing the native export's prompt", async () => {
    const result = await harness({
      events: [{ type: "step_finish", sessionID: "ses", part: { id: "live", messageID: "first", reason: "tool-calls" } }],
      sessionExport: {
        info: { id: "ses" },
        messages: [
          { type: "user", text: "Secret exported prompt." },
          { type: "assistant", id: "first", finish: "tool-calls", time: { completed: 1000 }, tokens: { input: 10 } },
          { type: "compaction", id: "compact", time: { completed: 1500 }, summary: "Secret compaction summary.", tokens: { input: 5 }, cost: 0.01, providerState: { secret: "private compaction state" } },
          { type: "assistant", id: "second", finish: "stop", time: { completed: 2000 }, tokens: { input: 20 }, cost: 0.01, providerState: { secret: "private provider state" } },
        ],
      },
    });
    expect(result.process.exitCode).toBeUndefined();
    expect(result.execFile.mock.calls[0].slice(0, 2)).toEqual(["opencode", ["session", "export", "--standalone", "ses"]]);
    expect(result.execFile.mock.calls[0][2]).toMatchObject({ timeout: 60000, maxBuffer: 64 * 1024 * 1024, env: result.options.env });
    const records = result.stdout.trim().split("\n").map(JSON.parse);
    expect(records[0]).toMatchObject({ type: "step_finish", timestamp: 1000, part: { id: "live", messageID: "first", reason: "tool-calls", tokens: { input: 10 } } });
    expect(records[1]).toMatchObject({ type: "opencode.compaction", timestamp: 1500, data: { sessionID: "ses", messageID: "compact", tokens: { input: 5 }, cost: 0.01 } });
    expect(records[2]).toMatchObject({ type: "step_finish", timestamp: 2000, part: { messageID: "second", reason: "stop", tokens: { input: 20 }, cost: 0.01 } });
    for (const secret of ["Secret exported prompt.", "Secret compaction summary.", "private compaction state", "private provider state"]) {
      expect(result.stdout).not.toContain(secret);
      expect(result.stderr).not.toContain(secret);
    }
  });

  it("fails explicitly on session export errors or malformed exports without leaking their content", async () => {
    for (const options of [
      { exportError: new Error("Secret exported prompt.") },
      { exportOutput: "Secret exported prompt." },
      { sessionExport: { info: { id: "wrong" }, messages: [] } },
      { sessionExport: { info: { id: "ses" }, messages: [] } },
    ]) {
      const result = await harness(options);
      expect(result.process.exitCode).toBe(1);
      expect(result.stdout).toBe("");
      expect(result.stderr).toContain("OpenCode session");
      expect(result.stderr).not.toContain("Secret exported prompt.");
    }
    const empty = await harness({ events: [] });
    expect(empty.process.exitCode).toBe(1);
    expect(empty.stderr).toContain("produced no session events");
    expect(empty.execFile).not.toHaveBeenCalled();
  });

  it("rejects the native v2 maximum-step tool error even without step_finish events", async () => {
    const result = await harness({
      env: { GH_AW_MAX_TURNS: "1" },
      events: [{ type: "tool_use", sessionID: "ses", part: { partID: "part", id: "call", tool: "shell", state: { status: "error", error: "Tools are disabled after the maximum agent steps" } } }],
    });
    expect(result.process.exitCode).toBe(1);
    expect(JSON.parse(result.stdout)).toMatchObject({ type: "opencode.max_turns", data: { maxTurns: 1 } });
    expect(result.execFile).not.toHaveBeenCalled();
  });

  it("enforces max-turns from reconciled steps when the CLI omitted live step events", async () => {
    const result = await harness({
      env: { GH_AW_MAX_TURNS: "1" },
      sessionExport: { info: { id: "ses" }, messages: [{ type: "assistant", id: "first", finish: "tool-calls", time: { completed: 1000 } }] },
    });
    expect(result.process.exitCode).toBe(1);
    expect(
      result.stdout
        .trim()
        .split("\n")
        .map(line => JSON.parse(line).type)
    ).toEqual(["step_finish", "opencode.max_turns"]);
  });

  it("enforces max-turns on any recovered tool-calling step at or beyond the limit", async () => {
    const result = await harness({
      env: { GH_AW_MAX_TURNS: "2" },
      sessionExport: {
        info: { id: "ses" },
        messages: [
          { type: "assistant", id: "first", finish: "tool-calls", time: { completed: 1000 } },
          { type: "assistant", id: "second", finish: "unknown", time: { completed: 2000 } },
          { type: "assistant", id: "third", finish: "tool-calls", time: { completed: 3000 } },
          { type: "assistant", id: "fourth", finish: "stop", time: { completed: 4000 } },
        ],
      },
    });
    expect(result.process.exitCode).toBe(1);
    expect(JSON.parse(result.stdout.trim().split("\n").at(-1))).toMatchObject({ type: "opencode.max_turns", data: { maxTurns: 2 } });
  });

  it("merges exported usage into an observed step without duplicating its turn", async () => {
    const result = await harness({
      events: [{ type: "step_finish", sessionID: "ses", part: { id: "live-first", messageID: "first", reason: "tool-calls" } }],
      sessionExport: {
        info: { id: "ses" },
        messages: [
          { type: "assistant", id: "first", finish: "tool-calls", time: { completed: 1000 }, tokens: { input: 10 }, cost: 0.01 },
          { type: "compaction", id: "compact", time: { completed: 1500 }, tokens: { input: 5 }, cost: 0.01 },
          { type: "assistant", id: "second", finish: "stop", time: { completed: 2000 }, tokens: { input: 20 }, cost: 0.02 },
        ],
      },
    });
    const records = result.stdout.trim().split("\n").map(JSON.parse);
    expect(records).toHaveLength(3);
    expect(records[0]).toMatchObject({ type: "step_finish", part: { id: "live-first", messageID: "first", tokens: { input: 10 }, cost: 0.01 } });
    expect(records[1]).toMatchObject({ type: "opencode.compaction", data: { sessionID: "ses", messageID: "compact", tokens: { input: 5 }, cost: 0.01 } });
    const parsed = require("./parse_opencode_log.cjs").parseOpenCodeLog(result.stdout);
    expect(parsed.logEntries.at(-1).data).toMatchObject({ numTurns: 2, totalCostUsd: 0.04, usage: { input_tokens: 35 } });
  });

  it("preserves session-export execution errors as causes without logging their contents", async () => {
    const exportError = new Error("Secret exported prompt.");
    const result = await harness({ exportError });
    expect(result.process.exitCode).toBe(1);
    expect(result.stderr).toContain("Unable to export the OpenCode session for step reconciliation");
    expect(result.stderr).not.toContain("Secret exported prompt.");
    expect(result.errors.find(error => error.message.includes("Unable to export"))?.cause).toBe(exportError);
  });
});

describe("OpenCode MCP adapter", () => {
  it("rewrites URLs, preserves headers, excludes CLI servers, and emits only native fields", () => {
    const writeSecureOutput = vi.fn();
    vm.runInNewContext(engineScript("config-adapter"), {
      require: () => ({
        ...gateway,
        loadGatewayContext: () => ({
          urlPrefix: "http://host.docker.internal:80",
          cliServers: new Set(["safeoutputs"]),
          servers: {
            github: { type: "http", url: "http://gateway:80/mcp/github", tools: ["*"], headers: { Authorization: "Bearer fixture" } },
            safeoutputs: { url: "http://gateway:80/mcp/safeoutputs" },
          },
        }),
        writeSecureOutput,
      }),
    });
    expect(writeSecureOutput.mock.calls[0][0]).toBe("/tmp/gh-aw/opencode-mcp.json");
    expect(JSON.parse(writeSecureOutput.mock.calls[0][1])).toEqual({
      mcp: { servers: { github: { type: "remote", url: "http://host.docker.internal:80/mcp/github", headers: { Authorization: "Bearer fixture" }, disabled: false, codemode: false, oauth: false } } },
    });
  });

  it("does not silently accept a malformed gateway server", () => {
    expect(() =>
      vm.runInNewContext(engineScript("config-adapter"), {
        require: () => ({
          ...gateway,
          loadGatewayContext: () => ({ urlPrefix: "http://gateway", cliServers: new Set(), servers: { bad: {} } }),
        }),
      })
    ).toThrow("requires a gateway HTTP URL");
  });
});
