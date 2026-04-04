import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// ---------------------------------------------------------------------------
// Module import
// ---------------------------------------------------------------------------

const { isValidTraceId, generateTraceId, generateSpanId, toNanoString, buildAttr, buildOTLPPayload, sendOTLPSpan, sendJobSetupSpan } = await import("./send_otlp_span.cjs");

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
      attributes: [buildAttr("foo", "bar")],
    });

    expect(payload.resourceSpans).toHaveLength(1);
    const rs = payload.resourceSpans[0];

    // Resource
    expect(rs.resource.attributes).toContainEqual({ key: "service.name", value: { stringValue: "gh-aw" } });

    // Scope
    expect(rs.scopeSpans).toHaveLength(1);
    expect(rs.scopeSpans[0].scope.name).toBe("gh-aw.setup");

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
});

// ---------------------------------------------------------------------------
// sendOTLPSpan
// ---------------------------------------------------------------------------

describe("sendOTLPSpan", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
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

  it("throws when server returns non-2xx status", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: false, status: 400, statusText: "Bad Request" });
    vi.stubGlobal("fetch", mockFetch);

    await expect(sendOTLPSpan("https://traces.example.com", {})).rejects.toThrow("OTLP export failed: HTTP 400 Bad Request");
  });
});

// ---------------------------------------------------------------------------
// sendJobSetupSpan
// ---------------------------------------------------------------------------

describe("sendJobSetupSpan", () => {
  /** @type {Record<string, string | undefined>} */
  const savedEnv = {};
  const envKeys = ["OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_SERVICE_NAME", "INPUT_JOB_NAME", "INPUT_TRACE_ID", "GH_AW_INFO_WORKFLOW_NAME", "GH_AW_INFO_ENGINE_ID", "GITHUB_RUN_ID", "GITHUB_ACTOR", "GITHUB_REPOSITORY"];

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
    for (const k of envKeys) {
      savedEnv[k] = process.env[k];
      delete process.env[k];
    }
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

  it("returns a trace ID even when OTEL_EXPORTER_OTLP_ENDPOINT is not set", async () => {
    const traceId = await sendJobSetupSpan();
    expect(traceId).toMatch(/^[0-9a-f]{32}$/);
    expect(fetch).not.toHaveBeenCalled();
  });

  it("returns the same trace ID when called with INPUT_TRACE_ID and no endpoint", async () => {
    process.env.INPUT_TRACE_ID = "a".repeat(32);
    const traceId = await sendJobSetupSpan();
    expect(traceId).toBe("a".repeat(32));
    expect(fetch).not.toHaveBeenCalled();
  });

  it("generates a new trace ID when INPUT_TRACE_ID is invalid", async () => {
    process.env.INPUT_TRACE_ID = "not-a-valid-trace-id";
    const traceId = await sendJobSetupSpan();
    expect(traceId).toMatch(/^[0-9a-f]{32}$/);
    expect(traceId).not.toBe("not-a-valid-trace-id");
  });

  it("sends a span when endpoint is configured and returns the trace ID", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.INPUT_JOB_NAME = "agent";
    process.env.GH_AW_INFO_WORKFLOW_NAME = "my-workflow";
    process.env.GH_AW_INFO_ENGINE_ID = "copilot";
    process.env.GITHUB_RUN_ID = "123456789";
    process.env.GITHUB_ACTOR = "octocat";
    process.env.GITHUB_REPOSITORY = "owner/repo";

    const traceId = await sendJobSetupSpan();

    expect(traceId).toMatch(/^[0-9a-f]{32}$/);
    expect(mockFetch).toHaveBeenCalledOnce();
    const [url, init] = mockFetch.mock.calls[0];
    expect(url).toBe("https://traces.example.com/v1/traces");
    expect(init.method).toBe("POST");

    const body = JSON.parse(init.body);
    const span = body.resourceSpans[0].scopeSpans[0].spans[0];
    expect(span.name).toBe("gh-aw.job.setup");
    // Span traceId must match the returned value (cross-job correlation)
    expect(span.traceId).toBe(traceId);
    expect(span.spanId).toMatch(/^[0-9a-f]{16}$/);

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

    const returned = await sendJobSetupSpan({ traceId: correlationTraceId });

    expect(returned).toBe(correlationTraceId);
    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    expect(body.resourceSpans[0].scopeSpans[0].spans[0].traceId).toBe(correlationTraceId);
  });

  it("uses trace ID from INPUT_TRACE_ID env var when options.traceId is absent", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.INPUT_TRACE_ID = "c".repeat(32);

    const returned = await sendJobSetupSpan();

    expect(returned).toBe("c".repeat(32));
    const body = JSON.parse(mockFetch.mock.calls[0][1].body);
    expect(body.resourceSpans[0].scopeSpans[0].spans[0].traceId).toBe("c".repeat(32));
  });

  it("options.traceId takes priority over INPUT_TRACE_ID", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, statusText: "OK" });
    vi.stubGlobal("fetch", mockFetch);

    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://traces.example.com";
    process.env.INPUT_TRACE_ID = "d".repeat(32);

    const returned = await sendJobSetupSpan({ traceId: "e".repeat(32) });

    expect(returned).toBe("e".repeat(32));
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
