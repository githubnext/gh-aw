// @ts-check

"use strict";

const AI_CREDITS_EXCEEDED_PATTERNS = [/\bmax[\s_-]*ai[\s_-]*credits[\s_-]*exceeded\b/i, /\bai[\s_-]*credits[\s_-]*rate[\s_-]*limit[\s_-]*error\b/i, /ai[\s_-]*credits?.*(?:rate[\s-]*limit|limit exceeded|budget exceeded|exceeded)/i];

const AWF_API_PROXY_BLOCKING_REQUESTS_PATTERNS = [/\bawf\b.*\bapi[\s_-]*proxy\b.*\bblocking requests\b/i, /\bapi[\s_-]*proxy\b.*\bblocking requests\b/i, /\bapi[\s_-]*proxy\b.*\bblocked requests?\b/i, /\bDIFC_FILTERED\b/];
const GOAL_ALREADY_ACTIVE_PATTERNS = [/\bthis thread already has a goal\b[\s\S]*?\buse update_goal\b/i];

/**
 * Detect retry guard conditions that should stop harness retries immediately.
 * @param {unknown} output
 * @returns {{ aiCreditsExceeded: boolean, awfAPIProxyBlockingRequests: boolean, goalAlreadyActive: boolean }}
 */
function detectNonRetryableHarnessGuard(output) {
  const safeOutput = typeof output === "string" ? output : "";
  return {
    aiCreditsExceeded: AI_CREDITS_EXCEEDED_PATTERNS.some(pattern => pattern.test(safeOutput)),
    awfAPIProxyBlockingRequests: AWF_API_PROXY_BLOCKING_REQUESTS_PATTERNS.some(pattern => pattern.test(safeOutput)),
    goalAlreadyActive: GOAL_ALREADY_ACTIVE_PATTERNS.some(pattern => pattern.test(safeOutput)),
  };
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    detectNonRetryableHarnessGuard,
    AI_CREDITS_EXCEEDED_PATTERNS,
    AWF_API_PROXY_BLOCKING_REQUESTS_PATTERNS,
    GOAL_ALREADY_ACTIVE_PATTERNS,
  };
}
