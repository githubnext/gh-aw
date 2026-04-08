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
 * Run a command with the given arguments, transparently forwarding stdin/stdout/stderr.
 * Also collects output for error pattern detection.
 *
 * @param {string} command - The executable to run
 * @param {string[]} args - Arguments to pass to the command
 * @returns {Promise<{exitCode: number, output: string, hasOutput: boolean}>}
 */
function runProcess(command, args) {
  return new Promise(resolve => {
    const child = spawn(command, args, {
      stdio: ["inherit", "pipe", "pipe"],
      env: process.env,
    });

    let collectedOutput = "";
    let hasOutput = false;

    child.stdout.on(
      "data",
      /** @param {Buffer} data */ data => {
        hasOutput = true;
        collectedOutput += data.toString();
        process.stdout.write(data);
      }
    );

    child.stderr.on(
      "data",
      /** @param {Buffer} data */ data => {
        hasOutput = true;
        collectedOutput += data.toString();
        process.stderr.write(data);
      }
    );

    child.on("exit", code => {
      resolve({
        exitCode: code ?? 1,
        output: collectedOutput,
        hasOutput,
      });
    });

    child.on("error", err => {
      process.stderr.write(`copilot-driver: Failed to start process '${command}': ${err.message}\n`);
      resolve({
        exitCode: 1,
        output: collectedOutput,
        hasOutput,
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

  let delay = INITIAL_DELAY_MS;
  let lastExitCode = 1;

  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    // Add --resume flag on retries so the copilot session resumes from where it left off
    const currentArgs = attempt > 0 ? [...args, "--resume"] : args;

    if (attempt > 0) {
      process.stderr.write(`copilot-driver: Retry attempt ${attempt}/${MAX_RETRIES} with --resume after ${delay}ms delay...\n`);
      await sleep(delay);
      delay = Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS);
    }

    const result = await runProcess(command, currentArgs);
    lastExitCode = result.exitCode;

    // Success — exit immediately
    if (result.exitCode === 0) {
      process.exit(0);
    }

    // Check if this is a transient CAPIError 400 that occurred after the session started
    // (hasOutput indicates the CLI ran for a while before failing, making it worth retrying)
    if (attempt < MAX_RETRIES && result.hasOutput && isTransientCAPIError(result.output)) {
      process.stderr.write(`copilot-driver: Detected transient CAPIError 400 on attempt ${attempt + 1}/${MAX_RETRIES + 1}. Will retry with --resume.\n`);
      continue;
    }

    // Non-retryable error or retries exhausted — propagate exit code
    break;
  }

  process.exit(lastExitCode);
}

main().catch(err => {
  process.stderr.write(`copilot-driver: Unexpected error: ${err.message}\n`);
  process.exit(1);
});
