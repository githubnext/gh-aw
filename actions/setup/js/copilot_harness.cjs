// @ts-check

/**
 * Copilot CLI Harness with Retry Logic
 *
 * Wraps the Copilot CLI command with retry logic for failures that occur after the session
 * has been partially executed.  Passes all arguments to the copilot subprocess, transparently
 * forwarding stdin/stdout/stderr.
 *
 * Retry policy:
 *   - If the process produced any output (hasOutput) and exits with a non-zero code, the
 *     session is considered partially executed.  The driver retries with --continue so the
 *     Copilot CLI can continue from where it left off.
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

const fs = require("fs");
const path = require("path");
const { runProcess, formatDuration, sleep, isCopilotSDKEnabled, buildCopilotSDKEnv } = require("./process_runner.cjs");
const { buildCopilotSDKServerArgs, getCopilotSDKServerPort, startCopilotSDKServer, stopCopilotSDKServer, waitForCopilotSDKServer } = require("./copilot_sdk_sidecar.cjs");
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
const NO_AUTH_INFO_PATTERN = /No authentication information found/;
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
const PERMISSION_DENIED_PATTERN = /\b(?:permission denied|permissions denied|EACCES|EPERM)\b/gi;
const NUMEROUS_PERMISSION_DENIED_THRESHOLD = 3;

/**
 * Emit a timestamped diagnostic log line to stderr.
 * All driver messages are prefixed with "[copilot-harness]" so they are easy to
 * grep out of the combined agent-stdio.log.
 * @param {string} message
 */
function log(message) {
  const ts = new Date().toISOString();
  process.stderr.write(`[copilot-harness] ${ts} ${message}\n`);
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
 * Count permission-denied indicators in process output.
 * @param {string} output
 * @returns {number}
 */
function countPermissionDeniedIssues(output) {
  if (!output) return 0;
  const matches = output.match(PERMISSION_DENIED_PATTERN);
  return matches ? matches.length : 0;
}

/**
 * Detect whether output contains numerous permission-denied issues.
 * @param {string} output
 * @returns {boolean}
 */
function hasNumerousPermissionDeniedIssues(output) {
  return countPermissionDeniedIssues(output) >= NUMEROUS_PERMISSION_DENIED_THRESHOLD;
}

/**
 * Extract the commands that were denied from process output.
 * Scans for lines using the Copilot CLI pipe marker (│) that appear
 * within three lines before each "permission denied" occurrence.
 * Returns a deduplicated array of command strings (may be empty if
 * the output format does not contain extractable commands).
 * @param {string} output
 * @returns {string[]}
 */
function extractDeniedCommands(output) {
  if (!output) return [];
  const lines = output.split("\n");
  const deniedCommands = new Set();
  for (let i = 0; i < lines.length; i++) {
    if (/\bpermission denied\b/i.test(lines[i])) {
      // Look back up to 3 lines for a command displayed with the
      // Copilot CLI box-drawing pipe marker (│ U+2502) or plain pipe (|).
      for (let j = i - 1; j >= Math.max(0, i - 3); j--) {
        const cmdMatch = lines[j].match(/^\s*[\u2502|]\s+(.+)\s*$/);
        if (cmdMatch && cmdMatch[1].trim()) {
          deniedCommands.add(cmdMatch[1].trim());
          break;
        }
      }
    }
  }
  return [...deniedCommands];
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
 * Build a structured missing_tool payload for repeated permission-denied failures.
 * @param {string[]} [deniedCommands] - Commands that were denied (may be empty)
 * @returns {string}
 */
function buildMissingToolPermissionIssuePayload(deniedCommands) {
  return JSON.stringify({
    type: "missing_tool",
    tool: "tool/permission",
    reason: "missing tool/permission issue: numerous permission denied errors detected",
    alternatives: "Verify token scopes, repository permissions, and MCP/tool access configuration.",
    denied_commands: deniedCommands && deniedCommands.length > 0 ? deniedCommands : [],
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
  const resolvedArgs = resolvePromptFileArgs(args);

  // Build SDK env additions. When COPILOT_SDK_URI is set the harness will start a separate
  // headless Copilot CLI sidecar and this helper merges COPILOT_SDK_URI into the child
  // process env so that every started process (including retry attempts) inherits the
  // correct SDK endpoint URI.
  const sdkEnv = buildCopilotSDKEnv();
  const copilotSDKMode = isCopilotSDKEnabled();
  if (copilotSDKMode) {
    log(`copilot-sdk mode active: COPILOT_SDK_URI=${sdkEnv.COPILOT_SDK_URI || "(not set)"}`);
  }
  // Merge SDK env additions into the child process env only when the SDK helper
  // returned at least one variable; otherwise leave the env undefined so that
  // runProcess inherits the full process.env (the common case).
  const childEnv = Object.keys(sdkEnv).length > 0 ? { ...process.env, ...sdkEnv } : undefined;

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
  /** @type {Awaited<ReturnType<typeof startCopilotSDKServer>>} */
  let copilotSDKServer = null;
  try {
    if (copilotSDKMode) {
      copilotSDKServer = await startCopilotSDKServer({
        command,
        env: childEnv ?? process.env,
        logger: log,
      });
    }

    for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
      // Add --continue flag on retries so the copilot session continues from where it left off
      const currentArgs = attempt > 0 && useContinueOnRetry ? [...resolvedArgs, "--continue"] : resolvedArgs;

      if (attempt > 0) {
        const retryMode = useContinueOnRetry ? "--continue" : "fresh run";
        log(`retry ${attempt}/${MAX_RETRIES}: sleeping ${delay}ms before next attempt (${retryMode})`);
        await sleep(delay);
        delay = Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS);
        log(`retry ${attempt}/${MAX_RETRIES}: woke up, next delay cap will be ${Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS)}ms`);
      }

      // Redact --prompt / -p value from logs to avoid leaking prompt content
      const safeArgs = currentArgs.map((arg, i) => (currentArgs[i - 1] === "--prompt" || currentArgs[i - 1] === "-p" ? "<redacted>" : arg));
      const result = await runProcess({ command, args: currentArgs, attempt, log, logArgs: safeArgs, env: childEnv });
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
      // Retry whenever the session was partially executed (hasOutput), using --continue so that
      // the Copilot CLI can continue from where it left off.  CAPIError 400 is the well-known
      // transient case, but any partial-execution failure is eligible for a continue retry.
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
      const isNullTypeToolCall = isNullTypeToolCallError(result.output);
      const permissionDeniedCount = countPermissionDeniedIssues(result.output);
      const hasNumerousPermissionDenied = hasNumerousPermissionDeniedIssues(result.output);
      log(
        `attempt ${attempt + 1} failed:` +
          ` exitCode=${result.exitCode}` +
          ` isCAPIError400=${isCAPIError}` +
          ` isMCPPolicyError=${isMCPPolicy}` +
          ` isModelNotSupportedError=${isModelNotSupported}` +
          ` isNullTypeToolCallError=${isNullTypeToolCall}` +
          ` isAuthError=${isAuthErr}` +
          ` isAuthenticationFailedError=${isAuthenticationFailed}` +
          ` permissionDeniedCount=${permissionDeniedCount}` +
          ` hasNumerousPermissionDenied=${hasNumerousPermissionDenied}` +
          ` hasOutput=${result.hasOutput}` +
          ` retriesRemaining=${MAX_RETRIES - attempt}`
      );

      if (attempt === 0 && isAuthenticationFailed) {
        log(`attempt ${attempt + 1}: authentication failed — not retrying (first-attempt auth failure is non-retryable)`);
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

      // Auth error: behavior depends on whether this was a --continue attempt.
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
        useContinueOnRetry = !continueDisabledPermanently;
        const retryMode = useContinueOnRetry ? "--continue" : "fresh run (--continue permanently disabled)";
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
    buildCopilotSDKServerArgs,
    getCopilotSDKServerPort,
    isDetectionPhase,
    isModelAvailableInReflectData,
    isModelAvailableInReflectFile,
    countPermissionDeniedIssues,
    detectCopilotErrors,
    hasNumerousPermissionDeniedIssues,
    INFERENCE_ACCESS_ERROR_PATTERN,
    AGENTIC_ENGINE_TIMEOUT_PATTERN,
    buildMissingToolPermissionIssuePayload,
    isAuthenticationFailedError,
    startCopilotSDKServer,
    stopCopilotSDKServer,
    waitForCopilotSDKServer,
    writeCopilotOutputs,
    resolvePromptFileArgs,
  };
}

if (require.main === module) {
  main().catch(err => {
    log(`unexpected error: ${err.message}`);
    process.exit(1);
  });
}
