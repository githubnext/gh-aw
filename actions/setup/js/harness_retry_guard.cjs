// @ts-check

"use strict";

const { emitInfrastructureIncomplete } = require("./safeoutputs_cli.cjs");

// Stop retrying this long before the step hard timeout so the harness can emit
// structured safe-output diagnostics instead of being terminated by Actions.
const SOFT_TIMEOUT_BUFFER_MS = 90 * 1000;

// Pattern to detect write-once tool probe rejections (-32602 empty-argument errors).
// Matches the MCP server error message emitted when the agent calls a write-once
// safe-output tool with empty arguments as a discovery probe.
// See mcp_server_core.cjs: "write-once, not a discovery probe".
const WRITE_ONCE_PROBE_REJECTION_PATTERN = /write-once, not a discovery probe/g;

// Threshold: number of write-once probe rejections before the kill guard fires.
// Three rejections indicate a persistent probe loop — the tool schema does not
// change between calls, so no further retries will succeed.
const WRITE_ONCE_PROBE_REJECTION_THRESHOLD = 3;

const AI_CREDITS_EXCEEDED_PATTERNS = [/\bmax[\s_-]*ai[\s_-]*credits[\s_-]*exceeded\b/i, /\bai[\s_-]*credits[\s_-]*rate[\s_-]*limit[\s_-]*error\b/i, /ai[\s_-]*credits?.*(?:rate[\s-]*limit|limit exceeded|budget exceeded|exceeded)/i];

const AWF_API_PROXY_BLOCKING_REQUESTS_PATTERNS = [/\bawf\b.*\bapi[\s_-]*proxy\b.*\bblocking requests\b/i, /\bapi[\s_-]*proxy\b.*\bblocking requests\b/i, /\bapi[\s_-]*proxy\b.*\bblocked requests?\b/i, /\bDIFC_FILTERED\b/];
const GOAL_ALREADY_ACTIVE_PATTERNS = [/\bthis thread already has a goal\b[\s\S]*?\buse update_goal\b/i, /\bcannot create a new goal because this thread has an unfinished goal\b;\s*\bcomplete the existing goal first\b/i];

// Patterns to detect Anthropic "max_runs_exceeded" (HTTP 403).
// This occurs when the per-session LLM invocation quota is exhausted.
// Retrying is pointless because each fresh-run attempt immediately fails with
// the same 403 until the quota resets.  Matches both the JSON error type
// ("max_runs_exceeded") and the human-readable message
// ("Maximum LLM invocations exceeded").
const MAX_RUNS_EXCEEDED_PATTERNS = [/\bmax_runs_exceeded\b/i, /Maximum LLM invocations exceeded/i];

/**
 * Detect retry guard conditions that should stop harness retries immediately.
 * @param {unknown} output
 * @returns {{ aiCreditsExceeded: boolean, awfAPIProxyBlockingRequests: boolean, goalAlreadyActive: boolean, maxRunsExceeded: boolean }}
 */
function detectNonRetryableHarnessGuard(output) {
  const safeOutput = typeof output === "string" ? output : "";
  return {
    aiCreditsExceeded: AI_CREDITS_EXCEEDED_PATTERNS.some(pattern => pattern.test(safeOutput)),
    awfAPIProxyBlockingRequests: AWF_API_PROXY_BLOCKING_REQUESTS_PATTERNS.some(pattern => pattern.test(safeOutput)),
    goalAlreadyActive: GOAL_ALREADY_ACTIVE_PATTERNS.some(pattern => pattern.test(safeOutput)),
    maxRunsExceeded: MAX_RUNS_EXCEEDED_PATTERNS.some(pattern => pattern.test(safeOutput)),
  };
}

/**
 * Count write-once probe rejection errors in process output.
 * Each occurrence of the pattern indicates the agent called a write-once safe-output
 * tool with empty arguments instead of using tools/list or calling noop.
 * @param {unknown} output
 * @returns {number}
 */
function countWriteOnceProbeRejections(output) {
  const safeOutput = typeof output === "string" ? output : "";
  if (!safeOutput) return 0;
  const matches = safeOutput.match(WRITE_ONCE_PROBE_REJECTION_PATTERN);
  return matches ? matches.length : 0;
}

/**
 * Detect whether output contains an excessive number of write-once probe rejections.
 * When this threshold is reached the agent is stuck in a probe loop — further
 * execution will never succeed and the process should be killed rather than allowed
 * to run to the full 15-minute step timeout.
 * @param {unknown} output
 * @returns {boolean}
 */
function hasExcessiveWriteOnceProbeRejections(output) {
  return countWriteOnceProbeRejections(output) >= WRITE_ONCE_PROBE_REJECTION_THRESHOLD;
}

/**
 * Compute a soft timeout deadline for the harness based on GH_AW_TIMEOUT_MINUTES.
 * Returns null when timeout is unset/invalid.
 * @param {number} driverStartTime
 * @param {NodeJS.ProcessEnv} [env]
 * @returns {{ timeoutMinutes: number, softDeadlineMs: number } | null}
 */
function buildSoftTimeoutGuard(driverStartTime, env = process.env) {
  const timeoutMinutes = Number(env.GH_AW_TIMEOUT_MINUTES);
  if (!Number.isFinite(timeoutMinutes) || timeoutMinutes <= 0) {
    return null;
  }
  const hardTimeoutMs = Math.floor(timeoutMinutes * 60 * 1000);
  const softDeadlineMs = driverStartTime + Math.max(hardTimeoutMs - SOFT_TIMEOUT_BUFFER_MS, 1000);
  return { timeoutMinutes, softDeadlineMs };
}

/**
 * Emit infrastructure incomplete signal and log when the soft timeout guard fires.
 * @param {{ timeoutMinutes: number, softDeadlineMs: number }} guard
 * @param {string} context - Short label for where the check fired (e.g. "before attempt 2")
 * @param {string} harnessName - Human-readable name of the harness (e.g. "Copilot harness")
 * @param {(message: string) => void} logFn - Harness-specific log function
 */
function emitSoftTimeoutSignal(guard, context, harnessName, logFn) {
  emitInfrastructureIncomplete(`${harnessName} reached soft retry budget before the ${guard.timeoutMinutes}-minute step timeout. ` + "Stopping retries early to preserve structured failure output.");
  logFn(`soft-timeout guard reached ${context}: timeoutMinutes=${guard.timeoutMinutes} bufferMs=${SOFT_TIMEOUT_BUFFER_MS}`);
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    detectNonRetryableHarnessGuard,
    AI_CREDITS_EXCEEDED_PATTERNS,
    AWF_API_PROXY_BLOCKING_REQUESTS_PATTERNS,
    GOAL_ALREADY_ACTIVE_PATTERNS,
    MAX_RUNS_EXCEEDED_PATTERNS,
    SOFT_TIMEOUT_BUFFER_MS,
    WRITE_ONCE_PROBE_REJECTION_PATTERN,
    WRITE_ONCE_PROBE_REJECTION_THRESHOLD,
    countWriteOnceProbeRejections,
    hasExcessiveWriteOnceProbeRejections,
    buildSoftTimeoutGuard,
    emitSoftTimeoutSignal,
  };
}
