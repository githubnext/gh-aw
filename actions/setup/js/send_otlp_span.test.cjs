import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";

// ---------------------------------------------------------------------------
// Module import
// ---------------------------------------------------------------------------

const { isValidTraceId, isValidSpanId, generateTraceId, generateSpanId, toNanoString, buildAttr, buildOTLPPayload, parseOTLPHeaders, sendOTLPSpan, sendJobSetupSpan, sendJobConclusionSpan, OTEL_JSONL_PATH, appendToOTLPJSONL } =
  await import("./send_otlp_span.cjs");

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
    expect(span.kind).toBe(2);
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
    "GITHUB_ACTOR",
    "GITHUB_REPOSITORY",
    "GITHUB_EVENT_NAME",
    "GH_AW_INFO_VERSION",
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
    expect(attrs["gh-aw.run.actor"]).toBe("octocat");
    expect(attrs["gh-aw.repository"]).toBe("owner/repo");
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
    "GITHUB_ACTOR",
    "GITHUB_REPOSITORY",
    "GITHUB_EVENT_NAME",
    "INPUT_JOB_NAME",
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

  it("is a no-op when OTEL_EXPORTER_OTLP_ENDPOINT is not set", async () => {
    await sendJobConclusionSpan("gh-aw.job.conclusion");
    expect(fetch).not.toHaveBeenCalled();
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
});
