import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);
const { createCopilotSDKModelObserver, readCopilotModelSubstitution, persistCopilotModelSelection } = require("./copilot_model_selection.cjs");
const { runWithCopilotSDK } = require("./copilot_sdk_session.cjs");
const { qualifyModelForMultiProvider } = require("./copilot_sdk_multi_provider.cjs");
const { detectErrors } = require("./agent_error_patterns.cjs");

// Captured, credential-free model-bearing records from Smoke Copilot SDK Routed run 39529495361
// (native envelopes and main/sub-agent scope preserved) and the CLI control run 39529502581.
const fixtures = path.join(path.dirname(require.resolve("./copilot_sdk_session.cjs")), "fixtures");
const sdkDir = path.join(fixtures, "copilot-sdk-model-observation-39529495361");
const cliDir = path.join(fixtures, "copilot-cli-model-observation-39529502581");
const readJsonl = file =>
  fs
    .readFileSync(file, "utf8")
    .trim()
    .split("\n")
    .map(line => JSON.parse(line));
const sdkEvents = () => readJsonl(path.join(sdkDir, "events.jsonl"));
const providerConfig = () => JSON.parse(fs.readFileSync(path.join(sdkDir, "multi-provider.json"), "utf8"));
const routingSelection = JSON.parse(fs.readFileSync(path.join(sdkDir, "model-routing-selection.json"), "utf8"));
const SESSION_ID = "aee0ab8e-b091-4008-864d-4ca73e3f95c7";
const mainEvents = () => sdkEvents().filter(event => event.agentId === undefined);
const haikuEvents = () => sdkEvents().filter(event => event.agentId !== undefined);
const startEvent = () => sdkEvents().find(event => event.type === "session.start");
const shutdownEvent = () => sdkEvents().find(event => event.type === "session.shutdown");
// Derived test fixture (not a substitution observed in the supplied runs): the captured Haiku
// model change with its sub-agent scope removed, i.e. the main session switching model.
const derivedMainModelChange = () => {
  const { agentId: _agentId, ...event } = sdkEvents().find(candidate => candidate.type === "session.model_change");
  return event;
};

describe("Copilot SDK main-session model observation", () => {
  let root, selectionPath;
  beforeEach(() => {
    root = fs.mkdtempSync(path.join(os.tmpdir(), "sdk-model-observation-"));
    selectionPath = path.join(root, "sandbox/agent/logs/copilot-model-selection.json");
    vi.stubEnv("GH_AW_TMP_DIR", root);
    vi.stubEnv("HOME", root);
    vi.stubEnv("COPILOT_HOME", path.join(root, ".copilot"));
    vi.stubEnv("GH_AW_PHASE", "agent");
    vi.stubEnv("GH_AW_ENGINE_ID", "copilot");
    vi.stubEnv("GH_AW_SESSION_STATE_BASE_DIR", path.join(root, "driver-session-state"));
    vi.stubEnv("GH_AW_ENGINE_CWD", root);
    vi.stubEnv("GH_AW_MODEL_FALLBACK", "");
    vi.stubEnv("GH_AW_MODEL_ROUTING", "");
    vi.stubEnv("GH_AW_COPILOT_SDK_SERVER_ARGS", "[]");
    vi.stubEnv("GH_AW_COPILOT_SDK_MULTI_PROVIDER_JSON", JSON.stringify(providerConfig()));
    vi.spyOn(process.stderr, "write").mockImplementation(() => true);
  });
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.restoreAllMocks();
    fs.rmSync(root, { recursive: true, force: true });
  });

  const selection = () => JSON.parse(fs.readFileSync(selectionPath, "utf8"));
  const observer = (configuredModel, expectedModel) => createCopilotSDKModelObserver({ configuredModel, expectedModel, providerConfig: providerConfig(), sessionDirs: [path.join(root, "native")] });

  /**
   * Run the SDK session runner against a fake SDK session that replays native events.
   * @param {{ replay: (handler: (event: any) => void) => any, onDisconnect?: (handler: (event: any) => void) => void }} options
   */
  async function runSession({ replay, onDisconnect = () => {} }) {
    let handler = () => {};
    const disconnect = vi.fn(async () => onDisconnect(handler));
    const createSession = vi.fn(async () => ({
      sessionId: SESSION_ID,
      on: callback => {
        handler = callback;
      },
      sendAndWait: async () => replay(handler),
      disconnect,
    }));
    class FakeCopilotClient {
      start = async () => {};
      stop = async () => {};
      createSession = createSession;
    }
    const logger = vi.fn();
    const result = await runWithCopilotSDK({
      sdkUri: "http://127.0.0.1:3002",
      prompt: "test",
      logger,
      workingDirectory: root,
      agentsBaseDir: root,
      sdkModule: { CopilotClient: FakeCopilotClient, RuntimeConnection: { forUri: () => ({}) }, approveAll: () => "allow" },
    });
    return { result, disconnect, createSession, logger };
  }
  const finalMessage = () => {
    const message = mainEvents()
      .filter(event => event.type === "assistant.message")
      .at(-1);
    return { ...message, data: { ...message.data, content: "Done." } };
  };

  it("sdk_main_model_matches_records_observed", async () => {
    vi.stubEnv("COPILOT_MODEL", "gpt-5.6-luna");
    vi.stubEnv("GH_AW_COPILOT_CONFIGURED_MODEL", "gpt-5.6-luna");
    const { result, createSession } = await runSession({
      replay: handler => {
        for (const event of mainEvents().filter(event => event.type !== "session.shutdown")) handler(event);
        return finalMessage();
      },
    });
    expect(createSession.mock.calls[0][0].model).toBe("copilot-responses/gpt-5.6-luna");
    expect(result.exitCode).toBe(0);
    expect(selection()).toEqual({ status: "observed", configured_model: "gpt-5.6-luna", resolved_model: "gpt-5.6-luna", effective_model: "gpt-5.6-luna" });
    expect(readCopilotModelSubstitution()).toBe("");
    // Same schema and outcome as the CLI control run's selection evidence.
    expect(Object.keys(selection()).sort()).toEqual(Object.keys(JSON.parse(fs.readFileSync(path.join(cliDir, "copilot-model-selection.json"), "utf8"))).sort());
  });

  it("sdk_provider_qualified_model_matches_bare_events", () => {
    const watch = observer("copilot-responses/gpt-5.6-luna", "copilot-responses/gpt-5.6-luna");
    watch.setSessionId(SESSION_ID);
    expect(startEvent().data.selectedModel).toBe("copilot-responses/gpt-5.6-luna");
    for (const event of mainEvents()) {
      expect(watch.observe(event)).toBe("");
      expect(selection().status).toBe("observed");
    }
    expect(shutdownEvent().data.currentModel).toBe("gpt-5.6-luna");
    expect(selection()).toMatchObject({ resolved_model: "gpt-5.6-luna", effective_model: "gpt-5.6-luna" });
  });

  it("sdk_main_model_switch_is_substituted_and_stops", async () => {
    vi.stubEnv("COPILOT_MODEL", "gpt-5.6-luna");
    vi.stubEnv("GH_AW_COPILOT_CONFIGURED_MODEL", "gpt-5.6-luna");
    const [start, firstMessage] = mainEvents();
    let afterSwitch = 0;
    const { result, disconnect } = await runSession({
      replay: handler => {
        handler(start);
        handler(firstMessage);
        handler(derivedMainModelChange());
        // A later matching main-session record cannot clear the mismatch.
        handler(firstMessage);
        afterSwitch++;
        return new Promise(() => {});
      },
    });
    expect(afterSwitch).toBe(1);
    expect(result.exitCode).toBe(1);
    expect(disconnect).toHaveBeenCalled();
    expect(result.output).toContain("Copilot substituted 'copilot-completions/claude-haiku-4.5'. Refusing silent model substitution.");
    expect(selection()).toEqual({ status: "substituted", configured_model: "gpt-5.6-luna", resolved_model: "gpt-5.6-luna", effective_model: "copilot-completions/claude-haiku-4.5" });
    // The harness reads the recorded refusal, fails the attempt as copilot_model_verification
    // (nonzero, no retry) and reports it through the CLI substitution classification.
    const diagnostic = readCopilotModelSubstitution();
    expect(result.output).toContain(diagnostic);
    expect(detectErrors(diagnostic).modelNotSupportedError).toBe(true);
    const harness = fs.readFileSync(require.resolve("./copilot_harness.cjs"), "utf8");
    expect(harness).toContain("copilotSDKMode ? readCopilotModelSubstitution() : modelObserver.check(true)");
    expect(harness).toMatch(/if \(result\.runtimeGuardReason === "copilot_model_verification"\) return \{ action: "stop" \};/);
    const info = path.join(root, "aw_info.json");
    fs.writeFileSync(info, "{}");
    persistCopilotModelSelection();
    expect(JSON.parse(fs.readFileSync(info, "utf8"))).toMatchObject({ model: "copilot-completions/claude-haiku-4.5", model_selection: { status: "substituted" } });
  });

  it.each([
    ["matching", "gpt-5.6-luna", "observed", 0],
    // Derived test fixture: the main session served a different model than the router selected.
    ["derived different", "gpt-5.6-sol", "substituted", 1],
  ])("sdk_routed_model_is_checked_against_selection_wire_model (%s main model)", async (_name, mainModel, status, exitCode) => {
    expect(routingSelection).toMatchObject({ stage: "selection", wire_model: "gpt-5.6-luna" });
    const routedModel = qualifyModelForMultiProvider(routingSelection.wire_model, providerConfig());
    vi.stubEnv("GH_AW_MODEL_ROUTING", "1");
    vi.stubEnv("COPILOT_MODEL", routedModel);
    vi.stubEnv("GH_AW_COPILOT_CONFIGURED_MODEL", "auto");
    const { result } = await runSession({
      replay: handler => {
        handler(startEvent());
        for (const event of mainEvents().filter(event => event.type === "assistant.message")) handler({ ...event, data: { ...event.data, model: mainModel } });
        return { ...finalMessage(), data: { ...finalMessage().data, model: mainModel } };
      },
    });
    expect(result.exitCode).toBe(exitCode);
    expect(selection()).toEqual({ status, configured_model: "auto", resolved_model: "gpt-5.6-luna", effective_model: mainModel });
  });

  it("sdk_auto_does_not_report_substitution", () => {
    const watch = createCopilotSDKModelObserver({ configuredModel: "auto", expectedModel: "auto", providerConfig: null, sessionDirs: [] });
    for (const event of mainEvents()) expect(watch.observe(event)).toBe("");
    expect(watch.finalize(SESSION_ID)).toBe("");
    expect(selection()).toEqual({ status: "observed", configured_model: "auto", resolved_model: "auto", effective_model: "gpt-5.6-luna" });
    expect(readCopilotModelSubstitution()).toBe("");
  });

  it("sdk_subagent_model_events_do_not_change_main_observation", () => {
    expect(haikuEvents().map(event => [event.type, event.data.newModel || event.data.model])).toEqual([
      ["session.model_change", "copilot-completions/claude-haiku-4.5"],
      ["assistant.message", "claude-haiku-4.5"],
    ]);
    const native = path.join(root, "native", SESSION_ID, "events.jsonl");
    fs.mkdirSync(path.dirname(native), { recursive: true });
    fs.writeFileSync(
      native,
      sdkEvents()
        .map(event => JSON.stringify(event))
        .join("\n") + "\n"
    );
    const watch = observer("auto", "copilot-responses/gpt-5.6-luna");
    for (const event of haikuEvents()) expect(watch.observe(event)).toBe("");
    expect(fs.existsSync(selectionPath)).toBe(false);
    expect(watch.finalize(SESSION_ID)).toBe("");
    expect(selection()).toEqual({ status: "observed", configured_model: "auto", resolved_model: "gpt-5.6-luna", effective_model: "gpt-5.6-luna" });
  });

  it.each(["listener", "native session file"])("sdk_shutdown_mismatch_cannot_be_reported_as_success (%s)", async delivery => {
    vi.stubEnv("COPILOT_MODEL", "gpt-5.6-luna");
    vi.stubEnv("GH_AW_COPILOT_CONFIGURED_MODEL", "gpt-5.6-luna");
    // Derived test fixture: the captured shutdown envelope reporting a different current model.
    const shutdown = { ...shutdownEvent(), data: { ...shutdownEvent().data, currentModel: "gpt-5.6-sol" } };
    const { result, logger } = await runSession({
      replay: handler => {
        for (const event of mainEvents().filter(event => event.type !== "session.shutdown")) handler(event);
        return finalMessage();
      },
      onDisconnect: handler => {
        if (delivery === "listener") return handler(shutdown);
        const native = path.join(root, "sandbox/agent/session-state", SESSION_ID, "events.jsonl");
        fs.mkdirSync(path.dirname(native), { recursive: true });
        fs.writeFileSync(native, [...mainEvents().slice(0, -1), shutdown].map(event => JSON.stringify(event)).join("\n") + "\n");
      },
    });
    expect(result.exitCode).toBe(1);
    expect(result.output).toContain("Copilot substituted 'gpt-5.6-sol'");
    expect(selection()).toMatchObject({ status: "substituted", effective_model: "gpt-5.6-sol" });
    expect(readCopilotModelSubstitution()).toContain("Refusing silent model substitution");
    expect(logger.mock.calls.flat().some(message => message.includes("Refusing silent model substitution"))).toBe(true);
  });
});
