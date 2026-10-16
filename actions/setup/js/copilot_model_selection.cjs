// @ts-check
"use strict";

const { iterateJsonl } = require("./jsonl_helpers.cjs");
const fs = require("fs");
const path = require("path");
const os = require("os");
const { validateModelIdentifier } = require("./model_attribution.cjs");
const { normalizeForCopilotCLI, splitModelIdentifier } = require("./resolve_model_alias.cjs");
const { copilotSessionContext } = require("./copilot_session.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

function modelID(value) {
  return validateModelIdentifier(normalizeForCopilotCLI(splitModelIdentifier(String(value || "")).base));
}

function recordCopilotModelSelection(selection, env = process.env) {
  if (env.GH_AW_PHASE && env.GH_AW_PHASE !== "agent") return;
  const file = path.join(env.GH_AW_TMP_DIR || "/tmp/gh-aw", "aw_info.json");
  try {
    const info = fs.existsSync(file) ? JSON.parse(fs.readFileSync(file, "utf8")) : {};
    info.requested_model ||= selection.configured_model;
    info.model = selection.effective_model;
    info.model_selection = selection;
    fs.mkdirSync(path.dirname(file), { recursive: true });
    fs.writeFileSync(file, JSON.stringify(info, null, 2));
  } catch (error) {
    throw new Error(`Cannot record Copilot model selection: ${getErrorMessage(error)}`, { cause: error });
  }
}

/**
 * Observe only the invocation's exact session and newly appended records, not other sessions or subagents.
 * Native CLI startup fallback is not an authorized AWF fallback.
 * @param {{ configuredModel: string, resolvedModel: string, sessionId?: string, env?: NodeJS.ProcessEnv, sessionDirs?: string[], logger?: (message: string) => void }} options
 */
function createCopilotModelObserver({ configuredModel, resolvedModel, sessionId, env = process.env, sessionDirs, logger = () => {} }) {
  const root = env.GH_AW_TMP_DIR || "/tmp/gh-aw";
  const directories = sessionDirs || [
    path.join(env.COPILOT_HOME || path.join(env.HOME || os.homedir(), ".copilot"), "session-state"),
    path.join(root, "sandbox/agent/session-state"),
    path.join(root, "sandbox/agent/logs/copilot-session-state"),
  ];
  /** @type {Map<string, number>} */
  const offsets = new Map();
  /** @type {Map<string, string>} */
  const partial = new Map();
  let diagnostic = "";
  let observedModel = "";
  const expected = modelID(resolvedModel);
  if (!expected || (env.GH_AW_PHASE && env.GH_AW_PHASE !== "agent")) return { check: () => "" };
  if (!sessionId || !/^[a-z0-9_-]+$/i.test(sessionId)) throw new Error("Copilot model verification requires an exact invocation session ID");
  const ownedSessionId = sessionId;

  function eventFiles() {
    const files = new Set();
    for (const directory of directories) {
      const file = path.join(directory, ownedSessionId, "events.jsonl");
      try {
        fs.statSync(file);
      } catch (error) {
        if (error !== null && typeof error === "object" && "code" in error && error.code === "ENOENT") continue;
        throw error;
      }
      files.add(file);
    }
    return files;
  }

  try {
    for (const file of eventFiles()) offsets.set(file, fs.statSync(file).size);
  } catch (error) {
    throw new Error(`Cannot initialize Copilot model verification: ${getErrorMessage(error)}`, { cause: error });
  }

  function check(final = false) {
    try {
      if (diagnostic) return diagnostic;
      for (const file of eventFiles()) {
        const size = fs.statSync(file).size;
        let offset = offsets.get(file) || 0;
        if (size < offset) {
          offset = 0;
          partial.delete(file);
        }
        const content = Buffer.alloc(size - offset);
        const fd = fs.openSync(file, "r");
        let bytesRead;
        try {
          bytesRead = fs.readSync(fd, content, 0, content.length, offset);
        } finally {
          fs.closeSync(fd);
        }
        const text = (partial.get(file) || "") + content.subarray(0, bytesRead).toString("utf8");
        offsets.set(file, offset + bytesRead);
        const lines = text.split("\n");
        partial.set(file, final ? "" : lines.pop() || "");
        for (const line of lines) {
          let event;
          try {
            event = iterateJsonl([line], { strict: true }).next().value;
          } catch {
            continue; // Native logs can contain incomplete or non-JSON records.
          }
          const context = copilotSessionContext(event);
          if (context.agentId !== undefined || context.parentToolUseId || (context.sessionId !== undefined && context.sessionId !== ownedSessionId)) continue;
          const model = modelID(event.type === "session.model_change" && event.data?.source === "startup" ? event.data.newModel : event.type === "assistant.message" ? event.data?.model : "");
          if (!model || model === observedModel) continue;
          observedModel = model;
          const substituted = expected.toLowerCase() !== "auto" && model.toLowerCase() !== expected.toLowerCase();
          recordCopilotModelSelection({ status: substituted ? "substituted" : "observed", configured_model: configuredModel, resolved_model: resolvedModel, effective_model: model }, env);
          if (substituted) {
            diagnostic = `Model '${expected}' is not available as configured: Copilot substituted '${model}'. Refusing silent model substitution. Select a model offered to this workflow's agentic-workflows integration; do not change the integration ID to bypass model access policy.`;
            logger(diagnostic);
            return diagnostic;
          }
        }
      }
      return "";
    } catch (error) {
      diagnostic = `Copilot model verification failed: ${getErrorMessage(error)}. Refusing to report an unverified run as successful.`;
      logger(diagnostic);
      return diagnostic;
    }
  }
  return { check };
}

module.exports = { createCopilotModelObserver, recordCopilotModelSelection };
