// @ts-check

/**
 * Copilot SDK Driver
 *
 * Uses @github/copilot-sdk to drive a Copilot session against a running headless
 * Copilot CLI server (started by copilot_sdk_sidecar.cjs).  Serializes all SDK
 * session events to a JSONL file so that unified_timeline.cjs can render them.
 *
 * Event mapping:
 *   SDK "user.message"          → JSONL "user.message"
 *   SDK "tool.execution_start"  → JSONL "tool.execution_start"  (toolName, mcpServerName)
 *   SDK "tool.execution_complete" → JSONL "tool.execution_complete" (toolName, mcpServerName, success)
 *   SDK "assistant.message"     → JSONL "assistant.message"     (content)
 *
 * The JSONL file is written to:
 *   /tmp/gh-aw/sandbox/agent/logs/copilot-session-state/{sessionId}/events.jsonl
 * which mirrors the path that copy_copilot_session_state.sh produces and that
 * unified_timeline.cjs reads.
 */

"use strict";

const fs = require("fs");
const path = require("path");
const os = require("os");

// Default timeout for a single sendAndWait call: 10 minutes.
// This is intentionally generous — the headless Copilot CLI has its own internal
// timeouts for individual tool calls and model inference.
// Override via the COPILOT_SDK_SEND_TIMEOUT_MS environment variable.
const SDK_SEND_TIMEOUT_MS_DEFAULT = 10 * 60 * 1000;

/**
 * Extract the prompt text from a resolved args array.
 * Looks for the first occurrence of "-p <value>" or "--prompt <value>".
 *
 * @param {string[]} args - Resolved args (after resolvePromptFileArgs has run).
 * @returns {string | null} The prompt text, or null if not found.
 */
function extractPromptFromArgs(args) {
  for (let i = 0; i < args.length - 1; i++) {
    if (args[i] === "-p" || args[i] === "--prompt") {
      return args[i + 1];
    }
  }
  return null;
}

/**
 * Run a Copilot agentic session using the @github/copilot-sdk.
 *
 * Connects to the already-running headless Copilot CLI server at sdkUri, creates
 * a session, sends the prompt, waits for the session to go idle, and returns a
 * result shape that mirrors what runProcess() returns so that callers can treat
 * both modes uniformly.
 *
 * All SDK events are serialised to a JSONL file under the session state directory
 * so that unified_timeline.cjs can render them in the step summary.
 *
 * @param {{
 *   sdkUri: string,
 *   prompt: string,
 *   logger: (msg: string) => void,
 *   attempt?: number,
 *   sdkModule?: {
 *     CopilotClient: typeof import("@github/copilot-sdk").CopilotClient,
 *     RuntimeConnection: typeof import("@github/copilot-sdk").RuntimeConnection,
 *     approveAll: typeof import("@github/copilot-sdk").approveAll
 *   },
 * }} options
 * @returns {Promise<{exitCode: number, output: string, hasOutput: boolean, durationMs: number}>}
 */
async function runWithCopilotSDK({ sdkUri, prompt, logger, attempt = 0, sdkModule }) {
  // Lazy-require to avoid loading the SDK when it is not needed.
  // The SDK is large and has side-effects on import (worker threads, etc.).
  const { CopilotClient, RuntimeConnection, approveAll } = sdkModule ?? require("@github/copilot-sdk");

  const startTime = Date.now();
  let output = "";
  let hasOutput = false;

  const log = msg => logger(`[sdk-driver] ${msg}`);
  log(`attempt ${attempt + 1}: connecting to Copilot SDK at ${sdkUri}`);

  // Session state directory — mirrors the target path used by unified_timeline.cjs.
  // /tmp/gh-aw/sandbox/agent/logs/copilot-session-state/{sessionId}/events.jsonl
  const sessionStateBase = path.join(os.tmpdir(), "gh-aw", "sandbox", "agent", "logs", "copilot-session-state");

  /** @type {ReadonlyArray<NonNullable<import("@github/copilot-sdk").CopilotClientOptions["logLevel"]>>} */
  const VALID_LOG_LEVELS = ["none", "error", "warning", "info", "debug", "all"];
  const rawLogLevel = process.env.COPILOT_SDK_LOG_LEVEL ?? "";
  /** @type {import("@github/copilot-sdk").CopilotClientOptions["logLevel"]} */
  const logLevel = /** @type {any} */ VALID_LOG_LEVELS.includes(/** @type {any} */ rawLogLevel) ? rawLogLevel : "warning";

  const client = new CopilotClient({
    connection: RuntimeConnection.forUri(sdkUri, {}),
    workingDirectory: process.env.GITHUB_WORKSPACE || process.cwd(),
    logLevel,
  });
  let session = null;
  /** @type {fs.WriteStream | null} */
  let eventsStream = null;
  let clientStarted = false;

  try {
    await client.start();
    clientStarted = true;
    log("client started");

    session = await client.createSession({
      model: process.env.COPILOT_MODEL || undefined,
      onPermissionRequest: approveAll,
    });
    log(`session created: sessionId=${session.sessionId}`);

    // Prepare JSONL output file for this session.
    const sessionDir = path.join(sessionStateBase, session.sessionId);
    fs.mkdirSync(sessionDir, { recursive: true });
    const eventsPath = path.join(sessionDir, "events.jsonl");
    eventsStream = fs.createWriteStream(eventsPath, { flags: "a" });
    log(`serialising SDK events to ${eventsPath}`);

    /**
     * Map from toolCallId → {toolName, mcpServerName} so that tool.execution_complete
     * events (which carry no mcpServerName) can be enriched from the matching start event.
     * @type {Map<string, {toolName: string, mcpServerName: string}>}
     */
    const pendingToolCalls = new Map();

    /**
     * Write one JSONL entry to the events file.
     * Uses the event's own ISO-8601 timestamp when available.
     *
     * @param {string} type
     * @param {object} data
     * @param {string | undefined} [timestamp]
     */
    function writeEvent(type, data, timestamp) {
      const entry = { type, timestamp: timestamp ?? new Date().toISOString(), data };
      eventsStream.write(JSON.stringify(entry) + "\n");
    }

    // Subscribe to all session events and serialise the ones we care about.
    session.on(event => {
      // Skip transient events that are not persisted by the server.
      if (event.ephemeral) return;

      switch (event.type) {
        case "user.message":
          writeEvent("user.message", {}, event.timestamp);
          break;

        case "tool.execution_start": {
          const toolName = event.data?.toolName ?? "unknown";
          const mcpServerName = event.data?.mcpServerName ?? "";
          const toolCallId = event.data?.toolCallId;
          if (toolCallId) {
            pendingToolCalls.set(toolCallId, { toolName, mcpServerName });
          }
          writeEvent("tool.execution_start", { toolName, mcpServerName }, event.timestamp);
          break;
        }

        case "tool.execution_complete": {
          const toolCallId = event.data?.toolCallId;
          // Resolve toolName/mcpServerName from the matching start event when available.
          const pending = toolCallId ? pendingToolCalls.get(toolCallId) : undefined;
          const toolName = pending?.toolName ?? event.data?.toolDescription?.name ?? "unknown";
          const mcpServerName = pending?.mcpServerName ?? "";
          if (toolCallId) pendingToolCalls.delete(toolCallId);
          const success = event.data?.success ?? !event.data?.error;
          writeEvent("tool.execution_complete", { toolName, mcpServerName, success }, event.timestamp);
          break;
        }

        case "assistant.message": {
          const content = event.data?.content ?? "";
          if (content) {
            hasOutput = true;
            output += content;
          }
          writeEvent("assistant.message", { content }, event.timestamp);
          break;
        }

        default:
          // Other event types are not consumed by unified_timeline.cjs; skip them.
          break;
      }
    });

    log("sending prompt...");
    const sendTimeoutMs = Number(process.env.COPILOT_SDK_SEND_TIMEOUT_MS) || SDK_SEND_TIMEOUT_MS_DEFAULT;
    const result = await session.sendAndWait({ prompt }, sendTimeoutMs);

    // sendAndWait returns the last assistant.message event; capture its content
    // as a fallback in case the on() handler missed it.
    if (result && !hasOutput) {
      const content = result.data?.content ?? "";
      if (content) {
        output = content;
        hasOutput = true;
      }
    }

    const durationMs = Date.now() - startTime;
    log(`session completed: hasOutput=${hasOutput} durationMs=${durationMs}`);

    return { exitCode: 0, output, hasOutput, durationMs };
  } catch (err) {
    const durationMs = Date.now() - startTime;
    log(`error: ${err instanceof Error ? err.message : String(err)}`);
    return {
      exitCode: 1,
      output: err instanceof Error ? err.message : String(err),
      hasOutput: false,
      durationMs,
    };
  } finally {
    if (eventsStream) {
      await new Promise(resolve => eventsStream.end(resolve));
    }
    if (session) {
      try {
        await session.disconnect();
      } catch {
        // best-effort cleanup
      }
    }
    if (clientStarted) {
      try {
        await client.stop();
      } catch {
        // best-effort cleanup
      }
    }
  }
}

module.exports = { extractPromptFromArgs, runWithCopilotSDK };
