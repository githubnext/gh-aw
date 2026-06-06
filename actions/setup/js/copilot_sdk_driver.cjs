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
 *
 * When run as a standalone program (require.main === module), the driver reads
 * configuration from environment variables and connects to the sidecar server
 * that has already been started by copilot_harness.cjs:
 *
 *   GH_AW_PROMPT              — path to the prompt file
 *   COPILOT_SDK_URI            — SDK server URI (set by the harness)
 *   COPILOT_CONNECTION_TOKEN   — shared secret for the SDK session (set by the harness)
 *   COPILOT_MODEL              — model override (optional)
 *
 * The sidecar is started and stopped by the harness; the driver only opens a
 * client connection, runs the session, and exits.  This makes the driver a
 * simple client extension that can be started by the harness like any other
 * command, while serving as a sample showing how to create a Copilot SDK driver
 * extension in agentic-workflows.
 */

"use strict";

const fs = require("fs");
const path = require("path");
const os = require("os");
const { minimatch } = require("minimatch");

// Default timeout for a single sendAndWait call: 10 minutes.
// This is intentionally generous — the headless Copilot CLI has its own internal
// timeouts for individual tool calls and model inference.
// Override via the COPILOT_SDK_SEND_TIMEOUT_MS environment variable.
const SDK_SEND_TIMEOUT_MS_DEFAULT = 10 * 60 * 1000;
const MAX_TOOL_DENIALS_DEFAULT = 5;

/**
 * @typedef {{
 *   allowAllTools?: boolean,
 *   allowedTools?: string[],
 * }} CopilotSDKPermissionConfig
 */

/**
 * Parse a strict positive integer from a number or string.
 * Returns undefined when the input is not a whole positive integer.
 *
 * @param {unknown} value
 * @returns {number | undefined}
 */
function parseStrictPositiveInteger(value) {
  if (typeof value === "number" && Number.isSafeInteger(value) && value > 0) {
    return value;
  }
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (/^\d+$/.test(trimmed)) {
      const parsed = Number.parseInt(trimmed, 10);
      if (Number.isSafeInteger(parsed) && parsed > 0) {
        return parsed;
      }
    }
  }
  return undefined;
}

/**
 * Parse max tool denials threshold from input.
 * Falls back to MAX_TOOL_DENIALS_DEFAULT when unset/invalid.
 *
 * @param {unknown} value
 * @returns {number}
 */
function parseMaxToolDenialsLimit(value) {
  return parseStrictPositiveInteger(value) ?? MAX_TOOL_DENIALS_DEFAULT;
}

/**
 * Read a positive integer from an environment variable with fallback.
 *
 * @param {string} key
 * @param {number} fallback
 * @returns {number}
 */
function getEnvPositiveIntOrDefault(key, fallback) {
  return parseStrictPositiveInteger(process.env[key]) ?? fallback;
}

/**
 * @typedef {{
 *   info?: (message: string) => void,
 *   warning?: (message: string) => void,
 * }} CopilotSDKCoreLogger
 */

/**
 * Create a compact, human-readable permission-request summary for diagnostics.
 * Examples: shell(git status), mcp(github.get_file_contents), url(https://example.com).
 *
 * @param {import("@github/copilot-sdk").PermissionRequest} request
 * @returns {string}
 */
function summarizePermissionRequest(request) {
  switch (request.kind) {
    case "shell":
      return `shell(${String(request.fullCommandText || "").trim() || "unknown"})`;
    case "mcp":
      return `mcp(${request.serverName || "unknown"}.${request.toolName || "unknown"})`;
    case "url":
      return `url(${request.url || "unknown"})`;
    case "write":
      return `write(${request.fileName || "unknown"})`;
    case "read":
      return `read(${request.path || "unknown"})`;
    case "custom-tool":
      return `custom-tool(${request.toolName || "unknown"})`;
    default:
      return request.kind;
  }
}

/**
 * @param {CopilotSDKCoreLogger | undefined} coreLogger
 * @param {(msg: string) => void} logger
 * @param {import("@github/copilot-sdk").PermissionRequest} request
 */
function logPermissionDenied(coreLogger, logger, request) {
  const requestSummary = summarizePermissionRequest(request);
  logger(`permission denied by workflow tool permissions: ${requestSummary}`);
  if (coreLogger?.info) {
    coreLogger.info(`Copilot SDK permission denied: ${requestSummary}`);
  }
  if (coreLogger?.warning) {
    coreLogger.warning(`Copilot SDK permission denied by workflow tool permissions: ${requestSummary}`);
  }
}

/**
 * @param {string} value
 * @returns {string}
 */
function normalizePermissionPath(value) {
  return (
    String(value || "")
      .trim()
      .replace(/\\/g, "/")
      .replace(/\/+$/, "") || "/"
  );
}

/**
 * @param {string} shellRule
 * @returns {string[]}
 */
function extractReadablePathPatternsFromShellRule(shellRule) {
  const trimmed = String(shellRule || "").trim();
  if (!trimmed) return [];

  if (trimmed.startsWith("cat ")) {
    return [trimmed.slice("cat ".length).trim()];
  }

  const xargsCatMatch = trimmed.match(/^xargs\s+-a\s+(\S+)\s+cat(?:\s|$)/);
  if (xargsCatMatch) {
    return [xargsCatMatch[1]];
  }

  const lsMatch = trimmed.match(/^ls\s+(\S+)(?:\s|$)/);
  if (lsMatch) {
    return [lsMatch[1]];
  }

  return [];
}

/**
 * @param {string | undefined} requestedPath
 * @param {string[]} allowedPathPatterns
 * @returns {boolean}
 */
function isReadPathAllowedByShellRules(requestedPath, allowedPathPatterns) {
  if (typeof requestedPath !== "string" || requestedPath.trim().length === 0) {
    return false;
  }

  const normalizedRequestedPath = normalizePermissionPath(requestedPath);

  return allowedPathPatterns.some(pattern => {
    const normalizedPattern = normalizePermissionPath(pattern);
    if (normalizedRequestedPath === normalizedPattern) {
      return true;
    }
    return minimatch(normalizedRequestedPath, normalizedPattern, { dot: true });
  });
}

/**
 * Build an SDK on-permission handler from Copilot CLI allow-tool rules.
 * A handler is always returned so session creation consistently wires explicit
 * permission behavior derived from configuration input.
 *
 * @param {CopilotSDKPermissionConfig | undefined} permissionConfig
 * @param {import("@github/copilot-sdk").PermissionHandler} approveAll
 * @param {{coreLogger?: CopilotSDKCoreLogger, logger?: (msg: string) => void, onDenied?: (requestSummary: string) => void}=} logOptions
 * @returns {import("@github/copilot-sdk").PermissionHandler}
 */
function buildCopilotSDKPermissionHandler(permissionConfig, approveAll, logOptions) {
  const logger = logOptions?.logger ?? (() => {});

  const allowAll = permissionConfig?.allowAllTools === true;
  const allowedTools = Array.isArray(permissionConfig?.allowedTools) ? permissionConfig.allowedTools : [];
  const normalizedAllowedTools = allowedTools
    .filter(tool => typeof tool === "string")
    .map(tool => tool.trim())
    .filter(tool => tool.length > 0);
  const allowedToolEntries = new Set(normalizedAllowedTools);

  // Keep explicit allow-all behavior when requested by config input.
  if (allowAll || allowedToolEntries.size === 0) {
    return approveAll;
  }

  const shellRules = [...allowedToolEntries]
    .filter(tool => tool.startsWith("shell(") && tool.endsWith(")"))
    .map(tool => tool.slice("shell(".length, -1).trim())
    .filter(Boolean);
  const readablePathPatterns = shellRules.flatMap(extractReadablePathPatternsFromShellRule);

  /**
   * @param {import("@github/copilot-sdk").PermissionRequest} request
   * @returns {boolean}
   */
  function isAllowed(request) {
    switch (request.kind) {
      case "shell": {
        if (allowedToolEntries.has("shell")) return true;
        const commandIdentifiers = Array.isArray(request.commands) ? request.commands.map(cmd => cmd?.identifier).filter(Boolean) : [];
        const fullCommand = String(request.fullCommandText || "").trim();
        return shellRules.some(rule => {
          if (rule.endsWith(":*")) {
            const prefix = rule.slice(0, -2).trim();
            return prefix.length > 0 && commandIdentifiers.includes(prefix);
          }
          if (!rule.includes(" ")) {
            return commandIdentifiers.includes(rule);
          }
          return fullCommand === rule;
        });
      }
      case "write":
        return allowedToolEntries.has("write");
      case "read":
        return allowedToolEntries.has("read") || allowedToolEntries.has("shell") || isReadPathAllowedByShellRules(request.path, readablePathPatterns);
      case "url":
        return allowedToolEntries.has("web_fetch");
      case "mcp":
        // Server-only entries (for example: "github") allow all tools from that server.
        // Server+tool entries (for example: "github(get_file_contents)") allow only that tool.
        return allowedToolEntries.has(request.serverName) || allowedToolEntries.has(`${request.serverName}(${request.toolName})`);
      case "custom-tool":
        return allowedToolEntries.has(request.toolName);
      default:
        return false;
    }
  }

  return request => {
    if (isAllowed(request)) {
      return { kind: "approve-once" };
    }
    const requestSummary = summarizePermissionRequest(request);
    logPermissionDenied(logOptions?.coreLogger, logger, request);
    if (logOptions?.onDenied) {
      logOptions.onDenied(requestSummary);
    }
    return { kind: "reject", feedback: "Tool invocation is not allowed by workflow tool permissions." };
  };
}

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
 *   model?: string,
 *   connectionToken?: string,
 *   provider?: import("@github/copilot-sdk").ProviderConfig,
 *   maxToolDenials?: number | string,
 *   permissionConfig?: {
 *     allowAllTools?: boolean,
 *     allowedTools?: string[],
 *   },
 *   coreLogger?: CopilotSDKCoreLogger,
 *   sdkModule?: {
 *     CopilotClient: typeof import("@github/copilot-sdk").CopilotClient,
 *     RuntimeConnection: typeof import("@github/copilot-sdk").RuntimeConnection,
 *     approveAll: typeof import("@github/copilot-sdk").approveAll
 *   },
 * }} options
 * @returns {Promise<{exitCode: number, output: string, hasOutput: boolean, durationMs: number}>}
 */
async function runWithCopilotSDK({ sdkUri, prompt, logger, attempt = 0, model, connectionToken, provider, maxToolDenials, permissionConfig, coreLogger, sdkModule }) {
  // Lazy-require to avoid loading the SDK when it is not needed.
  // The SDK is large and has side-effects on import (worker threads, etc.).
  const { CopilotClient, RuntimeConnection, approveAll } = sdkModule ?? require("@github/copilot-sdk");

  const startTime = Date.now();
  let output = "";
  let hasOutput = false;

  const log = msg => logger(`[sdk-driver] ${msg}`);
  log(`attempt ${attempt + 1}: connecting to Copilot SDK at ${sdkUri}`);
  let maxToolDenialsLimit = MAX_TOOL_DENIALS_DEFAULT;
  if (maxToolDenials === undefined) {
    maxToolDenialsLimit = getEnvPositiveIntOrDefault("GH_AW_MAX_TOOL_DENIALS", MAX_TOOL_DENIALS_DEFAULT);
  } else {
    maxToolDenialsLimit = parseMaxToolDenialsLimit(maxToolDenials);
  }
  log(`max-tool-denials threshold: ${maxToolDenialsLimit}`);

  // Session state directory — mirrors the target path used by unified_timeline.cjs.
  // /tmp/gh-aw/sandbox/agent/logs/copilot-session-state/{sessionId}/events.jsonl
  const sessionStateBase = path.join(os.tmpdir(), "gh-aw", "sandbox", "agent", "logs", "copilot-session-state");

  /** @type {ReadonlyArray<NonNullable<import("@github/copilot-sdk").CopilotClientOptions["logLevel"]>>} */
  const VALID_LOG_LEVELS = ["none", "error", "warning", "info", "debug", "all"];
  const rawLogLevel = process.env.COPILOT_SDK_LOG_LEVEL ?? "";
  /**
   * @param {string} value
   * @returns {value is NonNullable<import("@github/copilot-sdk").CopilotClientOptions["logLevel"]>}
   */
  const isValidLogLevel = value => {
    /** @type {readonly string[]} */
    const validLogLevels = VALID_LOG_LEVELS;
    return validLogLevels.includes(value);
  };
  /** @type {import("@github/copilot-sdk").CopilotClientOptions["logLevel"]} */
  const logLevel = isValidLogLevel(rawLogLevel) ? rawLogLevel : "warning";

  const connection = RuntimeConnection.forUri(sdkUri, {
    connectionToken,
  });
  const client = new CopilotClient({
    connection,
    workingDirectory: process.env.GITHUB_WORKSPACE || process.cwd(),
    logLevel,
  });
  let session = null;
  /** @type {fs.WriteStream | null} */
  let eventsStream = null;
  let clientStarted = false;
  let toolDenialCount = 0;
  let catastrophicToolDenialsError = null;
  let catastrophicToolDenialsTriggered = false;

  /**
   * @param {string} reason
   */
  function recordToolDenial(reason) {
    toolDenialCount += 1;
    log(`tool denial ${toolDenialCount}/${maxToolDenialsLimit}: ${reason}`);
    if (catastrophicToolDenialsTriggered || toolDenialCount < maxToolDenialsLimit) {
      return;
    }
    catastrophicToolDenialsTriggered = true;
    catastrophicToolDenialsError = new Error(`max tool denials threshold reached (${toolDenialCount}/${maxToolDenialsLimit})`);
    log(`${catastrophicToolDenialsError.message}; stopping SDK session early`);
    if (session) {
      void session.disconnect().catch(() => {
        // best-effort early stop
      });
    }
  }

  try {
    await client.start();
    clientStarted = true;
    log("client started");

    /**
     * Build the session on-permission handler from configuration input.
     * @type {import("@github/copilot-sdk").PermissionHandler}
     */
    const onPermissionRequest = buildCopilotSDKPermissionHandler(permissionConfig, approveAll, {
      coreLogger,
      logger: log,
      onDenied: requestSummary => recordToolDenial(`permission denied: ${requestSummary}`),
    });

    /** @type {import("@github/copilot-sdk").SessionConfig} */
    const sessionConfig = {
      model: model || process.env.COPILOT_MODEL || undefined,
      provider,
      onPermissionRequest,
    };
    session = await client.createSession(sessionConfig);
    log(`session created: sessionId=${session.sessionId}`);

    // Prepare JSONL output file for this session.
    const sessionDir = path.join(sessionStateBase, session.sessionId);
    fs.mkdirSync(sessionDir, { recursive: true });
    const eventsPath = path.join(sessionDir, "events.jsonl");
    eventsStream = fs.createWriteStream(eventsPath, { flags: "a" });
    // Snapshot to a non-null local for closure-safe writes (JSDoc nullability narrowing).
    const stream = eventsStream;
    log(`serialising SDK events to ${eventsPath}`);

    /**
     * Map from toolCallId → {toolName, mcpServerName} so that tool.execution_complete
     * events (which carry no mcpServerName) can be enriched from the matching start event.
     * @type {Map<string, {toolName: string, mcpServerName: string}>}
     */
    const pendingToolCalls = new Map();

    /**
     * Write one JSONL entry to the events file and stderr.
     * Uses the event's own ISO-8601 timestamp when available.
     *
     * @param {string} type
     * @param {object} data
     * @param {string | undefined} [timestamp]
     */
    function writeEvent(type, data, timestamp) {
      const entry = { type, timestamp: timestamp ?? new Date().toISOString(), data };
      const jsonl = JSON.stringify(entry) + "\n";
      stream.write(jsonl);
      process.stderr.write(jsonl);
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
          // max-tool-denials intentionally tracks permission denials only.
          // Tool execution failures are still logged, but do not increment the guardrail counter.
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
    const sendTimeoutMs = getEnvPositiveIntOrDefault("COPILOT_SDK_SEND_TIMEOUT_MS", SDK_SEND_TIMEOUT_MS_DEFAULT);
    const result = await session.sendAndWait({ prompt }, sendTimeoutMs);

    if (catastrophicToolDenialsError) {
      throw catastrophicToolDenialsError;
    }

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
    const failure = catastrophicToolDenialsError ?? (err instanceof Error ? err : new Error(String(err)));
    log(`error: ${failure.message}`);
    return {
      exitCode: 1,
      output: failure.message,
      hasOutput: false,
      durationMs,
    };
  } finally {
    // Snapshot for null-safe cleanup in this scope.
    const stream = eventsStream;
    if (stream) {
      await new Promise(resolve => stream.end(resolve));
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

/**
 * Parse a CopilotSDKPermissionConfig from a JSON-encoded sidecar args array.
 *
 * Extracts --allow-tool values and the --allow-all-tools flag from the raw
 * GH_AW_COPILOT_SDK_SERVER_ARGS string that the Go engine writes. Returns
 * undefined when no permission-related flags are present so the session
 * on-permission handler can interpret config absence as unrestricted behavior.
 *
 * @param {string | undefined} serverArgsJson - Raw JSON value of GH_AW_COPILOT_SDK_SERVER_ARGS
 * @returns {CopilotSDKPermissionConfig | undefined}
 */
function parsePermissionConfigFromServerArgs(serverArgsJson) {
  if (!serverArgsJson) {
    return undefined;
  }
  /** @type {unknown} */
  let parsed;
  try {
    parsed = JSON.parse(serverArgsJson);
  } catch {
    return undefined;
  }
  if (!Array.isArray(parsed)) {
    return undefined;
  }
  const args = /** @type {unknown[]} */ parsed;

  // --allow-all-tools takes precedence: the sidecar was launched with blanket
  // tool approval, so the driver should mirror that policy.
  if (args.includes("--allow-all-tools")) {
    return { allowAllTools: true };
  }

  // Collect the value of every --allow-tool <entry> pair.
  /** @type {string[]} */
  const allowedTools = [];
  for (let i = 0; i < args.length - 1; i++) {
    if (args[i] === "--allow-tool" && typeof args[i + 1] === "string") {
      allowedTools.push(/** @type {string} */ args[i + 1]);
      i += 1; // consume the value so it is not re-examined as a flag
    }
  }

  return allowedTools.length > 0 ? { allowedTools } : undefined;
}

module.exports = { extractPromptFromArgs, runWithCopilotSDK, parsePermissionConfigFromServerArgs };

// ---------------------------------------------------------------------------
// Standalone entry point
// ---------------------------------------------------------------------------

/**
 * Log a message prefixed with [copilot-sdk-driver] to stderr.
 * @param {string} msg
 */
function log(msg) {
  process.stderr.write(`[copilot-sdk-driver] ${msg}\n`);
}

/**
 * Entry point when the driver is run directly with Node:
 *   node copilot_sdk_driver.cjs
 *
 * Reads configuration from environment variables and connects to the headless
 * Copilot CLI sidecar that has already been started by copilot_harness.cjs.
 * Runs a single SDK session and exits with the session's exit code.
 * Any unhandled error causes a non-zero exit.
 */
async function main() {
  // --- Read configuration from environment ---------------------

  const promptFile = process.env.GH_AW_PROMPT;
  if (!promptFile) {
    process.stderr.write("[copilot-sdk-driver] error: GH_AW_PROMPT is not set\n");
    process.exit(1);
  }

  const sdkUri = process.env.COPILOT_SDK_URI;
  if (!sdkUri) {
    process.stderr.write("[copilot-sdk-driver] error: COPILOT_SDK_URI is not set\n");
    process.exit(1);
  }

  const model = process.env.COPILOT_MODEL || undefined;
  const connectionToken = process.env.COPILOT_CONNECTION_TOKEN;
  if (!connectionToken) {
    process.stderr.write("[copilot-sdk-driver] error: COPILOT_CONNECTION_TOKEN is required. This token is generated by copilot_harness.cjs and must be passed to the driver environment\n");
    process.exit(1);
  }

  // --- Read the prompt -------------------------------------------------

  let prompt;
  try {
    prompt = fs.readFileSync(promptFile, "utf8");
  } catch (err) {
    process.stderr.write(`[copilot-sdk-driver] error: failed to read prompt file ${promptFile}: ${err}\n`);
    process.exit(1);
  }

  log(`connecting to sidecar at ${sdkUri}`);

  // --- Resolve BYOK custom provider from environment ------------------
  // The harness resolves the BYOK provider from live AWF reflect data before launching
  // this driver and injects the result as GH_AW_COPILOT_SDK_PROVIDER_BASE_URL.
  // BYOK is the only supported mode — fail immediately if the env var is missing.
  const providerBaseUrl = process.env.GH_AW_COPILOT_SDK_PROVIDER_BASE_URL;
  if (!providerBaseUrl) {
    process.stderr.write("[copilot-sdk-driver] error: GH_AW_COPILOT_SDK_PROVIDER_BASE_URL is not set — " + "BYOK provider is required; ensure the harness resolved a custom provider from awf-reflect data\n");
    process.exit(1);
  }
  /** @type {import("@github/copilot-sdk").ProviderConfig} */
  const provider = { type: "openai", baseUrl: providerBaseUrl };

  // --- Build permission config from sidecar server args ----------------
  // GH_AW_COPILOT_SDK_SERVER_ARGS holds the JSON-encoded --allow-tool flags
  // that the Go engine passed to the sidecar. Mirror those same rules in the
  // SDK session so the driver's onPermissionRequest handler aligns with the
  // sidecar's pre-configured allow list (e.g. shell(safeoutputs:*) for
  // workflows with safe-outputs enabled and a restricted bash allowlist).
  const permissionConfig = parsePermissionConfigFromServerArgs(process.env.GH_AW_COPILOT_SDK_SERVER_ARGS);
  if (permissionConfig) {
    if (permissionConfig.allowAllTools) {
      log("permission config: allow-all-tools (sidecar launched with --allow-all-tools)");
    } else {
      log(`permission config: ${(permissionConfig.allowedTools ?? []).length} allow-tool entries from GH_AW_COPILOT_SDK_SERVER_ARGS`);
    }
  } else {
    log("permission config: none (onPermissionRequest will use unrestricted behavior)");
  }

  // --- Run SDK session -------------------------------------------------

  const result = await runWithCopilotSDK({
    sdkUri,
    prompt,
    logger: log,
    model,
    connectionToken,
    provider,
    permissionConfig,
  });

  process.exit(result.exitCode);
}

if (require.main === module) {
  main().catch(err => {
    process.stderr.write(`[copilot-sdk-driver] unhandled error: ${err instanceof Error ? err.stack : String(err)}\n`);
    process.exit(1);
  });
}
