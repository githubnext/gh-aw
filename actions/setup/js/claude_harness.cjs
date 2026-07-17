// @ts-check

/**
 * Claude Code CLI Harness with Retry Logic
 *
 * Wraps the Claude Code CLI command with retry logic for failures that occur after the session
 * has been partially executed.  Passes all arguments to the claude subprocess, transparently
 * forwarding stdin/stdout/stderr.
 *
 * Retry policy:
 *   - If the process produced any output (hasOutput) and exits with a non-zero code, the
 *     session is considered partially executed.  The driver retries with --continue so the
 *     Claude Code CLI can continue from where it left off.
 *   - Overloaded API errors (HTTP 529 / "overloaded_error") and rate-limit errors (HTTP 429 /
 *     "rate_limit_error") are well-known transient failure modes and are logged explicitly, but
 *     any partial-execution failure is retried — not just those specific errors.
 *   - If the process produced no output (failed to start / auth error before any work), the
 *     driver does not retry because there is nothing to resume.
 *   - On a `--continue` retry the initial prompt is omitted: Claude Code resumes the session
 *     from its on-disk state rather than re-processing the original instructions.
 *   - Retries use exponential backoff: 5s → 10s → 20s (capped at 60s) by default.
 *   - Maximum 3 retry attempts after the initial run by default.
 *
 * Prompt handling:
 *   - The harness expects a `--prompt-file <path>` argument in the args list.
 *   - For the initial run it reads the file and appends the content as the last positional arg.
 *   - For `--continue` retries the prompt is omitted (Claude resumes from session state).
 *
 * Usage: node claude_harness.cjs <command> [args...]
 * Example: node claude_harness.cjs claude --print --prompt-file /tmp/gh-aw/aw-prompts/prompt.txt
 */

"use strict";

const { getErrorMessage } = require("./error_helpers.cjs");
const fs = require("fs");
const { runProcess, formatDuration, sleep } = require("./process_runner.cjs");
const { resolveRetryConfig: resolveSharedRetryConfig } = require("./harness_retry_config.cjs");
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
  normalizeReflectProviderName,
  resolveProviderEndpointFromReflect,
} = require("./awf_reflect.cjs");
const { emitMissingToolPermissionIssue, hasExpectedSafeOutputs, hasNoopInSafeOutputs } = require("./safeoutputs_cli.cjs");
const { countPermissionDeniedIssues, hasNumerousPermissionDeniedIssues, extractDeniedCommands, buildMissingToolPermissionIssuePayload } = require("./permission_denied_helpers.cjs");
const { detectNonRetryableHarnessGuard, buildSoftTimeoutGuard, emitSoftTimeoutSignal, isAuthenticationFailedError } = require("./harness_retry_guard.cjs");
const { MODEL_NOT_SUPPORTED_PATTERN: INVALID_MODEL_ERROR_PATTERN } = require("./detect_agent_errors.cjs");

// Pattern to detect Anthropic API overload errors (HTTP 529).
// Matches "overloaded_error" from the Anthropic error type field, and the
// "Overloaded" human-readable message that Claude Code emits in its stream-json output.
const OVERLOADED_ERROR_PATTERN = /overloaded_error|"overloaded"/i;

// Pattern to detect Anthropic rate-limit errors (HTTP 429).
// Claude CLI may surface this as:
//   - transport-style text (e.g. "429 Too Many Requests")
//   - embedded stream-json result fields (e.g. "api_error_status":429)
//   - human-readable message text ("rate limit")
const RATE_LIMIT_ERROR_PATTERN = /rate_limit_error|429 Too Many Requests|"api_error_status"\s*:\s*429|request rejected \(429\)|rate limit/i;

// Pattern to detect a clean max-turns exit from Claude Code.
// Claude Code emits a JSON result object with "subtype":"error_max_turns" when the
// session ends because the turn limit was reached.  This is a deterministic terminal
// condition — --continue cannot recover it because no deferred tool marker was written.
const MAX_TURNS_EXIT_PATTERN = /"subtype"\s*:\s*"error_max_turns"/;

// Pattern to detect a "no deferred tool marker" error from Claude Code.
// This occurs when --continue is attempted but the session either was never deferred,
// the deferred marker is stale (tool already ran), or it falls outside the tail-scan
// window.  Retrying with --continue will always produce the same instant failure, so
// this path must not be retried via --continue (fall back to a fresh run if budget remains).
const NO_DEFERRED_MARKER_PATTERN = /No deferred tool marker found/i;
const SIGNAL_TERMINATION_EXIT_CODES = new Set([137, 143]);
const MAX_STARTUP_RETRIES = 2;

/**
 * Emit a timestamped diagnostic log line to stderr.
 * All driver messages are prefixed with "[claude-harness]" so they are easy to
 * grep out of the combined agent-stdio.log.
 * @param {string} message
 */
function log(message) {
  const ts = new Date().toISOString();
  process.stderr.write(`[claude-harness] ${ts} ${message}\n`);
}

/**
 * @param {NodeJS.ProcessEnv} [env]
 * @param {(message: string) => void} [logger]
 * @returns {{maxRetries: number, initialDelayMs: number, backoffMultiplier: number, maxDelayMs: number}}
 */
function resolveRetryConfig(env = process.env, logger = log) {
  return resolveSharedRetryConfig(env, logger);
}

/**
 * Parse bounded startup retries for zero-output startup failures.
 * Retries are always fresh runs (never --continue).
 *
 * @param {NodeJS.ProcessEnv} [env]
 * @param {(message: string) => void} [logger]
 * @returns {number}
 */
function resolveStartupRetryLimit(env = process.env, logger = log) {
  const raw = env.GH_AW_CLAUDE_STARTUP_RETRIES;
  if (raw == null || raw === "") {
    return 1;
  }
  if (!/^[+-]?\d+$/.test(raw)) {
    logger(`invalid GH_AW_CLAUDE_STARTUP_RETRIES='${raw}' (expected integer); using default startup retries=1`);
    return 1;
  }
  const parsed = Number.parseInt(raw, 10);
  if (!Number.isFinite(parsed)) {
    logger(`invalid GH_AW_CLAUDE_STARTUP_RETRIES='${raw}' (not finite); using default startup retries=1`);
    return 1;
  }
  const clamped = Math.min(Math.max(parsed, 0), MAX_STARTUP_RETRIES);
  if (clamped !== parsed) {
    logger(`GH_AW_CLAUDE_STARTUP_RETRIES=${parsed} out of range; clamped to ${clamped}`);
  }
  return clamped;
}

/**
 * Determines if the collected output contains an Anthropic overload error.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isOverloadedError(output) {
  return OVERLOADED_ERROR_PATTERN.test(output);
}

/**
 * Determines if the collected output contains an Anthropic rate-limit error.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isRateLimitError(output) {
  return RATE_LIMIT_ERROR_PATTERN.test(output);
}

/**
 * Determines if the collected output signals a clean max-turns exit.
 * When Claude Code hits its turn limit it emits a result object with
 * "subtype":"error_max_turns".  This is not a transient error — retrying
 * with --continue will always fail because no deferred tool marker was written.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isMaxTurnsExit(output) {
  return MAX_TURNS_EXIT_PATTERN.test(output);
}

/**
 * Determines if the collected output contains a "no deferred tool marker" error.
 * This occurs when Claude Code is invoked with --continue but the session was never
 * deferred, the deferred marker is stale (tool already ran), or it falls outside the
 * tail-scan window.  Each retry with --continue will instantly produce the same error,
 * so this should not be retried via --continue (fall back to fresh run retries).
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isNoDeferredMarkerError(output) {
  return NO_DEFERRED_MARKER_PATTERN.test(output);
}

/**
 * Determines if the collected output indicates an invalid or unavailable model name.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isInvalidModelError(output) {
  return INVALID_MODEL_ERROR_PATTERN.test(output);
}

/**
 * Determines whether the exit code corresponds to signal-style termination
 * (SIGKILL=137 / SIGTERM=143), typically from timeout/cancellation.
 * @param {number} exitCode
 * @returns {boolean}
 */
function isSignalTerminationExitCode(exitCode) {
  return SIGNAL_TERMINATION_EXIT_CODES.has(exitCode);
}

/**
 * Decide whether the next retry should use --continue.
 * @param {{
 *   attempt: number,
 *   maxRetries: number,
 *   exitCode: number,
 *   hasOutput: boolean,
 *   isNoDeferredMarker: boolean,
 *   continueDisabledPermanently: boolean
 * }} input
 * @returns {boolean}
 */
function shouldRetryWithContinue({ attempt, maxRetries, exitCode, hasOutput, isNoDeferredMarker, continueDisabledPermanently }) {
  if (attempt >= maxRetries || !hasOutput || continueDisabledPermanently) {
    return false;
  }
  if (isSignalTerminationExitCode(exitCode)) {
    return false;
  }
  if (isNoDeferredMarker) {
    return false;
  }
  return true;
}

/**
 * Resolve --prompt-file arguments for the initial Claude run.
 * Strips the --prompt-file <path> pair from args and appends -- followed by
 * the file content as the last positional argument.
 *
 * The end-of-options marker (--) is essential: Claude Code 2.x treats any
 * non-flag argument that follows --mcp-config as an additional config file
 * path (variadic flag).  Without --, a long prompt appended after
 * --mcp-config <path> would be used as a file path, producing an
 * ENAMETOOLONG error when the prompt exceeds PATH_MAX (~4096 bytes).
 *
 * For --continue retries the prompt should be omitted entirely (Claude resumes
 * from its on-disk session state).  Call this function only for the initial run.
 *
 * @param {string[]} args
 * @returns {string[]} Args with --prompt-file resolved to ["--", <content>]
 */
function resolveClaudePromptFileArgs(args) {
  /** @type {string[]} */
  const filteredArgs = [];
  /** @type {string|null} */
  let promptContent = null;

  for (let i = 0; i < args.length; i++) {
    if (args[i] !== "--prompt-file") {
      filteredArgs.push(args[i]);
      continue;
    }

    if (i + 1 >= args.length) {
      log("warning: --prompt-file provided without a path; leaving arguments unchanged");
      filteredArgs.push(args[i]);
      continue;
    }

    const promptFile = args[i + 1];
    try {
      const stat = fs.statSync(promptFile);
      log(`resolved --prompt-file: path=${promptFile} size=${stat.size}B`);
      promptContent = fs.readFileSync(promptFile, "utf8");
    } catch (error) {
      const err = /** @type {Error} */ error;
      // An unreadable prompt file means no task instructions can be delivered to Claude.
      // Propagate as a fatal error rather than forwarding the harness-only flag to the
      // claude subprocess (which would fail with an "unknown option" error).
      throw new Error(`--prompt-file '${promptFile}' is not readable: ${err.message}`, { cause: err });
    }
    i++; // Skip the prompt-file path argument
  }

  // Append an end-of-options marker followed by the prompt content.
  // The '--' prevents Claude Code from treating the prompt text as an additional
  // --mcp-config value (Claude Code 2.x accepts that flag variadically, so any
  // non-flag positional argument that follows --mcp-config <path> would otherwise
  // be tried as a second config file path, causing ENAMETOOLONG for long prompts).
  if (promptContent !== null) {
    filteredArgs.push("--");
    filteredArgs.push(promptContent);
  }

  return filteredArgs;
}

/**
 * Strip --prompt-file and its path argument from args.
 * Used for --continue retries where Claude resumes from on-disk session state
 * and should not be given the original prompt again.
 *
 * @param {string[]} args
 * @returns {string[]} Args with --prompt-file pair removed
 */
function stripPromptFileArgs(args) {
  /** @type {string[]} */
  const filteredArgs = [];
  for (let i = 0; i < args.length; i++) {
    if (args[i] === "--prompt-file" && i + 1 < args.length) {
      i++; // Skip path too
      continue;
    }
    filteredArgs.push(args[i]);
  }
  return filteredArgs;
}

/**
 * Strip any user-supplied --continue flags from args.
 * The harness decides when --continue should be used on retries.
 *
 * @param {string[]} args
 * @returns {string[]}
 */
function stripContinueArgs(args) {
  return args.filter(arg => arg !== "--continue");
}

/**
 * Build Claude child process env with provider endpoint overrides resolved from /reflect.
 * @returns {Promise<NodeJS.ProcessEnv>}
 */
async function buildClaudeChildEnv() {
  const childEnv = { ...process.env };
  const provider = normalizeReflectProviderName(process.env.GH_AW_LLM_PROVIDER, "anthropic");
  try {
    const raw = fs.readFileSync(AWF_REFLECT_OUTPUT_PATH, "utf8");
    const reflectData = JSON.parse(raw);
    const resolved = resolveProviderEndpointFromReflect({ provider, reflectData, logger: log });
    if (resolved && resolved.baseUrl) {
      childEnv.ANTHROPIC_BASE_URL = resolved.baseUrl;
      log(`configured ANTHROPIC_BASE_URL from /reflect for provider=${provider}: ${resolved.baseUrl}`);
    }
  } catch (error) {
    const err = /** @type {Error} */ error;
    log(`warning: unable to resolve provider endpoint from /reflect: ${err.message}`);
  }
  return childEnv;
}

/**
 * Main entry point: run claude with retry logic for transient API failures.
 */
async function main() {
  const [, , command, ...args] = process.argv;
  const retryConfig = resolveRetryConfig(process.env, log);
  const { maxRetries, initialDelayMs, backoffMultiplier, maxDelayMs } = retryConfig;
  const startupRetryLimit = resolveStartupRetryLimit(process.env, log);

  if (!command) {
    process.stderr.write("claude-harness: Usage: node claude_harness.cjs <command> [args...]\n");
    process.exit(1);
  }

  log(`starting: command=${command} maxRetries=${maxRetries} initialDelayMs=${initialDelayMs}` + ` backoffMultiplier=${backoffMultiplier} maxDelayMs=${maxDelayMs}` + ` nodeVersion=${process.version} platform=${process.platform}`);

  // Resolve the prompt for the initial run (reads --prompt-file content).
  // A missing or unreadable prompt file is treated as a fatal startup error.
  let initialArgs;
  try {
    initialArgs = resolveClaudePromptFileArgs(args);
  } catch (err) {
    const e = /** @type {Error} */ err;
    log(`fatal: ${e.message}`);
    process.exit(1);
  }
  const freshRetryArgs = stripContinueArgs(initialArgs);
  // Args without --prompt-file, used as the base for --continue retries.
  const continueBaseArgs = stripContinueArgs(stripPromptFileArgs(args));

  // Detect whether the original args included --prompt-file so we know whether
  // initialArgs carries prompt text as its last positional arg.
  const hadPromptFile = args.includes("--prompt-file");

  // Safe arg list for logging: when --prompt-file was present, the last two elements of
  // initialArgs are the -- end-of-options marker and the resolved prompt content.
  // Strip both and replace with a placeholder so task instructions are never written
  // to stderr or captured in agent logs.
  const safeInitialArgs = hadPromptFile && initialArgs.length > 0 ? [...initialArgs.slice(0, -2), "<prompt omitted>"] : initialArgs;
  const safeFreshRetryArgs = hadPromptFile && freshRetryArgs.length > 0 ? [...freshRetryArgs.slice(0, -2), "<prompt omitted>"] : freshRetryArgs;

  // Fetch AWF API proxy reflection data before running the agent to capture initial proxy state.
  // This is best-effort: failures are logged but do not affect the agent run.
  await fetchAWFReflect({ logger: log });
  const childEnv = await buildClaudeChildEnv();

  // Pre-flight: skip the agent entirely when a noop has already been written by a prior step.
  // A noop indicates the work is complete or there is nothing to do — starting the agent
  // would be wasteful and potentially harmful.
  const safeOutputsPath = process.env.GH_AW_SAFE_OUTPUTS || "";
  if (safeOutputsPath && hasNoopInSafeOutputs(safeOutputsPath, { logger: log })) {
    log("pre-flight: noop message found in safe-outputs — skipping agent (work is already complete or no work needed)");
    process.exit(0);
  }

  let delay = initialDelayMs;
  let lastExitCode = 1;
  let useContinueOnRetry = false;
  let continueDisabledPermanently = false;
  let startupRetriesUsed = 0;
  const driverStartTime = Date.now();
  // Soft-timeout guard: polled at the top of the retry loop and after each backoff sleep.
  // It does not preempt a running attempt — if a single invocation runs past the soft
  // deadline the guard fires on the next iteration. Individual attempts are expected to
  // complete within the SOFT_TIMEOUT_BUFFER_MS window.
  const softTimeoutGuard = buildSoftTimeoutGuard(driverStartTime);

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    if (softTimeoutGuard && Date.now() >= softTimeoutGuard.softDeadlineMs) {
      emitSoftTimeoutSignal(softTimeoutGuard, `before attempt ${attempt + 1}`, "Claude harness", log);
      lastExitCode = 1;
      break;
    }

    // For --continue retries: omit the original prompt and add --continue.
    // Claude Code resumes the session from on-disk state; re-sending the original
    // instructions would re-execute the full task from scratch.
    let currentArgs;
    if (attempt > 0 && useContinueOnRetry) {
      currentArgs = [...continueBaseArgs, "--continue"];
    } else {
      currentArgs = attempt === 0 ? initialArgs : freshRetryArgs;
    }

    // Use redacted args for logging when the run carries the prompt text.
    const logArgs = attempt === 0 ? safeInitialArgs : useContinueOnRetry ? currentArgs : safeFreshRetryArgs;

    if (attempt > 0) {
      const retryMode = useContinueOnRetry ? "--continue" : "fresh run";
      log(`retry ${attempt}/${maxRetries}: sleeping ${delay}ms before next attempt (${retryMode})`);
      await sleep(delay);
      delay = Math.min(delay * backoffMultiplier, maxDelayMs);
      log(`retry ${attempt}/${maxRetries}: woke up, next delay cap will be ${Math.min(delay * backoffMultiplier, maxDelayMs)}ms`);
      if (softTimeoutGuard && Date.now() >= softTimeoutGuard.softDeadlineMs) {
        emitSoftTimeoutSignal(softTimeoutGuard, "after backoff sleep", "Claude harness", log);
        lastExitCode = 1;
        break;
      }
    }

    const result = await runProcess({ command, args: currentArgs, attempt, log, logArgs, env: childEnv });
    lastExitCode = result.exitCode;

    // Success — stop retrying
    if (result.exitCode === 0) {
      log(`success on attempt ${attempt + 1}: totalDuration=${formatDuration(Date.now() - driverStartTime)}`);
      lastExitCode = 0;
      break;
    }

    const isOverloaded = isOverloadedError(result.output);
    const isRateLimit = isRateLimitError(result.output);
    const isAuthenticationFailed = isAuthenticationFailedError(result.output);
    const isMaxTurns = isMaxTurnsExit(result.output);
    const isNoDeferredMarker = isNoDeferredMarkerError(result.output);
    const isInvalidModel = isInvalidModelError(result.output);
    const permissionDeniedCount = countPermissionDeniedIssues(result.output);
    const hasNumerousPermissionDenied = hasNumerousPermissionDeniedIssues(result.output);
    log(
      `attempt ${attempt + 1} failed:` +
        ` exitCode=${result.exitCode}` +
        ` isOverloadedError=${isOverloaded}` +
        ` isRateLimitError=${isRateLimit}` +
        ` isAuthenticationFailedError=${isAuthenticationFailed}` +
        ` isMaxTurnsExit=${isMaxTurns}` +
        ` isNoDeferredMarkerError=${isNoDeferredMarker}` +
        ` isInvalidModelError=${isInvalidModel}` +
        ` permissionDeniedCount=${permissionDeniedCount}` +
        ` hasNumerousPermissionDenied=${hasNumerousPermissionDenied}` +
        ` hasOutput=${result.hasOutput}` +
        ` retriesRemaining=${maxRetries - attempt}`
    );

    // If a noop was written to safe-outputs during the failed run, the agent determined
    // there was nothing to do (or the user indicated so before the agent ran).  Retrying
    // would not produce different results and could waste resources.
    if (safeOutputsPath && hasNoopInSafeOutputs(safeOutputsPath, { logger: log })) {
      log(`attempt ${attempt + 1}: noop message found in safe-outputs — not retrying (work is already complete or no work needed)`);
      lastExitCode = 0;
      break;
    }

    const nonRetryableGuard = detectNonRetryableHarnessGuard(result.output);
    if (nonRetryableGuard.aiCreditsExceeded || nonRetryableGuard.awfAPIProxyBlockingRequests || nonRetryableGuard.maxRunsExceeded) {
      const reasons = [];
      if (nonRetryableGuard.aiCreditsExceeded) reasons.push("AI credits budget exceeded");
      if (nonRetryableGuard.awfAPIProxyBlockingRequests) reasons.push("AWF API proxy is blocking requests");
      if (nonRetryableGuard.maxRunsExceeded) reasons.push("maximum LLM invocations exceeded");
      log(`attempt ${attempt + 1}: ${reasons.join(" and ")} — not retrying (non-retryable guard condition)`);
      break;
    }

    if (attempt === 0 && isAuthenticationFailed) {
      log(`attempt ${attempt + 1}: authentication failed — not retrying (first-attempt auth failure is non-retryable)`);
      break;
    }

    if (isInvalidModel) {
      log(`attempt ${attempt + 1}: invalid/unsupported model configuration — not retrying (specify a valid engine model name in workflow frontmatter)`);
      break;
    }

    if (hasNumerousPermissionDenied) {
      // If the agent already produced expected safe-outputs, the permission-denied
      // signals are from optional/exploratory commands — not from the core task work.
      // Suppress the terminal verdict and exit 0 to avoid a false-red run.
      if (safeOutputsPath && hasExpectedSafeOutputs(safeOutputsPath, { logger: log })) {
        log(`attempt ${attempt + 1}: detected numerous permission-denied issues but safe-outputs already contain expected output — suppressing terminal verdict (false-red: core work succeeded)`);
        lastExitCode = 0;
        break;
      }
      const deniedCommands = extractDeniedCommands(result.output);
      emitMissingToolPermissionIssue({ deniedCommands, logger: log });
      log(`attempt ${attempt + 1}: detected numerous permission-denied issues — not retrying (classified as missing tool/permission issue)`);
      break;
    }

    // max_turns is a deterministic terminal condition: the session ended cleanly after
    // exhausting the allowed number of turns.  --continue cannot resume it because no
    // deferred tool marker was written.  Retrying would immediately fail with "No deferred
    // tool marker found", wasting time and masking the real exit reason.
    if (isMaxTurns) {
      log(`attempt ${attempt + 1}: max_turns exit — not retriable via --continue`);
      break;
    }

    // "No deferred tool marker found" is a deterministic terminal condition: the session
    // was never deferred, the marker is stale (tool already ran), or it falls outside the
    // tail-scan window. If this happens on a --continue attempt, restart fresh and disable
    // --continue permanently so we do not re-enter the same invalid retry path.
    if (isNoDeferredMarker) {
      if (attempt < maxRetries && result.hasOutput) {
        useContinueOnRetry = false;
        continueDisabledPermanently = true;
        log(`attempt ${attempt + 1}: no deferred tool marker on --continue — retrying as fresh run (failure_reason=harness_retry_path_invalid, --continue disabled permanently, attempt ${attempt + 2}/${maxRetries + 1})`);
        continue;
      }
      log(`attempt ${attempt + 1}: no deferred tool marker — not retriable via --continue (failure_reason=harness_retry_path_invalid)`);
      break;
    }

    // Retry when the session was partially executed (has output).
    // Use --continue so Claude Code can resume from its saved session state.
    if (attempt < maxRetries && result.hasOutput) {
      const isSignalTermination = isSignalTerminationExitCode(result.exitCode);
      const retryWithContinue = shouldRetryWithContinue({
        attempt,
        maxRetries,
        exitCode: result.exitCode,
        hasOutput: result.hasOutput,
        isNoDeferredMarker,
        continueDisabledPermanently,
      });
      if (isSignalTermination) {
        continueDisabledPermanently = true;
      }
      const reason = isSignalTermination
        ? `signal-style termination exitCode=${result.exitCode} (failure_reason=cancelled_or_timed_out)`
        : isOverloaded
          ? "overloaded_error (transient)"
          : isRateLimit
            ? "rate_limit_error (transient)"
            : "partial execution";
      useContinueOnRetry = retryWithContinue;
      const retryMode = retryWithContinue ? "--continue" : "fresh run (--continue disabled permanently)";
      log(`attempt ${attempt + 1}: ${reason} — will retry with ${retryMode} (attempt ${attempt + 2}/${maxRetries + 1})`);
      continue;
    }

    if (attempt >= maxRetries) {
      log(`all ${maxRetries} retries exhausted — giving up (exitCode=${lastExitCode})`);
    } else {
      if (startupRetriesUsed < startupRetryLimit) {
        startupRetriesUsed++;
        useContinueOnRetry = false;
        delay = initialDelayMs;
        // attempt is 0-based; logs are 1-based, so "next attempt" is +2.
        const nextAttemptNumber = attempt + 2;
        const totalAttempts = maxRetries + 1;
        log(`attempt ${attempt + 1}: no output produced — retrying startup as fresh run ` + `(startup retry ${startupRetriesUsed}/${startupRetryLimit}, next attempt ${nextAttemptNumber} of ${totalAttempts} total attempts)`);
        continue;
      }
      log(
        `attempt ${attempt + 1}: no output produced — not retrying (startup retry budget exhausted: ${startupRetriesUsed}/${startupRetryLimit}; possible causes: binary not found, permission denied, auth failure, or silent startup crash)`
      );
    }

    break;
  }

  // Fetch AWF API proxy reflection data and persist to disk for post-run step summary.
  await fetchAWFReflect({ logger: log });

  log(`done: exitCode=${lastExitCode} totalDuration=${formatDuration(Date.now() - driverStartTime)}`);
  process.exit(lastExitCode);
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    resolveClaudePromptFileArgs,
    stripPromptFileArgs,
    isRateLimitError,
    isAuthenticationFailedError,
    isMaxTurnsExit,
    isNoDeferredMarkerError,
    isInvalidModelError,
    isSignalTerminationExitCode,
    shouldRetryWithContinue,
    countPermissionDeniedIssues,
    hasNumerousPermissionDeniedIssues,
    extractDeniedCommands,
    buildMissingToolPermissionIssuePayload,
    emitMissingToolPermissionIssue,
    hasNoopInSafeOutputs,
    hasExpectedSafeOutputs,
    resolveRetryConfig,
    resolveStartupRetryLimit,
  };
}

if (require.main === module) {
  main().catch(err => {
    log(`unexpected error: ${getErrorMessage(err)}`);
    process.exit(1);
  });
}
