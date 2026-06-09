// @ts-check

import { describe, expect, it } from "vitest";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { detectNonRetryableHarnessGuard } = require("./harness_retry_guard.cjs");

describe("harness_retry_guard.cjs", () => {
  it("detects AI credits exceeded markers", () => {
    const result = detectNonRetryableHarnessGuard("error: max_ai_credits_exceeded=true");
    expect(result.aiCreditsExceeded).toBe(true);
    expect(result.awfAPIProxyBlockingRequests).toBe(false);
  });

  it("detects AI credits rate-limit markers", () => {
    const result = detectNonRetryableHarnessGuard("error: ai_credits_rate_limit_error=true");
    expect(result.aiCreditsExceeded).toBe(true);
    expect(result.awfAPIProxyBlockingRequests).toBe(false);
  });

  it("detects AI credits budget markers", () => {
    const result = detectNonRetryableHarnessGuard("error: ai credits budget exceeded");
    expect(result.aiCreditsExceeded).toBe(true);
    expect(result.awfAPIProxyBlockingRequests).toBe(false);
  });

  it("detects AWF API proxy blocking request markers", () => {
    const result = detectNonRetryableHarnessGuard("awf api proxy is blocking requests for this run");
    expect(result.aiCreditsExceeded).toBe(false);
    expect(result.awfAPIProxyBlockingRequests).toBe(true);
  });

  it("detects API proxy blocking request markers without AWF prefix", () => {
    const result = detectNonRetryableHarnessGuard("api-proxy is blocking requests");
    expect(result.aiCreditsExceeded).toBe(false);
    expect(result.awfAPIProxyBlockingRequests).toBe(true);
  });

  it("detects API proxy blocked request markers", () => {
    const result = detectNonRetryableHarnessGuard("api proxy blocked request");
    expect(result.aiCreditsExceeded).toBe(false);
    expect(result.awfAPIProxyBlockingRequests).toBe(true);
  });

  it("detects DIFC filtered proxy block markers", () => {
    const result = detectNonRetryableHarnessGuard('{"type":"DIFC_FILTERED","reason":"blocked"}');
    expect(result.aiCreditsExceeded).toBe(false);
    expect(result.awfAPIProxyBlockingRequests).toBe(true);
  });

  it("returns false for non-string input", () => {
    const result = detectNonRetryableHarnessGuard(null);
    expect(result.aiCreditsExceeded).toBe(false);
    expect(result.awfAPIProxyBlockingRequests).toBe(false);
  });

  it("detects both flags when output contains both signals", () => {
    const result = detectNonRetryableHarnessGuard("max_ai_credits_exceeded=true DIFC_FILTERED");
    expect(result.aiCreditsExceeded).toBe(true);
    expect(result.awfAPIProxyBlockingRequests).toBe(true);
  });

  it("returns false when output has no guard markers", () => {
    const result = detectNonRetryableHarnessGuard("transient network timeout");
    expect(result.aiCreditsExceeded).toBe(false);
    expect(result.awfAPIProxyBlockingRequests).toBe(false);
  });
});
