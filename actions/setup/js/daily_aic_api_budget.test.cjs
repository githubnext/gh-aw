import { createRequire } from "node:module";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const require = createRequire(import.meta.url);
const { API_READ_OPERATIONS, apiError, createAPIBudget, isTransientError, retryNotBefore, getAPIReadFailure } = require("./daily_aic_api_budget.cjs");

beforeEach(() => {
  vi.useFakeTimers();
  global.core = { info: vi.fn() };
});
afterEach(() => {
  vi.useRealTimers();
  delete global.core;
});

const response = (status = 200, headers = { "x-ratelimit-remaining": "4000" }) => ({ status, headers });

describe("bounded daily AIC read retries", () => {
  it("classifies a headerless installation rate limit without retrying or publishing its payload", async () => {
    const error = apiError(403, {}, "API rate limit exceeded for installation; secret-test-value");
    const request = vi.fn().mockRejectedValue(error);
    await expect(createAPIBudget().request(request, true, API_READ_OPERATIONS.CURRENT_RUN)).rejects.toBe(error);
    expect(getAPIReadFailure(error)).toMatchObject({ attempts: 1, reason: "quota_stop", operation: "current_run" });
    expect(request).toHaveBeenCalledOnce();
    const logs = global.core.info.mock.calls.flat().join("\n");
    expect(logs).toContain("current_run: GitHub API rate_limited");
    expect(logs).toContain("remaining=unknown");
    expect(logs).not.toContain("secret-test-value");
  });

  it("does not let malformed reset metadata replace the original failure", async () => {
    const request = vi.fn().mockRejectedValue(apiError(403, { "x-ratelimit-remaining": "0", "x-ratelimit-reset": "9999999999999999" }, "private payload"));
    const budget = createAPIBudget();
    await expect(budget.request(request)).rejects.toMatchObject({ code: "DAILY_AIC_QUOTA_STOP" });
    expect(budget.snapshot().reset).toBe("");
    expect(retryNotBefore({ "x-ratelimit-remaining": "0", "x-ratelimit-reset": "9999999999999999" })).toBe("");
    expect(request).toHaveBeenCalledOnce();
  });

  it("retries only the failed read and observes each response once", async () => {
    const budget = createAPIBudget();
    const request = vi
      .fn()
      .mockRejectedValueOnce(apiError(502, { "x-ratelimit-remaining": "4001" }, "Server Error"))
      .mockResolvedValue(response());
    const pending = budget.request(request);
    await vi.runAllTimersAsync();
    await expect(pending).resolves.toEqual(response());
    expect(request).toHaveBeenCalledTimes(2);
    expect(budget.snapshot()).toMatchObject({ requests: 2, remaining: 4000 });
    expect(global.core.info).toHaveBeenCalledWith('[daily-workflow-aic] Retrying transient read: {"attempt":1,"maxAttempts":3,"status":502,"delayMs":1000,"operation":"unknown"}');
    expect(global.core.info).toHaveBeenCalledWith('[daily-workflow-aic] Transient read recovered: {"attempts":2,"status":200,"requests":2,"operation":"unknown"}');
  });

  it.each([500, 502, 503, 504])("stops HTTP %i after three attempts", async status => {
    const request = vi.fn(async () => response(status));
    const budget = createAPIBudget();
    const assertion = expect(budget.request(request)).rejects.toMatchObject({ status });
    await vi.runAllTimersAsync();
    await assertion;
    expect(request).toHaveBeenCalledTimes(3);
    expect(budget.snapshot().requests).toBe(3);
    expect(global.core.info).toHaveBeenCalledWith(`[daily-workflow-aic] Stopped daily AIC read: {"attempts":3,"status":${status},"reason":"retry_exhausted","requests":3,"operation":"unknown"}`);
  });

  it.each([401, 403, 404, 429])("never retries HTTP %i", async status => {
    const request = vi.fn().mockRejectedValue(apiError(status, {}, "denied"));
    await expect(createAPIBudget().request(request)).rejects.toMatchObject({ status });
    expect(request).toHaveBeenCalledOnce();
    expect(vi.getTimerCount()).toBe(0);
    expect(global.core.info).toHaveBeenCalledWith(`[daily-workflow-aic] Stopped daily AIC read: {"attempts":1,"status":${status},"reason":"${status === 429 ? "quota_stop" : "non_transient"}","requests":1,"operation":"unknown"}`);
  });

  it.each([200, 502])("stops at the quota reserve even on HTTP %i", async status => {
    const request = vi.fn(async () => response(status, { "x-ratelimit-remaining": "100" }));
    await expect(createAPIBudget().request(request)).rejects.toMatchObject({ code: "DAILY_AIC_QUOTA_STOP" });
    expect(request).toHaveBeenCalledOnce();
    expect(vi.getTimerCount()).toBe(0);
    expect(global.core.info).toHaveBeenCalledWith(`[daily-workflow-aic] Stopped daily AIC read: {"attempts":1,"status":${status},"reason":"quota_stop","requests":1,"operation":"unknown"}`);
  });

  it("does not retry a server error with an explicit retry restriction", async () => {
    const request = vi.fn(async () => response(503, { "retry-after": "120" }));
    await expect(createAPIBudget().request(request)).rejects.toMatchObject({ status: 503 });
    expect(request).toHaveBeenCalledOnce();
    expect(global.core.info).toHaveBeenCalledWith('[daily-workflow-aic] Stopped daily AIC read: {"attempts":1,"status":503,"reason":"retry_restricted","requests":1,"operation":"unknown"}');
  });

  it.each([
    Object.assign(new Error("connection reset"), { code: "ECONNRESET" }),
    Object.assign(new Error("timeout"), { name: "TimeoutError" }),
    new Error("fetch failed", { cause: Object.assign(new Error("timeout"), { code: "UND_ERR_CONNECT_TIMEOUT" }) }),
    Object.assign(new Error("failed to read response body", { cause: Object.assign(new Error("body timeout"), { code: "UND_ERR_BODY_TIMEOUT" }) }), { response: response() }),
  ])("recovers recognized transport failures", async error => {
    const request = vi.fn().mockRejectedValueOnce(error).mockResolvedValue(response());
    const budget = createAPIBudget();
    const assertion = expect(budget.request(request)).resolves.toEqual(response());
    await vi.runAllTimersAsync();
    await assertion;
    expect(request).toHaveBeenCalledTimes(2);
    expect(budget.snapshot().requests).toBe(error.response ? 2 : 1);
  });

  it.each([{ "x-ratelimit-remaining": "100" }, { "retry-after": "120" }])("never retries body timeouts through response restrictions %j", async headers => {
    const error = Object.assign(new Error("failed to read response body", { cause: Object.assign(new Error("body timeout"), { code: "UND_ERR_BODY_TIMEOUT" }) }), { response: response(200, headers) });
    const request = vi.fn().mockRejectedValue(error);
    const budget = createAPIBudget();
    await expect(budget.request(request)).rejects.toThrow();
    expect(request).toHaveBeenCalledOnce();
    expect(budget.snapshot().requests).toBe(1);
    expect(vi.getTimerCount()).toBe(0);
  });

  it("does not retry arbitrary accounting or programming errors", async () => {
    const request = vi.fn().mockRejectedValue(new Error("invalid accounting"));
    await expect(createAPIBudget().request(request)).rejects.toThrow("invalid accounting");
    expect(request).toHaveBeenCalledOnce();
    expect(isTransientError(new Error("invalid accounting"))).toBe(false);
  });

  it("does not count artifact storage responses as GitHub quota", async () => {
    const budget = createAPIBudget();
    await budget.request(async () => response(), false);
    expect(budget.snapshot().requests).toBe(0);
  });

  it("never logs error payloads, credentials, headers or signed download URLs", async () => {
    const sensitive = "sensitive-test-value";
    const error = apiError(502, { authorization: sensitive, location: `https://example.invalid/blob?token=${sensitive}` }, sensitive);
    error.request = { headers: { authorization: sensitive } };
    const request = vi.fn().mockRejectedValue(error);
    const assertion = expect(createAPIBudget().request(request)).rejects.toMatchObject({ status: 502 });
    await vi.runAllTimersAsync();
    await assertion;
    const logs = global.core.info.mock.calls.map(([message]) => message).join("\n");
    expect(logs).toContain('"reason":"retry_exhausted"');
    expect(logs).not.toContain(sensitive);
    expect(logs).not.toContain("example.invalid");
  });

  it.each(Object.values(API_READ_OPERATIONS))("attributes %s failures without changing error identity", async operation => {
    const error = apiError(502, {}, "private API payload");
    const request = vi.fn().mockRejectedValue(error);
    const assertion = expect(createAPIBudget().request(request, operation !== "artifact_download", operation)).rejects.toBe(error);
    await vi.runAllTimersAsync();
    await assertion;
    expect(getAPIReadFailure(error)).toEqual({ attempts: 3, status: 502, reason: "retry_exhausted", requests: operation === "artifact_download" ? 0 : 3, operation });
    expect(getAPIReadFailure(new Error("wrapper", { cause: error }))).toEqual(getAPIReadFailure(error));
    expect(Object.isFrozen(getAPIReadFailure(error))).toBe(true);
    expect(global.core.info.mock.calls.map(([message]) => message).join("\n")).not.toContain("private API payload");
  });

  it("rejects unknown operation labels without publishing them", async () => {
    const sensitive = "https://example.invalid/download?token=private";
    const operation = vi.fn();
    await expect(createAPIBudget().request(operation, true, sensitive)).rejects.toThrow("Unknown daily AIC read operation");
    expect(operation).not.toHaveBeenCalled();
    expect(global.core.info.mock.calls.map(([message]) => message).join("\n")).not.toContain(sensitive);
    const unrelated = new Error("unrelated");
    unrelated.cause = unrelated;
    expect(getAPIReadFailure(unrelated)).toBeNull();
  });

  it("does not trust caller-supplied failure metadata", async () => {
    const sensitive = "https://example.invalid/download?token=private";
    const error = Object.assign(apiError(403, {}, sensitive), { dailyAICReadFailure: { operation: sensitive } });
    const assertion = expect(createAPIBudget().request(() => Promise.reject(error), true, API_READ_OPERATIONS.WORKFLOW_HISTORY)).rejects.toBe(error);
    await assertion;
    expect(getAPIReadFailure(error)).toMatchObject({ operation: API_READ_OPERATIONS.WORKFLOW_HISTORY, attempts: 1, reason: "non_transient" });
    expect(global.core.info.mock.calls.map(([message]) => message).join("\n")).not.toContain(sensitive);
  });

  it("labels recovered reads without retaining the earlier failure as terminal", async () => {
    const error = apiError(502, {}, "private payload");
    const request = vi.fn().mockRejectedValueOnce(error).mockResolvedValue(response());
    const assertion = expect(createAPIBudget().request(request, true, API_READ_OPERATIONS.WORKFLOW_HISTORY)).resolves.toEqual(response());
    await vi.runAllTimersAsync();
    await assertion;
    expect(getAPIReadFailure(error)).toBeNull();
    expect(global.core.info).toHaveBeenCalledWith('[daily-workflow-aic] Transient read recovered: {"attempts":2,"status":200,"requests":2,"operation":"workflow_history"}');
  });
});

describe("daily AIC retry deadlines", () => {
  const now = Date.parse("2026-10-09T12:00:00Z");
  const reset = String(now / 1000 + 3600);

  it("ignores the rate-limit reset on an ordinary server failure", () => {
    expect(retryNotBefore({ "x-ratelimit-reset": reset, "x-ratelimit-remaining": "4000" }, now)).toBe("");
  });

  it("retains reset deadlines for quota reserve stops and exhausted quota", () => {
    expect(retryNotBefore({ "x-ratelimit-reset": reset }, now, true)).toBe(new Date(now + 3600000).toISOString());
    expect(retryNotBefore({ "x-ratelimit-reset": reset, "x-ratelimit-remaining": "0" }, now)).toBe(new Date(now + 3600000).toISOString());
  });

  it("honors Retry-After without an unrelated quota reset", () => {
    expect(retryNotBefore({ "x-ratelimit-reset": reset, "retry-after": "120" }, now)).toBe(new Date(now + 120000).toISOString());
    expect(retryNotBefore({ "retry-after": new Date(now + 120000).toUTCString() }, now)).toBe(new Date(now + 120000).toISOString());
  });
});
