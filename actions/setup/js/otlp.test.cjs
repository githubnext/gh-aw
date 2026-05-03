// @ts-check
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);

// ---------------------------------------------------------------------------
// Load otlp.cjs (module under test) and patch its send_otlp_span dependency.
// Both share the same CJS module cache, so we can replace exports on the
// already-loaded send_otlp_span module and the otlp module picks them up.
// ---------------------------------------------------------------------------

const sendOtlpModule = req("./send_otlp_span.cjs");
const otlp = req("./otlp.cjs");

/** 32 lowercase hex chars — valid OTLP trace ID */
const VALID_TRACE_ID = "aabbccdd00112233aabbccdd00112233";
/** 16 lowercase hex chars — valid OTLP span ID */
const VALID_SPAN_ID = "aabbccdd00112233";

// Stable stubs that we swap in before each test
const mockBuildAttr = vi.fn();
const mockBuildOTLPPayload = vi.fn();
const mockSendOTLPSpan = vi.fn();
const mockSendOTLPToAllEndpoints = vi.fn();
const mockSanitizeOTLPPayload = vi.fn();
const mockAppendToOTLPJSONL = vi.fn();
const mockGenerateSpanId = vi.fn();
const mockIsValidTraceId = vi.fn();
const mockIsValidSpanId = vi.fn();

// Capture originals so we can restore them after each test
const PATCHED_KEYS = ["buildAttr", "buildOTLPPayload", "sendOTLPSpan", "sendOTLPToAllEndpoints", "sanitizeOTLPPayload", "appendToOTLPJSONL", "generateSpanId", "isValidTraceId", "isValidSpanId"];
const originals = Object.fromEntries(PATCHED_KEYS.map(k => [k, sendOtlpModule[k]]));

describe("otlp.cjs", () => {
  /** @type {Record<string, string | undefined>} */
  let savedEnv;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, "warn").mockImplementation(() => {});

    // Re-apply default implementations after clearAllMocks (which resets them)
    mockBuildAttr.mockImplementation((key, value) => ({ key, value }));
    mockBuildOTLPPayload.mockReturnValue({ resourceSpans: [] });
    mockSendOTLPSpan.mockResolvedValue(undefined);
    // sendOTLPToAllEndpoints delegates to mockSendOTLPSpan so existing per-span
    // assertions still work without changing each individual test.
    mockSendOTLPToAllEndpoints.mockImplementation(async (endpoints, payload, opts) => {
      for (const ep of endpoints) {
        await mockSendOTLPSpan(ep.url, payload, { ...opts, headersOverride: ep.headers !== undefined ? ep.headers : "" });
      }
    });
    mockSanitizeOTLPPayload.mockImplementation(p => p);
    mockAppendToOTLPJSONL.mockReturnValue(undefined);
    mockGenerateSpanId.mockReturnValue(VALID_SPAN_ID);
    mockIsValidTraceId.mockImplementation(id => id === VALID_TRACE_ID);
    mockIsValidSpanId.mockImplementation(id => id === VALID_SPAN_ID);

    // Patch the shared CJS module exports
    sendOtlpModule.buildAttr = mockBuildAttr;
    sendOtlpModule.buildOTLPPayload = mockBuildOTLPPayload;
    sendOtlpModule.sendOTLPSpan = mockSendOTLPSpan;
    sendOtlpModule.sendOTLPToAllEndpoints = mockSendOTLPToAllEndpoints;
    sendOtlpModule.sanitizeOTLPPayload = mockSanitizeOTLPPayload;
    sendOtlpModule.appendToOTLPJSONL = mockAppendToOTLPJSONL;
    sendOtlpModule.generateSpanId = mockGenerateSpanId;
    sendOtlpModule.isValidTraceId = mockIsValidTraceId;
    sendOtlpModule.isValidSpanId = mockIsValidSpanId;
    // Keep SPAN_KIND_CLIENT as-is (it's a constant and does not need a stub)

    savedEnv = {
      GH_AW_OTLP_ENDPOINTS: process.env.GH_AW_OTLP_ENDPOINTS,
      GITHUB_AW_OTEL_TRACE_ID: process.env.GITHUB_AW_OTEL_TRACE_ID,
      GITHUB_AW_OTEL_PARENT_SPAN_ID: process.env.GITHUB_AW_OTEL_PARENT_SPAN_ID,
    };

    process.env.GITHUB_AW_OTEL_TRACE_ID = VALID_TRACE_ID;
    process.env.GITHUB_AW_OTEL_PARENT_SPAN_ID = VALID_SPAN_ID;
    process.env.GH_AW_OTLP_ENDPOINTS = JSON.stringify([{ url: "https://otel.example.com" }]);
  });

  afterEach(() => {
    for (const [k, v] of Object.entries(originals)) {
      sendOtlpModule[k] = v;
    }
    for (const [k, v] of Object.entries(savedEnv)) {
      if (v === undefined) {
        delete process.env[k];
      } else {
        process.env[k] = v;
      }
    }
    vi.restoreAllMocks();
  });

  // ---------------------------------------------------------------------------
  // shim.cjs integration — global.core must be available after require
  // ---------------------------------------------------------------------------

  describe("shim integration", () => {
    it("populates global.core when otlp.cjs is loaded", () => {
      // otlp.cjs requires shim.cjs at module load time; by the time we reach
      // this test global.core must already be set (either by the real
      // github-script runtime or by the shim).
      expect(global.core).toBeDefined();
      expect(typeof global.core.warning).toBe("function");
      expect(typeof global.core.info).toBe("function");
    });
  });

  // ---------------------------------------------------------------------------
  // logSpan — happy path
  // ---------------------------------------------------------------------------

  describe("logSpan", () => {
    it("calls sendOTLPSpan with a payload that includes the tool name as service name", async () => {
      await otlp.logSpan("my-scanner", { "my-scanner.issues_found": 3 });

      expect(mockSendOTLPSpan).toHaveBeenCalledOnce();
      expect(mockBuildOTLPPayload).toHaveBeenCalledOnce();
      const payloadOpts = mockBuildOTLPPayload.mock.calls[0][0];
      expect(payloadOpts.serviceName).toBe("my-scanner");
      expect(payloadOpts.spanName).toBe("my-scanner.run");
    });

    it("uses the trace ID from GITHUB_AW_OTEL_TRACE_ID", async () => {
      await otlp.logSpan("my-scanner", {});

      const payloadOpts = mockBuildOTLPPayload.mock.calls[0][0];
      expect(payloadOpts.traceId).toBe(VALID_TRACE_ID);
    });

    it("uses the parent span ID from GITHUB_AW_OTEL_PARENT_SPAN_ID", async () => {
      await otlp.logSpan("my-scanner", {});

      const payloadOpts = mockBuildOTLPPayload.mock.calls[0][0];
      expect(payloadOpts.parentSpanId).toBe(VALID_SPAN_ID);
    });

    it("reads the endpoint from GH_AW_OTLP_ENDPOINTS", async () => {
      await otlp.logSpan("my-scanner", {});

      expect(mockSendOTLPSpan).toHaveBeenCalledWith("https://otel.example.com", expect.anything(), expect.objectContaining({ skipJSONL: true }));
    });

    it("converts attributes object to buildAttr calls", async () => {
      await otlp.logSpan("my-scanner", { "my-scanner.count": 5, "my-scanner.ok": true, "my-scanner.label": "x" });

      expect(mockBuildAttr).toHaveBeenCalledWith("my-scanner.count", 5);
      expect(mockBuildAttr).toHaveBeenCalledWith("my-scanner.ok", true);
      expect(mockBuildAttr).toHaveBeenCalledWith("my-scanner.label", "x");
    });

    it("uses statusCode 1 (OK) by default", async () => {
      await otlp.logSpan("my-scanner", {});

      const payloadOpts = mockBuildOTLPPayload.mock.calls[0][0];
      expect(payloadOpts.statusCode).toBe(1);
    });

    it("uses statusCode 2 (ERROR) when isError is true", async () => {
      await otlp.logSpan("my-scanner", {}, { isError: true, errorMessage: "scan failed" });

      const payloadOpts = mockBuildOTLPPayload.mock.calls[0][0];
      expect(payloadOpts.statusCode).toBe(2);
      expect(payloadOpts.statusMessage).toBe("scan failed");
    });

    it("accepts options.traceId override", async () => {
      const customTrace = "ccddee0011223344ccddee0011223344";
      mockIsValidTraceId.mockImplementation(id => id === customTrace);

      await otlp.logSpan("my-scanner", {}, { traceId: customTrace });

      const payloadOpts = mockBuildOTLPPayload.mock.calls[0][0];
      expect(payloadOpts.traceId).toBe(customTrace);
    });

    it("accepts options.endpoint override", async () => {
      await otlp.logSpan("my-scanner", {}, { endpoint: "https://custom.otel.io" });

      expect(mockSendOTLPSpan).toHaveBeenCalledWith("https://custom.otel.io", expect.anything(), { skipJSONL: true });
    });

    it("does not attempt HTTP export when GH_AW_OTLP_ENDPOINTS is not set", async () => {
      delete process.env.GH_AW_OTLP_ENDPOINTS;

      await otlp.logSpan("my-scanner", {});

      expect(mockSendOTLPSpan).not.toHaveBeenCalled();
      expect(mockAppendToOTLPJSONL).toHaveBeenCalledOnce();
    });

    it("sanitizes the payload before writing to the JSONL mirror", async () => {
      const rawPayload = { resourceSpans: ["raw"] };
      const sanitizedPayload = { resourceSpans: ["sanitized"] };
      mockBuildOTLPPayload.mockReturnValue(rawPayload);
      mockSanitizeOTLPPayload.mockReturnValue(sanitizedPayload);

      await otlp.logSpan("my-scanner", {});

      expect(mockSanitizeOTLPPayload).toHaveBeenCalledWith(rawPayload);
      expect(mockAppendToOTLPJSONL).toHaveBeenCalledWith(sanitizedPayload);
      // Wire export still uses the original payload (sendOTLPSpan sanitizes internally)
      expect(mockSendOTLPSpan).toHaveBeenCalledWith(expect.any(String), rawPayload, expect.objectContaining({ skipJSONL: true }));
    });
  });

  // ---------------------------------------------------------------------------
  // logSpan — missing / invalid trace ID
  // ---------------------------------------------------------------------------

  describe("logSpan — missing trace ID", () => {
    it("silently skips the span when GITHUB_AW_OTEL_TRACE_ID is not set", async () => {
      delete process.env.GITHUB_AW_OTEL_TRACE_ID;
      mockIsValidTraceId.mockReturnValue(false);

      await otlp.logSpan("my-scanner", { "my-scanner.count": 1 });

      expect(mockSendOTLPSpan).not.toHaveBeenCalled();
      expect(console.warn).not.toHaveBeenCalled();
    });
  });

  // ---------------------------------------------------------------------------
  // logSpan — error resilience
  // ---------------------------------------------------------------------------

  describe("logSpan — error resilience", () => {
    it("does not throw when sendOTLPSpan rejects", async () => {
      mockSendOTLPSpan.mockRejectedValue(new Error("network failure"));

      await expect(otlp.logSpan("my-scanner", {})).resolves.toBeUndefined();
      expect(console.warn).toHaveBeenCalledWith(expect.stringContaining("network failure"));
    });

    it("does not throw when an internal helper throws synchronously", async () => {
      mockBuildOTLPPayload.mockImplementation(() => {
        throw new Error("unexpected");
      });

      await expect(otlp.logSpan("my-scanner", {})).resolves.toBeUndefined();
      expect(console.warn).toHaveBeenCalledWith(expect.stringContaining("unexpected"));
    });
  });

  // ---------------------------------------------------------------------------
  // logSpan — omits parentSpanId when invalid
  // ---------------------------------------------------------------------------

  describe("logSpan — parent span ID handling", () => {
    it("omits parentSpanId when GITHUB_AW_OTEL_PARENT_SPAN_ID is not set", async () => {
      delete process.env.GITHUB_AW_OTEL_PARENT_SPAN_ID;
      mockIsValidSpanId.mockReturnValue(false);

      await otlp.logSpan("my-scanner", {});

      const payloadOpts = mockBuildOTLPPayload.mock.calls[0][0];
      expect(payloadOpts.parentSpanId).toBeUndefined();
    });
  });
});
