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
 *   - error_max_turns exits are NOT retried — reaching max_turns is a deterministic
 *     termination condition, not a transient API failure.  Retrying with --continue always
 *     fails for max_turns exits because no deferred tool marker was written for a clean
 *     max_turns session termination.
 *   - If the process produced no output (failed to start / auth error before any work), the
 *     driver does not retry because there is nothing to resume.
 *   - Retries use exponential backoff: 5s → 10s → 20s (capped at 60s).
 *   - Maximum 3 retry attempts after the initial run.
 *   - --continue retries omit the prompt (Claude resumes from its internal session state).
 *
 * Usage: node claude_harness.cjs <command> [args...] [--prompt-file <path>]
 * Example: node claude_harness.cjs claude --print --no-chrome ... --prompt-file /tmp/gh-aw/aw-prompts/prompt.txt
 */

"use strict";

const { spawn } = require("child_process");
const fs = require("fs");

// Maximum number of retry attempts after the initial run
const MAX_RETRIES = 3;
// Initial delay in milliseconds before the first retry
const INITIAL_DELAY_MS = 5000;
// Multiplier applied to delay after each retry
const BACKOFF_MULTIPLIER = 2;
// Maximum delay cap in milliseconds
const MAX_DELAY_MS = 60000;

// Pattern to detect error_max_turns in Claude Code CLI output (JSONL stream-json format).
// Format: {"type":"result","subtype":"error_max_turns","is_error":true,...}
// This is a deterministic termination — the agent reached the configured max_turns limit.
// Retrying with --continue always fails because no deferred tool marker is written when
// the session ends cleanly due to max_turns.
const ERROR_MAX_TURNS_PATTERN = /"subtype"\s*:\s*"error_max_turns"/;

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
 * Determines if the collected output contains an error_max_turns signal.
 * This is a deterministic termination — the agent reached the max turns limit.
 * Retrying with --continue will not help because no deferred tool marker was written.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isMaxTurnsError(output) {
  return ERROR_MAX_TURNS_PATTERN.test(output);
}

/**
 * Sleep for a specified duration
 * @param {number} ms - Duration in milliseconds
 * @returns {Promise<void>}
 */
function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Format elapsed milliseconds as a human-readable string (e.g. "3m 12s").
 * @param {number} ms
 * @returns {string}
 */
function formatDuration(ms) {
  const totalSeconds = Math.floor(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  }
  return `${seconds}s`;
}

/**
 * Run a command with the given arguments, transparently forwarding stdin/stdout/stderr.
 * Also collects output for error pattern detection.
 *
 * @param {string} command - The executable to run
 * @param {string[]} args - Arguments to pass to the command
 * @param {number} attempt - Current attempt index (0-based), used for logging
 * @returns {Promise<{exitCode: number, output: string, hasOutput: boolean, durationMs: number}>}
 */
function runProcess(command, args, attempt) {
  return new Promise(resolve => {
    const startTime = Date.now();
    log(`attempt ${attempt + 1}: spawning: ${command} ${args.join(" ")}`);

    const child = spawn(command, args, {
      stdio: ["inherit", "pipe", "pipe"],
      env: process.env,
    });

    log(`attempt ${attempt + 1}: process started (pid=${child.pid ?? "unknown"})`);

    let collectedOutput = "";
    let hasOutput = false;
    let stdoutBytes = 0;
    let stderrBytes = 0;

    child.stdout.on(
      "data",
      /** @param {Buffer} data */ data => {
        hasOutput = true;
        stdoutBytes += data.length;
        collectedOutput += data.toString();
        process.stdout.write(data);
      }
    );

    child.stderr.on(
      "data",
      /** @param {Buffer} data */ data => {
        hasOutput = true;
        stderrBytes += data.length;
        collectedOutput += data.toString();
        process.stderr.write(data);
      }
    );

    child.on("exit", (code, signal) => {
      log(`attempt ${attempt + 1}: process exit event exitCode=${code ?? 1}${signal ? ` signal=${signal}` : ""}`);
    });

    // Resolve on 'close', not 'exit'.  'close' fires after stdio streams are fully drained,
    // guaranteeing that collectedOutput and hasOutput are complete before we make the retry
    // decision and that the final exit code is faithfully propagated.
    child.on("close", (code, signal) => {
      const durationMs = Date.now() - startTime;
      const exitCode = code ?? 1;
      log(`attempt ${attempt + 1}: process closed` + ` exitCode=${exitCode}` + (signal ? ` signal=${signal}` : "") + ` duration=${formatDuration(durationMs)}` + ` stdout=${stdoutBytes}B stderr=${stderrBytes}B hasOutput=${hasOutput}`);
      resolve({ exitCode, output: collectedOutput, hasOutput, durationMs });
    });

    child.on("error", err => {
      const durationMs = Date.now() - startTime;
      // prettier-ignore
      const errno = /** @type {NodeJS.ErrnoException} */ (err);
      const errCode = errno.code ?? "unknown";
      const errSyscall = errno.syscall ?? "unknown";
      log(`attempt ${attempt + 1}: failed to start process '${command}': ${err.message}` + ` (code=${errCode} syscall=${errSyscall})`);
      resolve({ exitCode: 1, output: collectedOutput, hasOutput, durationMs });
    });
  });
}

/**
 * Parse args into initial args (with prompt content) and continue args (without prompt,
 * with --continue appended).
 *
 * Claude Code CLI takes the prompt as a positional argument (not a flag).  The harness
 * accepts --prompt-file <path> and reads the file content, appending it as the last
 * positional argument for the initial run.  For --continue retries the prompt is omitted
 * because Claude resumes from its internal session state.
 *
 * @param {string[]} args - Raw args including optional --prompt-file <path>
 * @returns {{ initialArgs: string[], continueArgs: string[] }}
 */
function parseArgs(args) {
  /** @type {string[]} */
  const baseArgs = [];
  let promptContent = null;

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg !== "--prompt-file") {
      baseArgs.push(arg);
      continue;
    }

    if (i + 1 >= args.length) {
      log("warning: --prompt-file provided without a path; leaving arguments unchanged");
      baseArgs.push(arg);
      continue;
    }

    const promptFile = args[i + 1];
    try {
      promptContent = fs.readFileSync(promptFile, "utf8");
      log(`resolved --prompt-file: path=${promptFile} size=${Buffer.byteLength(promptContent, "utf8")}B`);
    } catch (error) {
      const err = /** @type {Error} */ error;
      log(`warning: failed to resolve --prompt-file ${promptFile}: ${err.message}; leaving arguments unchanged`);
      baseArgs.push(arg, promptFile);
    }
    i++; // skip the prompt-file path argument
  }

  const initialArgs = promptContent !== null ? [...baseArgs, promptContent] : baseArgs;
  const continueArgs = [...baseArgs, "--continue"];

  return { initialArgs, continueArgs };
}

/**
 * Main entry point: run Claude Code CLI with retry logic for partially-executed sessions.
 */
async function main() {
  const [, , command, ...args] = process.argv;

  if (!command) {
    process.stderr.write("claude-harness: Usage: node claude_harness.cjs <command> [args...]\n");
    process.exit(1);
  }

  log(`starting: command=${command} maxRetries=${MAX_RETRIES} initialDelayMs=${INITIAL_DELAY_MS}` + ` backoffMultiplier=${BACKOFF_MULTIPLIER} maxDelayMs=${MAX_DELAY_MS}` + ` nodeVersion=${process.version} platform=${process.platform}`);

  const { initialArgs, continueArgs } = parseArgs(args);

  let delay = INITIAL_DELAY_MS;
  let lastExitCode = 1;
  let useContinueOnRetry = false;
  const driverStartTime = Date.now();

  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    // Use --continue args on retries (omits prompt, resumes from session state)
    const currentArgs = attempt > 0 && useContinueOnRetry ? continueArgs : initialArgs;

    if (attempt > 0) {
      const retryMode = useContinueOnRetry ? "--continue" : "fresh run";
      log(`retry ${attempt}/${MAX_RETRIES}: sleeping ${delay}ms before next attempt (${retryMode})`);
      await sleep(delay);
      delay = Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS);
      log(`retry ${attempt}/${MAX_RETRIES}: woke up, next delay cap will be ${Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS)}ms`);
    }

    const result = await runProcess(command, currentArgs, attempt);
    lastExitCode = result.exitCode;

    // Success — stop retrying
    if (result.exitCode === 0) {
      log(`success on attempt ${attempt + 1}: totalDuration=${formatDuration(Date.now() - driverStartTime)}`);
      lastExitCode = 0;
      break;
    }

    const maxTurns = isMaxTurnsError(result.output);
    log(`attempt ${attempt + 1} failed:` + ` exitCode=${result.exitCode}` + ` isMaxTurnsError=${maxTurns}` + ` hasOutput=${result.hasOutput}` + ` retriesRemaining=${MAX_RETRIES - attempt}`);

    // error_max_turns is a deterministic termination — not retriable via --continue.
    // The session ended cleanly at the configured max_turns limit; no deferred tool
    // marker was written, so --continue always fails with "No deferred tool marker
    // found in the resumed session".
    if (maxTurns) {
      log(`attempt ${attempt + 1}: max_turns exit — not retriable via --continue (session ended deterministically at turn limit)`);
      break;
    }

    if (attempt < MAX_RETRIES && result.hasOutput) {
      useContinueOnRetry = true;
      log(`attempt ${attempt + 1}: partial execution — will retry with --continue (attempt ${attempt + 2}/${MAX_RETRIES + 1})`);
      continue;
    }

    if (attempt >= MAX_RETRIES) {
      log(`all ${MAX_RETRIES} retries exhausted — giving up (exitCode=${lastExitCode})`);
    } else {
      log(`attempt ${attempt + 1}: no output produced — not retrying` + ` (possible causes: binary not found, permission denied, auth failure, or silent startup crash)`);
    }

    break;
  }

  log(`done: exitCode=${lastExitCode} totalDuration=${formatDuration(Date.now() - driverStartTime)}`);
  process.exit(lastExitCode);
}

if (require.main === module) {
  main();
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    ERROR_MAX_TURNS_PATTERN,
    isMaxTurnsError,
    parseArgs,
  };
}
