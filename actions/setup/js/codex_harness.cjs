// @ts-check

/**
 * OpenAI Codex CLI Harness with Retry Logic
 *
 * Wraps the OpenAI Codex CLI command with retry logic for failures that occur after the
 * session has been partially executed.  Passes all arguments to the codex subprocess,
 * transparently forwarding stdin/stdout/stderr.
 *
 * Retry policy:
 *   - If the process produced any output (hasOutput) and exits with a non-zero code, the
 *     session is considered partially executed.  The driver retries with a fresh run
 *     because Codex does not support a --continue-style session resumption.
 *   - Rate-limit errors (HTTP 429 / "rate_limit_exceeded") and server errors (HTTP 500,
 *     503) are well-known transient failure modes and are logged explicitly, but
 *     any partial-execution failure is retried — not just those specific errors.
 *   - If the process produced no output (failed to start / auth error before any work), the
 *     driver does not retry because there is nothing to resume.
 *   - Retries use exponential backoff: 5s → 10s → 20s (capped at 60s).
 *   - Maximum 3 retry attempts after the initial run.
 *
 * Prompt handling:
 *   - The harness expects a `--prompt-file <path>` argument in the args list.
 *   - It reads the file and appends the content as the last positional argument, which is
 *     where the Codex CLI (`codex exec`) expects the prompt.
 *   - The `--prompt-file` flag is a harness-only argument and is not forwarded to codex.
 *
 * Usage: node codex_harness.cjs <command> [args...]
 * Example: node codex_harness.cjs codex exec --dangerously-bypass-approvals-and-sandbox --skip-git-repo-check --prompt-file /tmp/gh-aw/aw-prompts/prompt.txt
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

// Pattern to detect OpenAI rate-limit errors (HTTP 429).
// Matches "rate_limit_exceeded" from the OpenAI error type field and the "429" status code
// that Codex emits when the API rate limit is hit.
const RATE_LIMIT_ERROR_PATTERN = /rate_limit_exceeded|429 Too Many Requests|RateLimitError/i;

// Pattern to detect OpenAI server-side errors (HTTP 500, 503).
// These are transient infrastructure failures that may resolve on retry.
const SERVER_ERROR_PATTERN = /InternalServerError|ServiceUnavailableError|500 Internal Server Error|503 Service Unavailable/i;

/**
 * Emit a timestamped diagnostic log line to stderr.
 * All driver messages are prefixed with "[codex-harness]" so they are easy to
 * grep out of the combined agent-stdio.log.
 * @param {string} message
 */
function log(message) {
  const ts = new Date().toISOString();
  process.stderr.write(`[codex-harness] ${ts} ${message}\n`);
}

/**
 * Determines if the collected output contains an OpenAI rate-limit error.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isRateLimitError(output) {
  return RATE_LIMIT_ERROR_PATTERN.test(output);
}

/**
 * Determines if the collected output contains an OpenAI server error.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isServerError(output) {
  return SERVER_ERROR_PATTERN.test(output);
}

/**
 * Resolve --prompt-file arguments for the Codex run.
 * Strips the --prompt-file <path> pair from args and appends the file content
 * as the last positional argument, which is where `codex exec` expects the prompt.
 *
 * @param {string[]} args
 * @returns {string[]} Args with --prompt-file resolved to inline prompt content
 */
function resolveCodexPromptFileArgs(args) {
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
      // An unreadable prompt file means no task instructions can be delivered to Codex.
      // Propagate as a fatal error rather than forwarding the harness-only flag to the
      // codex subprocess (which would fail with an "unknown option" error).
      throw new Error(`--prompt-file '${promptFile}' is not readable: ${err.message}`);
    }
    i++; // Skip the prompt-file path argument
  }

  // Append the prompt content as the last positional argument (codex exec convention).
  if (promptContent !== null) {
    filteredArgs.push(promptContent);
  }

  return filteredArgs;
}

/**
 * Main entry point: run codex with retry logic for transient API failures.
 * Codex does not support --continue session resumption, so all retries are fresh runs.
 */
async function main() {
  const [, , command, ...args] = process.argv;

  if (!command) {
    process.stderr.write("codex-harness: Usage: node codex_harness.cjs <command> [args...]\n");
    process.exit(1);
  }

  log(`starting: command=${command} maxRetries=${MAX_RETRIES} initialDelayMs=${INITIAL_DELAY_MS}` + ` backoffMultiplier=${BACKOFF_MULTIPLIER} maxDelayMs=${MAX_DELAY_MS}` + ` nodeVersion=${process.version} platform=${process.platform}`);

  // Resolve the prompt for the initial run (reads --prompt-file content).
  // A missing or unreadable prompt file is treated as a fatal startup error.
  let resolvedArgs;
  try {
    resolvedArgs = resolveCodexPromptFileArgs(args);
  } catch (err) {
    const e = /** @type {Error} */ err;
    log(`fatal: ${e.message}`);
    process.exit(1);
  }

  // Safe arg list for logging: when --prompt-file was present, the last element of
  // resolvedArgs is the resolved prompt content. Replace it with a placeholder so that
  // task instructions are never written to stderr or captured in agent logs.
  const hadPromptFile = args.includes("--prompt-file");
  const safeArgs = hadPromptFile && resolvedArgs.length > 0 ? [...resolvedArgs.slice(0, -1), "<prompt omitted>"] : resolvedArgs;

  // Fetch AWF API proxy reflection data before running the agent to capture initial proxy state.
  // This is best-effort: failures are logged but do not affect the agent run.
  await fetchAWFReflect({ logger: log });

  let delay = INITIAL_DELAY_MS;
  let lastExitCode = 1;
  const driverStartTime = Date.now();

  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    // Codex does not support --continue: every retry is a fresh run from scratch.
    // Context from the interrupted session is not recoverable, but transient API
    // failures (rate limits, server errors) may resolve on the next attempt.

    if (attempt > 0) {
      log(`retry ${attempt}/${MAX_RETRIES}: sleeping ${delay}ms before next attempt (fresh run)`);
      await sleep(delay);
      delay = Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS);
      log(`retry ${attempt}/${MAX_RETRIES}: woke up, next delay cap will be ${Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS)}ms`);
    }

    const result = await runProcess({ command, args: resolvedArgs, attempt, log, logArgs: safeArgs });
    lastExitCode = result.exitCode;

    // Success — stop retrying
    if (result.exitCode === 0) {
      log(`success on attempt ${attempt + 1}: totalDuration=${formatDuration(Date.now() - driverStartTime)}`);
      lastExitCode = 0;
      break;
    }

    const isRateLimit = isRateLimitError(result.output);
    const isServer = isServerError(result.output);
    log(`attempt ${attempt + 1} failed:` + ` exitCode=${result.exitCode}` + ` isRateLimitError=${isRateLimit}` + ` isServerError=${isServer}` + ` hasOutput=${result.hasOutput}` + ` retriesRemaining=${MAX_RETRIES - attempt}`);

    // Retry when the session was partially executed (has output) or on well-known
    // transient errors (rate limit, server error) even without output.
    const isTransient = isRateLimit || isServer;
    if (attempt < MAX_RETRIES && (result.hasOutput || isTransient)) {
      const reason = isRateLimit ? "rate_limit_exceeded (transient)" : isServer ? "server_error (transient)" : "partial execution";
      log(`attempt ${attempt + 1}: ${reason} — will retry as fresh run (attempt ${attempt + 2}/${MAX_RETRIES + 1})`);
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
    resolveCodexPromptFileArgs,
    isRateLimitError,
    isServerError,
  };
}

if (require.main === module) {
  main().catch(err => {
    log(`unexpected error: ${err.message}`);
    process.exit(1);
  });
}
