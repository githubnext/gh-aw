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
const { runProcess, formatDuration, sleep } = require("./process_runner.cjs");
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

// Pattern to detect null-type tool_call error that poisons conversation history.
// Matches the Copilot API 400 error:
//   "Invalid type for '...tool_calls[N].type': expected one of 'function', ..., but got null instead."
// The model emitted a malformed tool call with type: null.  Retrying with --continue
// re-injects the same broken history, producing the same 400 on every subsequent attempt.
// A fresh restart is required to discard the poisoned history.
const NULL_TYPE_TOOL_CALL_PATTERN = /tool_calls\[.*?\]\.type.*null/;

/**
 * @typedef {(path: import("node:fs").PathOrFileDescriptor, data: string | Uint8Array, options?: import("node:fs").WriteFileOptions) => void} AppendFileSyncLike
 */

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
 * @param {AppendFileSyncLike} appendFileSync
 * @param {string} safeOutputsPath
 * @param {string} payload
 */
function appendSafeOutputLine(appendFileSync, safeOutputsPath, payload) {
  appendFileSync(safeOutputsPath, payload + "\n", { encoding: "utf8" });
}

/**
 * Append a structured report_incomplete signal when infrastructure failures prevent completion.
 * This allows downstream failure handling to classify transient infrastructure errors explicitly.
 * @param {string} details
 * @param {{
 *   safeOutputsPath?: string,
 *   appendFileSync?: AppendFileSyncLike,
 *   logger?: (message: string) => void
 * }=} options
 */
function emitInfrastructureIncomplete(details, options) {
  const safeOutputsPath = options && typeof options.safeOutputsPath === "string" ? options.safeOutputsPath : process.env.GH_AW_SAFE_OUTPUTS || "";
  const appendFileSync = options && options.appendFileSync ? options.appendFileSync : fs.appendFileSync;
  const logger = options && options.logger ? options.logger : log;

  if (!safeOutputsPath) {
    logger("report_incomplete skipped: GH_AW_SAFE_OUTPUTS is not set");
    return;
  }
  try {
    const payload = buildInfrastructureIncompletePayload(details);
    appendSafeOutputLine(appendFileSync, safeOutputsPath, payload);
    logger(`report_incomplete emitted: ${safeOutputsPath}`);
  } catch (error) {
    const err = /** @type {Error} */ error;
    logger(`report_incomplete emission failed: ${err.message}`);
  }
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

  // Fetch AWF API proxy reflection data before running the agent to capture initial proxy state.
  // This is best-effort: failures are logged but do not affect the agent run.
  await fetchAWFReflect({ logger: log });

  let delay = INITIAL_DELAY_MS;
  let lastExitCode = 1;
  const isScheduledRun = process.env.GITHUB_EVENT_NAME === "schedule";
  let scheduledExit2Retries = 0;
  let scheduledExit2RetryAttempted = false;
  let useContinueOnRetry = false;
  // Once set to true, --continue is never re-enabled for the remainder of this run.
  // This prevents a broken --continue recovery from resurrecting --continue on the next attempt.
  let continueDisabledPermanently = false;
  const driverStartTime = Date.now();

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
    const result = await runProcess({ command, args: currentArgs, attempt, log, logArgs: safeArgs });
    lastExitCode = result.exitCode;

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
    const isNullTypeToolCall = isNullTypeToolCallError(result.output);
    log(
      `attempt ${attempt + 1} failed:` +
        ` exitCode=${result.exitCode}` +
        ` isCAPIError400=${isCAPIError}` +
        ` isMCPPolicyError=${isMCPPolicy}` +
        ` isModelNotSupportedError=${isModelNotSupported}` +
        ` isNullTypeToolCallError=${isNullTypeToolCall}` +
        ` isAuthError=${isAuthErr}` +
        ` hasOutput=${result.hasOutput}` +
        ` retriesRemaining=${MAX_RETRIES - attempt}`
    );

    // MCP policy errors are persistent — retrying will not help.
    if (isMCPPolicy) {
      log(`attempt ${attempt + 1}: MCP servers blocked by policy — not retrying (this is a policy configuration issue, not a transient error)`);
      break;
    }

    // Model-not-supported errors are persistent — retrying will not help.
    if (isModelNotSupported) {
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
  await fetchAWFReflect({ logger: log });

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
    buildPromptFileFallbackInstruction,
    buildInfrastructureIncompletePayload,
    emitInfrastructureIncomplete,
    enrichReflectModels,
    extractModelIds,
    fetchAWFReflect,
    fetchModelsFromUrl,
    resolvePromptFileArgs,
  };
}

if (require.main === module) {
  main().catch(err => {
    log(`unexpected error: ${err.message}`);
    process.exit(1);
  });
}
