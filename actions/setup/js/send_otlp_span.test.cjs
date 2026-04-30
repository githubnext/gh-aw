import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";

// ---------------------------------------------------------------------------
// Module import
// ---------------------------------------------------------------------------

const {
  isValidTraceId,
  isValidSpanId,
  generateTraceId,
  generateSpanId,
  toNanoString,
  buildAttr,
  buildOTLPPayload,
  sanitizeOTLPPayload,
  parseOTLPHeaders,
  sendOTLPSpan,
  sendJobSetupSpan,
  sendJobConclusionSpan,
  readLastRateLimitEntry,
  GITHUB_RATE_LIMITS_JSONL_PATH,
  OTEL_JSONL_PATH,
  appendToOTLPJSONL,
  SPAN_KIND_INTERNAL,
  SPAN_KIND_SERVER,
} = await import("./send_otlp_span.cjs");

// ---------------------------------------------------------------------------
// isValidTraceId
// ---------------------------------------------------------------------------

describe("isValidTraceId", () => {
  it("accepts a valid 32-character lowercase hex trace ID", () => {
    expect(isValidTraceId("a".repeat(32))).toBe(true);
    expect(isValidTraceId("0123456789abcdef0123456789abcdef")).toBe(true);
  });

  it("rejects uppercase hex characters", () => {
    expect(isValidTraceId("A".repeat(32))).toBe(false);
  });

  it("rejects strings that are too short or too long", () => {
    expect(isValidTraceId("a".repeat(31))).toBe(false);
    expect(isValidTraceId("a".repeat(33))).toBe(false);
  });

  it("rejects empty string", () => {
    expect(isValidTraceId("")).toBe(false);
  });

  it("rejects non-hex characters", () => {
    expect(isValidTraceId("z".repeat(32))).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// isValidSpanId
// ---------------------------------------------------------------------------

describe("isValidSpanId", () => {
  it("accepts a valid 16-character lowercase hex span ID", () => {
    expect(isValidSpanId("b".repeat(16))).toBe(true);
    expect(isValidSpanId("0123456789abcdef")).toBe(true);
  });

  it("rejects uppercase hex characters", () => {
    expect(isValidSpanId("B".repeat(16))).toBe(false);
  });

  it("rejects strings that are too short or too long", () => {
    expect(isValidSpanId("b".repeat(15))).toBe(false);
    expect(isValidSpanId("b".repeat(17))).toBe(false);
  });

  it("rejects empty string", () => {
    expect(isValidSpanId("")).toBe(false);
  });

  it("rejects non-hex characters", () => {
    expect(isValidSpanId("z".repeat(16))).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// generateTraceId
// ---------------------------------------------------------------------------

describe("generateTraceId", () => {
  it("returns a 32-character hex string", () => {
    const id = generateTraceId();
    expect(id).toMatch(/^[0-9a-f]{32}$/);
  });

  it("returns a unique value on each call", () => {
    expect(generateTraceId()).not.toBe(generateTraceId());
  });
});

// ---------------------------------------------------------------------------
// generateSpanId
// ---------------------------------------------------------------------------

describe("generateSpanId", () => {
  it("returns a 16-character hex string", () => {
    const id = generateSpanId();
    expect(id).toMatch(/^[0-9a-f]{16}$/);
  });

  it("returns a unique value on each call", () => {
    expect(generateSpanId()).not.toBe(generateSpanId());
  });
});

// ---------------------------------------------------------------------------
// toNanoString
// ---------------------------------------------------------------------------

describe("toNanoString", () => {
  it("converts milliseconds to nanoseconds string", () => {
    expect(toNanoString(1000)).toBe("1000000000");
  });

  it("handles zero", () => {
    expect(toNanoString(0)).toBe("0");
  });

  it("handles a realistic GitHub Actions timestamp without precision loss", () => {
    const ms = 1700000000000; // 2023-11-14T22:13:20Z
    const nanos = toNanoString(ms);
    expect(nanos).toBe("1700000000000000000");
  });

  it("truncates fractional milliseconds", () => {
    // 1500.9 ms should truncate to 1500
    expect(toNanoString(1500.9)).toBe("1500000000");
  });
});

// ---------------------------------------------------------------------------
// buildAttr
// ---------------------------------------------------------------------------

describe("buildAttr", () => {
  it("returns stringValue for string input", () => {
    expect(buildAttr("k", "v")).toEqual({ key: "k", value: { stringValue: "v" } });
  });

  it("returns intValue for number input", () => {
    expect(buildAttr("k", 42)).toEqual({ key: "k", value: { intValue: 42 } });
  });

  it("returns boolValue for boolean input", () => {
    expect(buildAttr("k", true)).toEqual({ key: "k", value: { boolValue: true } });
    expect(buildAttr("k", false)).toEqual({ key: "k", value: { boolValue: false } });
  });

  it("coerces non-string non-number non-boolean to stringValue", () => {
    // @ts-expect-error intentional type violation for coverage
    expect(buildAttr("k", null).value).toHaveProperty("stringValue");
  });
});

// ---------------------------------------------------------------------------
// buildOTLPPayload
// ---------------------------------------------------------------------------

describe("buildOTLPPayload", () => {
  it("produces a valid OTLP resourceSpans structure", () => {
    const traceId = "a".repeat(32);
    const spanId = "b".repeat(16);
    const payload = buildOTLPPayload({
      traceId,
      spanId,
      spanName: "gh-aw.job.setup",
      startMs: 1000,
      endMs: 2000,
      serviceName: "gh-aw",
      scopeVersion: "v1.2.3",
      attributes: [buildAttr("foo", "bar")],
    });

    expect(payload.resourceSpans).toHaveLength(1);
    const rs = payload.resourceSpans[0];

    // Resource
    expect(rs.resource.attributes).toContainEqual({ key: "service.name", value: { stringValue: "gh-aw" } });
    expect(rs.resource.attributes).toContainEqual({ key: "service.version", value: { stringValue: "v1.2.3" } });

    // Scope — name is always "gh-aw"; version comes from scopeVersion
    expect(rs.scopeSpans).toHaveLength(1);
    expect(rs.scopeSpans[0].scope.name).toBe("gh-aw");
    expect(rs.scopeSpans[0].scope.version).toBe("v1.2.3");

    // Span
    const span = rs.scopeSpans[0].spans[0];
    expect(span.traceId).toBe(traceId);
    expect(span.spanId).toBe(spanId);
    expect(span.name).toBe("gh-aw.job.setup");
    expect(span.kind).toBe(SPAN_KIND_INTERNAL);
    expect(span.startTimeUnixNano).toBe(toNanoString(1000));
    expect(span.endTimeUnixNano).toBe(toNanoString(2000));
    expect(span.status.code).toBe(1);
    expect(span.attributes).toContainEqual({ key: "foo", value: { stringValue: "bar" } });
  });

  it("uses 'unknown' as scope version when scopeVersion is omitted", () => {
    const payload = buildOTLPPayload({
      traceId: "a".repeat(32),
      spanId: "b".repeat(16),
      spanName: "test",
      startMs: 0,
      endMs: 1,
      serviceName: "gh-aw",
      attributes: [],
    });
    expect(payload.resourceSpans[0].scopeSpans[0].scope.version).toBe("unknown");
  });

  it("omits service.version from resource attributes when scopeVersion is 'unknown'", () => {
    const payload = buildOTLPPayload({
      traceId: "a".repeat(32),
      spanId: "b".repeat(16),
      spanName: "test",
      startMs: 0,
      endMs: 1,
      serviceName: "gh-aw",
      scopeVersion: "unknown",
      attributes: [],
    });
    const resourceKeys = payload.resourceSpans[0].resource.attributes.map(a => a.key);
    expect(resourceKeys).not.toContain("service.version");
  });

  it("omits service.version from resource attributes when scopeVersion is omitted", () => {
    const payload = buildOTLPPayload({
      traceId: "a".repeat(32),
      spanId: "b".repeat(16),
      spanName: "test",
      startMs: 0,
      endMs: 1,
      serviceName: "gh-aw",
      attributes: [],
    });
    const resourceKeys = payload.resourceSpans[0].resource.attributes.map(a => a.key);
    expect(resourceKeys).not.toContain("service.version");
  });

  it("merges caller-supplied resourceAttributes into the resource block", () => {
    const payload = buildOTLPPayload({
      traceId: "a".repeat(32),
      spanId: "b".repeat(16),
      spanName: "test",
      startMs: 0,
      endMs: 1,
      serviceName: "gh-aw",
      scopeVersion: "v1.0.0",
      attributes: [],
      resourceAttributes: [buildAttr("github.repository", "owner/repo"), buildAttr("github.run_id", "123")],
    });
    const rs = payload.resourceSpans[0];
    expect(rs.resource.attributes).toContainEqual({ key: "github.repository", value: { stringValue: "owner/repo" } });
    expect(rs.resource.attributes).toContainEqual({ key: "github.run_id", value: { stringValue: "123" } });
    expect(rs.resource.attributes).toContainEqual({ key: "service.version", value: { stringValue: "v1.0.0" } });
  });

  it("includes parentSpanId in span when provided", () => {
    const payload = buildOTLPPayload({
      traceId: "a".repeat(32),
      spanId: "b".repeat(16),
      parentSpanId: "c".repeat(16),
      spanName: "test",
      startMs: 0,
      endMs: 1,
      serviceName: "gh-aw",
      attributes: [],
    });
    const span = payload.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.parentSpanId).toBe("c".repeat(16));
  });

  it("omits parentSpanId from span when not provided", () => {
    const payload = buildOTLPPayload({
      traceId: "a".repeat(32),
      spanId: "b".repeat(16),
      spanName: "test",
      startMs: 0,
      endMs: 1,
      serviceName: "gh-aw",
      attributes: [],
    });
    const span = payload.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.parentSpanId).toBeUndefined();
  });

  it("uses SPAN_KIND_INTERNAL (1) by default when kind is not specified", () => {
    const payload = buildOTLPPayload({
      traceId: "a".repeat(32),
      spanId: "b".repeat(16),
      spanName: "test",
      startMs: 0,
      endMs: 1,
      serviceName: "gh-aw",
      attributes: [],
    });
    const span = payload.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.kind).toBe(SPAN_KIND_INTERNAL);
  });

  it("uses the caller-supplied kind when specified (e.g. SPAN_KIND_SERVER)", () => {
    const payload = buildOTLPPayload({
      traceId: "a".repeat(32),
      spanId: "b".repeat(16),
      spanName: "test",
      startMs: 0,
      endMs: 1,
      serviceName: "gh-aw",
      attributes: [],
      kind: SPAN_KIND_SERVER,
    });
    const span = payload.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.kind).toBe(SPAN_KIND_SERVER);
  });

  it("includes events array in span when events are provided", () => {
    const events = [
      {
        timeUnixNano: toNanoString(1000),
        name: "exception",
        attributes: [buildAttr("exception.message", "something failed")],
      },
    ];
    const payload = buildOTLPPayload({
      traceId: "a".repeat(32),
      spanId: "b".repeat(16),
      spanName: "test",
      startMs: 0,
      endMs: 1,
      serviceName: "gh-aw",
      attributes: [],
      events,
    });
    const span = payload.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.events).toHaveLength(1);
    expect(span.events[0].name).toBe("exception");
    expect(span.events[0].attributes).toContainEqual({ key: "exception.message", value: { stringValue: "something failed" } });
  });

  it("includes multiple events when provided", () => {
    const events = [
      { timeUnixNano: toNanoString(1000), name: "exception", attributes: [buildAttr("exception.message", "error A")] },
      { timeUnixNano: toNanoString(1000), name: "exception", attributes: [buildAttr("exception.message", "error B")] },
    ];
    const payload = buildOTLPPayload({
      traceId: "a".repeat(32),
      spanId: "b".repeat(16),
      spanName: "test",
      startMs: 0,
      endMs: 1,
      serviceName: "gh-aw",
      attributes: [],
      events,
    });
    const span = payload.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.events).toHaveLength(2);
    expect(span.events[0].attributes[0].value.stringValue).toBe("error A");
    expect(span.events[1].attributes[0].value.stringValue).toBe("error B");
  });

  it("omits events from span when events array is empty", () => {
    const payload = buildOTLPPayload({
      traceId: "a".repeat(32),
      spanId: "b".repeat(16),
      spanName: "test",
      startMs: 0,
      endMs: 1,
      serviceName: "gh-aw",
      attributes: [],
      events: [],
    });
    const span = payload.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.events).toBeUndefined();
  });

  it("omits events from span when events is not provided", () => {
    const payload = buildOTLPPayload({
      traceId: "a".repeat(32),
      spanId: "b".repeat(16),
      spanName: "test",
      startMs: 0,
      endMs: 1,
      serviceName: "gh-aw",
      attributes: [],
    });
    const span = payload.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.events).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// sanitizeOTLPPayload
// ---------------------------------------------------------------------------

describe("sanitizeOTLPPayload", () => {
  /** Build a minimal OTLP payload with the given span and resource attributes. */
  function makePayload(spanAttrs = [], resourceAttrs = []) {
    return buildOTLPPayload({
      traceId: "a".repeat(32),
      spanId: "b".repeat(16),
      spanName: "test",
      startMs: 0,
      endMs: 1,
      serviceName: "gh-aw",
      attributes: spanAttrs,
      resourceAttributes: resourceAttrs,
    });
  }

  it("redacts span attribute values whose keys match sensitive patterns", () => {
    const payload = makePayload([buildAttr("gh.auth_token", "super-secret"), buildAttr("safe.label", "value")]);
    const sanitized = sanitizeOTLPPayload(payload);
    const attrs = sanitized.resourceSpans[0].scopeSpans[0].spans[0].attributes;
    const tokenAttr = attrs.find(a => a.key === "gh.auth_token");
    expect(tokenAttr.value.stringValue, "sensitive attribute should be redacted").toBe("[REDACTED]");
    const safeAttr = attrs.find(a => a.key === "safe.label");
    expect(safeAttr.value.stringValue, "non-sensitive attribute should be unchanged").toBe("value");
  });

  it("redacts span attributes matching all sensitive key patterns", () => {
    const sensitiveKeys = ["token", "secret", "password", "passwd", "key", "api_key", "auth", "credential", "access_key"];
    const attrs = sensitiveKeys.map(k => buildAttr(k, "should-be-redacted"));
    const payload = makePayload(attrs);
    const sanitized = sanitizeOTLPPayload(payload);
    const spanAttrs = sanitized.resourceSpans[0].scopeSpans[0].spans[0].attributes;
    for (const k of sensitiveKeys) {
      const attr = spanAttrs.find(a => a.key === k);
      expect(attr.value.stringValue, `${k} should be redacted`).toBe("[REDACTED]");
    }
  });

  it("does not redact non-sensitive compound keys containing 'key' after underscore", () => {
    const nonSensitiveKeys = ["sort_key", "cache_key", "primary_key", "partition_key"];
    const attrs = nonSensitiveKeys.map(k => buildAttr(k, "safe-value"));
    const payload = makePayload(attrs);
    const sanitized = sanitizeOTLPPayload(payload);
    const spanAttrs = sanitized.resourceSpans[0].scopeSpans[0].spans[0].attributes;
    for (const k of nonSensitiveKeys) {
      const attr = spanAttrs.find(a => a.key === k);
      expect(attr.value.stringValue, `${k} should not be redacted`).toBe("safe-value");
    }
  });

  it("redacts resource attribute values whose keys match sensitive patterns", () => {
    const payload = makePayload([], [buildAttr("db.password", "hunter2"), buildAttr("service.name", "gh-aw")]);
    const sanitized = sanitizeOTLPPayload(payload);
    const resourceAttrs = sanitized.resourceSpans[0].resource.attributes;
    const pwAttr = resourceAttrs.find(a => a.key === "db.password");
    expect(pwAttr.value.stringValue, "sensitive resource attribute should be redacted").toBe("[REDACTED]");
    const svcAttr = resourceAttrs.find(a => a.key === "service.name");
    expect(svcAttr.value.stringValue, "service.name should be unchanged").toBe("gh-aw");
  });

  it("truncates string values exceeding 1024 characters", () => {
    const longValue = "x".repeat(2000);
    const payload = makePayload([buildAttr("large.output", longValue)]);
    const sanitized = sanitizeOTLPPayload(payload);
    const attr = sanitized.resourceSpans[0].scopeSpans[0].spans[0].attributes.find(a => a.key === "large.output");
    expect(attr.value.stringValue, "value should be truncated to 1024 chars").toBe(longValue.slice(0, 1024));
  });

  it("does not redact non-string sensitive attribute values (e.g. intValue, boolValue)", () => {
    const intAttr = { key: "auth_count", value: { intValue: 42 } };
    const boolAttr = { key: "token_valid", value: { boolValue: true } };
    const payload = makePayload([intAttr, boolAttr]);
    const sanitized = sanitizeOTLPPayload(payload);
    const spanAttrs = sanitized.resourceSpans[0].scopeSpans[0].spans[0].attributes;
    const sanitizedInt = spanAttrs.find(a => a.key === "auth_count");
    expect(sanitizedInt.value.intValue, "intValue sensitive attribute should not be redacted").toBe(42);
    const sanitizedBool = spanAttrs.find(a => a.key === "token_valid");
    expect(sanitizedBool.value.boolValue, "boolValue sensitive attribute should not be redacted").toBe(true);
  });

  it("does not mutate the original payload", () => {
    const payload = makePayload([buildAttr("auth_token", "secret-value")]);
    const originalAttr = payload.resourceSpans[0].scopeSpans[0].spans[0].attributes[0];
    sanitizeOTLPPayload(payload);
    expect(originalAttr.value.stringValue, "original payload should not be mutated").toBe("secret-value");
  });

  it("returns the payload unchanged when resourceSpans is absent", () => {
    const payload = { custom: "data" };
    expect(sanitizeOTLPPayload(payload), "payload without resourceSpans should be returned as-is").toBe(payload);
  });

  it("redacts sensitive keys in span event attributes", () => {
    const payload = makePayload([]);
    // Manually add events with sensitive attributes to the span
    const span = payload.resourceSpans[0].scopeSpans[0].spans[0];
    span.events = [
      {
        timeUnixNano: "1000000000",
        name: "exception",
        attributes: [buildAttr("exception.message", "safe message"), buildAttr("auth_token", "super-secret-token")],
      },
    ];
    const sanitized = sanitizeOTLPPayload(payload);
    const events = sanitized.resourceSpans[0].scopeSpans[0].spans[0].events;
    expect(events).toHaveLength(1);
    const msgAttr = events[0].attributes.find(a => a.key === "exception.message");
    expect(msgAttr.value.stringValue, "non-sensitive event attribute should be unchanged").toBe("safe message");
    const tokenAttr = events[0].attributes.find(a => a.key === "auth_token");
    expect(tokenAttr.value.stringValue, "sensitive event attribute should be redacted").toBe("[REDACTED]");
  });

  it("truncates long string values in span event attributes", () => {
    const payload = makePayload([]);
    const span = payload.resourceSpans[0].scopeSpans[0].spans[0];
    const longValue = "y".repeat(2000);
    span.events = [
      {
        timeUnixNano: "1000000000",
        name: "exception",
        attributes: [buildAttr("exception.message", longValue)],
      },
    ];
    const sanitized = sanitizeOTLPPayload(payload);
    const events = sanitized.resourceSpans[0].scopeSpans[0].spans[0].events;
    const msgAttr = events[0].attributes.find(a => a.key === "exception.message");
    expect(msgAttr.value.stringValue.length, "long event attribute should be truncated").toBe(1024);
  });

  it("preserves span events without attributes unchanged", () => {
    const payload = makePayload([]);
    const span = payload.resourceSpans[0].scopeSpans[0].spans[0];
    span.events = [{ timeUnixNano: "1000000000", name: "custom-event" }];
    const sanitized = sanitizeOTLPPayload(payload);
    const events = sanitized.resourceSpans[0].scopeSpans[0].spans[0].events;
    expect(events).toHaveLength(1);
    expect(events[0].name).toBe("custom-event");
  });
});

// ---------------------------------------------------------------------------
// sendOTLPSpan
// ---------------------------------------------------------------------------

describe("sendOTLPSpan", () => {
  let mkdirSpy, appendSpy;

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
    mkdirSpy = vi.spyOn(fs, "mkdirSync").mockImplementation(() => {});
    appendSpy = vi.spyOn(fs, "appendFileSync").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    mkdirSpy.mockRestore();
    appendSpy.mockRestore();
  });

  it("POSTs JSON payload to endpoint/v1/traces", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    const payload = { resourceSpans: [] };
    await sendOTLPSpan("https://traces.example.com:4317", payload);

    expect(mockFetch).toHaveBeenCalledOnce();
    const [url, init] = mockFetch.mock.calls[0];
    expect(url).toBe("https://traces.example.com:4317/v1/traces");
    expect(init.method).toBe("POST");
    expect(init.headers["Content-Type"]).toBe("application/json");
    expect(JSON.parse(init.body)).toEqual(payload);
  });

  it("strips trailing slash from endpoint before appending /v1/traces", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    await sendOTLPSpan("https://traces.example.com/", {});
    const [url] = mockFetch.mock.calls[0];
    expect(url).toBe("https://traces.example.com/v1/traces");
  });

  it("warns (does not throw) when server returns non-2xx status on all retries", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: false, status: 400, statusText: "Bad Request" });
    vi.stubGlobal("fetch", mockFetch);
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    // Should not throw
    await expect(sendOTLPSpan("https://traces.example.com", {}, { maxRetries: 1, baseDelayMs: 1 })).resolves.toBeUndefined();

    // Two attempts (1 initial + 1 retry)
    expect(mockFetch).toHaveBeenCalledTimes(2);
    expect(warnSpy).toHaveBeenCalledTimes(2);
    expect(warnSpy.mock.calls[0][0]).toContain("attempt 1/2 failed");
    expect(warnSpy.mock.calls[1][0]).toContain("failed after 2 attempts");

    warnSpy.mockRestore();
  });

  it("retries on failure and succeeds on second attempt", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({ ok: false, status: 503, statusText: "Service Unavailable" }).mockResolvedValueOnce({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    await sendOTLPSpan("https://traces.example.com", {}, { maxRetries: 2, baseDelayMs: 1 });

    expect(mockFetch).toHaveBeenCalledTimes(2);
    expect(warnSpy).toHaveBeenCalledTimes(1);
    expect(warnSpy.mock.calls[0][0]).toContain("attempt 1/3 failed");

    warnSpy.mockRestore();
  });

  it("warns (does not throw) when fetch rejects on all retries", async () => {
    const mockFetch = vi.fn().mockRejectedValue(new Error("network error"));
    vi.stubGlobal("fetch", mockFetch);
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    await expect(sendOTLPSpan("https://traces.example.com", {}, { maxRetries: 1, baseDelayMs: 1 })).resolves.toBeUndefined();

    expect(mockFetch).toHaveBeenCalledTimes(2);
    expect(warnSpy.mock.calls[1][0]).toContain("error after 2 attempts");

    warnSpy.mockRestore();
  });
});

// ---------------------------------------------------------------------------
// readLastRateLimitEntry
// ---------------------------------------------------------------------------

describe("readLastRateLimitEntry", () => {
  let readFileSpy;

  beforeEach(() => {
    readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(() => {
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });
  });

  afterEach(() => {
    readFileSpy.mockRestore();
  });

  it("returns null when the file does not exist", () => {
    expect(readLastRateLimitEntry()).toBeNull();
  });

  it("returns null when the file is empty", () => {
    readFileSpy.mockImplementation(filePath => {
      if (filePath === GITHUB_RATE_LIMITS_JSONL_PATH) return "";
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });
    expect(readLastRateLimitEntry()).toBeNull();
  });

  it("returns null when the file contains only blank lines", () => {
    readFileSpy.mockImplementation(filePath => {
      if (filePath === GITHUB_RATE_LIMITS_JSONL_PATH) return "\n\n  \n";
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });
    expect(readLastRateLimitEntry()).toBeNull();
  });

  it("returns the parsed entry for a single-line file", () => {
    const entry = { resource: "core", limit: 5000, remaining: 4823, used: 177 };
    readFileSpy.mockImplementation(filePath => {
      if (filePath === GITHUB_RATE_LIMITS_JSONL_PATH) return JSON.stringify(entry) + "\n";
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });
    expect(readLastRateLimitEntry()).toEqual(entry);
  });

  it("returns the last entry for a multi-line file", () => {
    const first = { resource: "core", remaining: 4900 };
    const last = { resource: "core", remaining: 4500 };
    readFileSpy.mockImplementation(filePath => {
      if (filePath === GITHUB_RATE_LIMITS_JSONL_PATH) {
        return JSON.stringify(first) + "\n" + JSON.stringify(last) + "\n";
      }
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });
    expect(readLastRateLimitEntry()).toEqual(last);
  });

  it("returns null when the last line is invalid JSON", () => {
    readFileSpy.mockImplementation(filePath => {
      if (filePath === GITHUB_RATE_LIMITS_JSONL_PATH) return "not valid json\n";
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });
    expect(readLastRateLimitEntry()).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// appendToOTLPJSONL
// ---------------------------------------------------------------------------

describe("appendToOTLPJSONL", () => {
  let mkdirSpy, appendSpy;

  beforeEach(() => {
    mkdirSpy = vi.spyOn(fs, "mkdirSync").mockImplementation(() => {});
    appendSpy = vi.spyOn(fs, "appendFileSync").mockImplementation(() => {});
  });

  afterEach(() => {
    mkdirSpy.mockRestore();
    appendSpy.mockRestore();
  });

  it("writes payload as a JSON line to OTEL_JSONL_PATH", () => {
    const payload = { resourceSpans: [{ spans: [] }] };
    appendToOTLPJSONL(payload);

    expect(appendSpy).toHaveBeenCalledOnce();
    const [filePath, content] = appendSpy.mock.calls[0];
    expect(filePath).toBe(OTEL_JSONL_PATH);
    expect(content).toBe(JSON.stringify(payload) + "\n");
  });

  it("ensures /tmp/gh-aw directory exists before writing", () => {
    appendToOTLPJSONL({});

    expect(mkdirSpy).toHaveBeenCalledWith("/tmp/gh-aw", { recursive: true });
  });

  it("does not throw when appendFileSync fails", () => {
    appendSpy.mockImplementation(() => {
      throw new Error("disk full");
    });

    expect(() => appendToOTLPJSONL({ spans: [] })).not.toThrow();
  });
});

// ---------------------------------------------------------------------------
// sendOTLPSpan – JSONL mirror
// ---------------------------------------------------------------------------

describe("sendOTLPSpan JSONL mirror", () => {
  let mkdirSpy, appendSpy;

  beforeEach(() => {
    mkdirSpy = vi.spyOn(fs, "mkdirSync").mockImplementation(() => {});
    appendSpy = vi.spyOn(fs, "appendFileSync").mockImplementation(() => {});
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" }));
  });

  afterEach(() => {
    mkdirSpy.mockRestore();
    appendSpy.mockRestore();
    vi.unstubAllGlobals();
  });

  it("mirrors the payload to otel.jsonl even when fetch succeeds", async () => {
    const payload = { resourceSpans: [] };
    await sendOTLPSpan("https://traces.example.com", payload);

    expect(appendSpy).toHaveBeenCalledOnce();
    const [filePath, content] = appendSpy.mock.calls[0];
    expect(filePath).toBe(OTEL_JSONL_PATH);
    expect(content).toBe(JSON.stringify(payload) + "\n");
  });

  it("mirrors the payload to otel.jsonl even when fetch fails all retries", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false, status: 503, statusText: "Unavailable" }));
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    const payload = { resourceSpans: [{ note: "retry-test" }] };
    await sendOTLPSpan("https://traces.example.com", payload, { maxRetries: 1, baseDelayMs: 1 });

    expect(appendSpy).toHaveBeenCalledOnce();
    expect(appendSpy.mock.calls[0][1]).toBe(JSON.stringify(payload) + "\n");

    warnSpy.mockRestore();
  });

  it("skips JSONL mirror when skipJSONL is true", async () => {
    const payload = { resourceSpans: [{ note: "skip-test" }] };
    await sendOTLPSpan("https://traces.example.com", payload, { skipJSONL: true });

    expect(appendSpy).not.toHaveBeenCalled();
    expect(fetch).toHaveBeenCalledOnce();
  });
});

// ---------------------------------------------------------------------------
// parseOTLPHeaders
// ---------------------------------------------------------------------------

describe("parseOTLPHeaders", () => {
  it("returns empty object for empty/null/whitespace input", () => {
    expect(parseOTLPHeaders("")).toEqual({});
    expect(parseOTLPHeaders("   ")).toEqual({});
  });

  it("parses a single key=value pair", () => {
    expect(parseOTLPHeaders("Authorization=Bearer mytoken")).toEqual({ Authorization: "Bearer mytoken" });
  });

  it("parses multiple comma-separated key=value pairs", () => {
    expect(parseOTLPHeaders("X-Tenant=acme,X-Region=us-east-1")).toEqual({
      "X-Tenant": "acme",
      "X-Region": "us-east-1",
    });
  });

  it("handles percent-encoded values", () => {
    expect(parseOTLPHeaders("Authorization=Bearer%20tok%3Dvalue")).toEqual({ Authorization: "Bearer tok=value" });
  });

  it("decodes before trimming so encoded whitespace at edges is preserved", () => {
    // %20 at start/end of value should survive: decode first, then trim removes nothing
    expect(parseOTLPHeaders("X-Token=abc%20def")).toEqual({ "X-Token": "abc def" });
  });

  it("handles values containing = signs (only first = is delimiter)", () => {
    expect(parseOTLPHeaders("Authorization=Bearer base64==")).toEqual({ Authorization: "Bearer base64==" });
  });

  it("parses Sentry OTLP header format (value contains space and embedded = sign)", () => {
    // Sentry's OTLP auth header: x-sentry-auth: Sentry sentry_key=<key>
    // The value "Sentry sentry_key=abc123" contains both a space and an embedded =.
    expect(parseOTLPHeaders("x-sentry-auth=Sentry sentry_key=abc123def456")).toEqual({
      "x-sentry-auth": "Sentry sentry_key=abc123def456",
    });
  });

  it("parses Sentry header combined with another header", () => {
    expect(parseOTLPHeaders("x-sentry-auth=Sentry sentry_key=mykey,x-custom=value")).toEqual({
      "x-sentry-auth": "Sentry sentry_key=mykey",
      "x-custom": "value",
    });
  });

  it("skips malformed pairs with no =", () => {
    const result = parseOTLPHeaders("Valid=value,malformedNoEquals");
    expect(result).toEqual({ Valid: "value" });
  });

  it("skips pairs with empty key", () => {
    const result = parseOTLPHeaders("=value,Good=ok");
    expect(result).toEqual({ Good: "ok" });
  });
});

// ---------------------------------------------------------------------------
// sendOTLPSpan headers
// ---------------------------------------------------------------------------

describe("sendOTLPSpan with OTEL_EXPORTER_OTLP_HEADERS", () => {
  const savedHeaders = process.env.OTEL_EXPORTER_OTLP_HEADERS;
  let mkdirSpy, appendSpy;

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
    delete process.env.OTEL_EXPORTER_OTLP_HEADERS;
    mkdirSpy = vi.spyOn(fs, "mkdirSync").mockImplementation(() => {});
    appendSpy = vi.spyOn(fs, "appendFileSync").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    mkdirSpy.mockRestore();
    appendSpy.mockRestore();
    if (savedHeaders !== undefined) {
      process.env.OTEL_EXPORTER_OTLP_HEADERS = savedHeaders;
    } else {
      delete process.env.OTEL_EXPORTER_OTLP_HEADERS;
    }
  });

  it("includes custom headers when OTEL_EXPORTER_OTLP_HEADERS is set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_HEADERS = "Authorization=Bearer mytoken,X-Tenant=acme";
    await sendOTLPSpan("https://traces.example.com", {});

    const [, init] = mockFetch.mock.calls[0];
    expect(init.headers["Authorization"]).toBe("Bearer mytoken");
    expect(init.headers["X-Tenant"]).toBe("acme");
    expect(init.headers["Content-Type"]).toBe("application/json");
  });

  it("does not add extra headers when OTEL_EXPORTER_OTLP_HEADERS is absent", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    await sendOTLPSpan("https://traces.example.com", {});

    const [, init] = mockFetch.mock.calls[0];
    expect(Object.keys(init.headers)).toEqual(["Content-Type"]);
  });
});

// ---------------------------------------------------------------------------
// sendJobSetupSpan
// ---------------------------------------------------------------------------

describe("sendJobSetupSpan", () => {
  /** @type {Record<string, string | undefined>} */
  const savedEnv = {};
  const envKeys = [
    "OTEL_EXPORTER_OTLP_ENDPOINT",
    "OTEL_SERVICE_NAME",
    "INPUT_JOB_NAME",
    "INPUT_TRACE_ID",
    "GH_AW_INFO_WORKFLOW_NAME",
    "GH_AW_INFO_ENGINE_ID",
    "GITHUB_RUN_ID",
    "GITHUB_RUN_ATTEMPT",
    "GITHUB_ACTOR",
    "GITHUB_REPOSITORY",
    "GITHUB_EVENT_NAME",
    "GITHUB_REF",
    "GITHUB_REF_NAME",
    "GITHUB_HEAD_REF",
    "GITHUB_SHA",
    "GITHUB_WORKFLOW_REF",
    "GH_AW_INFO_VERSION",
    "GH_AW_INFO_STAGED",
  ];
  let mkdirSpy, appendSpy;

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
    for (const k of envKeys) {
      savedEnv[k] = process.env[k];
      delete process.env[k];
    }
    mkdirSpy = vi.spyOn(fs, "mkdirSync").mockImplementation(() => {});
    appendSpy = vi.spyOn(fs, "appendFileSync").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    for (const k of envKeys) {
      if (savedEnv[k] !== undefined) {
        process.env[k] = savedEnv[k];
      } else {
        delete process.env[k];
      }
    }
    mkdirSpy.mockRestore();
    appendSpy.mockRestore();
  });

  /**
   * Extract the scalar value from an OTLP attribute's `value` union, covering all
   * known OTLP value types (stringValue, intValue, boolValue).
   *
   * @param {{ key: string, value: { stringValue?: string, intValue?: number, boolValue?: boolean } }} attr
   * @returns {string | number | boolean | undefined}
   */
  function attrValue(attr) {
    if (attr.value.stringValue !== undefined) return attr.value.stringValue;
    if (attr.value.intValue !== undefined) return attr.value.intValue;
    if (attr.value.boolValue !== undefined) return attr.value.boolValue;
    return undefined;
  }

  it("returns a trace ID and span ID even when OTEL_EXPORTER_OTLP_ENDPOINT is not set", async () => {
    const { traceId, spanId } = await sendJobSetupSpan();
    expect(traceId).toMatch(/^[0-9a-f]{32}$/);
    expect(spanId).toMatch(/^[0-9a-f]{16}$/);
    expect(fetch).not.toHaveBeenCalled();
  });

  it("writes JSONL mirror even when OTEL_EXPORTER_OTLP_ENDPOINT is not set", async () => {
    await sendJobSetupSpan();
    expect(appendSpy).toHaveBeenCalledOnce();
    const [filePath, content] = appendSpy.mock.calls[0];
    expect(filePath).toBe(OTEL_JSONL_PATH);
    const payload = JSON.parse(content.trim());
    expect(payload).toHaveProperty("resourceSpans");
  });

  it("returns the same trace ID when called with INPUT_TRACE_ID and no endpoint", async () => {
    process.env.INPUT_TRACE_ID = "a".repeat(32);
    const { traceId } = await sendJobSetupSpan();
    expect(traceId).toBe("a".repeat(32));
    expect(fetch).not.toHaveBeenCalled();
  });

  it("generates a new trace ID when INPUT_TRACE_ID is invalid", async () => {
    process.env.INPUT_TRACE_ID = "not-a-valid-trace-id";
    const { traceId } = await sendJobSetupSpan();
    expect(traceId).toMatch(/^[0-9a-f]{32}$/);
    expect(traceId).not.toBe("not-a-valid-trace-id");
  });

  it("normalizes uppercase INPUT_TRACE_ID to lowercase and accepts it", async () => {
    // Trace IDs pasted from external tools may be uppercase; we normalise them.
    process.env.INPUT_TRACE_ID = "A".repeat(32);
    const { traceId } = await sendJobSetupSpan();
    expect(traceId).toBe("a".repeat(32));
  });

  it("rejects an invalid options.traceId and generates a new trace ID", async () => {
    const { traceId } = await sendJobSetupSpan({ traceId: "too-short" });
    expect(traceId).toMatch(/^[0-9a-f]{32}$/);
    expect(traceId).not.toBe("too-short");
  });

  it("sends a span when endpoint is configured and returns the trace ID and span ID", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.INPUT_JOB_NAME = "agent";
    process.env.GH_AW_INFO_WORKFLOW_NAME = "my-workflow";
    process.env.GH_AW_INFO_ENGINE_ID = "copilot";
    process.env.GITHUB_RUN_ID = "123456789";
    process.env.GITHUB_RUN_ATTEMPT = "2";
    process.env.GITHUB_ACTOR = "octocat";
    process.env.GITHUB_REPOSITORY = "owner/repo";

    const { traceId, spanId } = await sendJobSetupSpan();

    expect(traceId).toMatch(/^[0-9a-f]{32}$/);
    expect(spanId).toMatch(/^[0-9a-f]{16}$/);
    expect(mockFetch).toHaveBeenCalledOnce();
    const [url, init] = mockFetch.mock.calls[0];
    expect(url).toBe("https://traces.example.com/v1/traces");
    expect(init.method).toBe("POST");

    const body = JSON.parse(init.body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.name).toBe("gh-aw.agent.setup");
    expect(span.traceId).toBe(traceId);
    expect(span.spanId).toBe(spanId);

    const attrs = Object.fromEntries(span.attributes.map(a => [a.key, attrValue(a)]));
    expect(attrs["gh-aw.job.name"]).toBe("agent");
    expect(attrs["gh-aw.workflow.name"]).toBe("my-workflow");
    expect(attrs["gh-aw.engine.id"]).toBe("copilot");
    expect(attrs["gh-aw.run.id"]).toBe("123456789");
    expect(attrs["gh-aw.run.attempt"]).toBe("2");
    expect(attrs["gh-aw.run.actor"]).toBe("octocat");
    expect(attrs["gh-aw.repository"]).toBe("owner/repo");
  });

  it("defaults gh-aw.run.attempt to '1' when GITHUB_RUN_ATTEMPT is not set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    const attrs = Object.fromEntries(span.attributes.map(a => [a.key, a.value.stringValue]));
    expect(attrs["gh-aw.run.attempt"]).toBe("1");
  });

  it("uses trace ID from options.traceId for cross-job correlation", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    const correlationTraceId = "b".repeat(32);

    const { traceId } = await sendJobSetupSpan({ traceId: correlationTraceId });

    expect(traceId).toBe(correlationTraceId);
    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    expect(body.resourceSpans[0].scopeSpans[0].spans[0].traceId).toBe(correlationTraceId);
  });

  it("uses trace ID from INPUT_TRACE_ID env var when options.traceId is absent", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.INPUT_TRACE_ID = "c".repeat(32);

    const { traceId } = await sendJobSetupSpan();

    expect(traceId).toBe("c".repeat(32));
    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    expect(body.resourceSpans[0].scopeSpans[0].spans[0].traceId).toBe("c".repeat(32));
  });

  it("options.traceId takes priority over INPUT_TRACE_ID", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.INPUT_TRACE_ID = "d".repeat(32);

    const { traceId } = await sendJobSetupSpan({ traceId: "e".repeat(32) });

    expect(traceId).toBe("e".repeat(32));
    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    expect(body.resourceSpans[0].scopeSpans[0].spans[0].traceId).toBe("e".repeat(32));
  });

  it("uses the provided startMs for the span start time", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    const startMs = 1_700_000_000_000;
    await sendJobSetupSpan({ startMs });

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.startTimeUnixNano).toBe(toNanoString(startMs));
  });

  it("uses OTEL_SERVICE_NAME for the resource service.name attribute", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.OTEL_SERVICE_NAME = "my-service";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({ key: "service.name", value: { stringValue: "my-service" } });
  });

  it("includes github.repository and github.run_id as resource attributes", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_REPOSITORY = "owner/repo";
    process.env.GITHUB_RUN_ID = "987654321";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({ key: "github.repository", value: { stringValue: "owner/repo" } });
    expect(resourceAttrs).toContainEqual({ key: "github.run_id", value: { stringValue: "987654321" } });
  });

  it("includes github.event_name as resource attribute when GITHUB_EVENT_NAME is set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_EVENT_NAME = "workflow_dispatch";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({ key: "github.event_name", value: { stringValue: "workflow_dispatch" } });
  });

  it("omits github.event_name resource attribute when GITHUB_EVENT_NAME is not set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    const resourceKeys = resourceAttrs.map(a => a.key);
    expect(resourceKeys).not.toContain("github.event_name");
  });

  it("includes gh-aw.event_name as span attribute when GITHUB_EVENT_NAME is set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_EVENT_NAME = "workflow_dispatch";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.attributes).toContainEqual({ key: "gh-aw.event_name", value: { stringValue: "workflow_dispatch" } });
  });

  it("omits gh-aw.event_name span attribute when GITHUB_EVENT_NAME is not set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    const keys = span.attributes.map(a => a.key);
    expect(keys).not.toContain("gh-aw.event_name");
  });

  it("includes github.ref as resource attribute when GITHUB_REF is set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_REF = "refs/heads/main";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({ key: "github.ref", value: { stringValue: "refs/heads/main" } });
  });

  it("includes github.ref_name and github.head_ref as resource attributes when set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_REF_NAME = "main";
    process.env.GITHUB_HEAD_REF = "feature-branch";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({ key: "github.ref_name", value: { stringValue: "main" } });
    expect(resourceAttrs).toContainEqual({ key: "github.head_ref", value: { stringValue: "feature-branch" } });
  });

  it("omits github.ref resource attribute when GITHUB_REF is not set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    const resourceKeys = resourceAttrs.map(a => a.key);
    expect(resourceKeys).not.toContain("github.ref");
  });

  it("omits github.ref_name and github.head_ref resource attributes when not set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    const resourceKeys = resourceAttrs.map(a => a.key);
    expect(resourceKeys).not.toContain("github.ref_name");
    expect(resourceKeys).not.toContain("github.head_ref");
  });

  it("includes github.sha as resource attribute when GITHUB_SHA is set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_SHA = "abc1234567890def";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({ key: "github.sha", value: { stringValue: "abc1234567890def" } });
  });

  it("omits github.sha resource attribute when GITHUB_SHA is not set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    const resourceKeys = resourceAttrs.map(a => a.key);
    expect(resourceKeys).not.toContain("github.sha");
  });

  it("includes github.workflow_ref as resource attribute when GITHUB_WORKFLOW_REF is set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_WORKFLOW_REF = "owner/repo/.github/workflows/my-workflow.yml@refs/heads/main";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({ key: "github.workflow_ref", value: { stringValue: "owner/repo/.github/workflows/my-workflow.yml@refs/heads/main" } });
  });

  it("omits github.workflow_ref resource attribute when GITHUB_WORKFLOW_REF is not set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    const resourceKeys = resourceAttrs.map(a => a.key);
    expect(resourceKeys).not.toContain("github.workflow_ref");
  });

  it("includes github.actions.run_url as resource attribute when repository and run_id are set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_REPOSITORY = "owner/repo";
    process.env.GITHUB_RUN_ID = "987654321";
    delete process.env.GITHUB_SERVER_URL;

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({
      key: "github.actions.run_url",
      value: { stringValue: "https://github.com/owner/repo/actions/runs/987654321" },
    });
  });

  it("uses GITHUB_SERVER_URL for github.actions.run_url in sendJobSetupSpan", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_REPOSITORY = "owner/repo";
    process.env.GITHUB_RUN_ID = "987654321";
    process.env.GITHUB_SERVER_URL = "https://github.example.com";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({
      key: "github.actions.run_url",
      value: { stringValue: "https://github.example.com/owner/repo/actions/runs/987654321" },
    });
  });

  it("omits github.actions.run_url when repository or run_id is missing in sendJobSetupSpan", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    delete process.env.GITHUB_REPOSITORY;
    delete process.env.GITHUB_RUN_ID;

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    const resourceKeys = resourceAttrs.map(a => a.key);
    expect(resourceKeys).not.toContain("github.actions.run_url");
  });

  it("includes service.version resource attribute when GH_AW_INFO_VERSION is set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GH_AW_INFO_VERSION = "v1.2.3";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({ key: "service.version", value: { stringValue: "v1.2.3" } });
  });

  it("omits gh-aw.engine.id attribute when engine is not set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobSetupSpan();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    const keys = span.attributes.map(a => a.key);
    expect(keys).not.toContain("gh-aw.engine.id");
  });

  describe("cross-job parent span propagation via aw_context", () => {
    let readFileSpy;

    beforeEach(() => {
      readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(() => {
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });
    });

    afterEach(() => {
      readFileSpy.mockRestore();
    });

    it("uses otel_parent_span_id from aw_context as parentSpanId for cross-job trace hierarchy", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      const parentSpanId = "abcdef1234567890";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/aw_info.json") {
          return JSON.stringify({ context: { otel_trace_id: "a".repeat(32), otel_parent_span_id: parentSpanId } });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobSetupSpan();

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.parentSpanId).toBe(parentSpanId);
    });

    it("omits parentSpanId when aw_context.otel_parent_span_id is absent", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/aw_info.json") {
          return JSON.stringify({ context: { otel_trace_id: "a".repeat(32) } });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobSetupSpan();

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.parentSpanId).toBeUndefined();
    });

    it("ignores invalid otel_parent_span_id from aw_context and omits parentSpanId", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/aw_info.json") {
          return JSON.stringify({ context: { otel_trace_id: "a".repeat(32), otel_parent_span_id: "not-a-valid-span-id" } });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobSetupSpan();

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.parentSpanId).toBeUndefined();
    });
  });

  describe("staged / deployment.environment", () => {
    let readFileSpy;

    beforeEach(() => {
      readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(() => {
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });
    });

    afterEach(() => {
      readFileSpy.mockRestore();
    });

    it("sets deployment.environment=production when aw_info.json is absent", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      await sendJobSetupSpan();

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const resourceAttrs = body.resourceSpans[0].resource.attributes;
      expect(resourceAttrs).toContainEqual({ key: "deployment.environment", value: { stringValue: "production" } });
    });

    it("sets deployment.environment=staging when aw_info.json is absent and GH_AW_INFO_STAGED=true", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_INFO_STAGED = "true";

      await sendJobSetupSpan();

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.attributes).toContainEqual({ key: "gh-aw.staged", value: { boolValue: true } });

      const resourceAttrs = body.resourceSpans[0].resource.attributes;
      expect(resourceAttrs).toContainEqual({ key: "deployment.environment", value: { stringValue: "staging" } });
    });

    it("sets deployment.environment=staging when awInfo.staged=true", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/aw_info.json") {
          return JSON.stringify({ staged: true });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobSetupSpan();

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const resourceAttrs = body.resourceSpans[0].resource.attributes;
      expect(resourceAttrs).toContainEqual({ key: "deployment.environment", value: { stringValue: "staging" } });
    });

    it("sets deployment.environment=production when awInfo.staged=false", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/aw_info.json") {
          return JSON.stringify({ staged: false });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobSetupSpan();

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.attributes).toContainEqual({ key: "gh-aw.staged", value: { boolValue: false } });

      const resourceAttrs = body.resourceSpans[0].resource.attributes;
      expect(resourceAttrs).toContainEqual({ key: "deployment.environment", value: { stringValue: "production" } });
    });
  });

  describe("trigger item context from aw_info.json", () => {
    let readFileSpy;

    beforeEach(() => {
      readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(() => {
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });
    });

    afterEach(() => {
      readFileSpy.mockRestore();
    });

    it("emits gh-aw.trigger.item_type and gh-aw.trigger.item_number from aw_info.context", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/aw_info.json") {
          return JSON.stringify({ context: { item_type: "issue", item_number: "42", trigger_label: "" } });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobSetupSpan();

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.attributes).toContainEqual({ key: "gh-aw.trigger.item_type", value: { stringValue: "issue" } });
      expect(span.attributes).toContainEqual({ key: "gh-aw.trigger.item_number", value: { stringValue: "42" } });
      const keys = span.attributes.map(a => a.key);
      expect(keys).not.toContain("gh-aw.trigger.label");
    });

    it("emits gh-aw.trigger.label when trigger_label is non-empty", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/aw_info.json") {
          return JSON.stringify({ context: { item_type: "pull_request", item_number: "99", trigger_label: "copilot" } });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobSetupSpan();

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.attributes).toContainEqual({ key: "gh-aw.trigger.item_type", value: { stringValue: "pull_request" } });
      expect(span.attributes).toContainEqual({ key: "gh-aw.trigger.item_number", value: { stringValue: "99" } });
      expect(span.attributes).toContainEqual({ key: "gh-aw.trigger.label", value: { stringValue: "copilot" } });
    });

    it("omits trigger attributes when aw_info.json is absent", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      await sendJobSetupSpan();

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const keys = span.attributes.map(a => a.key);
      expect(keys).not.toContain("gh-aw.trigger.item_type");
      expect(keys).not.toContain("gh-aw.trigger.item_number");
      expect(keys).not.toContain("gh-aw.trigger.label");
    });
  });
});

// ---------------------------------------------------------------------------
// sendJobConclusionSpan
// ---------------------------------------------------------------------------

describe("sendJobConclusionSpan", () => {
  /** @type {Record<string, string | undefined>} */
  const savedEnv = {};
  const envKeys = [
    "OTEL_EXPORTER_OTLP_ENDPOINT",
    "OTEL_SERVICE_NAME",
    "GH_AW_EFFECTIVE_TOKENS",
    "GH_AW_INFO_VERSION",
    "GITHUB_AW_OTEL_TRACE_ID",
    "GITHUB_AW_OTEL_PARENT_SPAN_ID",
    "GITHUB_RUN_ID",
    "GITHUB_RUN_ATTEMPT",
    "GITHUB_ACTOR",
    "GITHUB_REPOSITORY",
    "GITHUB_EVENT_NAME",
    "GITHUB_REF",
    "GITHUB_REF_NAME",
    "GITHUB_HEAD_REF",
    "GITHUB_SHA",
    "GITHUB_WORKFLOW_REF",
    "INPUT_JOB_NAME",
    "GH_AW_AGENT_CONCLUSION",
    "GH_AW_INFO_WORKFLOW_NAME",
    "GITHUB_WORKFLOW",
  ];
  let mkdirSpy, appendSpy;

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
    for (const k of envKeys) {
      savedEnv[k] = process.env[k];
      delete process.env[k];
    }
    mkdirSpy = vi.spyOn(fs, "mkdirSync").mockImplementation(() => {});
    appendSpy = vi.spyOn(fs, "appendFileSync").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    for (const k of envKeys) {
      if (savedEnv[k] !== undefined) {
        process.env[k] = savedEnv[k];
      } else {
        delete process.env[k];
      }
    }
    mkdirSpy.mockRestore();
    appendSpy.mockRestore();
  });

  it("skips OTLP export but writes JSONL mirror when OTEL_EXPORTER_OTLP_ENDPOINT is not set", async () => {
    await sendJobConclusionSpan("gh-aw.job.conclusion");
    expect(fetch).not.toHaveBeenCalled();
    expect(appendSpy).toHaveBeenCalledOnce();
    const [filePath, content] = appendSpy.mock.calls[0];
    expect(filePath).toBe(OTEL_JSONL_PATH);
    const payload = JSON.parse(content.trim());
    expect(payload).toHaveProperty("resourceSpans");
  });

  it("sends a span with the given span name", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_RUN_ID = "111";
    process.env.GITHUB_ACTOR = "octocat";
    process.env.GITHUB_REPOSITORY = "owner/repo";

    await sendJobConclusionSpan("gh-aw.job.safe-outputs");

    expect(mockFetch).toHaveBeenCalledOnce();
    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.name).toBe("gh-aw.job.safe-outputs");
    expect(span.traceId).toMatch(/^[0-9a-f]{32}$/);
    expect(span.spanId).toMatch(/^[0-9a-f]{16}$/);
  });

  it("emits a dedicated gh-aw.<job>.agent span when startMs and agent_output mtime are available", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.INPUT_JOB_NAME = "agent";
    process.env.GITHUB_AW_OTEL_TRACE_ID = "f".repeat(32);
    process.env.GITHUB_AW_OTEL_PARENT_SPAN_ID = "abcdef1234567890";

    const startMs = 1_700_000_000_000;
    const endMs = 1_700_000_005_000;
    const statSpy = vi.spyOn(fs, "statSync").mockReturnValue(/** @type {Partial<fs.Stats>} */ { mtimeMs: endMs });
    const readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(filePath => {
      if (filePath === "/tmp/gh-aw/agent_output.json") {
        return JSON.stringify({ items: [{ type: "issue" }, { type: "pull_request" }] });
      }
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });

    await sendJobConclusionSpan("gh-aw.agent.conclusion", { startMs });

    statSpy.mockRestore();
    readFileSpy.mockRestore();
    expect(mockFetch).toHaveBeenCalledTimes(2);

    const agentBody = JSON.parse(mockFetch.mock.calls[0][1].body);
    const agentSpan = agentBody.resourceSpans[0].scopeSpans[0].spans[0];
    expect(agentSpan.name).toBe("gh-aw.agent.agent");
    expect(agentSpan.startTimeUnixNano).toBe(toNanoString(startMs));
    expect(agentSpan.endTimeUnixNano).toBe(toNanoString(endMs));

    const conclusionBody = JSON.parse(mockFetch.mock.calls[1][1].body);
    const conclusionSpan = conclusionBody.resourceSpans[0].scopeSpans[0].spans[0];
    expect(conclusionSpan.name).toBe("gh-aw.agent.conclusion");
    expect(agentSpan.traceId).toBe(conclusionSpan.traceId);
    expect(agentSpan.parentSpanId).toBe(conclusionSpan.spanId);
    expect(agentSpan.parentSpanId).not.toBe("abcdef1234567890");
    expect(conclusionSpan.parentSpanId).toBe("abcdef1234567890");
    expect(agentSpan.attributes).toContainEqual({ key: "gh-aw.output.item_count", value: { intValue: 2 } });
    expect(conclusionSpan.attributes).toContainEqual({ key: "gh-aw.output.item_count", value: { intValue: 2 } });
  });

  it("does not emit a dedicated agent span when agent_output mtime is unavailable", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    const statSpy = vi.spyOn(fs, "statSync").mockImplementation(() => {
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });

    await sendJobConclusionSpan("gh-aw.job.conclusion", { startMs: 1_700_000_000_000 });

    statSpy.mockRestore();
    expect(mockFetch).toHaveBeenCalledOnce();
    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.name).toBe("gh-aw.job.conclusion");
  });

  it("emits a dedicated agent span on timed_out when agent_output mtime is unavailable", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.INPUT_JOB_NAME = "agent";
    process.env.GH_AW_AGENT_CONCLUSION = "timed_out";

    const startMs = 1_700_000_000_000;
    const statSpy = vi.spyOn(fs, "statSync").mockImplementation(() => {
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });

    await sendJobConclusionSpan("gh-aw.agent.conclusion", { startMs });

    statSpy.mockRestore();
    expect(mockFetch).toHaveBeenCalledTimes(2);

    const agentBody = JSON.parse(mockFetch.mock.calls[0][1].body);
    const agentSpan = agentBody.resourceSpans[0].scopeSpans[0].spans[0];
    expect(agentSpan.name).toBe("gh-aw.agent.agent");
    expect(agentSpan.startTimeUnixNano).toBe(toNanoString(startMs));
    expect(BigInt(agentSpan.endTimeUnixNano)).toBeGreaterThan(BigInt(toNanoString(startMs)));

    const conclusionBody = JSON.parse(mockFetch.mock.calls[1][1].body);
    const conclusionSpan = conclusionBody.resourceSpans[0].scopeSpans[0].spans[0];
    expect(conclusionSpan.name).toBe("gh-aw.agent.conclusion");
    expect(agentSpan.parentSpanId).toBe(conclusionSpan.spanId);
    expect(conclusionSpan.status.code).toBe(2);
    expect(conclusionSpan.status.message).toContain("agent timed_out");
  });

  it("emits a dedicated agent span on cancelled when agent_output mtime is unavailable", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.INPUT_JOB_NAME = "agent";
    process.env.GH_AW_AGENT_CONCLUSION = "cancelled";

    const startMs = 1_700_000_000_000;
    const statSpy = vi.spyOn(fs, "statSync").mockImplementation(() => {
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });

    await sendJobConclusionSpan("gh-aw.agent.conclusion", { startMs });

    statSpy.mockRestore();
    expect(mockFetch).toHaveBeenCalledTimes(2);

    const agentBody = JSON.parse(mockFetch.mock.calls[0][1].body);
    const agentSpan = agentBody.resourceSpans[0].scopeSpans[0].spans[0];
    expect(agentSpan.name).toBe("gh-aw.agent.agent");
    expect(agentSpan.startTimeUnixNano).toBe(toNanoString(startMs));
    expect(BigInt(agentSpan.endTimeUnixNano)).toBeGreaterThan(BigInt(toNanoString(startMs)));

    const conclusionBody = JSON.parse(mockFetch.mock.calls[1][1].body);
    const conclusionSpan = conclusionBody.resourceSpans[0].scopeSpans[0].spans[0];
    expect(conclusionSpan.name).toBe("gh-aw.agent.conclusion");
    expect(agentSpan.parentSpanId).toBe(conclusionSpan.spanId);
    expect(conclusionSpan.status.code).toBe(2);
    expect(conclusionSpan.status.message).toContain("agent cancelled");
  });

  it("does not emit a dedicated agent span for non-agent jobs", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.INPUT_JOB_NAME = "safe-outputs";

    const statSpy = vi.spyOn(fs, "statSync").mockReturnValue(/** @type {Partial<fs.Stats>} */ { mtimeMs: 1_700_000_005_000 });

    await sendJobConclusionSpan("gh-aw.safe-outputs.conclusion", { startMs: 1_700_000_000_000 });

    statSpy.mockRestore();
    expect(mockFetch).toHaveBeenCalledOnce();
    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    expect(body.resourceSpans[0].scopeSpans[0].spans).toHaveLength(1);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.name).toBe("gh-aw.safe-outputs.conclusion");
  });

  it("emits the agent span with SPAN_KIND_CLIENT (3)", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.INPUT_JOB_NAME = "agent";

    const startMs = 1_700_000_000_000;
    const endMs = 1_700_000_005_000;
    const statSpy = vi.spyOn(fs, "statSync").mockReturnValue(/** @type {Partial<fs.Stats>} */ { mtimeMs: endMs });
    const readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(() => {
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });

    await sendJobConclusionSpan("gh-aw.agent.conclusion", { startMs });

    statSpy.mockRestore();
    readFileSpy.mockRestore();

    const agentBody = JSON.parse(mockFetch.mock.calls[0][1].body);
    const agentSpan = agentBody.resourceSpans[0].scopeSpans[0].spans[0];
    expect(agentSpan.name).toBe("gh-aw.agent.agent");
    expect(agentSpan.kind).toBe(3); // SPAN_KIND_CLIENT
  });

  it("includes gen_ai.request.model, gen_ai.system, gh-aw.engine, gen_ai.operation.name and gen_ai.workflow.name on the agent span from aw_info.json", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.INPUT_JOB_NAME = "agent";

    const startMs = 1_700_000_000_000;
    const endMs = 1_700_000_005_000;
    const statSpy = vi.spyOn(fs, "statSync").mockReturnValue(/** @type {Partial<fs.Stats>} */ { mtimeMs: endMs });
    const readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(filePath => {
      if (filePath === "/tmp/gh-aw/aw_info.json") {
        return JSON.stringify({ model: "claude-3-5-sonnet-20241022", engine_id: "claude", workflow_name: "otel-advisor" });
      }
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });

    await sendJobConclusionSpan("gh-aw.agent.conclusion", { startMs });

    statSpy.mockRestore();
    readFileSpy.mockRestore();

    const agentBody = JSON.parse(mockFetch.mock.calls[0][1].body);
    const agentSpan = agentBody.resourceSpans[0].scopeSpans[0].spans[0];
    expect(agentSpan.name).toBe("gh-aw.agent.agent");
    const attrs = Object.fromEntries(agentSpan.attributes.map(a => [a.key, a.value.stringValue ?? a.value.intValue]));
    expect(attrs["gen_ai.operation.name"]).toBe("chat");
    expect(attrs["gen_ai.request.model"]).toBe("claude-3-5-sonnet-20241022");
    expect(attrs["gen_ai.system"]).toBe("anthropic");
    expect(attrs["gh-aw.engine"]).toBe("claude");
    expect(attrs["gen_ai.workflow.name"]).toBe("otel-advisor");
  });

  it("omits gen_ai.request.model, gen_ai.system, gh-aw.engine and gen_ai.workflow.name from the agent span when model, engine_id and workflow_name are absent", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.INPUT_JOB_NAME = "agent";

    const startMs = 1_700_000_000_000;
    const endMs = 1_700_000_005_000;
    const statSpy = vi.spyOn(fs, "statSync").mockReturnValue(/** @type {Partial<fs.Stats>} */ { mtimeMs: endMs });
    const readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(() => {
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });

    await sendJobConclusionSpan("gh-aw.agent.conclusion", { startMs });

    statSpy.mockRestore();
    readFileSpy.mockRestore();

    const agentBody = JSON.parse(mockFetch.mock.calls[0][1].body);
    const agentSpan = agentBody.resourceSpans[0].scopeSpans[0].spans[0];
    const attrs = Object.fromEntries(agentSpan.attributes.map(a => [a.key, a.value.stringValue ?? a.value.intValue]));
    // gen_ai.operation.name is always present
    expect(attrs["gen_ai.operation.name"]).toBe("chat");
    const keys = agentSpan.attributes.map(a => a.key);
    expect(keys).not.toContain("gen_ai.request.model");
    expect(keys).not.toContain("gen_ai.system");
    expect(keys).not.toContain("gh-aw.engine");
    expect(keys).not.toContain("gen_ai.workflow.name");
  });

  it("uses the raw engine ID as gen_ai.system fallback for unknown engines on the agent span", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.INPUT_JOB_NAME = "agent";

    const startMs = 1_700_000_000_000;
    const endMs = 1_700_000_005_000;
    const statSpy = vi.spyOn(fs, "statSync").mockReturnValue(/** @type {Partial<fs.Stats>} */ { mtimeMs: endMs });
    const readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(filePath => {
      if (filePath === "/tmp/gh-aw/aw_info.json") {
        return JSON.stringify({ engine_id: "custom-engine" });
      }
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });

    await sendJobConclusionSpan("gh-aw.agent.conclusion", { startMs });

    statSpy.mockRestore();
    readFileSpy.mockRestore();

    const agentBody = JSON.parse(mockFetch.mock.calls[0][1].body);
    const agentSpan = agentBody.resourceSpans[0].scopeSpans[0].spans[0];
    const attrs = Object.fromEntries(agentSpan.attributes.map(a => [a.key, a.value.stringValue ?? a.value.intValue]));
    // Unknown engine ID falls back to the raw value for gen_ai.system
    expect(attrs["gen_ai.system"]).toBe("custom-engine");
    expect(attrs["gh-aw.engine"]).toBe("custom-engine");
  });

  it("includes gen_ai.request.model on the conclusion span when model is set in aw_info.json", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    const readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(filePath => {
      if (filePath === "/tmp/gh-aw/aw_info.json") {
        return JSON.stringify({ model: "claude-3-5-sonnet-20241022", engine_id: "claude" });
      }
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });

    await sendJobConclusionSpan("gh-aw.job.conclusion");
    readFileSpy.mockRestore();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.name).toBe("gh-aw.job.conclusion");
    const attrs = Object.fromEntries(span.attributes.map(a => [a.key, a.value.stringValue ?? a.value.intValue]));
    expect(attrs["gen_ai.request.model"]).toBe("claude-3-5-sonnet-20241022");
  });

  it("omits gen_ai.request.model from the conclusion span when model is absent in aw_info.json", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.name).toBe("gh-aw.job.conclusion");
    const keys = span.attributes.map(a => a.key);
    expect(keys).not.toContain("gen_ai.request.model");
  });

  it("includes gh-aw.run.attempt attribute from GITHUB_RUN_ATTEMPT env var", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_RUN_ATTEMPT = "3";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    const attrs = Object.fromEntries(span.attributes.map(a => [a.key, a.value.stringValue]));
    expect(attrs["gh-aw.run.attempt"]).toBe("3");
  });

  it("defaults gh-aw.run.attempt to '1' when neither awInfo nor GITHUB_RUN_ATTEMPT is set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    const attrs = Object.fromEntries(span.attributes.map(a => [a.key, a.value.stringValue]));
    expect(attrs["gh-aw.run.attempt"]).toBe("1");
  });

  it("reads gh-aw.workflow.name from aw_info.json when present", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GH_AW_INFO_WORKFLOW_NAME = "env-workflow";
    process.env.GITHUB_WORKFLOW = "github-workflow";

    const readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(filePath => {
      if (filePath === "/tmp/gh-aw/aw_info.json") {
        return JSON.stringify({ workflow_name: "aw-info-workflow" });
      }
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });

    await sendJobConclusionSpan("gh-aw.job.conclusion");
    readFileSpy.mockRestore();

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    const attrs = Object.fromEntries(span.attributes.map(a => [a.key, a.value.stringValue]));
    expect(attrs["gh-aw.workflow.name"]).toBe("aw-info-workflow");
  });

  it("falls back to GH_AW_INFO_WORKFLOW_NAME when aw_info.json is absent", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GH_AW_INFO_WORKFLOW_NAME = "env-workflow";
    process.env.GITHUB_WORKFLOW = "github-workflow";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    const attrs = Object.fromEntries(span.attributes.map(a => [a.key, a.value.stringValue]));
    expect(attrs["gh-aw.workflow.name"]).toBe("env-workflow");
  });

  it("falls back to GITHUB_WORKFLOW when aw_info.json and GH_AW_INFO_WORKFLOW_NAME are absent", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_WORKFLOW = "github-workflow";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    const attrs = Object.fromEntries(span.attributes.map(a => [a.key, a.value.stringValue]));
    expect(attrs["gh-aw.workflow.name"]).toBe("github-workflow");
  });

  it("sets gh-aw.workflow.name to empty string when all sources are absent", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    const attrs = Object.fromEntries(span.attributes.map(a => [a.key, a.value.stringValue]));
    expect(attrs["gh-aw.workflow.name"]).toBe("");
  });

  it("includes effective_tokens attribute when GH_AW_EFFECTIVE_TOKENS is set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GH_AW_EFFECTIVE_TOKENS = "5000";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    const etAttr = span.attributes.find(a => a.key === "gh-aw.effective_tokens");
    expect(etAttr).toBeDefined();
    expect(etAttr.value.intValue).toBe(5000);
  });

  it("omits effective_tokens attribute when GH_AW_EFFECTIVE_TOKENS is absent", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    const keys = span.attributes.map(a => a.key);
    expect(keys).not.toContain("gh-aw.effective_tokens");
  });

  it("uses GH_AW_INFO_VERSION as scope version when aw_info.json is absent", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GH_AW_INFO_VERSION = "v2.0.0";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    expect(body.resourceSpans[0].scopeSpans[0].scope.version).toBe("v2.0.0");
  });

  it("uses GITHUB_AW_OTEL_TRACE_ID from env as trace ID (1 trace per run)", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_AW_OTEL_TRACE_ID = "f".repeat(32);

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.traceId).toBe("f".repeat(32));
  });

  it("uses GITHUB_AW_OTEL_PARENT_SPAN_ID as parentSpanId (1 parent span per job)", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    const parentSpanId = "abcdef1234567890";
    process.env.GITHUB_AW_OTEL_PARENT_SPAN_ID = parentSpanId;

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.parentSpanId).toBe(parentSpanId);
  });

  it("omits parentSpanId when GITHUB_AW_OTEL_PARENT_SPAN_ID is absent", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.parentSpanId).toBeUndefined();
  });

  it("normalizes uppercase GITHUB_AW_OTEL_TRACE_ID to lowercase", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_AW_OTEL_TRACE_ID = "F".repeat(32); // uppercase — should be normalised

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.traceId).toBe("f".repeat(32));
  });

  it("includes github.repository and github.run_id as resource attributes", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_REPOSITORY = "owner/repo";
    process.env.GITHUB_RUN_ID = "987654321";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({ key: "github.repository", value: { stringValue: "owner/repo" } });
    expect(resourceAttrs).toContainEqual({ key: "github.run_id", value: { stringValue: "987654321" } });
  });

  it("includes github.event_name as resource attribute when GITHUB_EVENT_NAME is set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_EVENT_NAME = "pull_request";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({ key: "github.event_name", value: { stringValue: "pull_request" } });
  });

  it("omits github.event_name resource attribute when GITHUB_EVENT_NAME is not set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    const resourceKeys = resourceAttrs.map(a => a.key);
    expect(resourceKeys).not.toContain("github.event_name");
  });

  it("includes gh-aw.event_name as span attribute when GITHUB_EVENT_NAME is set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_EVENT_NAME = "pull_request";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.attributes).toContainEqual({ key: "gh-aw.event_name", value: { stringValue: "pull_request" } });
  });

  it("omits gh-aw.event_name span attribute when GITHUB_EVENT_NAME is not set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    const keys = span.attributes.map(a => a.key);
    expect(keys).not.toContain("gh-aw.event_name");
  });

  it("includes github.ref as resource attribute when GITHUB_REF is set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_REF = "refs/heads/main";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({ key: "github.ref", value: { stringValue: "refs/heads/main" } });
  });

  it("includes github.ref_name and github.head_ref as resource attributes when set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_REF_NAME = "123/merge";
    process.env.GITHUB_HEAD_REF = "feature-branch";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({ key: "github.ref_name", value: { stringValue: "123/merge" } });
    expect(resourceAttrs).toContainEqual({ key: "github.head_ref", value: { stringValue: "feature-branch" } });
  });

  it("omits github.ref resource attribute when GITHUB_REF is not set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    const resourceKeys = resourceAttrs.map(a => a.key);
    expect(resourceKeys).not.toContain("github.ref");
  });

  it("omits github.ref_name and github.head_ref resource attributes when not set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    const resourceKeys = resourceAttrs.map(a => a.key);
    expect(resourceKeys).not.toContain("github.ref_name");
    expect(resourceKeys).not.toContain("github.head_ref");
  });

  it("includes github.sha as resource attribute when GITHUB_SHA is set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_SHA = "abc1234567890def";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({ key: "github.sha", value: { stringValue: "abc1234567890def" } });
  });

  it("omits github.sha resource attribute when GITHUB_SHA is not set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    const resourceKeys = resourceAttrs.map(a => a.key);
    expect(resourceKeys).not.toContain("github.sha");
  });

  it("includes github.actions.run_url as resource attribute when repository and run_id are set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_REPOSITORY = "owner/repo";
    process.env.GITHUB_RUN_ID = "987654321";
    delete process.env.GITHUB_SERVER_URL;

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({
      key: "github.actions.run_url",
      value: { stringValue: "https://github.com/owner/repo/actions/runs/987654321" },
    });
  });

  it("uses GITHUB_SERVER_URL for github.actions.run_url in sendJobConclusionSpan", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_REPOSITORY = "owner/repo";
    process.env.GITHUB_RUN_ID = "987654321";
    process.env.GITHUB_SERVER_URL = "https://github.example.com";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({
      key: "github.actions.run_url",
      value: { stringValue: "https://github.example.com/owner/repo/actions/runs/987654321" },
    });
  });

  it("omits github.actions.run_url when repository or run_id is missing in sendJobConclusionSpan", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    delete process.env.GITHUB_REPOSITORY;
    delete process.env.GITHUB_RUN_ID;

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    const resourceKeys = resourceAttrs.map(a => a.key);
    expect(resourceKeys).not.toContain("github.actions.run_url");
  });

  it("includes service.version resource attribute when version is known", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GH_AW_INFO_VERSION = "v3.0.0";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({ key: "service.version", value: { stringValue: "v3.0.0" } });
  });

  describe("agent_output.json error enrichment", () => {
    let readFileSpy;

    beforeEach(() => {
      readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(filePath => {
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });
    });

    afterEach(() => {
      readFileSpy.mockRestore();
    });

    it("adds gh-aw.error.count and gh-aw.error.messages attributes when agent_output.json has errors on failure", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ errors: [{ message: "Rate limit exceeded" }, { message: "Tool call failed" }] });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const attrs = span.attributes;
      const errorCount = attrs.find(a => a.key === "gh-aw.error.count");
      const errorMessages = attrs.find(a => a.key === "gh-aw.error.messages");
      expect(errorCount).toBeDefined();
      expect(errorCount.value.intValue).toBe(2);
      expect(errorMessages).toBeDefined();
      expect(errorMessages.value.stringValue).toBe("Rate limit exceeded | Tool call failed");
    });

    it("adds gh-aw.error attributes when agent_output.json has errors on success", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "success";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ errors: [{ message: "partial failure one" }, { message: "partial failure two" }] });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const attrs = span.attributes;
      expect(span.status.code).toBe(1);
      expect(attrs).toContainEqual({ key: "gh-aw.error.count", value: { intValue: 2 } });
      expect(attrs).toContainEqual({ key: "gh-aw.error.messages", value: { stringValue: "partial failure one | partial failure two" } });
      expect(span.events).toHaveLength(2);
      expect(span.events[0].name).toBe("exception");
      expect(span.events[0].attributes).toContainEqual({ key: "exception.type", value: { stringValue: "gh-aw.AgentError" } });
      expect(span.events[0].attributes).toContainEqual({ key: "exception.message", value: { stringValue: "partial failure one" } });
    });

    it("enriches statusMessage with the first error message on failure", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ errors: [{ message: "Rate limit exceeded on claude-3-5-sonnet" }] });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.status.message).toBe("agent failure: Rate limit exceeded on claude-3-5-sonnet");
    });

    it("enriches statusMessage with the first error message on timed_out", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "timed_out";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ errors: [{ message: "Execution exceeded 30 minute limit" }] });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.status.message).toBe("agent timed_out: Execution exceeded 30 minute limit");
    });

    it("marks cancelled conclusion spans as errors", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "cancelled";

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.status.code).toBe(2);
      expect(span.status.message).toBe("agent cancelled");
    });

    it("caps error messages at 5 entries", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      const manyErrors = [1, 2, 3, 4, 5, 6, 7].map(i => ({ message: `Error ${i}` }));
      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ errors: manyErrors });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const errorMessages = span.attributes.find(a => a.key === "gh-aw.error.messages");
      expect(errorMessages).toBeDefined();
      expect(errorMessages.value.stringValue).toBe("Error 1 | Error 2 | Error 3 | Error 4 | Error 5");
      // gh-aw.error.count reflects the full error count (7), not the capped count
      const errorCount = span.attributes.find(a => a.key === "gh-aw.error.count");
      expect(errorCount.value.intValue).toBe(7);
    });

    it("truncates statusMessage to 256 characters", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      const longMessage = "x".repeat(300);
      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ errors: [{ message: longMessage }] });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.status.message.length).toBe(256);
      expect(span.status.message.startsWith("agent failure: ")).toBe(true);
    });

    it("does not add error attributes when agent_output.json has no errors array", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ items: [] });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const keys = span.attributes.map(a => a.key);
      expect(keys).not.toContain("gh-aw.error.count");
      expect(keys).not.toContain("gh-aw.error.messages");
      expect(span.status.message).toBe("agent failure");
    });

    it("reads agent_output.json and adds output metrics when agent conclusion is success", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "success";
      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({
            items: [{ type: "pull_request" }, { type: "issue" }, { type: "pull_request" }, {}],
          });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const agentOutputCalls = readFileSpy.mock.calls.filter(([p]) => p === "/tmp/gh-aw/agent_output.json");
      expect(agentOutputCalls).toHaveLength(1);
      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const attrs = span.attributes;
      expect(attrs).toContainEqual({ key: "gh-aw.output.item_count", value: { intValue: 4 } });
      expect(attrs).toContainEqual({ key: "gh-aw.output.item_types", value: { stringValue: "issue,pull_request" } });
    });

    it("does not add error attributes when agent_output.json is absent on failure", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      // readFileSpy already throws ENOENT for all paths (set in beforeEach)

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const keys = span.attributes.map(a => a.key);
      expect(keys).not.toContain("gh-aw.error.count");
      expect(keys).not.toContain("gh-aw.error.messages");
      expect(span.status.message).toBe("agent failure");
    });

    it("emits one exception span event per error on agent failure", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ errors: [{ message: "Rate limit exceeded" }, { message: "Tool call failed" }] });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.events).toHaveLength(2);
      expect(span.events[0].name).toBe("exception");
      expect(span.events[0].attributes).toContainEqual({ key: "exception.type", value: { stringValue: "gh-aw.AgentError" } });
      expect(span.events[0].attributes).toContainEqual({ key: "exception.message", value: { stringValue: "Rate limit exceeded" } });
      expect(span.events[1].name).toBe("exception");
      expect(span.events[1].attributes).toContainEqual({ key: "exception.type", value: { stringValue: "gh-aw.AgentError" } });
      expect(span.events[1].attributes).toContainEqual({ key: "exception.message", value: { stringValue: "Tool call failed" } });
    });

    it("truncates exception.message to 1024 characters", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      const longMessage = "x".repeat(2000);
      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ errors: [{ message: longMessage }] });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.events).toHaveLength(1);
      const msg = span.events[0].attributes.find(a => a.key === "exception.message");
      expect(msg.value.stringValue.length).toBe(1024);
    });

    it("does not emit exception events when agent conclusion is success", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "success";

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.events).toBeUndefined();
    });

    it("does not emit exception events when agent_output.json is absent on failure", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      // readFileSpy already throws ENOENT for all paths (set in beforeEach)

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.events).toBeUndefined();
    });

    it("emits exception events for all errors (not capped at 5 like error messages attribute)", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      const manyErrors = [1, 2, 3, 4, 5, 6, 7].map(i => ({ message: `Error ${i}` }));
      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ errors: manyErrors });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.events).toHaveLength(7);
      for (let i = 0; i < 7; i++) {
        expect(span.events[i].name).toBe("exception");
        expect(span.events[i].attributes).toContainEqual({ key: "exception.type", value: { stringValue: "gh-aw.AgentError" } });
        expect(span.events[i].attributes).toContainEqual({ key: "exception.message", value: { stringValue: `Error ${i + 1}` } });
      }
    });

    it("sets valid timeUnixNano on each exception event", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ errors: [{ message: "test error" }] });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.events).toHaveLength(1);
      expect(span.events[0].timeUnixNano).toMatch(/^\d+$/);
    });

    it("extracts exception.type from colon-prefixed error messages", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ errors: [{ message: "push_to_pull_request_branch:Cannot push to remote" }] });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.events).toHaveLength(1);
      const typeAttr = span.events[0].attributes.find(a => a.key === "exception.type");
      expect(typeAttr.value.stringValue).toBe("gh-aw.push_to_pull_request_branch");
      const msgAttr = span.events[0].attributes.find(a => a.key === "exception.message");
      expect(msgAttr.value.stringValue).toBe("Cannot push to remote");
    });

    it("normalizes uppercase exception.type prefix to lowercase", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ errors: [{ message: "Push_To_PR:Cannot push to remote" }] });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const typeAttr = span.events[0].attributes.find(a => a.key === "exception.type");
      expect(typeAttr.value.stringValue).toBe("gh-aw.push_to_pr");
      const msgAttr = span.events[0].attributes.find(a => a.key === "exception.message");
      expect(msgAttr.value.stringValue).toBe("Cannot push to remote");
    });

    it("falls back to gh-aw.AgentError when message has no colon prefix", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ errors: [{ message: "Something went wrong" }] });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.events).toHaveLength(1);
      const typeAttr = span.events[0].attributes.find(a => a.key === "exception.type");
      expect(typeAttr.value.stringValue).toBe("gh-aw.AgentError");
      const msgAttr = span.events[0].attributes.find(a => a.key === "exception.message");
      expect(msgAttr.value.stringValue).toBe("Something went wrong");
    });

    it("falls back to gh-aw.AgentError when colon prefix contains invalid characters", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ errors: [{ message: "Error with spaces:details here" }] });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const typeAttr = span.events[0].attributes.find(a => a.key === "exception.type");
      expect(typeAttr.value.stringValue).toBe("gh-aw.AgentError");
      // Full original message kept when type extraction fails
      const msgAttr = span.events[0].attributes.find(a => a.key === "exception.message");
      expect(msgAttr.value.stringValue).toBe("Error with spaces:details here");
    });

    it("falls back to gh-aw.AgentError when colon prefix exceeds 64 characters", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      const longPrefix = "a".repeat(65);
      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_output.json") {
          return JSON.stringify({ errors: [{ message: `${longPrefix}:some error` }] });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const typeAttr = span.events[0].attributes.find(a => a.key === "exception.type");
      expect(typeAttr.value.stringValue).toBe("gh-aw.AgentError");
    });
  });

  describe("rate-limit enrichment in conclusion span", () => {
    let readFileSpy;

    beforeEach(() => {
      readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(() => {
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });
    });

    afterEach(() => {
      readFileSpy.mockRestore();
    });

    it("includes rate-limit attributes when github_rate_limits.jsonl has entries", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      const entry = { timestamp: "2026-04-05T09:00:00.000Z", source: "response_headers", operation: "issues.get", resource: "core", limit: 5000, remaining: 4823, used: 177, reset: "2026-04-05T09:30:00.000Z" };
      readFileSpy.mockImplementation(filePath => {
        if (filePath === GITHUB_RATE_LIMITS_JSONL_PATH) {
          return JSON.stringify(entry) + "\n";
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const attrs = Object.fromEntries(span.attributes.map(a => [a.key, a.value.intValue ?? a.value.stringValue]));
      expect(attrs["gh-aw.github.rate_limit.remaining"]).toBe(4823);
      expect(attrs["gh-aw.github.rate_limit.limit"]).toBe(5000);
      expect(attrs["gh-aw.github.rate_limit.used"]).toBe(177);
      expect(attrs["gh-aw.github.rate_limit.resource"]).toBe("core");
      expect(attrs["gh-aw.github.rate_limit.reset"]).toBe("2026-04-05T09:30:00.000Z");
    });

    it("uses the last entry when the file contains multiple lines", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      const first = { resource: "core", limit: 5000, remaining: 4900, used: 100 };
      const last = { resource: "core", limit: 5000, remaining: 4500, used: 500 };
      readFileSpy.mockImplementation(filePath => {
        if (filePath === GITHUB_RATE_LIMITS_JSONL_PATH) {
          return JSON.stringify(first) + "\n" + JSON.stringify(last) + "\n";
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const attrs = Object.fromEntries(span.attributes.map(a => [a.key, a.value.intValue ?? a.value.stringValue]));
      expect(attrs["gh-aw.github.rate_limit.remaining"]).toBe(4500);
      expect(attrs["gh-aw.github.rate_limit.used"]).toBe(500);
    });

    it("omits rate-limit attributes when github_rate_limits.jsonl is absent", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      // readFileSpy already throws ENOENT for all paths

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const keys = span.attributes.map(a => a.key);
      expect(keys).not.toContain("gh-aw.github.rate_limit.remaining");
      expect(keys).not.toContain("gh-aw.github.rate_limit.limit");
      expect(keys).not.toContain("gh-aw.github.rate_limit.used");
      expect(keys).not.toContain("gh-aw.github.rate_limit.resource");
      expect(keys).not.toContain("gh-aw.github.rate_limit.reset");
    });

    it("omits reset attribute when entry has no reset field", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      const entry = { resource: "core", limit: 5000, remaining: 4823, used: 177 };
      readFileSpy.mockImplementation(filePath => {
        if (filePath === GITHUB_RATE_LIMITS_JSONL_PATH) {
          return JSON.stringify(entry) + "\n";
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const attrs = Object.fromEntries(span.attributes.map(a => [a.key, a.value.intValue ?? a.value.stringValue]));
      expect(attrs["gh-aw.github.rate_limit.remaining"]).toBe(4823);
      expect(Object.keys(attrs)).not.toContain("gh-aw.github.rate_limit.reset");
    });

    it("omits rate-limit attributes when the file contains only invalid JSON", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === GITHUB_RATE_LIMITS_JSONL_PATH) {
          return "not valid json\n";
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const keys = span.attributes.map(a => a.key);
      expect(keys).not.toContain("gh-aw.github.rate_limit.remaining");
    });
  });

  describe("token breakdown enrichment in agent span", () => {
    let readFileSpy;
    let statSpy;

    beforeEach(() => {
      process.env.INPUT_JOB_NAME = "agent";
      const agentEndMs = 1_700_000_005_000;
      statSpy = vi.spyOn(fs, "statSync").mockReturnValue(/** @type {Partial<fs.Stats>} */ { mtimeMs: agentEndMs });
      readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(() => {
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });
    });

    afterEach(() => {
      readFileSpy.mockRestore();
      statSpy.mockRestore();
    });

    it("includes all four gen_ai token breakdown attributes on the agent span when agent_usage.json is present", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      const usage = { input_tokens: 48200, output_tokens: 1350, cache_read_tokens: 41000, cache_write_tokens: 3100, effective_tokens: 9800 };
      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_usage.json") {
          return JSON.stringify(usage);
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.agent.conclusion", { startMs: 1_700_000_000_000 });

      const agentBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      const agentSpan = agentBody.resourceSpans[0].scopeSpans[0].spans[0];
      const attrs = Object.fromEntries(agentSpan.attributes.map(a => [a.key, a.value.intValue ?? a.value.stringValue]));
      expect(attrs["gen_ai.usage.input_tokens"]).toBe(48200);
      expect(attrs["gen_ai.usage.output_tokens"]).toBe(1350);
      expect(attrs["gen_ai.usage.cache_read.input_tokens"]).toBe(41000);
      expect(attrs["gen_ai.usage.cache_creation.input_tokens"]).toBe(3100);
    });

    it("omits all gen_ai token breakdown attributes when agent_usage.json is absent", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      // readFileSpy already throws ENOENT for all paths

      await sendJobConclusionSpan("gh-aw.agent.conclusion", { startMs: 1_700_000_000_000 });

      const agentBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      const agentSpan = agentBody.resourceSpans[0].scopeSpans[0].spans[0];
      const keys = agentSpan.attributes.map(a => a.key);
      expect(keys).not.toContain("gen_ai.usage.input_tokens");
      expect(keys).not.toContain("gen_ai.usage.output_tokens");
      expect(keys).not.toContain("gen_ai.usage.cache_read.input_tokens");
      expect(keys).not.toContain("gen_ai.usage.cache_creation.input_tokens");
    });

    it("omits a gen_ai token attribute when its value is zero", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      const usage = { input_tokens: 1000, output_tokens: 0, cache_read_tokens: 500, cache_write_tokens: 0 };
      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_usage.json") {
          return JSON.stringify(usage);
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.agent.conclusion", { startMs: 1_700_000_000_000 });

      const agentBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      const agentSpan = agentBody.resourceSpans[0].scopeSpans[0].spans[0];
      const attrs = Object.fromEntries(agentSpan.attributes.map(a => [a.key, a.value.intValue ?? a.value.stringValue]));
      expect(attrs["gen_ai.usage.input_tokens"]).toBe(1000);
      expect(attrs["gen_ai.usage.cache_read.input_tokens"]).toBe(500);
      const keys = agentSpan.attributes.map(a => a.key);
      expect(keys).not.toContain("gen_ai.usage.output_tokens");
      expect(keys).not.toContain("gen_ai.usage.cache_creation.input_tokens");
    });

    it("omits gen_ai token breakdown attributes when agent_usage.json contains invalid JSON", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/agent_usage.json") {
          return "not valid json";
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.agent.conclusion", { startMs: 1_700_000_000_000 });

      const agentBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      const agentSpan = agentBody.resourceSpans[0].scopeSpans[0].spans[0];
      const keys = agentSpan.attributes.map(a => a.key);
      expect(keys).not.toContain("gen_ai.usage.input_tokens");
      expect(keys).not.toContain("gen_ai.usage.output_tokens");
      expect(keys).not.toContain("gen_ai.usage.cache_read.input_tokens");
      expect(keys).not.toContain("gen_ai.usage.cache_creation.input_tokens");
    });
  });

  it("includes github.workflow_ref as resource attribute when GITHUB_WORKFLOW_REF is set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.GITHUB_WORKFLOW_REF = "owner/repo/.github/workflows/my-workflow.yml@refs/heads/main";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    expect(resourceAttrs).toContainEqual({ key: "github.workflow_ref", value: { stringValue: "owner/repo/.github/workflows/my-workflow.yml@refs/heads/main" } });
  });

  it("omits github.workflow_ref resource attribute when GITHUB_WORKFLOW_REF is not set", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

    await sendJobConclusionSpan("gh-aw.job.conclusion");

    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    const resourceAttrs = body.resourceSpans[0].resource.attributes;
    const resourceKeys = resourceAttrs.map(a => a.key);
    expect(resourceKeys).not.toContain("github.workflow_ref");
  });

  describe("staged / deployment.environment", () => {
    let readFileSpy;

    beforeEach(() => {
      readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(() => {
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });
    });

    afterEach(() => {
      readFileSpy.mockRestore();
    });

    it("sets gh-aw.staged=false and deployment.environment=production when staged is not set", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const stagedAttr = span.attributes.find(a => a.key === "gh-aw.staged");
      expect(stagedAttr).toBeDefined();
      expect(stagedAttr.value.boolValue).toBe(false);

      const resourceAttrs = body.resourceSpans[0].resource.attributes;
      expect(resourceAttrs).toContainEqual({ key: "deployment.environment", value: { stringValue: "production" } });
    });

    it("sets gh-aw.staged=true and deployment.environment=staging when awInfo.staged=true", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/aw_info.json") {
          return JSON.stringify({ staged: true });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const stagedAttr = span.attributes.find(a => a.key === "gh-aw.staged");
      expect(stagedAttr).toBeDefined();
      expect(stagedAttr.value.boolValue).toBe(true);

      const resourceAttrs = body.resourceSpans[0].resource.attributes;
      expect(resourceAttrs).toContainEqual({ key: "deployment.environment", value: { stringValue: "staging" } });
    });

    it("sets gh-aw.staged=false and deployment.environment=production when awInfo.staged=false", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/aw_info.json") {
          return JSON.stringify({ staged: false });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const stagedAttr = span.attributes.find(a => a.key === "gh-aw.staged");
      expect(stagedAttr).toBeDefined();
      expect(stagedAttr.value.boolValue).toBe(false);

      const resourceAttrs = body.resourceSpans[0].resource.attributes;
      expect(resourceAttrs).toContainEqual({ key: "deployment.environment", value: { stringValue: "production" } });
    });
  });

  describe("trigger item context from aw_info.json", () => {
    let readFileSpy;

    beforeEach(() => {
      readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(() => {
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });
    });

    afterEach(() => {
      readFileSpy.mockRestore();
    });

    it("emits gh-aw.trigger.item_type and gh-aw.trigger.item_number from aw_info.context", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/aw_info.json") {
          return JSON.stringify({ context: { item_type: "issue", item_number: "7", trigger_label: "" } });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.attributes).toContainEqual({ key: "gh-aw.trigger.item_type", value: { stringValue: "issue" } });
      expect(span.attributes).toContainEqual({ key: "gh-aw.trigger.item_number", value: { stringValue: "7" } });
      const keys = span.attributes.map(a => a.key);
      expect(keys).not.toContain("gh-aw.trigger.label");
    });

    it("emits gh-aw.trigger.label when trigger_label is non-empty", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      readFileSpy.mockImplementation(filePath => {
        if (filePath === "/tmp/gh-aw/aw_info.json") {
          return JSON.stringify({ context: { item_type: "pull_request", item_number: "456", trigger_label: "bug" } });
        }
        throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
      });

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      expect(span.attributes).toContainEqual({ key: "gh-aw.trigger.item_type", value: { stringValue: "pull_request" } });
      expect(span.attributes).toContainEqual({ key: "gh-aw.trigger.item_number", value: { stringValue: "456" } });
      expect(span.attributes).toContainEqual({ key: "gh-aw.trigger.label", value: { stringValue: "bug" } });
    });

    it("omits trigger attributes when aw_info.json is absent", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
      vi.stubGlobal("fetch", mockFetch);

      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";

      await sendJobConclusionSpan("gh-aw.job.conclusion");

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      const span = body.resourceSpans[0].scopeSpans[0].spans[0];
      const keys = span.attributes.map(a => a.key);
      expect(keys).not.toContain("gh-aw.trigger.item_type");
      expect(keys).not.toContain("gh-aw.trigger.item_number");
      expect(keys).not.toContain("gh-aw.trigger.label");
    });
  });
});
