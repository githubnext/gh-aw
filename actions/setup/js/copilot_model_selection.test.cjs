import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { createRequire } from "module";
import { spawnSync } from "child_process";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);
const { createCopilotModelObserver, recordCopilotModelSelection } = require("./copilot_model_selection.cjs");
const { resolveEffectiveModel, getEffectiveModelLabel } = require("./model_attribution.cjs");
const { detectErrors } = require("./agent_error_patterns.cjs");

describe("Copilot configured versus observed model", () => {
  let root, env, session, infoPath;
  beforeEach(() => {
    root = fs.mkdtempSync(path.join(os.tmpdir(), "copilot-model-selection-"));
    env = { GH_AW_TMP_DIR: root, HOME: root, GH_AW_ENGINE_MODEL: "gemini-3.8-flash" };
    session = path.join(root, ".copilot/session-state/session/events.jsonl");
    infoPath = path.join(root, "aw_info.json");
    fs.mkdirSync(path.dirname(session), { recursive: true });
    fs.writeFileSync(infoPath, JSON.stringify({ model: "gemini-3.8-flash" }));
  });
  afterEach(() => fs.rmSync(root, { recursive: true, force: true }));

  const startup = model => ({ type: "session.model_change", data: { source: "startup", oldModel: "gemini-3.8-flash", newModel: model } });
  function append(event) {
    fs.appendFileSync(session, JSON.stringify(event) + "\n");
  }
  function observer(resolvedModel = "gemini-3.8-flash") {
    return createCopilotModelObserver({ configuredModel: "gemini-3.8-flash", resolvedModel, env });
  }

  it("rejects a startup vendor substitution and preserves configured and effective models", () => {
    const watch = observer();
    append(startup("claude-sonnet-5"));
    const diagnostic = watch.check();
    expect(diagnostic).toContain("Copilot substituted 'claude-sonnet-5'");
    expect(detectErrors(diagnostic).modelNotSupportedError).toBe(true);
    expect(JSON.parse(fs.readFileSync(infoPath, "utf8"))).toMatchObject({
      requested_model: "gemini-3.8-flash",
      model: "claude-sonnet-5",
      model_selection: { status: "substituted", effective_model: "claude-sonnet-5" },
    });
    expect(resolveEffectiveModel(infoPath, "agent", env)).toMatchObject({ model: "claude-sonnet-5", requestedModel: "gemini-3.8-flash" });
    expect(getEffectiveModelLabel(infoPath, "agent", env)).toContain("sonnet");
    expect(watch.check()).toBe(diagnostic);
  });

  it("also detects substitution from main assistant messages without a startup event", () => {
    const watch = observer();
    append({ type: "assistant.message", data: { model: "claude-sonnet-5", content: "Hello" } });
    expect(watch.check()).toContain("Refusing silent model substitution");
  });

  it("accepts resolved aliases, provider prefixes, and reasoning query parameters", () => {
    const watch = observer("copilot/gpt-6.1-sol?effort=high");
    append({ type: "assistant.message", data: { model: "gpt-6.1-sol" } });
    expect(watch.check()).toBe("");
    expect(resolveEffectiveModel(infoPath, "agent", env).model).toBe("gpt-6.1-sol");
  });

  it("ignores cached records, nested agents, quoted event text, and partial lines", () => {
    append(startup("claude-sonnet-5"));
    const watch = observer();
    append({ ...startup("claude-sonnet-5"), agentId: "explore" });
    append({ type: "assistant.message", parentToolCallId: "task-1", data: { model: "claude-sonnet-5" } });
    append({ type: "assistant.message", data: { content: JSON.stringify(startup("claude-sonnet-5")) } });
    fs.appendFileSync(session, JSON.stringify(startup("gemini-3.8-flash")));
    expect(watch.check()).toBe("");
    fs.appendFileSync(session, "\n");
    expect(watch.check()).toBe("");
    expect(JSON.parse(fs.readFileSync(infoPath, "utf8")).model_selection.status).toBe("observed");
  });

  it("does not inspect or change detection-phase models", () => {
    env.GH_AW_PHASE = "detection";
    const watch = observer();
    append(startup("claude-sonnet-5"));
    expect(watch.check()).toBe("");
    expect(JSON.parse(fs.readFileSync(infoPath, "utf8")).model).toBe("gemini-3.8-flash");
    expect(JSON.parse(fs.readFileSync(infoPath, "utf8")).model_selection).toBeUndefined();
  });

  it("does not attribute execution to an unavailable model", () => {
    recordCopilotModelSelection({ status: "unavailable", configured_model: "gemini-3.8-flash", resolved_model: "", effective_model: "" }, env);
    expect(resolveEffectiveModel(infoPath, "agent", env)).toMatchObject({ model: "", requestedModel: "gemini-3.8-flash" });
  });

  it("accepts native auto selection but retains the requested model", () => {
    const watch = observer("auto");
    append(startup("claude-sonnet-5"));
    expect(watch.check()).toBe("");
    expect(JSON.parse(fs.readFileSync(infoPath, "utf8")).model_selection.status).toBe("observed");
  });

  it("checks a final complete event even without a trailing newline", () => {
    const watch = observer();
    fs.writeFileSync(session, JSON.stringify(startup("claude-sonnet-5")));
    expect(watch.check()).toBe("");
    expect(watch.check(true)).toContain("substituted");
  });

  it("preserves authorized proxy fallback attribution", () => {
    const watch = observer();
    append(startup("gemini-3.8-flash"));
    expect(watch.check()).toBe("");
    const info = JSON.parse(fs.readFileSync(infoPath, "utf8"));
    info.fallback_model = "gpt-6.1-sol";
    fs.writeFileSync(infoPath, JSON.stringify(info));
    expect(resolveEffectiveModel(infoPath, "agent", env).model).toBe("gpt-6.1-sol");
  });

  it.each([false, true])("fails an otherwise successful direct CLI run without retrying (long-lived=%s)", longLived => {
    const stub = path.join(root, "cli.cjs");
    fs.writeFileSync(
      stub,
      `
      const fs = require("fs"), path = require("path");
      const file = path.join(process.env.HOME, ".copilot/session-state/session/events.jsonl");
      fs.appendFileSync(file, JSON.stringify({ type: "session.model_change", data: { source: "startup", newModel: "claude-sonnet-5" } }) + "\\n");
      console.log("CLI says success");
      ${longLived ? "setInterval(() => {}, 1000);" : ""}
    `
    );
    const run = spawnSync(process.execPath, [require.resolve("./copilot_harness.cjs"), process.execPath, stub], {
      env: {
        ...process.env,
        ...env,
        COPILOT_MODEL: "gemini-3.8-flash",
        GH_AW_PHASE: "agent",
        GH_AW_MODEL_ROUTING: "0",
        GH_AW_SAFE_OUTPUTS: "",
        AWF_REFLECT_ENABLED: "0",
        GH_AW_COPILOT_SDK: "0",
        GH_AW_HARNESS_INITIAL_DELAY_MS: "1",
        GH_AW_HARNESS_MAX_RETRIES: "1",
      },
      encoding: "utf8",
      timeout: 10000,
    });
    expect(run.error, run.stderr).toBeUndefined();
    expect(run.status).toBe(1);
    expect(run.stderr).toContain("Refusing silent model substitution");
    expect(run.stderr).not.toContain("attempt 2: spawning");
    expect(JSON.parse(fs.readFileSync(infoPath, "utf8")).model).toBe("claude-sonnet-5");
  });
});
