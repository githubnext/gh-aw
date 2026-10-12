// @ts-check

const SAFE_HEADERS = ["x-ratelimit-limit", "x-ratelimit-remaining", "x-ratelimit-used", "x-ratelimit-reset", "retry-after"];

function safeResponseHeaders(headers) {
  return Object.fromEntries(
    SAFE_HEADERS.flatMap(name => {
      const value = typeof headers?.get === "function" ? headers.get(name) : headers?.[name];
      return value == null ? [] : [[name, String(value)]];
    })
  );
}

function apiError(status, headers, message) {
  const error = new Error(message);
  return Object.assign(error, { status, response: { status, headers: safeResponseHeaders(headers) } });
}

function retryNotBefore(headers, now = Date.now(), quotaStopped = false) {
  const safe = safeResponseHeaders(headers);
  const reset = quotaStopped || safe["x-ratelimit-remaining"] === "0" ? Number(safe["x-ratelimit-reset"]) * 1000 : NaN;
  const retry = safe["retry-after"];
  const retryAt = retry == null ? NaN : /^\d+$/.test(retry) ? now + Number(retry) * 1000 : Date.parse(retry);
  const times = [reset, retryAt].filter(value => Number.isFinite(value) && value > now);
  return times.length ? new Date(Math.max(...times)).toISOString() : "";
}

function getErrorStatus(error) {
  return Number(error?.status || error?.response?.status || 0);
}

function isQuotaError(error) {
  const headers = safeResponseHeaders(error?.response?.headers);
  const status = getErrorStatus(error);
  return error?.code === "DAILY_AIC_QUOTA_STOP" || status === 429 || (status === 403 && (headers["x-ratelimit-remaining"] === "0" || headers["retry-after"] != null));
}

function isTransientError(error) {
  if (isQuotaError(error) || safeResponseHeaders(error?.response?.headers)["retry-after"] != null) return false;
  const status = getErrorStatus(error);
  if (status >= 400) return status >= 500 && status < 600;
  const codes = ["ECONNRESET", "ECONNREFUSED", "ETIMEDOUT", "EAI_AGAIN", "UND_ERR_CONNECT_TIMEOUT", "UND_ERR_HEADERS_TIMEOUT", "UND_ERR_BODY_TIMEOUT", "UND_ERR_SOCKET"];
  return ["TimeoutError", "AbortError"].includes(error?.name) || codes.includes(error?.code) || (!!error?.cause && isTransientError(error.cause));
}

/**
 * Observe business responses, including native-fetch artifact requests. Do not use
 * /rate_limit as the authority for a different request's quota or retry deadline.
 * Retry transient reads only; shared quota and server retry restrictions stop immediately.
 */
function createAPIBudget(reserve = 100) {
  let requests = 0;
  let latest = {};
  const budget = {
    observe(response) {
      requests++;
      latest = safeResponseHeaders(response.headers);
      const remaining = latest["x-ratelimit-remaining"];
      const limited = response.status === 429 || (response.status === 403 && (remaining === "0" || latest["retry-after"] != null));
      if (limited || (remaining != null && Number(remaining) <= reserve)) {
        throw Object.assign(apiError(response.status, latest, "Daily AIC inspection stopped to preserve GitHub API quota"), { code: "DAILY_AIC_QUOTA_STOP" });
      }
    },
    async request(operation, observe = true) {
      for (let attempt = 0; ; attempt++) {
        try {
          let response;
          try {
            response = await operation();
          } catch (error) {
            if (observe && error?.response) budget.observe(error.response);
            throw error;
          }
          if (observe) budget.observe(response);
          if (response.status >= 500 && response.status < 600) {
            throw apiError(response.status, response.headers, "Daily AIC read failed with a server error");
          }
          if (attempt > 0) {
            core.info(`[daily-workflow-aic] Transient read recovered: ${JSON.stringify({ attempts: attempt + 1, status: response.status || null, requests })}`);
          }
          return response;
        } catch (error) {
          const transient = isTransientError(error);
          const status = getErrorStatus(error) || null;
          if (attempt >= 2 || !transient) {
            const reason = isQuotaError(error) ? "quota_stop" : safeResponseHeaders(error?.response?.headers)["retry-after"] != null ? "retry_restricted" : transient ? "retry_exhausted" : "non_transient";
            core.info(`[daily-workflow-aic] Stopped daily AIC read: ${JSON.stringify({ attempts: attempt + 1, status, reason, requests })}`);
            throw error;
          }
          const delayMs = 1000 * 2 ** attempt;
          core.info(`[daily-workflow-aic] Retrying transient read: ${JSON.stringify({ attempt: attempt + 1, maxAttempts: 3, status, delayMs })}`);
          await new Promise(resolve => setTimeout(resolve, delayMs));
        }
      }
    },
    snapshot() {
      return {
        requests,
        remaining: Number(latest["x-ratelimit-remaining"] || 0),
        limit: Number(latest["x-ratelimit-limit"] || 0),
        used: Number(latest["x-ratelimit-used"] || 0),
        reset: latest["x-ratelimit-reset"] ? new Date(Number(latest["x-ratelimit-reset"]) * 1000).toISOString() : "",
      };
    },
  };
  return budget;
}

module.exports = { safeResponseHeaders, apiError, retryNotBefore, createAPIBudget, getErrorStatus, isQuotaError, isTransientError };
