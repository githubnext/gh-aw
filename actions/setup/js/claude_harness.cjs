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
 *   - Retries use exponential backoff: 5s → 10s → 20s (capped at 60s).
 *   - Maximum 3 retry attempts after the initial run.
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

const fs = require("fs");
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

// Pattern to detect Anthropic API overload errors (HTTP 529).
// Matches "overloaded_error" from the Anthropic error type field, and the
// "Overloaded" human-readable message that Claude Code emits in its stream-json output.
const OVERLOADED_ERROR_PATTERN = /overloaded_error|"overloaded"/i;

// Pattern to detect Anthropic rate-limit errors (HTTP 429).
const RATE_LIMIT_ERROR_PATTERN = /rate_limit_error|429 Too Many Requests/i;

// Pattern to detect a clean max-turns exit from Claude Code.
// Claude Code emits a JSON result object with "subtype":"error_max_turns" when the
// session ends because the turn limit was reached.  This is a deterministic terminal
// condition — --continue cannot recover it because no deferred tool marker was written.
const MAX_TURNS_EXIT_PATTERN = /"subtype"\s*:\s*"error_max_turns"/;

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
 * Resolve --prompt-file arguments for the initial Claude run.
 * Strips the --prompt-file <path> pair from args and appends the file content
 * as the last positional argument, which is where Claude Code expects the prompt.
 *
 * For --continue retries the prompt should be omitted entirely (Claude resumes
 * from its on-disk session state).  Call this function only for the initial run.
 *
 * @param {string[]} args
 * @returns {string[]} Args with --prompt-file resolved to inline prompt content
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
      throw new Error(`--prompt-file '${promptFile}' is not readable: ${err.message}`);
    }
    i++; // Skip the prompt-file path argument
  }

  // Append the prompt content as the last positional argument (Claude Code convention).
  if (promptContent !== null) {
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
 * Main entry point: run claude with retry logic for transient API failures.
 */
async function main() {
  const [, , command, ...args] = process.argv;

  if (!command) {
    process.stderr.write("claude-harness: Usage: node claude_harness.cjs <command> [args...]\n");
    process.exit(1);
  }

  log(`starting: command=${command} maxRetries=${MAX_RETRIES} initialDelayMs=${INITIAL_DELAY_MS}` + ` backoffMultiplier=${BACKOFF_MULTIPLIER} maxDelayMs=${MAX_DELAY_MS}` + ` nodeVersion=${process.version} platform=${process.platform}`);

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
  // Args without --prompt-file, used as the base for --continue retries.
  const continueBaseArgs = stripPromptFileArgs(args);

  // Detect whether the original args included --prompt-file so we know whether
  // initialArgs carries prompt text as its last positional arg.
  const hadPromptFile = args.includes("--prompt-file");

  // Safe arg list for logging: when --prompt-file was present, the last element of
  // initialArgs is the resolved prompt content. Replace it with a placeholder so that
  // task instructions are never written to stderr or captured in agent logs.
  const safeInitialArgs = hadPromptFile && initialArgs.length > 0 ? [...initialArgs.slice(0, -1), "<prompt omitted>"] : initialArgs;

  // Fetch AWF API proxy reflection data before running the agent to capture initial proxy state.
  // This is best-effort: failures are logged but do not affect the agent run.
  await fetchAWFReflect({ logger: log });

  let delay = INITIAL_DELAY_MS;
  let lastExitCode = 1;
  let useContinueOnRetry = false;
  const driverStartTime = Date.now();

  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    // For --continue retries: omit the original prompt and add --continue.
    // Claude Code resumes the session from on-disk state; re-sending the original
    // instructions would re-execute the full task from scratch.
    let currentArgs;
    if (attempt > 0 && useContinueOnRetry) {
      currentArgs = [...continueBaseArgs, "--continue"];
    } else {
      currentArgs = initialArgs;
    }

    // Use redacted args for logging when the run carries the prompt text.
    const logArgs = attempt === 0 || !useContinueOnRetry ? safeInitialArgs : currentArgs;

    if (attempt > 0) {
      const retryMode = useContinueOnRetry ? "--continue" : "fresh run";
      log(`retry ${attempt}/${MAX_RETRIES}: sleeping ${delay}ms before next attempt (${retryMode})`);
      await sleep(delay);
      delay = Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS);
      log(`retry ${attempt}/${MAX_RETRIES}: woke up, next delay cap will be ${Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS)}ms`);
    }

    const result = await runProcess({ command, args: currentArgs, attempt, log, logArgs });
    lastExitCode = result.exitCode;

    // Success — stop retrying
    if (result.exitCode === 0) {
      log(`success on attempt ${attempt + 1}: totalDuration=${formatDuration(Date.now() - driverStartTime)}`);
      lastExitCode = 0;
      break;
    }

    const isOverloaded = isOverloadedError(result.output);
    const isRateLimit = isRateLimitError(result.output);
    const isMaxTurns = isMaxTurnsExit(result.output);
    log(
      `attempt ${attempt + 1} failed:` +
        ` exitCode=${result.exitCode}` +
        ` isOverloadedError=${isOverloaded}` +
        ` isRateLimitError=${isRateLimit}` +
        ` isMaxTurnsExit=${isMaxTurns}` +
        ` hasOutput=${result.hasOutput}` +
        ` retriesRemaining=${MAX_RETRIES - attempt}`
    );

    // max_turns is a deterministic terminal condition: the session ended cleanly after
    // exhausting the allowed number of turns.  --continue cannot resume it because no
    // deferred tool marker was written.  Retrying would immediately fail with "No deferred
    // tool marker found", wasting time and masking the real exit reason.
    if (isMaxTurns) {
      log(`attempt ${attempt + 1}: max_turns exit — not retriable via --continue`);
      break;
    }

    // Retry when the session was partially executed (has output).
    // Use --continue so Claude Code can resume from its saved session state.
    if (attempt < MAX_RETRIES && result.hasOutput) {
      const reason = isOverloaded ? "overloaded_error (transient)" : isRateLimit ? "rate_limit_error (transient)" : "partial execution";
      useContinueOnRetry = true;
      log(`attempt ${attempt + 1}: ${reason} — will retry with --continue (attempt ${attempt + 2}/${MAX_RETRIES + 1})`);
      continue;
    }

    if (attempt >= MAX_RETRIES) {
      log(`all ${MAX_RETRIES} retries exhausted — giving up (exitCode=${lastExitCode})`);
    } else {
      log(`attempt ${attempt + 1}: no output produced — not retrying` + ` (possible causes: binary not found, permission denied, auth failure, or silent startup crash)`);
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
    isMaxTurnsExit,
  };
}

if (require.main === module) {
  main().catch(err => {
    log(`unexpected error: ${err.message}`);
    process.exit(1);
  });
}
