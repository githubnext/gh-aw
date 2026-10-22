import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "module";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

const require = createRequire(import.meta.url);
const piProvider = require("./pi_provider.cjs");
const { runPiSubagent } = require("./pi_subagent_extension.cjs");
const { runWithCopilotSDK } = require("./copilot_sdk_session.cjs");
const { resolveMultiProviderFromReflect } = require("./awf_reflect.cjs");
const { normalizeRuntimeEvent } = require("./unified_session.cjs");
const { normalizeUnifiedSessionEvent } = require("./unified_session_payload.cjs");
const { createSessionValidator } = require("./scripts/validate_session.cjs");

const ACTIONS_DIR = import.meta.dirname;
const CALLER_PATTERN = /^(?:main|subagent\/[A-Za-z0-9_.:/-]+)$/;

// Selected fields of agent artifact sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl
// from Smoke Pi Routed run 39540473269 (AWF v0.28.70). The rows omit purpose,
// agent_name and caller_source because the harness sent no x-awf-caller header.
const realRecord = fields => ({
  _schema: "token-usage/v0.28.70",
  event: "token_usage",
  provider: "copilot",
  status: 200,
  streaming: true,
  input_tokens: 1200,
  output_tokens: 40,
  cache_read_tokens: 0,
  cache_write_tokens: 0,
  duration_ms: 800,
  ...fields,
});
const PI_CLASSIFIER = realRecord({ timestamp: "2026-10-21T03:49:48.512Z", request_id: "classifier-1", model: "gpt-5.4-mini", path: "/chat/completions", purpose: "routing_classification", caller_source: "awf" });
const PI_MAIN = realRecord({ timestamp: "2026-10-21T03:49:50.709Z", request_id: "e92da47e-0815-427d-bb28-db4d501bcccf", model: "gpt-5.6-luna", path: "/responses", x_initiator: "agent" });
const PI_HAIKU = realRecord({ timestamp: "2026-10-21T03:49:51.799Z", request_id: "511ef3bb-e6e6-43a5-9b28-8a6f4082a147", model: "claude-haiku-4-5-20251001", path: "/v1/messages?beta=true", x_initiator: "agent" });
const PI_MINI = realRecord({ timestamp: "2026-10-21T03:49:58.208Z", request_id: "4d023499-e6e6-43a5-9b28-8a6f4082a147", model: "gpt-5.3-codex", path: "/responses", x_initiator: "agent" });

// Mirrors AWF callerRecordFields/parseCallerHeader (gh-aw-firewall#10222,
// containers/api-proxy/caller-header.js): the trusted AWF purpose wins, a
// valid caller adds purpose/agent_name/caller_source, anything else adds nothing.
function awfRecord(record, header, awfPurpose) {
  const { purpose: _purpose, caller_source: _source, agent_name: _name, ...base } = record;
  if (awfPurpose) return { ...base, purpose: awfPurpose, caller_source: "awf" };
  if (typeof header !== "string" || header.length > 128 || !CALLER_PATTERN.test(header)) return base;
  if (header === "main") return { ...base, purpose: "agent", caller_source: "caller_header" };
  return { ...base, purpose: "subagent", agent_name: header.slice("subagent/".length), caller_source: "caller_header" };
}

/** Collector path: raw api-proxy record -> runtime event -> unified projection. */
function project(record) {
  const event = normalizeUnifiedSessionEvent(normalizeRuntimeEvent("firewall", record));
  expect(event.type).toBe("firewall.token_usage");
  const validate = createSessionValidator("unified").event;
  const published = { ...event, provenance: { component: "firewall", phase: "agent", path: "sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl", index: 0 } };
  expect(validate(published), JSON.stringify(validate.errors)).toBe(true);
  expect(normalizeUnifiedSessionEvent(event)).toEqual(event);
  return event.data;
}

function callerFields(data) {
  return Object.fromEntries(["purpose", "agentName", "callerSource"].filter(key => Object.hasOwn(data, key)).map(key => [key, data[key]]));
}

/** Load pi_provider.cjs as Pi would and capture headers from its provider hook. */
function captureProviderHeaders(initial = {}) {
  const handlers = {};
  piProvider({ registerProvider: () => {}, on: (name, handler) => (handlers[name] = handler) });
  const headers = { authorization: "******", ...initial };
  handlers.before_provider_headers({ type: "before_provider_headers", headers });
  return headers;
}

describe("AWF x-awf-caller attribution", () => {
  let dir;
  beforeEach(() => {
    dir = fs.mkdtempSync(path.join(os.tmpdir(), "awf-caller-"));
    vi.spyOn(process.stderr, "write").mockImplementation(() => true);
    vi.stubEnv("COPILOT_GITHUB_TOKEN", "");
    vi.stubEnv("GITHUB_TOKEN", "");
    vi.stubEnv("ANTHROPIC_API_KEY", "");
    vi.stubEnv("CODEX_API_KEY", "");
    vi.stubEnv("OPENAI_API_KEY", "");
    vi.stubEnv("GH_AW_PI_SUBAGENT_CHILD", "");
    vi.stubEnv("GH_AW_PI_SUBAGENT_NAME", "");
  });
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllEnvs();
    fs.rmSync(dir, { recursive: true, force: true });
  });

  it("pi_main_and_subagent_caller_attribution", async () => {
    // Parent Pi process: the provider hook declares the main agent.
    const parentHeaders = captureProviderHeaders({ "X-AWF-Caller": "subagent/spoofed" });
    expect(parentHeaders).toEqual({ authorization: "******", "x-awf-caller": "main" });

    // Child Pi process: launched by the sub-agent extension with its own
    // environment; a real process loads pi_provider.cjs and fires the hook.
    const executable = path.join(dir, "fake-pi");
    fs.writeFileSync(
      executable,
      `#!/usr/bin/env node
const provider = require(${JSON.stringify(path.join(ACTIONS_DIR, "pi_provider.cjs"))});
const handlers = {};
provider({ registerProvider: () => {}, on: (name, handler) => (handlers[name] = handler) });
process.stdin.resume();
process.stdin.on("end", () => {
  const headers = {};
  handlers.before_provider_headers({ type: "before_provider_headers", headers });
  const event = { type: "message_end", message: { role: "assistant", content: [{ type: "text", text: JSON.stringify(headers) }] } };
  process.stdout.write(JSON.stringify(event) + "\\n");
});
`,
      { mode: 0o700 }
    );
    vi.stubEnv("GH_AW_PI_COMMAND", executable);
    vi.stubEnv("RUNNER_TEMP", "");
    vi.stubEnv("PI_CODING_AGENT_DIR", dir);
    const write = vi.spyOn(process.stdout, "write").mockImplementation(() => true);
    const agent = { name: "haiku-whoami", description: "Who am I", prompt: "Answer.", declaredModel: "claude-haiku-4.5", modelId: "claude-haiku-4.5", model: "aw-gateway/claude-haiku-4.5" };
    const result = await runPiSubagent(agent, "Who are you?", { cwd: dir });
    expect(JSON.parse(result.content[0].text)).toEqual({ "x-awf-caller": "subagent/haiku-whoami" });
    write.mockRestore();

    // AWF records of those requests keep their caller fields through the collector.
    expect(callerFields(project(awfRecord(PI_MAIN, parentHeaders["x-awf-caller"])))).toEqual({ purpose: "agent", callerSource: "caller_header" });
    expect(callerFields(project(awfRecord(PI_HAIKU, "subagent/haiku-whoami")))).toEqual({ purpose: "subagent", agentName: "haiku-whoami", callerSource: "caller_header" });
    expect(callerFields(project(awfRecord(PI_MINI, "subagent/mini-whoami")))).toEqual({ purpose: "subagent", agentName: "mini-whoami", callerSource: "caller_header" });
    // Canonical camelCase input is preserved too.
    expect(callerFields(normalizeUnifiedSessionEvent({ type: "firewall.token_usage", data: { purpose: "subagent", agentName: "haiku-whoami", callerSource: "caller_header" } }).data)).toEqual({
      purpose: "subagent",
      agentName: "haiku-whoami",
      callerSource: "caller_header",
    });
  });

  it("copilot_sdk_main_and_subagent_caller_attribution", async () => {
    // SDK 1.0.16 headers are provider-scoped (providers[].headers) or per turn
    // (MessageOptions.requestHeaders); neither identifies a child request. Main
    // and haiku-whoami share the copilot-completions provider and served model,
    // so any header would also label child traffic. The SDK sends none and
    // relies on native per-agent metrics instead.
    fs.mkdirSync(path.join(dir, ".github/agents"), { recursive: true });
    fs.writeFileSync(path.join(dir, ".github/agents/haiku-whoami.agent.md"), "---\nmodel: claude-haiku-4.5\n---\nSay which model you are.");
    vi.stubEnv("GH_AW_SESSION_STATE_BASE_DIR", dir);
    vi.stubEnv("GH_AW_ENGINE_CWD", dir);
    vi.stubEnv("COPILOT_MODEL", "");
    const config = resolveMultiProviderFromReflect({
      model: "claude-haiku-4.5",
      reflectData: {
        endpoints: [
          {
            provider: "copilot",
            configured: true,
            port: 10002,
            models: ["claude-haiku-4.5", "gpt-5.6-luna"],
            routing_models: [
              { model_id: "claude-haiku-4.5", supported_endpoints: ["/chat/completions", "/v1/messages"] },
              { model_id: "gpt-5.6-luna", supported_endpoints: ["/responses"] },
            ],
          },
        ],
      },
    });
    vi.stubEnv("GH_AW_COPILOT_SDK_MULTI_PROVIDER_JSON", JSON.stringify(config));
    const sendAndWait = vi.fn(async () => ({ data: { content: "done" } }));
    const createSession = vi.fn().mockResolvedValue({ sessionId: "caller-sdk", on: () => {}, sendAndWait, disconnect: async () => {} });
    class FakeCopilotClient {
      start = async () => {};
      stop = async () => {};
      createSession = createSession;
    }
    const result = await runWithCopilotSDK({
      sdkUri: "http://127.0.0.1:3002",
      prompt: "test",
      logger: () => {},
      workingDirectory: dir,
      agentsBaseDir: dir,
      sdkModule: { CopilotClient: FakeCopilotClient, RuntimeConnection: { forUri: () => ({}) }, approveAll: () => "allow" },
    });
    expect(result.exitCode).toBe(0);
    const session = createSession.mock.calls[0][0];
    // Wire APIs, providers and model resolution are unchanged.
    expect(session.providers).toEqual(config.providers);
    expect(session.models).toEqual(config.models);
    expect(session.model).toBe("copilot-completions/claude-haiku-4.5");
    expect(session.customAgents).toEqual([expect.objectContaining({ name: "haiku-whoami", model: "copilot-completions/claude-haiku-4.5" })]);
    expect(JSON.stringify(session).toLowerCase()).not.toContain("x-awf-caller");
    expect(session.providers.every(provider => provider.headers === undefined)).toBe(true);
    for (const call of sendAndWait.mock.calls) expect(JSON.stringify(call).toLowerCase()).not.toContain("x-awf-caller");
  });

  it("unsupported_engine_does_not_mislabel_subagents", () => {
    // A child process whose name cannot form a valid caller sends no header and
    // never inherits the parent's main declaration.
    for (const name of ["", "has space", "a".repeat(120)]) {
      vi.stubEnv("GH_AW_PI_SUBAGENT_CHILD", "1");
      vi.stubEnv("GH_AW_PI_SUBAGENT_NAME", name);
      expect(piProvider.resolveAwfCaller()).toBeNull();
      expect(captureProviderHeaders({ "x-awf-caller": "main" })).toEqual({ authorization: "******" });
    }
    // Boundary-length names are sent unchanged.
    vi.stubEnv("GH_AW_PI_SUBAGENT_NAME", "a".repeat(119));
    expect(captureProviderHeaders()["x-awf-caller"]).toBe(`subagent/${"a".repeat(119)}`);
    // Headerless child requests (SDK, CLI, Claude Code, Codex) stay unattributed.
    const child = project(awfRecord(PI_HAIKU, undefined));
    expect(callerFields(child)).toEqual({});
  });

  it("invalid_or_missing_caller_is_absent_and_backward_compatible", () => {
    // AWF rejects these values (caller-header.test.js) and writes no caller fields.
    for (const header of [undefined, `subagent/${"a".repeat(120)}`, "subagent/file summarizer", "subagent/", "admin", "main, main", "", "main\n", ["main"]]) {
      const data = project(awfRecord(PI_MAIN, header));
      expect(callerFields(data)).toEqual({});
      expect(data).toMatchObject({ model: "gpt-5.6-luna", requestId: PI_MAIN.request_id, path: "/responses", xInitiator: "agent", status: 200 });
    }
    // Older v0.28.70 records without the fields project without crashing.
    for (const record of [PI_MAIN, PI_HAIKU, PI_MINI]) expect(callerFields(project(record))).toEqual({});
    // Valid boundary-length values survive unchanged.
    const boundary = `subagent/${"a".repeat(119)}`;
    expect(boundary).toHaveLength(128);
    expect(project(awfRecord(PI_MINI, boundary))).toMatchObject({ purpose: "subagent", agentName: "a".repeat(119), callerSource: "caller_header" });
    expect(project(awfRecord(PI_MINI, "subagent/a_b.c:d/e-f")).agentName).toBe("a_b.c:d/e-f");
  });

  it("awf_owned_purpose_wins_over_caller", () => {
    // The real classifier record (raw L1) keeps purpose and caller source.
    expect(callerFields(project(PI_CLASSIFIER))).toEqual({ purpose: "routing_classification", callerSource: "awf" });
    // A conflicting caller is ignored by AWF; no child name is fabricated.
    expect(callerFields(project(awfRecord(PI_CLASSIFIER, "subagent/x", "routing_classification")))).toEqual({ purpose: "routing_classification", callerSource: "awf" });
  });
});
