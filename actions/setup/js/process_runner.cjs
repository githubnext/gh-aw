// @ts-check

/**
 * Shared process runner utilities for agent harnesses.
 *
 * Provides a common runProcess helper used by both the Claude and Copilot
 * harnesses to spawn child processes, forward stdin/stdout/stderr, collect
 * output for retry decisions, track byte counts, and surface spawn errors.
 *
 * Each harness retains its own logging prefix and argument-redaction logic;
 * the caller passes a log function and an optional logArgs array so that
 * sensitive values (e.g. prompt text) are never written to logs.
 */

"use strict";

const { spawn } = require("child_process");

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
 * Sleep for a specified duration.
 * @param {number} ms - Duration in milliseconds
 * @returns {Promise<void>}
 */
function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Run a command with the given arguments, transparently forwarding stdin/stdout/stderr.
 * Also collects combined stdout+stderr output for error pattern detection.
 *
 * @param {{
 *   command: string,
 *   args: string[],
 *   attempt: number,
 *   log: (message: string) => void,
 *   logArgs?: string[]
 * }} options
 *   - command   - The executable to run
 *   - args      - Arguments to pass to the command
 *   - attempt   - Current attempt index (0-based), used for logging
 *   - log       - Caller-supplied logging function (harness-specific prefix)
 *   - logArgs   - Safe arg list used only for logging; defaults to `args`.
 *                 Pass a redacted copy to avoid leaking sensitive values.
 * @returns {Promise<{exitCode: number, output: string, hasOutput: boolean, durationMs: number}>}
 */
function runProcess({ command, args, attempt, log, logArgs }) {
  return new Promise(resolve => {
    const startTime = Date.now();
    // Guard against the promise being settled more than once.  On some systems Node
    // emits 'close' after 'error' (or vice-versa); only the first terminal event should
    // log and resolve so callers receive a deterministic result.
    let settled = false;
    /** @param {{exitCode: number, output: string, hasOutput: boolean, durationMs: number}} result */
    function settle(result) {
      if (settled) return;
      settled = true;
      resolve(result);
    }

    const argsForLog = logArgs ?? args;
    log(`attempt ${attempt + 1}: spawning: ${command} ${argsForLog.join(" ").substring(0, 200)}`);

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
      log(`attempt ${attempt + 1}: process exit event` + ` exitCode=${code ?? 1}` + (signal ? ` signal=${signal}` : ""));
    });

    // Resolve on 'close', not 'exit', to ensure stdio streams are fully drained.
    child.on("close", (code, signal) => {
      const durationMs = Date.now() - startTime;
      const exitCode = code ?? 1;
      log(`attempt ${attempt + 1}: process closed` + ` exitCode=${exitCode}` + (signal ? ` signal=${signal}` : "") + ` duration=${formatDuration(durationMs)}` + ` stdout=${stdoutBytes}B stderr=${stderrBytes}B hasOutput=${hasOutput}`);
      settle({ exitCode, output: collectedOutput, hasOutput, durationMs });
    });

    child.on("error", err => {
      const durationMs = Date.now() - startTime;
      // prettier-ignore
      const errno = /** @type {NodeJS.ErrnoException} */ (err);
      const errCode = errno.code ?? "unknown";
      const errSyscall = errno.syscall ?? "unknown";
      log(`attempt ${attempt + 1}: failed to start process '${command}': ${err.message}` + ` (code=${errCode} syscall=${errSyscall})`);
      settle({
        exitCode: 1,
        output: collectedOutput,
        hasOutput,
        durationMs,
      });
    });
  });
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = { runProcess, formatDuration, sleep };
}
