// @ts-check
/// <reference types="@actions/github-script" />

const { fetchAndLogRateLimit } = require("./github_rate_limit_logger.cjs");

/**
 * Minimum rate limit remaining before we skip further operations.
 * This reserves capacity for other workflow jobs and API consumers.
 */
const MIN_RATE_LIMIT_REMAINING = 100;

/**
 * Check the current rate limit and determine if we should continue.
 * Returns the remaining requests count, or -1 if we couldn't check.
 * Also logs the rate limit snapshot for observability.
 *
 * @param {any} github - GitHub REST client
 * @param {string} [operation="rate_limit_check"] - Label for the log entry
 * @returns {Promise<number>} Remaining requests, or -1 on error
 */
async function getRateLimitRemaining(github, operation = "rate_limit_check") {
  try {
    await fetchAndLogRateLimit(github, operation);
    const { data } = await github.rest.rateLimit.get();
    return data.rate.remaining;
  } catch {
    return -1;
  }
}

/**
 * Check if the current rate limit is sufficient for operations.
 * Logs a warning and returns false if the rate limit is too low.
 *
 * @param {any} github - GitHub REST client
 * @param {string} [operation="rate_limit_check"] - Label for the log entry
 * @returns {Promise<{ok: boolean, remaining: number}>}
 */
async function checkRateLimit(github, operation = "rate_limit_check") {
  const remaining = await getRateLimitRemaining(github, operation);
  if (remaining !== -1 && remaining < MIN_RATE_LIMIT_REMAINING) {
    return { ok: false, remaining };
  }
  return { ok: true, remaining };
}

module.exports = {
  MIN_RATE_LIMIT_REMAINING,
  getRateLimitRemaining,
  checkRateLimit,
};
