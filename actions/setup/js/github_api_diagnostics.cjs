// @ts-check

const { isRateLimitError } = require("./error_helpers.cjs");

function getGitHubAPIQuota(headers, now = Date.now(), quotaStopped = false) {
  const header = name => (typeof headers?.get === "function" ? headers.get(name) : headers?.[name]);
  const integer = value => {
    if (typeof value !== "string" || !/^\d{1,16}$/.test(value)) return null;
    const number = Number(value);
    return Number.isSafeInteger(number) ? number : null;
  };
  const remaining = integer(header("x-ratelimit-remaining"));
  const limit = integer(header("x-ratelimit-limit"));
  const used = integer(header("x-ratelimit-used"));
  const resetSeconds = integer(header("x-ratelimit-reset"));
  const resetMs = resetSeconds === null ? NaN : resetSeconds * 1000;
  const reset = Number.isFinite(resetMs) && resetMs > 0 && resetMs <= 8640000000000000 ? new Date(resetMs).toISOString() : "";
  const retrySeconds = integer(header("retry-after"));
  const retryHeader = header("retry-after");
  const retryMs = retrySeconds === null ? (typeof retryHeader === "string" && retryHeader.length <= 64 ? Date.parse(retryHeader) : NaN) : now + retrySeconds * 1000;
  const retryAt = Number.isFinite(retryMs) && retryMs > now && retryMs <= 8640000000000000 ? new Date(retryMs).toISOString() : "";
  const deadlines = [reset && (remaining === 0 || quotaStopped) && resetMs > now ? resetMs : NaN, retryAt ? retryMs : NaN].filter(Number.isFinite);
  const deadline = Math.max(...deadlines);
  const retryNotBefore = Number.isFinite(deadline) && deadline > now && deadline <= 8640000000000000 ? new Date(deadline).toISOString() : "";
  const resourceHeader = header("x-ratelimit-resource");
  const resource = ["core", "search", "graphql", "code_search", "integration_manifest", "actions_runner_registration"].includes(resourceHeader) ? resourceHeader : "unknown";
  return { resource, remaining, limit, used, reset, retryRestricted: retryHeader != null, retryNotBefore };
}

/**
 * Derive diagnostics only from the failed business response, never another
 * client's /rate_limit precheck. Publish validated metadata, not API payloads.
 * @param {any} error
 * @param {number} [now]
 */
function getGitHubAPIDiagnostic(error, now = Date.now()) {
  const quotaStopped = error?.code === "DAILY_AIC_QUOTA_STOP";
  const metadata = getGitHubAPIQuota(error?.response?.headers, now, quotaStopped);
  const { resource, remaining, limit, used, reset, retryRestricted, retryNotBefore } = metadata;
  const status = Number(error?.status || error?.response?.status || 0);
  const secondary = typeof error?.message === "string" && /\bsecondary rate limit\b/i.test(error.message);
  const limited = status === 429 || (status === 403 && (remaining === 0 || retryRestricted || isRateLimitError(error) || secondary));
  const quota = quotaStopped || limited;
  if (!quota && status !== 401 && status !== 403) return null;

  const classification = status === 401 ? "authentication_failed" : status === 403 && !limited ? "permission_denied" : quota ? (remaining === 0 ? "quota_exhausted" : quotaStopped ? "quota_reserve" : "rate_limited") : "permission_denied";
  const details = quota
    ? `resource=${resource}; remaining=${remaining ?? "unknown"}; limit=${limit ?? "unknown"}; used=${used ?? "unknown"}; reset=${reset || "unknown"}; retry_not_before=${retryNotBefore || "unknown"}${quotaStopped && !limited && (status === 401 || status === 403) ? "; quota reserve guard also stopped further reads" : ""}`
    : "Check the credential's access to this operation; quota exhaustion was not established.";
  return { quota, classification, message: `GitHub API ${classification} (HTTP ${status >= 100 && status <= 599 && Number.isInteger(status) ? status : "unknown"}; ${details})` };
}

module.exports = { getGitHubAPIQuota, getGitHubAPIDiagnostic };
