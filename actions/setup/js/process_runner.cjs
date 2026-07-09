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
 * The child process is spawned with `cwd` set to `process.env.GH_AW_ENGINE_CWD` when
 * available, falling back to `process.env.GITHUB_WORKSPACE`, so that engines and their
 * skill-discovery paths resolve relative to the configured or repository checkout root
 * rather than the harness working directory.
 *
 * @param {{
 *   command: string,
 *   args: string[],
 *   attempt: number,
 *   log: (message: string) => void,
 *   logArgs?: string[],
 *   env?: NodeJS.ProcessEnv,
 *   postResultWatchdog?: {
 *     shouldArm: () => boolean,
 *     inactivityTimeoutMs: number,
 *     pollIntervalMs?: number,
 *     termGraceMs?: number
 *   }
 * }} options
 *   - command   - The executable to run
 *   - args      - Arguments to pass to the command
 *   - attempt   - Current attempt index (0-based), used for logging
 *   - log       - Caller-supplied logging function (harness-specific prefix)
 *   - logArgs   - Safe arg list used only for logging; defaults to `args`.
 *                 Pass a redacted copy to avoid leaking sensitive values.
 * @returns {Promise<{exitCode: number, output: string, hasOutput: boolean, durationMs: number}>}
 */
function runProcess({ command, args, attempt, log, logArgs, env, postResultWatchdog }) {
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
      if (postResultWatchdogTimer) clearInterval(postResultWatchdogTimer);
      resolve(result);
    }

    const argsForLog = logArgs ?? args;
    log(`attempt ${attempt + 1}: spawning: ${command} ${argsForLog.join(" ").substring(0, 200)}`);

    const child = spawn(command, args, {
      stdio: ["inherit", "pipe", "pipe"],
      env: env ?? process.env,
      cwd: process.env.GH_AW_ENGINE_CWD || process.env.GITHUB_WORKSPACE || undefined,
    });

    log(`attempt ${attempt + 1}: process started (pid=${child.pid ?? "unknown"})`);

    let collectedOutput = "";
    let hasOutput = false;
    let stdoutBytes = 0;
    let stderrBytes = 0;
    let lastActivityAt = Date.now();
    let watchdogArmed = false;
    let sentSigtermAt = 0;
    let sentSigkillAt = 0;
    const watchdogPollIntervalMs = Math.max(50, Number(postResultWatchdog?.pollIntervalMs) || 1000);
    const watchdogTermGraceMs = Math.max(50, Number(postResultWatchdog?.termGraceMs) || 5000);
    const rawInactivityTimeout = Number(postResultWatchdog?.inactivityTimeoutMs);
    const watchdogInactivityTimeoutMs = Number.isFinite(rawInactivityTimeout) && rawInactivityTimeout > 0 ? Math.max(50, rawInactivityTimeout) : 0;
    /** @type {NodeJS.Timeout | null} */
    let postResultWatchdogTimer = null;

    child.stdout.on(
      "data",
      /** @param {Buffer} data */ data => {
        hasOutput = true;
        stdoutBytes += data.length;
        collectedOutput += data.toString();
        lastActivityAt = Date.now();
        process.stdout.write(data);
      }
    );

    child.stderr.on(
      "data",
      /** @param {Buffer} data */ data => {
        hasOutput = true;
        stderrBytes += data.length;
        collectedOutput += data.toString();
        lastActivityAt = Date.now();
        process.stderr.write(data);
      }
    );

    if (postResultWatchdog && watchdogInactivityTimeoutMs > 0) {
      postResultWatchdogTimer = setInterval(() => {
        if (settled) return;
        if (!watchdogArmed) {
          try {
            watchdogArmed = postResultWatchdog.shouldArm();
          } catch {
            watchdogArmed = false;
          }
          if (watchdogArmed) {
            lastActivityAt = Date.now();
            log(`attempt ${attempt + 1}: post-result watchdog armed inactivityTimeout=${watchdogInactivityTimeoutMs}ms`);
          }
        }
        if (!watchdogArmed) return;
        const idleMs = Date.now() - lastActivityAt;
        if (sentSigtermAt === 0 && idleMs >= watchdogInactivityTimeoutMs) {
          sentSigtermAt = Date.now();
          log(`attempt ${attempt + 1}: post-result watchdog terminating idle process after ${idleMs}ms (SIGTERM)`);
          child.kill("SIGTERM");
          return;
        }
        if (sentSigtermAt > 0 && sentSigkillAt === 0 && Date.now() - sentSigtermAt >= watchdogTermGraceMs) {
          sentSigkillAt = Date.now();
          log(`attempt ${attempt + 1}: post-result watchdog forcing process exit after ${watchdogTermGraceMs}ms grace (SIGKILL)`);
          child.kill("SIGKILL");
        }
      }, watchdogPollIntervalMs);
    }

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

/**
 * @param {NodeJS.ProcessEnv} [env]
 * @returns {boolean}
 */
function isCopilotSDKEnabled(env) {
  const sourceEnv = env ?? process.env;
  return Boolean(sourceEnv.COPILOT_SDK_URI);
}

/**
 * Returns the Copilot SDK environment additions to inject into child processes
 * when SDK mode is active.
 *
 * When COPILOT_SDK_URI is set in process.env, returns an object with
 * { COPILOT_SDK_URI } so callers can merge it into their child-process env.
 * Returns an empty object when SDK mode is not active, making it safe to call
 * unconditionally.
 *
 * Intended to be shared by all engine harnesses (copilot_harness, claude_harness, …)
 * so that COPILOT_SDK_URI is forwarded consistently without duplicating the logic.
 *
 * @param {NodeJS.ProcessEnv} [env] - Source environment (defaults to process.env)
 * @returns {NodeJS.ProcessEnv}
 */
function buildCopilotSDKEnv(env) {
  const sourceEnv = env ?? process.env;
  if (!isCopilotSDKEnabled(sourceEnv)) return {};
  const uri = sourceEnv.COPILOT_SDK_URI;
  if (!uri) return {};
  /** @type {NodeJS.ProcessEnv} */
  const sdkEnv = { COPILOT_SDK_URI: uri };
  sdkEnv.COPILOT_SDK_LOG_LEVEL = sourceEnv.COPILOT_SDK_LOG_LEVEL || "all";
  if (sourceEnv.COPILOT_SDK_SEND_TIMEOUT_MS) {
    sdkEnv.COPILOT_SDK_SEND_TIMEOUT_MS = sourceEnv.COPILOT_SDK_SEND_TIMEOUT_MS;
    return sdkEnv;
  }

  const timeoutMinutes = Number(sourceEnv.GH_AW_TIMEOUT_MINUTES);
  if (!Number.isFinite(timeoutMinutes) || timeoutMinutes <= 0) {
    return sdkEnv;
  }

  // Keep SDK sendAndWait timeout below the job step timeout by 30 seconds.
  const timeoutMs = Math.max(Math.floor(timeoutMinutes * 60 * 1000) - 30 * 1000, 1000);
  sdkEnv.COPILOT_SDK_SEND_TIMEOUT_MS = String(timeoutMs);
  return sdkEnv;
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = { runProcess, formatDuration, sleep, isCopilotSDKEnabled, buildCopilotSDKEnv };
}
