// @ts-check

/**
 * Copilot Harness with Retry Logic
 *
 * Wraps the Copilot CLI command (or @github/copilot-sdk session in SDK mode) with retry logic
 * for failures that occur after the session has been partially executed.  Passes all arguments
 * to the copilot subprocess, transparently forwarding stdin/stdout/stderr.
 *
 * Retry policy (shared by CLI and SDK modes):
 *   - If the process produced any output (hasOutput) and exits with a non-zero code, the
 *     session is considered partially executed and is retried.
 *     - CLI mode: retries with --continue so the Copilot CLI can continue from on-disk state.
 *     - SDK mode: retries always restart the session fresh (--continue is a CLI concept).
 *   - CAPIError 400 is a well-known transient failure mode and is logged explicitly, but
 *     any partial-execution failure is retried — not just CAPIError 400.
 *   - If the process produced no output (failed to start / auth error before any work), the
 *     driver does not retry because there is nothing to resume.
 *   - "No authentication information found" errors are handled differently depending on context:
 *     - On a `--continue` attempt: the Copilot CLI's on-disk session credential written by the
 *       interrupted run may be incomplete/invalid.  The driver falls back to a single fresh run
 *       (without `--continue`) so env-var auth can succeed.  Mid-stream context is lost but the
 *       job has a recovery path.
 *     - On a fresh run (attempt 0 or after a `--continue`-auth fallback): the env-var token is
 *       genuinely absent or invalid.  All further retries will produce the same failure, so the
 *       driver bails immediately.
 *   - Null-type tool_call errors (400 "Invalid type for '...tool_calls[N].type': ... got null")
 *     poison the conversation history.  Retrying with `--continue` re-injects the same broken
 *     state on every subsequent attempt.  The driver restarts fresh to discard the poisoned
 *     history and permanently disables `--continue` for the remainder of the run so the corrupt
 *     state can never be reloaded.  Once `--continue` is disabled this way it is not re-enabled
 *     even if later retries produce output.
 *   - Retries use exponential backoff: 5s → 10s → 20s (capped at 60s).
 *   - Maximum 3 retry attempts after the initial run.
 *
 * Usage: node copilot_harness.cjs <command> [args...]
 * Example: node copilot_harness.cjs copilot --add-dir /tmp/ --prompt-file /tmp/gh-aw/aw-prompts/prompt.txt
 */

"use strict";

require("./shim.cjs");

const fs = require("fs");
const path = require("path");
const crypto = require("crypto");
const { runProcess, formatDuration, sleep, isCopilotSDKEnabled, buildCopilotSDKEnv } = require("./process_runner.cjs");
const { buildCopilotSDKServerArgs, getCopilotSDKServerPort, startCopilotSDKServer, stopCopilotSDKServer, waitForCopilotSDKServer } = require("./copilot_sdk_sidecar.cjs");
const { extractPromptFromArgs, runWithCopilotSDK } = require("./copilot_sdk_driver.cjs");
const { isMaxEffectiveTokensExceededError } = require("./effective_tokens_hard_rail.cjs");
const {
  AWF_API_PROXY_REFLECT_URL,
  AWF_REFLECT_OUTPUT_PATH,
  AWF_REFLECT_TIMEOUT_MS,
  AWF_MODELS_URL_TIMEOUT_MS,
  GEMINI_MODEL_NAME_PREFIX,
  enrichReflectModels,
  extractModelIds,
  fetchAWFReflect,
  fetchModelsFromUrl,
} = require("./awf_reflect.cjs");
const { runSafeOutputsCLI, buildMissingToolAlternatives, emitMissingToolPermissionIssue, emitInfrastructureIncomplete } = require("./safeoutputs_cli.cjs");
const { countPermissionDeniedIssues, hasNumerousPermissionDeniedIssues, extractDeniedCommands, buildMissingToolPermissionIssuePayload } = require("./permission_denied_helpers.cjs");

// Maximum number of retry attempts after the initial run
const MAX_RETRIES = 3;
// Initial delay in milliseconds before the first retry
const INITIAL_DELAY_MS = 5000;
// Multiplier applied to delay after each retry
const BACKOFF_MULTIPLIER = 2;
// Maximum delay cap in milliseconds
const MAX_DELAY_MS = 60000;
// Additional startup retry budget for scheduled runs when Copilot exits with code 2
// before producing any output (typically transient API interruption at startup).
const MAX_SCHEDULED_EXIT2_RETRIES = 1;
// If prompt files are larger than this threshold, avoid inlining into argv.
const PROMPT_FILE_INLINE_THRESHOLD_BYTES = 100 * 1024;
const PROMPT_FILE_INLINE_THRESHOLD_LABEL = "100KB";
const MAX_ENV_VAR_PREVIEW_LENGTH = 120;
// Pattern to detect transient CAPIError 400 in copilot output
const CAPI_ERROR_400_PATTERN = /CAPIError:\s*400/;

// Pattern to detect MCP servers blocked by enterprise/organization policy.
// This is a persistent policy configuration error — retrying will not help.
const MCP_POLICY_BLOCKED_PATTERN = /MCP servers were blocked by policy:/;

// Pattern to detect "model not supported" error (e.g. Copilot Pro/Education users hitting
// a model that is unavailable for their subscription tier).
// This is a persistent configuration error — retrying with --continue will not help.
const MODEL_NOT_SUPPORTED_PATTERN = /The requested model is not supported/;

// Pattern to detect missing authentication credentials.
// On a --continue attempt this may indicate that the Copilot CLI's on-disk session
// credential (written by a mid-stream interrupted run) is incomplete or invalid.  In that
// case the driver falls back to a fresh run (without --continue) to re-do env-var auth.
// On a fresh run the token is genuinely absent — retrying will not help.
const NO_AUTH_INFO_PATTERN = /No authentication information found|Session was not created with authentication info or custom provider/;
// Pattern to detect authentication failures returned by Copilot API.
// After a first-attempt auth failure, retrying is futile because the entrypoint unsets
// COPILOT_GITHUB_TOKEN between attempts.
const AUTHENTICATION_FAILED_PATTERN = /Authentication failed(?:\s*\(Request ID:[^)]+\))?/i;
// Pattern: Copilot CLI inference access denied
const INFERENCE_ACCESS_ERROR_PATTERN = /Access denied by policy settings|invalid access to inference/;
// Pattern: Agentic engine process killed by signal (timeout)
const AGENTIC_ENGINE_TIMEOUT_PATTERN = /signal=SIG(?:TERM|KILL|INT)/;

// Pattern to detect null-type tool_call error that poisons conversation history.
// Matches the Copilot API 400 error:
//   "Invalid type for '...tool_calls[N].type': expected one of 'function', ..., but got null instead."
// The model emitted a malformed tool call with type: null.  Retrying with --continue
// re-injects the same broken history, producing the same 400 on every subsequent attempt.
// A fresh restart is required to discard the poisoned history.
const NULL_TYPE_TOOL_CALL_PATTERN = /tool_calls\[.*?\]\.type.*null/;
/**
 * Emit a diagnostic log line to stderr.
 * All driver messages are prefixed with "[copilot-harness]" so they are easy to
 * grep out of the combined agent-stdio.log.
 * @param {string} message
 */
function log(message) {
  process.stderr.write(`[copilot-harness] ${message}\n`);
}

/**
 * Generate a per-run connection token for Copilot SDK headless authentication.
 * Produces 32 random bytes encoded as a 64-character hexadecimal string.
 * @param {{ randomBytes?: (size: number) => Buffer }} [options]
 * @returns {string} 64-character hexadecimal token (32 random bytes).
 */
function generateCopilotConnectionToken(options) {
  // randomBytes injection exists only for unit tests; production uses crypto.randomBytes.
  const randomBytes = options?.randomBytes ?? crypto.randomBytes;
  return randomBytes(32).toString("hex");
}

/**
 * Determines if the collected output contains a transient CAPIError 400
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isTransientCAPIError(output) {
  return CAPI_ERROR_400_PATTERN.test(output);
}

/**
 * Determines if the collected output indicates MCP servers were blocked by policy.
 * This is a persistent configuration error that cannot be resolved by retrying.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isMCPPolicyError(output) {
  return MCP_POLICY_BLOCKED_PATTERN.test(output);
}

/**
 * Determines if the collected output indicates the requested model is not supported.
 * This occurs when a Copilot Pro/Education user attempts to use a model that is not
 * available for their subscription tier.  Retrying will not help.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isModelNotSupportedError(output) {
  return MODEL_NOT_SUPPORTED_PATTERN.test(output);
}

/**
 * Determine whether the current run phase is threat detection.
 * @param {string | undefined | null} phase
 * @returns {boolean}
 */
function isDetectionPhase(phase) {
  return (
    String(phase || "")
      .trim()
      .toLowerCase() === "detection"
  );
}

/**
 * Check whether a model is present in AWF /reflect endpoint data.
 * @param {string} model
 * @param {unknown} reflectData
 * @returns {boolean}
 */
function isModelAvailableInReflectData(model, reflectData) {
  const normalizedModel = typeof model === "string" ? model.trim() : "";
  if (!normalizedModel) return false;
  if (!reflectData || typeof reflectData !== "object") return false;

  // TypeScript needs explicit 'in' check or cast before property access on narrowed object type
  const endpoints = "endpoints" in reflectData && Array.isArray(reflectData.endpoints) ? reflectData.endpoints : [];
  for (const endpoint of endpoints) {
    if (!endpoint || endpoint.configured !== true || !Array.isArray(endpoint.models)) {
      continue;
    }
    if (endpoint.models.includes(normalizedModel)) {
      return true;
    }
  }
  return false;
}

/**
 * Load saved AWF /reflect data and check whether a model is present.
 * @param {string} model
 * @param {{
 *   reflectPath?: string,
 *   readFileSync?: (path: string, encoding: string) => string,
 *   logger?: (msg: string) => void,
 * }} [options]
 * @returns {boolean}
 */
function isModelAvailableInReflectFile(model, options) {
  const normalizedModel = typeof model === "string" ? model.trim() : "";
  const reflectPath = (options && options.reflectPath) || AWF_REFLECT_OUTPUT_PATH;
  const readFile = (options && options.readFileSync) || fs.readFileSync;
  const logger = (options && options.logger) || log;
  if (!normalizedModel) {
    logger("awf-reflect: model availability check skipped (model is empty)");
    return false;
  }

  try {
    const raw = readFile(reflectPath, "utf8");
    const reflectData = JSON.parse(raw);
    return isModelAvailableInReflectData(normalizedModel, reflectData);
  } catch (error) {
    const err = /** @type {Error} */ error;
    logger(`awf-reflect: unable to read model availability from ${reflectPath}: ${err.message}`);
    return false;
  }
}

/**
 * Resolve Copilot SDK BYOK custom provider configuration from saved AWF /reflect data.
 * Chooses a configured endpoint and maps it to an OpenAI-compatible provider base URL.
 *
 * @param {{
 *   model?: string,
 *   reflectPath?: string,
 *   readFileSync?: (path: string, encoding: string) => string,
 *   logger?: (msg: string) => void,
 * }} [options]
 * @returns {{ model: string, provider: { type: "openai", baseUrl: string } } | null}
 */
function resolveCopilotSDKCustomProviderFromReflect(options) {
  const configuredModel = typeof options?.model === "string" ? options.model.trim() : "";
  const reflectPath = (options && options.reflectPath) || AWF_REFLECT_OUTPUT_PATH;
  const readFile = (options && options.readFileSync) || fs.readFileSync;
  const logger = (options && options.logger) || log;

  try {
    const raw = readFile(reflectPath, "utf8");
    const reflectData = JSON.parse(raw);
    const endpoints = Array.isArray(reflectData?.endpoints) ? reflectData.endpoints.filter(ep => ep && ep.configured === true) : [];
    if (endpoints.length === 0) {
      logger(`sdk-mode: no configured endpoints in ${reflectPath}; skipping custom provider config`);
      return null;
    }

    const endpoint = (configuredModel ? endpoints.find(ep => Array.isArray(ep.models) && ep.models.includes(configuredModel)) : null) || endpoints.find(ep => String(ep.provider || "").toLowerCase() === "copilot") || endpoints[0];

    let baseUrl = "";
    if (typeof endpoint?.models_url === "string" && endpoint.models_url) {
      try {
        baseUrl = new URL(endpoint.models_url).origin;
      } catch {
        // ignore malformed URL and fall back to port-based construction below
      }
    }
    if (!baseUrl && endpoint?.port != null) {
      baseUrl = `http://api-proxy:${String(endpoint.port)}`;
    }
    if (!baseUrl) {
      logger("sdk-mode: unable to derive provider baseUrl from awf-reflect endpoint data; skipping custom provider config");
      return null;
    }

    let model = configuredModel;
    if (!model && Array.isArray(endpoint?.models)) {
      const firstModel = endpoint.models.find(m => typeof m === "string" && m.trim().length > 0);
      model = typeof firstModel === "string" ? firstModel.trim() : "";
    }
    if (!model) {
      logger("sdk-mode: unable to derive model for custom provider from awf-reflect; skipping custom provider config");
      return null;
    }

    logger(`sdk-mode: custom provider resolved from awf-reflect (provider=${String(endpoint.provider || "unknown")} baseUrl=${baseUrl} model=${model})`);
    return {
      model,
      provider: { type: "openai", baseUrl },
    };
  } catch (error) {
    const err = /** @type {Error} */ error;
    logger(`sdk-mode: unable to read custom provider config from ${reflectPath}: ${err.message}`);
    return null;
  }
}

/**
 * Determines if the collected output contains a "No authentication information found" error.
 * This means no auth token (COPILOT_GITHUB_TOKEN, GH_TOKEN, or GITHUB_TOKEN) is available
 * in the environment.  Retrying will not help because the absent token will remain absent.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isNoAuthInfoError(output) {
  return NO_AUTH_INFO_PATTERN.test(output);
}

/**
 * Determines if the collected output contains an authentication failed error.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isAuthenticationFailedError(output) {
  return AUTHENTICATION_FAILED_PATTERN.test(output);
}

/**
 * Extract provider auth failure details from Copilot output when available.
 * @param {string} output
 * @returns {{ providerUrl: string, statusCode: string } | null}
 */
function parseProviderAuthFailure(output) {
  const match = output.match(/Authentication failed with provider at (\S+) \(HTTP (\d+)\)\.?/i);
  if (!match) {
    return null;
  }
  return {
    providerUrl: match[1],
    statusCode: match[2],
  };
}

/**
 * Determine whether a provider URL likely points at the gh-aw API proxy sidecar.
 * @param {string} providerUrl
 * @returns {boolean}
 */
function isLikelyAWFAPIProxyURL(providerUrl) {
  try {
    const { hostname, port } = new URL(providerUrl);
    const normalizedHostname = hostname.toLowerCase();
    if (port !== "10002") {
      return false;
    }
    return (
      normalizedHostname === "api-proxy" ||
      normalizedHostname === "host.docker.internal" ||
      normalizedHostname === "localhost" ||
      /^127(?:\.\d{1,3}){3}$/.test(normalizedHostname) ||
      /^10(?:\.\d{1,3}){3}$/.test(normalizedHostname) ||
      /^192\.168(?:\.\d{1,3}){2}$/.test(normalizedHostname) ||
      /^172\.(?:1[6-9]|2\d|3[01])(?:\.\d{1,3}){2}$/.test(normalizedHostname)
    );
  } catch {
    return false;
  }
}

/**
 * Infer which Copilot auth stage failed without exposing secrets.
 * @param {string} output
 * @returns {string}
 */
function detectCopilotAuthFailureStage(output) {
  if (/\b(?:validating|validate|validation)\b[\s\S]{0,40}\b(?:token|auth|authentication)\b/i.test(output)) {
    return "validating the token";
  }
  if (/\b(?:list|listing)\b[\s\S]{0,40}\bmodels?\b/i.test(output) || /\/models\b/i.test(output)) {
    return "listing models";
  }
  return "starting the Copilot CLI request";
}

/**
 * Build a more actionable Copilot auth diagnostic when a 401 came from the gh-aw API proxy.
 * @param {string} output
 * @param {NodeJS.ProcessEnv} [env]
 * @returns {string}
 */
function buildCopilotProxyAuthFailureDiagnostic(output, env = process.env) {
  const authFailure = parseProviderAuthFailure(output);
  if (!authFailure || authFailure.statusCode !== "401" || !isLikelyAWFAPIProxyURL(authFailure.providerUrl)) {
    return "";
  }

  const selectedModel = typeof env.COPILOT_MODEL === "string" && env.COPILOT_MODEL.trim() ? env.COPILOT_MODEL.trim() : "(unset)";
  const stage = detectCopilotAuthFailureStage(output);
  return (
    `Copilot authentication failed through the gh-aw API proxy (HTTP 401, model=${selectedModel}, stage=${stage}). ` +
    "Check that COPILOT_GITHUB_TOKEN is present, unexpired, and authorized for the selected COPILOT_MODEL. " +
    "If you configured GH_AW_MODEL_AGENT_COPILOT or GH_AW_DEFAULT_MODEL_COPILOT, verify that the token has access to that model."
  );
}

/**
 * Detect known Copilot error patterns for workflow outputs.
 * @param {string} output
 * @returns {{ inferenceAccessError: boolean, mcpPolicyError: boolean, agenticEngineTimeout: boolean, modelNotSupportedError: boolean }}
 */
function detectCopilotErrors(output) {
  return {
    inferenceAccessError: INFERENCE_ACCESS_ERROR_PATTERN.test(output),
    mcpPolicyError: isMCPPolicyError(output),
    agenticEngineTimeout: AGENTIC_ENGINE_TIMEOUT_PATTERN.test(output),
    modelNotSupportedError: isModelNotSupportedError(output),
  };
}

/**
 * Write Copilot detection outputs to $GITHUB_OUTPUT.
 * @param {{ inferenceAccessError: boolean, mcpPolicyError: boolean, agenticEngineTimeout: boolean, modelNotSupportedError: boolean }} results
 */
function writeCopilotOutputs(results) {
  const outputFile = process.env.GITHUB_OUTPUT;
  if (!outputFile) {
    log("GITHUB_OUTPUT not set — skipping copilot error outputs");
    return;
  }

  const lines = [
    `inference_access_error=${results.inferenceAccessError}`,
    `mcp_policy_error=${results.mcpPolicyError}`,
    `agentic_engine_timeout=${results.agenticEngineTimeout}`,
    `model_not_supported_error=${results.modelNotSupportedError}`,
  ];
  fs.appendFileSync(outputFile, lines.join("\n") + "\n");
}

/**
 * Determines if the collected output contains a null-type tool_call error.
 * This error occurs when the model emits a malformed tool call with type: null.
 * The Copilot API rejects it with a 400, and retrying with --continue will re-inject
 * the same broken history, causing the same failure on every subsequent attempt.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isNullTypeToolCallError(output) {
  return NULL_TYPE_TOOL_CALL_PATTERN.test(output);
}

/**
 * Build a structured report_incomplete payload for infrastructure failures.
 * @param {string} details
 * @returns {string}
 */
function buildInfrastructureIncompletePayload(details) {
  return JSON.stringify({
    type: "report_incomplete",
    reason: "infrastructure_error",
    details,
  });
}

/**
 * Append one safe-output entry line.
 * @param {(path: import("node:fs").PathOrFileDescriptor, data: string | Uint8Array, options?: import("node:fs").WriteFileOptions) => void} appendFileSync
 * @param {string} safeOutputsPath
 * @param {string} payload
 */
function appendSafeOutputLine(appendFileSync, safeOutputsPath, payload) {
  appendFileSync(safeOutputsPath, payload + "\n", { encoding: "utf8" });
}

/**
 * Check whether a command path is accessible and executable, logging the result.
 * Returns true if the command is usable, false otherwise.
 * @param {string} command - Absolute or relative path to the executable
 * @returns {Promise<boolean>}
 */
async function checkCommandAccessible(command) {
  try {
    await fs.promises.access(command, fs.constants.F_OK);
  } catch {
    log(`pre-flight: command not found: ${command} (F_OK check failed — binary does not exist at this path)`);
    return false;
  }
  try {
    await fs.promises.access(command, fs.constants.X_OK);
    log(`pre-flight: command is accessible and executable: ${command}`);
    return true;
  } catch {
    log(`pre-flight: command exists but is not executable: ${command} (X_OK check failed — permission denied)`);
    return false;
  }
}

/**
 * Read and parse the JSON options payload piped to stdin by the engine command.
 * Called in SDK mode where the Go engine pipes options via `printf '%s' '{"promptFile":"...","serverArgs":[...],"permissionConfig":{...}}'
 * | node harness`.
 * Returns null when stdin is a TTY, empty, or contains invalid JSON.
 * @returns {Promise<{promptFile?: string, serverArgs?: string[], addWorkspaceDir?: boolean, permissionConfig?: {allowAllTools?: boolean, allowedTools?: string[]}} | null>}
 */
async function readSDKOptionsFromStdin() {
  if (process.stdin.isTTY) return null;
  return new Promise(resolve => {
    /** @type {Buffer[]} */
    const chunks = [];
    process.stdin.on("data", chunk => {
      chunks.push(Buffer.from(chunk));
    });
    process.stdin.on("end", () => {
      const text = Buffer.concat(chunks).toString("utf8").trim();
      if (!text) {
        resolve(null);
        return;
      }

      try {
        resolve(JSON.parse(text));
      } catch {
        log(`warning: failed to parse SDK options from stdin: ${text.slice(0, 100)}`);
        resolve(null);
      }
    });
    process.stdin.on("error", () => resolve(null));
  });
}

/**
 * Parse GH_AW_COPILOT_SDK_SERVER_ARGS for SDK driver mode.
 * Returns [] when unset or invalid so sidecar defaults remain available.
 *
 * @param {string | undefined} serverArgsEnv
 * @param {{ logger?: (msg: string) => void }} [options]
 * @returns {string[]}
 */
function parseCopilotSDKServerArgsFromEnv(serverArgsEnv, options) {
  const logger = options?.logger ?? log;
  if (!serverArgsEnv) {
    logger("copilot-sdk driver mode: GH_AW_COPILOT_SDK_SERVER_ARGS is not set; using sidecar default args");
    return [];
  }

  try {
    const parsed = JSON.parse(serverArgsEnv);
    if (!Array.isArray(parsed) || parsed.some(arg => typeof arg !== "string")) {
      logger("copilot-sdk driver mode: GH_AW_COPILOT_SDK_SERVER_ARGS must be a JSON string array; using sidecar default args");
      return [];
    }
    logger(`copilot-sdk driver mode: parsed ${parsed.length} sidecar args from GH_AW_COPILOT_SDK_SERVER_ARGS`);
    return parsed;
  } catch (parseErr) {
    const preview = serverArgsEnv.length > MAX_ENV_VAR_PREVIEW_LENGTH ? serverArgsEnv.slice(0, MAX_ENV_VAR_PREVIEW_LENGTH) + "…" : serverArgsEnv;
    logger(`copilot-sdk driver mode: failed to parse GH_AW_COPILOT_SDK_SERVER_ARGS: ${parseErr} (value: ${preview})`);
    return [];
  }
}

/**
 * Build a compact fallback prompt that asks the agent to read instructions from disk.
 * @param {string} promptFile
 * @returns {string}
 */
function buildPromptFileFallbackInstruction(promptFile) {
  return `Read the full instructions from ${promptFile} and execute them exactly as written.`;
}

/**
 * Replace --prompt-file arguments with -p prompt text to support older Copilot CLIs.
 * For files over 100KB, emit a compact fallback prompt that instructs the agent to
 * read and execute the full prompt file from disk.
 * @param {string[]} args
 * @returns {string[]}
 */
function resolvePromptFileArgs(args) {
  /** @type {string[]} */
  const resolvedArgs = [];

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg !== "--prompt-file") {
      resolvedArgs.push(arg);
      continue;
    }

    if (i + 1 >= args.length) {
      log("warning: --prompt-file provided without a path; leaving arguments unchanged");
      resolvedArgs.push(arg);
      continue;
    }
    const promptFile = args[i + 1];

    try {
      const stat = fs.statSync(promptFile);
      log(`resolved --prompt-file: path=${promptFile} size=${stat.size}B`);

      if (stat.size > PROMPT_FILE_INLINE_THRESHOLD_BYTES) {
        log(`prompt file exceeds ${PROMPT_FILE_INLINE_THRESHOLD_LABEL}; using compact fallback prompt`);
        resolvedArgs.push("-p", buildPromptFileFallbackInstruction(promptFile));
      } else {
        const promptText = fs.readFileSync(promptFile, "utf8");
        resolvedArgs.push("-p", promptText);
      }
      i++; // Skip the prompt-file path argument
    } catch (error) {
      const err = /** @type {Error} */ error;
      log(`warning: failed to resolve --prompt-file ${promptFile}: ${err.message}; leaving arguments unchanged`);
      resolvedArgs.push(arg, promptFile);
      i++; // Skip the prompt-file path argument
    }
  }

  return resolvedArgs;
}

/**
 * Main entry point: run copilot with retry logic for partially-executed sessions.
 */
async function main() {
  const [, , command, ...args] = process.argv;

  if (!command) {
    process.stderr.write("copilot-harness: Usage: node copilot_harness.cjs <command> [args...]\n");
    process.exit(1);
  }

  log(`starting: command=${command} maxRetries=${MAX_RETRIES} initialDelayMs=${INITIAL_DELAY_MS}` + ` backoffMultiplier=${BACKOFF_MULTIPLIER} maxDelayMs=${MAX_DELAY_MS}` + ` nodeVersion=${process.version} platform=${process.platform}`);

  await checkCommandAccessible(command);

  // Build SDK env additions. When COPILOT_SDK_URI is set the harness will start a separate
  // headless Copilot CLI sidecar and this helper merges COPILOT_SDK_URI into the child
  // process env so that every started process (including retry attempts) inherits the
  // correct SDK endpoint URI.
  const sdkEnv = buildCopilotSDKEnv();
  const copilotSDKMode = isCopilotSDKEnabled();
  // Driver mode: the engine started copilot_sdk_driver.cjs as a standalone command.
  // The harness starts the sidecar and then runs the driver like any other subprocess;
  // the driver only opens an SDK client connection to the already-running server.
  const copilotSDKDriverMode = copilotSDKMode && process.env.GH_AW_COPILOT_SDK_DRIVER === "1";
  let copilotConnectionToken;
  if (copilotSDKMode) {
    // The harness always generates the connection token when SDK mode is active.
    // In driver mode the token is injected into the driver subprocess env so the
    // harness-managed sidecar and the driver's SDK client share the same token.
    copilotConnectionToken = generateCopilotConnectionToken();
    log("copilot-sdk mode active: generated per-run COPILOT_CONNECTION_TOKEN");
    log(`copilot-sdk mode active: COPILOT_SDK_URI=${sdkEnv.COPILOT_SDK_URI || "(not set)"} driverMode=${copilotSDKDriverMode}`);
  }
  // Merge SDK env additions into the child process env only when the SDK helper
  // returned at least one variable; otherwise leave the env undefined so that
  // runProcess inherits the full process.env (the common case).
  // sdkEnv already contains SDK-mode variables (e.g. COPILOT_SDK_URI) when enabled.
  // Always attach the generated per-run COPILOT_CONNECTION_TOKEN so both the sidecar
  // (started by the harness) and the SDK client share the same token.
  const sdkChildEnv = copilotSDKMode ? { ...sdkEnv, COPILOT_CONNECTION_TOKEN: copilotConnectionToken } : sdkEnv;
  const childEnv = Object.keys(sdkChildEnv).length > 0 ? { ...process.env, ...sdkChildEnv } : undefined;

  // In inline SDK mode, the engine pipes a JSON options payload via stdin containing the promptFile
  // path, serverArgs (complete CLI argument list for the headless server), and optionally addWorkspaceDir.
  // Read it before doing anything else so stdin is consumed before the process runs.
  // In driver mode and CLI mode, args are resolved normally.
  /** @type {{promptFile?: string, serverArgs?: string[], addWorkspaceDir?: boolean, permissionConfig?: {allowAllTools?: boolean, allowedTools?: string[]}} | null} */
  let sdkOptions = null;
  let resolvedArgs;
  if (copilotSDKMode && !copilotSDKDriverMode) {
    sdkOptions = await readSDKOptionsFromStdin();
    if (sdkOptions) {
      log(
        `sdk-options: promptFile=${sdkOptions.promptFile || "(none)"} serverArgs=${(sdkOptions.serverArgs || []).length} addWorkspaceDir=${!!sdkOptions.addWorkspaceDir} permissionRules=${(sdkOptions.permissionConfig?.allowedTools || []).length} allowAllTools=${sdkOptions.permissionConfig?.allowAllTools === true}`
      );
    }
    // Inline SDK mode does not use CLI prompt args; pass args through unmodified.
    resolvedArgs = args;
  } else if (copilotSDKMode) {
    // Driver mode: args are the driver command + copilot binary path; no stdin payload.
    resolvedArgs = args;
  } else {
    resolvedArgs = resolvePromptFileArgs(args);
  }

  // Fetch AWF API proxy reflection data before running the agent to capture initial proxy state.
  // This is best-effort: failures are logged but do not affect the agent run.
  // Skip when AWF_REFLECT_ENABLED is not "1" (e.g. sandbox.agent: false — no api-proxy running).
  if (process.env.AWF_REFLECT_ENABLED === "1") {
    await fetchAWFReflect({ logger: log });
  }

  let delay = INITIAL_DELAY_MS;
  let lastExitCode = 1;
  const isScheduledRun = process.env.GITHUB_EVENT_NAME === "schedule";
  let scheduledExit2Retries = 0;
  let scheduledExit2RetryAttempted = false;
  let useContinueOnRetry = false;
  let modelNotSupportedReflectRetryAttempted = false;
  // Once set to true, --continue is never re-enabled for the remainder of this run.
  // This prevents a broken --continue recovery from resurrecting --continue on the next attempt.
  let continueDisabledPermanently = false;
  const driverStartTime = Date.now();
  const detectedCopilotErrors = {
    inferenceAccessError: false,
    mcpPolicyError: false,
    agenticEngineTimeout: false,
    modelNotSupportedError: false,
  };
  // In inline SDK mode the prompt is required; read it from the promptFile in sdkOptions (piped
  // via stdin by the engine command).  Fall back to extracting from CLI args for backward compatibility.
  // In driver mode, the driver reads the prompt directly from GH_AW_PROMPT; no prompt needed here.
  let sdkPrompt = null;
  /** @type {{ model: string, provider: { type: "openai", baseUrl: string } } | null} */
  let sdkCustomProviderConfig = null;
  const sdkCoreLogger = copilotSDKMode ? global.core : undefined;
  if (copilotSDKMode && !copilotSDKDriverMode) {
    if (sdkOptions && sdkOptions.promptFile) {
      try {
        sdkPrompt = fs.readFileSync(sdkOptions.promptFile, "utf8");
        log(`sdk-mode: read prompt from ${sdkOptions.promptFile} (${sdkPrompt.length} chars)`);
      } catch (err) {
        const readErr = /** @type {Error} */ err;
        log(`sdk-mode: failed to read prompt from ${sdkOptions.promptFile}: ${readErr.message}`);
      }
    }
    if (!sdkPrompt) {
      // Fallback: try to extract from CLI args (backward compatibility with older engine versions)
      sdkPrompt = extractPromptFromArgs(resolvedArgs);
      if (sdkPrompt) {
        log("sdk-mode: prompt extracted from CLI args (fallback)");
      } else {
        log("sdk-mode: no prompt found in stdin JSON payload or CLI args");
      }
    }
    if (process.env.AWF_REFLECT_ENABLED === "1") {
      sdkCustomProviderConfig = resolveCopilotSDKCustomProviderFromReflect({
        model: process.env.COPILOT_MODEL,
        logger: log,
      });
    }
  }
  /** @type {Awaited<ReturnType<typeof startCopilotSDKServer>>} */
  let copilotSDKServer = null;
  try {
    if (copilotSDKMode) {
      if (copilotSDKDriverMode) {
        // Driver mode: the harness starts the sidecar; the driver subprocess only opens a client.
        // Server args are provided via GH_AW_COPILOT_SDK_SERVER_ARGS (JSON-encoded CLI arg list
        // generated by the Go engine).  The copilot binary is args[1] in the driver command:
        //   node copilot_harness.cjs $GH_AW_NODE_EXEC copilot_sdk_driver.cjs <copilot-binary>
        const copilotBin = args[1];
        if (!copilotBin) {
          log("copilot-sdk driver mode: missing copilot binary path in args[1]");
          lastExitCode = 1;
        } else {
          let driverServerArgs = parseCopilotSDKServerArgsFromEnv(process.env.GH_AW_COPILOT_SDK_SERVER_ARGS, { logger: log });
          if (process.env.GITHUB_WORKSPACE) {
            driverServerArgs = [...driverServerArgs, "--add-dir", process.env.GITHUB_WORKSPACE];
            log(`copilot-sdk driver mode: appended workspace --add-dir ${process.env.GITHUB_WORKSPACE}`);
          }
          log(`copilot-sdk driver mode: starting sidecar command=${copilotBin} args=${driverServerArgs.length}`);
          copilotSDKServer = await startCopilotSDKServer({
            command: copilotBin,
            env: childEnv ?? process.env,
            serverArgs: driverServerArgs.length > 0 ? driverServerArgs : undefined,
            logger: log,
          });
        }
      } else {
        // Inline SDK mode: harness manages the sidecar and SDK session directly.
        if (!sdkPrompt) {
          log("copilot-sdk mode: no prompt found (expected promptFile in stdin JSON payload or -p/--prompt in args)");
          lastExitCode = 1;
        } else {
          // Build the server args from the stdin JSON payload.
          // serverArgs carries the complete CLI argument list for the headless server (--headless,
          // --no-auto-update, --port, --add-dir, --log-level, etc.) generated by the Go engine.
          // addWorkspaceDir signals that the GITHUB_WORKSPACE env var should be appended at runtime.
          const serverArgs = [...(sdkOptions?.serverArgs ?? [])];
          if (sdkOptions?.addWorkspaceDir && process.env.GITHUB_WORKSPACE) {
            serverArgs.push("--add-dir", process.env.GITHUB_WORKSPACE);
          }
          copilotSDKServer = await startCopilotSDKServer({
            command,
            env: childEnv ?? process.env,
            serverArgs: serverArgs.length > 0 ? serverArgs : undefined,
            logger: log,
          });
        }
      }
    }

    // CLI mode always enters the retry loop.
    // Inline SDK mode only enters when a prompt was found; the missing-prompt case above sets lastExitCode=1.
    // Driver mode always enters when the sidecar started successfully.
    if (!copilotSDKMode || (copilotSDKDriverMode && copilotSDKServer) || sdkPrompt) {
      // Unified retry loop for CLI, driver, and inline-SDK modes.
      // --continue is a CLI concept; in SDK mode retries always restart the session fresh.
      for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
        // Add --continue flag on CLI retries so the copilot session continues from where it left off
        const currentArgs = !copilotSDKMode && attempt > 0 && useContinueOnRetry ? [...resolvedArgs, "--continue"] : resolvedArgs;

        if (attempt > 0) {
          const retryMode = !copilotSDKMode && useContinueOnRetry ? "--continue" : "fresh run";
          log(`retry ${attempt}/${MAX_RETRIES}: sleeping ${delay}ms before next attempt (${retryMode})`);
          await sleep(delay);
          delay = Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS);
          log(`retry ${attempt}/${MAX_RETRIES}: woke up, next delay cap will be ${Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS)}ms`);
        }

        // Redact --prompt / -p value from logs to avoid leaking prompt content
        const safeArgs = currentArgs.map((arg, i) => (currentArgs[i - 1] === "--prompt" || currentArgs[i - 1] === "-p" ? "<redacted>" : arg));
        let result;
        if (copilotSDKDriverMode) {
          // Driver mode: run copilot_sdk_driver.cjs as a normal subprocess. The harness has
          // already started the sidecar; the driver only opens an SDK client connection.
          result = await runProcess({ command, args: currentArgs, attempt, log, logArgs: safeArgs, env: childEnv });
        } else if (copilotSDKMode) {
          if (!sdkPrompt) {
            throw new Error("sdk-mode invariant violated: prompt must be resolved before execution");
          }
          result = await runWithCopilotSDK({
            sdkUri: sdkEnv.COPILOT_SDK_URI ?? process.env.COPILOT_SDK_URI ?? "",
            prompt: sdkPrompt,
            logger: log,
            attempt,
            model: sdkCustomProviderConfig?.model,
            connectionToken: copilotConnectionToken,
            provider: sdkCustomProviderConfig?.provider,
            permissionConfig: sdkOptions?.permissionConfig,
            coreLogger: sdkCoreLogger,
          });
        } else {
          result = await runProcess({ command, args: currentArgs, attempt, log, logArgs: safeArgs, env: childEnv });
        }
        lastExitCode = result.exitCode;
        const attemptDetections = detectCopilotErrors(result.output);
        detectedCopilotErrors.inferenceAccessError ||= attemptDetections.inferenceAccessError;
        detectedCopilotErrors.mcpPolicyError ||= attemptDetections.mcpPolicyError;
        detectedCopilotErrors.agenticEngineTimeout ||= attemptDetections.agenticEngineTimeout;
        detectedCopilotErrors.modelNotSupportedError ||= attemptDetections.modelNotSupportedError;

        // Success — record exit code and stop retrying
        if (result.exitCode === 0) {
          log(`success on attempt ${attempt + 1}: totalDuration=${formatDuration(Date.now() - driverStartTime)}`);
          lastExitCode = 0;
          break;
        }

        // Determine whether to retry.
        // Retry whenever the session was partially executed (hasOutput).
        //   - CLI mode: retry with --continue so the Copilot CLI can continue from on-disk state.
        //   - SDK mode: retry always restarts fresh — there is no CLI on-disk state to resume.
        // CAPIError 400 is the well-known transient case, but any partial-execution failure is
        // eligible for a retry.
        // Exceptions:
        //   - MCP policy errors and model-not-supported errors are persistent configuration issues.
        //   - Auth errors trigger a one-time fallback to a fresh run; after that --continue is
        //     permanently disabled.
        //   - Null-type tool_call 400 errors poison conversation history — always restart fresh and
        //     permanently disable --continue so the corrupt state is never reloaded.
        const isCAPIError = isTransientCAPIError(result.output);
        const isMCPPolicy = isMCPPolicyError(result.output);
        const isModelNotSupported = isModelNotSupportedError(result.output);
        const isAuthErr = isNoAuthInfoError(result.output);
        const isAuthenticationFailed = isAuthenticationFailedError(result.output);
        const proxyAuthDiagnostic = buildCopilotProxyAuthFailureDiagnostic(result.output, process.env);
        const isNullTypeToolCall = isNullTypeToolCallError(result.output);
        const isMaxEffectiveTokensExceeded = isMaxEffectiveTokensExceededError(result.output);
        const permissionDeniedCount = countPermissionDeniedIssues(result.output);
        const hasNumerousPermissionDenied = hasNumerousPermissionDeniedIssues(result.output);
        log(
          `attempt ${attempt + 1} failed:` +
            ` exitCode=${result.exitCode}` +
            ` isCAPIError400=${isCAPIError}` +
            ` isMCPPolicyError=${isMCPPolicy}` +
            ` isModelNotSupportedError=${isModelNotSupported}` +
            ` isNullTypeToolCallError=${isNullTypeToolCall}` +
            ` isMaxEffectiveTokensExceededError=${isMaxEffectiveTokensExceeded}` +
            ` isAuthError=${isAuthErr}` +
            ` isAuthenticationFailedError=${isAuthenticationFailed}` +
            ` permissionDeniedCount=${permissionDeniedCount}` +
            ` hasNumerousPermissionDenied=${hasNumerousPermissionDenied}` +
            ` hasOutput=${result.hasOutput}` +
            ` retriesRemaining=${MAX_RETRIES - attempt}`
        );

        if (attempt === 0 && isAuthenticationFailed) {
          if (proxyAuthDiagnostic) {
            log(`attempt ${attempt + 1}: ${proxyAuthDiagnostic} — not retrying (first-attempt auth failure is non-retryable)`);
          } else {
            log(`attempt ${attempt + 1}: authentication failed — not retrying (first-attempt auth failure is non-retryable)`);
          }
          break;
        }

        if (hasNumerousPermissionDenied) {
          const deniedCommands = extractDeniedCommands(result.output);
          emitMissingToolPermissionIssue({ deniedCommands, logger: log });
          log(`attempt ${attempt + 1}: detected numerous permission-denied issues — not retrying (classified as missing tool/permission issue)`);
          break;
        }

        // MCP policy errors are persistent — retrying will not help.
        if (isMCPPolicy) {
          log(`attempt ${attempt + 1}: MCP servers blocked by policy — not retrying (this is a policy configuration issue, not a transient error)`);
          break;
        }

        // Model-not-supported errors are persistent — retrying will not help.
        if (isModelNotSupported) {
          if (!modelNotSupportedReflectRetryAttempted && attempt < MAX_RETRIES && isDetectionPhase(process.env.GH_AW_PHASE) && process.env.AWF_REFLECT_ENABLED === "1") {
            const configuredModel = process.env.COPILOT_MODEL || "";
            modelNotSupportedReflectRetryAttempted = true;
            log(`attempt ${attempt + 1}: model not supported during detection — refreshing awf-reflect to rule out startup registry race`);
            await fetchAWFReflect({ logger: log });
            if (isModelAvailableInReflectFile(configuredModel, { logger: log })) {
              useContinueOnRetry = false;
              continueDisabledPermanently = true;
              log(`attempt ${attempt + 1}: refreshed awf-reflect now includes model '${configuredModel}' — retrying once as fresh run`);
              continue;
            }
            log(`attempt ${attempt + 1}: refreshed awf-reflect does not include model '${configuredModel || "(none)"}' — treating as non-retryable`);
          }
          log(`attempt ${attempt + 1}: model not supported — not retrying (the requested model is unavailable for this subscription tier; specify a supported model in the workflow frontmatter)`);
          break;
        }

        if (isMaxEffectiveTokensExceeded) {
          log(`attempt ${attempt + 1}: AWF effective-token hard rail hit — not retrying or continuing (further inference will be refused until budget resets)`);
          break;
        }

        // Auth error: behavior depends on whether this was a --continue attempt (CLI mode only).
        // On a --continue attempt: the Copilot CLI's on-disk session credential written by the
        // interrupted run may be incomplete/invalid.  Fall back to a fresh run (without --continue)
        // once so env-var auth can succeed.  Mid-stream context is lost but the job can recover.
        // On a fresh run: the auth token is genuinely absent or invalid — retrying will not help.
        if (isAuthErr) {
          if (useContinueOnRetry && attempt < MAX_RETRIES) {
            useContinueOnRetry = false;
            continueDisabledPermanently = true;
            log(`attempt ${attempt + 1}: auth error on --continue — retrying as fresh run (session credential may be corrupted; context will be lost)`);
            continue;
          }
          log(`attempt ${attempt + 1}: no authentication information found — not retrying (COPILOT_GITHUB_TOKEN, GH_TOKEN, and GITHUB_TOKEN are all absent or invalid)`);
          break;
        }

        // Null-type tool_call error: the model emitted a malformed tool call that poisons the
        // conversation history.  Retrying with --continue re-injects the same broken history and
        // produces the same 400 on every subsequent attempt.  Restart fresh to discard the poisoned
        // history, and permanently disable --continue so the corrupt state is never re-loaded.
        if (isNullTypeToolCall) {
          if (attempt < MAX_RETRIES && result.hasOutput) {
            const priorMode = attempt > 0 && useContinueOnRetry ? "--continue" : "fresh run";
            useContinueOnRetry = false;
            continueDisabledPermanently = true;
            log(`attempt ${attempt + 1}: null-type tool_call error (${priorMode}) — restarting fresh (poisoned history discarded; --continue disabled permanently)`);
            continue;
          }
        }

        // Scheduled runs: retry once on exit code 2 even when no output was produced.
        // This specifically targets transient Copilot API outages at startup where there is no
        // partial session state to continue from.
        if (isScheduledRun && result.exitCode === 2 && !result.hasOutput && scheduledExit2Retries < MAX_SCHEDULED_EXIT2_RETRIES && attempt < MAX_RETRIES) {
          scheduledExit2Retries += 1;
          scheduledExit2RetryAttempted = true;
          useContinueOnRetry = false;
          log(`attempt ${attempt + 1}: scheduled startup interruption (exit code 2, no output)` + ` — retrying once as fresh run (startupRetry=${scheduledExit2Retries}/${MAX_SCHEDULED_EXIT2_RETRIES})`);
          continue;
        }
        if (isScheduledRun && result.exitCode === 2 && !result.hasOutput && scheduledExit2Retries < MAX_SCHEDULED_EXIT2_RETRIES && attempt >= MAX_RETRIES) {
          log(`attempt ${attempt + 1}: scheduled startup interruption detected but retry budget exhausted — no attempts remain`);
        }

        if (attempt < MAX_RETRIES && result.hasOutput) {
          const reason = isCAPIError ? "CAPIError 400 (transient)" : "partial execution";
          // --continue is only meaningful in CLI mode; SDK mode always restarts fresh.
          useContinueOnRetry = !copilotSDKMode && !continueDisabledPermanently;
          const retryMode = useContinueOnRetry ? "--continue" : copilotSDKMode ? "fresh run" : "fresh run (--continue permanently disabled)";
          log(`attempt ${attempt + 1}: ${reason} — will retry with ${retryMode} (attempt ${attempt + 2}/${MAX_RETRIES + 1})`);
          continue;
        }

        if (attempt >= MAX_RETRIES) {
          log(`all ${MAX_RETRIES} retries exhausted — giving up (exitCode=${lastExitCode})`);
        } else {
          log(`attempt ${attempt + 1}: no output produced — not retrying` + ` (possible causes: binary not found, permission denied, auth failure, or silent startup crash)`);
        }

        // Non-retryable error or retries exhausted — propagate exit code
        break;
      }

      if (isScheduledRun && lastExitCode === 2 && scheduledExit2RetryAttempted) {
        emitInfrastructureIncomplete("Copilot API interruption (exit code 2) persisted after automatic retry in scheduled workflow run.");
      }
    }

    // Fetch AWF API proxy reflection data and persist to disk for post-run step summary.
    // This is best-effort: failures are logged but do not affect the agent exit code.
    // Skip when AWF_REFLECT_ENABLED is not "1" (e.g. sandbox.agent: false — no api-proxy running).
    if (process.env.AWF_REFLECT_ENABLED === "1") {
      await fetchAWFReflect({ logger: log });
    }
  } finally {
    await stopCopilotSDKServer(copilotSDKServer, { logger: log });
  }
  log(`done: exitCode=${lastExitCode} totalDuration=${formatDuration(Date.now() - driverStartTime)}`);
  process.exit(lastExitCode);
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    AWF_API_PROXY_REFLECT_URL,
    AWF_REFLECT_OUTPUT_PATH,
    AWF_REFLECT_TIMEOUT_MS,
    AWF_MODELS_URL_TIMEOUT_MS,
    GEMINI_MODEL_NAME_PREFIX,
    PROMPT_FILE_INLINE_THRESHOLD_BYTES,
    appendSafeOutputLine,
    buildMissingToolAlternatives,
    buildPromptFileFallbackInstruction,
    buildInfrastructureIncompletePayload,
    emitInfrastructureIncomplete,
    emitMissingToolPermissionIssue,
    enrichReflectModels,
    extractModelIds,
    extractDeniedCommands,
    fetchAWFReflect,
    fetchModelsFromUrl,
    buildCopilotProxyAuthFailureDiagnostic,
    generateCopilotConnectionToken,
    buildCopilotSDKServerArgs,
    getCopilotSDKServerPort,
    isDetectionPhase,
    isModelAvailableInReflectData,
    isModelAvailableInReflectFile,
    resolveCopilotSDKCustomProviderFromReflect,
    countPermissionDeniedIssues,
    detectCopilotErrors,
    hasNumerousPermissionDeniedIssues,
    INFERENCE_ACCESS_ERROR_PATTERN,
    AGENTIC_ENGINE_TIMEOUT_PATTERN,
    buildMissingToolPermissionIssuePayload,
    isMaxEffectiveTokensExceededError,
    isAuthenticationFailedError,
    startCopilotSDKServer,
    stopCopilotSDKServer,
    waitForCopilotSDKServer,
    writeCopilotOutputs,
    resolvePromptFileArgs,
    extractPromptFromArgs,
    readSDKOptionsFromStdin,
    parseCopilotSDKServerArgsFromEnv,
    runWithCopilotSDK,
  };
}

if (require.main === module) {
  main().catch(err => {
    log(`unexpected error: ${err.message}`);
    process.exit(1);
  });
}
