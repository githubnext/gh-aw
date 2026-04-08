// @ts-check

/**
 * Copilot CLI Driver with Retry Logic
 *
 * Wraps the Copilot CLI command with retry logic for transient CAPIError 400 errors.
 * Passes all arguments to the copilot subprocess, transparently forwarding stdin/stdout/stderr.
 * On a transient CAPIError 400 (which may occur mid-session), retries with --resume flag
 * using exponential backoff so that the session can continue from where it left off.
 *
 * Usage: node copilot_driver.cjs <command> [args...]
 * Example: node copilot_driver.cjs copilot --add-dir /tmp/ --prompt "..."
 */

"use strict";

const { spawn } = require("child_process");

// Maximum number of retry attempts after the initial run
const MAX_RETRIES = 3;
// Initial delay in milliseconds before the first retry
const INITIAL_DELAY_MS = 5000;
// Multiplier applied to delay after each retry
const BACKOFF_MULTIPLIER = 2;
// Maximum delay cap in milliseconds
const MAX_DELAY_MS = 60000;

// Pattern to detect transient CAPIError 400 in copilot output
const CAPI_ERROR_400_PATTERN = /CAPIError:\s*400/;

/**
 * Emit a timestamped diagnostic log line to stderr.
 * All driver messages are prefixed with "[copilot-driver]" so they are easy to
 * grep out of the combined agent-stdio.log.
 * @param {string} message
 */
function log(message) {
  const ts = new Date().toISOString();
  process.stderr.write(`[copilot-driver] ${ts} ${message}\n`);
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
    // Redact --prompt value from logs to avoid leaking prompt content
    const safeArgs = args.map((arg, i) => (args[i - 1] === "--prompt" ? "<redacted>" : arg));
    log(`attempt ${attempt + 1}: spawning: ${command} ${safeArgs.join(" ")}`);

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
      const durationMs = Date.now() - startTime;
      const exitCode = code ?? 1;
      log(`attempt ${attempt + 1}: process exited` + ` exitCode=${exitCode}` + (signal ? ` signal=${signal}` : "") + ` duration=${formatDuration(durationMs)}` + ` stdout=${stdoutBytes}B stderr=${stderrBytes}B hasOutput=${hasOutput}`);
      resolve({ exitCode, output: collectedOutput, hasOutput, durationMs });
    });

    child.on("error", err => {
      const durationMs = Date.now() - startTime;
      log(`attempt ${attempt + 1}: failed to start process '${command}': ${err.message}`);
      resolve({
        exitCode: 1,
        output: collectedOutput,
        hasOutput,
        durationMs,
      });
    });
  });
}

/**
 * Main entry point: run copilot with retry logic for transient CAPIError 400 errors.
 */
async function main() {
  const [, , command, ...args] = process.argv;

  if (!command) {
    process.stderr.write("copilot-driver: Usage: node copilot_driver.cjs <command> [args...]\n");
    process.exit(1);
  }

  log(`starting: command=${command} maxRetries=${MAX_RETRIES} initialDelayMs=${INITIAL_DELAY_MS} backoffMultiplier=${BACKOFF_MULTIPLIER} maxDelayMs=${MAX_DELAY_MS}`);

  let delay = INITIAL_DELAY_MS;
  let lastExitCode = 1;
  const driverStartTime = Date.now();

  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    // Add --resume flag on retries so the copilot session resumes from where it left off
    const currentArgs = attempt > 0 ? [...args, "--resume"] : args;

    if (attempt > 0) {
      log(`retry ${attempt}/${MAX_RETRIES}: sleeping ${delay}ms before next attempt with --resume`);
      await sleep(delay);
      delay = Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS);
      log(`retry ${attempt}/${MAX_RETRIES}: woke up, next delay cap will be ${Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS)}ms`);
    }

    const result = await runProcess(command, currentArgs, attempt);
    lastExitCode = result.exitCode;

    // Success — exit immediately
    if (result.exitCode === 0) {
      log(`success on attempt ${attempt + 1}: totalDuration=${formatDuration(Date.now() - driverStartTime)}`);
      process.exit(0);
    }

    // Determine whether to retry
    const isCAPIError = isTransientCAPIError(result.output);
    log(`attempt ${attempt + 1} failed:` + ` exitCode=${result.exitCode}` + ` isCAPIError400=${isCAPIError}` + ` hasOutput=${result.hasOutput}` + ` retriesRemaining=${MAX_RETRIES - attempt}`);

    // Check if this is a transient CAPIError 400 that occurred after the session started
    // (hasOutput indicates the CLI ran for a while before failing, making it worth retrying)
    if (attempt < MAX_RETRIES && result.hasOutput && isCAPIError) {
      log(`attempt ${attempt + 1}: detected transient CAPIError 400 — will retry with --resume (attempt ${attempt + 2}/${MAX_RETRIES + 1})`);
      continue;
    }

    if (attempt >= MAX_RETRIES) {
      log(`all ${MAX_RETRIES} retries exhausted — giving up (exitCode=${lastExitCode})`);
    } else if (!result.hasOutput) {
      log(`attempt ${attempt + 1}: no output produced — not retrying (process may have failed to start)`);
    } else {
      log(`attempt ${attempt + 1}: error is not a transient CAPIError 400 — not retrying`);
    }

    // Non-retryable error or retries exhausted — propagate exit code
    break;
  }

  log(`done: exitCode=${lastExitCode} totalDuration=${formatDuration(Date.now() - driverStartTime)}`);
  process.exit(lastExitCode);
}

main().catch(err => {
  log(`unexpected error: ${err.message}`);
  process.exit(1);
});
