// @ts-check

/**
 * Copilot CLI Driver with Retry Logic
 *
 * Wraps the Copilot CLI command with retry logic for failures that occur after the session
 * has been partially executed.  Passes all arguments to the copilot subprocess, transparently
 * forwarding stdin/stdout/stderr.
 *
 * Retry policy:
 *   - If the process produced any output (hasOutput) and exits with a non-zero code, the
 *     session is considered partially executed.  The driver retries with --resume so the
 *     Copilot CLI can continue from where it left off.
 *   - CAPIError 400 is a well-known transient failure mode and is logged explicitly, but
 *     any partial-execution failure is retried — not just CAPIError 400.
 *   - If the process produced no output (failed to start / auth error before any work), the
 *     driver does not retry because there is nothing to resume.
 *   - "No authentication information found" errors are non-retryable: the absent token will
 *     remain absent on every subsequent attempt.  The driver also applies an auth-token
 *     fallback before each spawn: if COPILOT_GITHUB_TOKEN is absent or empty, GITHUB_TOKEN
 *     (the standard GitHub Actions token, always forwarded into the container) is tried next,
 *     then GH_TOKEN.  This prevents auth failures when COPILOT_GITHUB_TOKEN has been
 *     excluded from the container environment (e.g. by AWF's --exclude-env) while the
 *     standard GITHUB_TOKEN is still available.
 *   - Retries use exponential backoff: 5s → 10s → 20s (capped at 60s).
 *   - Maximum 3 retry attempts after the initial run.
 *
 * Usage: node copilot_driver.cjs <command> [args...]
 * Example: node copilot_driver.cjs copilot --add-dir /tmp/ --prompt "..."
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

// Pattern to detect transient CAPIError 400 in copilot output
const CAPI_ERROR_400_PATTERN = /CAPIError:\s*400/;

// Pattern to detect MCP servers blocked by enterprise/organization policy.
// This is a persistent policy configuration error — retrying will not help.
const MCP_POLICY_BLOCKED_PATTERN = /MCP servers were blocked by policy:/;

// Pattern to detect missing authentication credentials.
// This error means no auth token is available in the environment; retrying will not help
// because the missing token will still be absent on every subsequent attempt.
const NO_AUTH_INFO_PATTERN = /No authentication information found/;

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
 * Determines if the collected output indicates MCP servers were blocked by policy.
 * This is a persistent configuration error that cannot be resolved by retrying.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isMCPPolicyError(output) {
  return MCP_POLICY_BLOCKED_PATTERN.test(output);
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
 * Build the environment object for spawning the copilot subprocess.
 *
 * Starts from the current process environment and applies auth token fallback:
 * if COPILOT_GITHUB_TOKEN is absent or empty, we substitute GITHUB_TOKEN or GH_TOKEN
 * (whichever is set first).  This ensures the copilot CLI can authenticate on --resume
 * attempts even when COPILOT_GITHUB_TOKEN was excluded from the container environment
 * (e.g. by AWF's --exclude-env to prevent the agent from reading the raw token) but the
 * standard GITHUB_TOKEN Actions token is still forwarded via --env-all.
 *
 * Auth token availability is always logged (names only, never values) so that failures
 * can be diagnosed without exposing secrets in the log.
 *
 * @returns {NodeJS.ProcessEnv}
 */
function buildSpawnEnv() {
  const env = { ...process.env };

  const hasTokenValue = /** @param {string} name */ name => typeof env[name] === "string" && env[name].length > 0;

  if (!hasTokenValue("COPILOT_GITHUB_TOKEN")) {
    if (hasTokenValue("GITHUB_TOKEN")) {
      env["COPILOT_GITHUB_TOKEN"] = env["GITHUB_TOKEN"];
      log("auth: COPILOT_GITHUB_TOKEN is absent — using GITHUB_TOKEN as fallback");
    } else if (hasTokenValue("GH_TOKEN")) {
      env["COPILOT_GITHUB_TOKEN"] = env["GH_TOKEN"];
      log("auth: COPILOT_GITHUB_TOKEN is absent — using GH_TOKEN as fallback");
    } else {
      log("auth: warning — COPILOT_GITHUB_TOKEN, GITHUB_TOKEN, and GH_TOKEN are all absent or empty; the copilot CLI may fail to authenticate");
    }
  } else {
    log("auth: COPILOT_GITHUB_TOKEN is set");
  }

  return env;
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
      env: buildSpawnEnv(),
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
      // Log the exit event early; the promise is resolved in 'close' (see below) once stdio
      // streams are fully drained so that collectedOutput and hasOutput are complete.
      log(`attempt ${attempt + 1}: process exit event` + ` exitCode=${code ?? 1}` + (signal ? ` signal=${signal}` : ""));
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
 * Main entry point: run copilot with retry logic for partially-executed sessions.
 */
async function main() {
  const [, , command, ...args] = process.argv;

  if (!command) {
    process.stderr.write("copilot-driver: Usage: node copilot_driver.cjs <command> [args...]\n");
    process.exit(1);
  }

  log(`starting: command=${command} maxRetries=${MAX_RETRIES} initialDelayMs=${INITIAL_DELAY_MS}` + ` backoffMultiplier=${BACKOFF_MULTIPLIER} maxDelayMs=${MAX_DELAY_MS}` + ` nodeVersion=${process.version} platform=${process.platform}`);

  await checkCommandAccessible(command);

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

    // Determine whether to retry.
    // Retry whenever the session was partially executed (hasOutput), using --resume so that
    // the Copilot CLI can continue from where it left off.  CAPIError 400 is the well-known
    // transient case, but any partial-execution failure is eligible for a resume retry.
    // Exceptions: MCP policy errors and auth errors are persistent — never retry.
    const isCAPIError = isTransientCAPIError(result.output);
    const isMCPPolicy = isMCPPolicyError(result.output);
    const isAuthErr = isNoAuthInfoError(result.output);
    log(
      `attempt ${attempt + 1} failed:` +
        ` exitCode=${result.exitCode}` +
        ` isCAPIError400=${isCAPIError}` +
        ` isMCPPolicyError=${isMCPPolicy}` +
        ` isAuthError=${isAuthErr}` +
        ` hasOutput=${result.hasOutput}` +
        ` retriesRemaining=${MAX_RETRIES - attempt}`
    );

    // MCP policy errors are persistent — retrying will not help.
    if (isMCPPolicy) {
      log(`attempt ${attempt + 1}: MCP servers blocked by policy — not retrying (this is a policy configuration issue, not a transient error)`);
      break;
    }

    // Auth errors are persistent for the duration of the job — retrying will not help.
    // "No authentication information found" means COPILOT_GITHUB_TOKEN / GH_TOKEN / GITHUB_TOKEN
    // are all absent or invalid.  Retrying with --resume will produce the same auth failure.
    if (isAuthErr) {
      log(`attempt ${attempt + 1}: no authentication information found — not retrying (COPILOT_GITHUB_TOKEN, GH_TOKEN, and GITHUB_TOKEN are all absent or invalid)`);
      break;
    }

    if (attempt < MAX_RETRIES && result.hasOutput) {
      const reason = isCAPIError ? "CAPIError 400 (transient)" : "partial execution";
      log(`attempt ${attempt + 1}: ${reason} — will retry with --resume (attempt ${attempt + 2}/${MAX_RETRIES + 1})`);
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

  log(`done: exitCode=${lastExitCode} totalDuration=${formatDuration(Date.now() - driverStartTime)}`);
  process.exit(lastExitCode);
}

main().catch(err => {
  log(`unexpected error: ${err.message}`);
  process.exit(1);
});
