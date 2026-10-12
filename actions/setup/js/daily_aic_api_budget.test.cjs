import { createRequire } from "node:module";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const require = createRequire(import.meta.url);
const { apiError, createAPIBudget, isTransientError, retryNotBefore } = require("./daily_aic_api_budget.cjs");

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
  });

  it.each([500, 502, 503, 504])("stops HTTP %i after three attempts", async status => {
    const request = vi.fn(async () => response(status));
    const budget = createAPIBudget();
    const assertion = expect(budget.request(request)).rejects.toMatchObject({ status });
    await vi.runAllTimersAsync();
    await assertion;
    expect(request).toHaveBeenCalledTimes(3);
    expect(budget.snapshot().requests).toBe(3);
  });

  it.each([401, 403, 404, 429])("never retries HTTP %i", async status => {
    const request = vi.fn().mockRejectedValue(apiError(status, {}, "denied"));
    await expect(createAPIBudget().request(request)).rejects.toMatchObject({ status });
    expect(request).toHaveBeenCalledOnce();
    expect(vi.getTimerCount()).toBe(0);
  });

  it.each([200, 502])("stops at the quota reserve even on HTTP %i", async status => {
    const request = vi.fn(async () => response(status, { "x-ratelimit-remaining": "100" }));
    await expect(createAPIBudget().request(request)).rejects.toMatchObject({ code: "DAILY_AIC_QUOTA_STOP" });
    expect(request).toHaveBeenCalledOnce();
    expect(vi.getTimerCount()).toBe(0);
  });

  it("does not retry a server error with an explicit retry restriction", async () => {
    const request = vi.fn(async () => response(503, { "retry-after": "120" }));
    await expect(createAPIBudget().request(request)).rejects.toMatchObject({ status: 503 });
    expect(request).toHaveBeenCalledOnce();
  });

  it.each([
    Object.assign(new Error("connection reset"), { code: "ECONNRESET" }),
    Object.assign(new Error("timeout"), { name: "TimeoutError" }),
    new Error("fetch failed", { cause: Object.assign(new Error("timeout"), { code: "UND_ERR_CONNECT_TIMEOUT" }) }),
  ])("recovers recognized transport failures", async error => {
    const request = vi.fn().mockRejectedValueOnce(error).mockResolvedValue(response());
    const pending = createAPIBudget().request(request);
    await vi.runAllTimersAsync();
    await expect(pending).resolves.toEqual(response());
    expect(request).toHaveBeenCalledTimes(2);
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
