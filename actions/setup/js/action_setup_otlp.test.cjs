// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";
import { tmpdir } from "os";
import { join } from "path";
import { writeFileSync, readFileSync, mkdtempSync, rmSync } from "fs";

const req = createRequire(import.meta.url);

// Load send_otlp_span.cjs first so we can patch its exports before run() calls require()
// inside action_setup_otlp.cjs. Both share the same CJS module cache, so patching the
// exports object here is reflected when run() destructures from require(...send_otlp_span.cjs).
const sendOtlpModule = req("./send_otlp_span.cjs");
const originalSendJobSetupSpan = sendOtlpModule.sendJobSetupSpan;
const originalIsValidTraceId = sendOtlpModule.isValidTraceId;
const originalIsValidSpanId = sendOtlpModule.isValidSpanId;

// Load the module under test after patching is possible
const { run } = req("./action_setup_otlp.cjs");

const mockSendJobSetupSpan = vi.fn();

/** 32 lowercase hex chars — valid OTLP trace ID */
const VALID_TRACE_ID = "0102030405060708090a0b0c0d0e0f10";
/** 16 lowercase hex chars — valid OTLP span ID */
const VALID_SPAN_ID = "0102030405060708";

describe("action_setup_otlp.cjs", () => {
  /** @type {string} */
  let outputFile;
  /** @type {string} */
  let envFile;
  /** @type {string} */
  let tempDir;
  /** @type {Record<string, string | undefined>} */
  let savedEnv;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, "log").mockImplementation(() => {});

    // Patch the shared CJS module exports — run() re-destructures on every call
    sendOtlpModule.sendJobSetupSpan = mockSendJobSetupSpan;
    mockSendJobSetupSpan.mockResolvedValue({ traceId: VALID_TRACE_ID, spanId: VALID_SPAN_ID });

    // Provide fresh temp files so GITHUB_OUTPUT and GITHUB_ENV writes are isolated
    tempDir = mkdtempSync(join(tmpdir(), "action-setup-otlp-test-"));
    outputFile = join(tempDir, "test_github_output");
    envFile = join(tempDir, "test_github_env");
    writeFileSync(outputFile, "");
    writeFileSync(envFile, "");

    savedEnv = {
      OTEL_EXPORTER_OTLP_ENDPOINT: process.env.OTEL_EXPORTER_OTLP_ENDPOINT,
      SETUP_START_MS: process.env.SETUP_START_MS,
      GITHUB_OUTPUT: process.env.GITHUB_OUTPUT,
      GITHUB_ENV: process.env.GITHUB_ENV,
      INPUT_TRACE_ID: process.env.INPUT_TRACE_ID,
      "INPUT_TRACE-ID": process.env["INPUT_TRACE-ID"],
      INPUT_JOB_NAME: process.env.INPUT_JOB_NAME,
      "INPUT_JOB-NAME": process.env["INPUT_JOB-NAME"],
    };

    delete process.env.OTEL_EXPORTER_OTLP_ENDPOINT;
    delete process.env.SETUP_START_MS;
    delete process.env.INPUT_TRACE_ID;
    delete process.env["INPUT_TRACE-ID"];
    delete process.env.INPUT_JOB_NAME;
    delete process.env["INPUT_JOB-NAME"];
    process.env.GITHUB_OUTPUT = outputFile;
    process.env.GITHUB_ENV = envFile;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    sendOtlpModule.sendJobSetupSpan = originalSendJobSetupSpan;
    sendOtlpModule.isValidTraceId = originalIsValidTraceId;
    sendOtlpModule.isValidSpanId = originalIsValidSpanId;

    rmSync(tempDir, { recursive: true, force: true });

    for (const [key, val] of Object.entries(savedEnv)) {
      if (val !== undefined) process.env[key] = val;
      else delete process.env[key];
    }
  });

  it("should export run as a function", () => {
    expect(typeof run).toBe("function");
  });

  describe("when OTEL_EXPORTER_OTLP_ENDPOINT is not set", () => {
    it("should log that OTLP export is being skipped", async () => {
      await run();

      expect(console.log).toHaveBeenCalledWith("[otlp] OTEL_EXPORTER_OTLP_ENDPOINT not set, skipping setup span");
    });

    it("should still call sendJobSetupSpan for JSONL mirror", async () => {
      await run();

      expect(mockSendJobSetupSpan).toHaveBeenCalledOnce();
    });

    it("should not log 'sending setup span to' when endpoint is absent", async () => {
      await run();

      const logged = vi.mocked(console.log).mock.calls.flat();
      expect(logged.some(msg => typeof msg === "string" && msg.includes("sending setup span to"))).toBe(false);
    });
  });

  describe("when OTEL_EXPORTER_OTLP_ENDPOINT is set", () => {
    beforeEach(() => {
      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:4318";
    });

    it("should log sending the setup span to the configured endpoint", async () => {
      await run();

      expect(console.log).toHaveBeenCalledWith("[otlp] sending setup span to http://localhost:4318");
    });

    it("should call sendJobSetupSpan exactly once", async () => {
      await run();

      expect(mockSendJobSetupSpan).toHaveBeenCalledOnce();
    });

    it("should log setup span sent with traceId and spanId", async () => {
      await run();

      expect(console.log).toHaveBeenCalledWith(expect.stringContaining(`traceId=${VALID_TRACE_ID}`));
      expect(console.log).toHaveBeenCalledWith(expect.stringContaining(`spanId=${VALID_SPAN_ID}`));
    });

    it("should log the resolved trace ID", async () => {
      await run();

      expect(console.log).toHaveBeenCalledWith(expect.stringContaining(`trace-id=${VALID_TRACE_ID}`));
    });
  });

  describe("SETUP_START_MS propagation", () => {
    it("should pass startMs=0 when SETUP_START_MS is not set", async () => {
      await run();

      expect(mockSendJobSetupSpan).toHaveBeenCalledWith(expect.objectContaining({ startMs: 0 }));
    });

    it("should pass the parsed integer startMs when SETUP_START_MS is a valid number", async () => {
      const jobStartMs = Date.now() - 60_000;
      process.env.SETUP_START_MS = String(jobStartMs);

      await run();

      expect(mockSendJobSetupSpan).toHaveBeenCalledWith(expect.objectContaining({ startMs: jobStartMs }));
    });

    it("should pass startMs=0 when SETUP_START_MS is not a number", async () => {
      process.env.SETUP_START_MS = "not-a-number";

      await run();

      expect(mockSendJobSetupSpan).toHaveBeenCalledWith(expect.objectContaining({ startMs: 0 }));
    });

    it("should pass startMs=0 when SETUP_START_MS is a partial number string like '123abc'", async () => {
      process.env.SETUP_START_MS = "123abc";

      await run();

      expect(mockSendJobSetupSpan).toHaveBeenCalledWith(expect.objectContaining({ startMs: 0 }));
    });

    it("should pass startMs=0 when SETUP_START_MS is scientific notation like '1e3'", async () => {
      process.env.SETUP_START_MS = "1e3";

      await run();

      expect(mockSendJobSetupSpan).toHaveBeenCalledWith(expect.objectContaining({ startMs: 0 }));
    });

    it("should pass startMs=0 when SETUP_START_MS is hex notation like '0x10'", async () => {
      process.env.SETUP_START_MS = "0x10";

      await run();

      expect(mockSendJobSetupSpan).toHaveBeenCalledWith(expect.objectContaining({ startMs: 0 }));
    });
  });

  describe("INPUT_TRACE_ID handling", () => {
    it("should pass traceId=undefined when INPUT_TRACE_ID is not set", async () => {
      await run();

      expect(mockSendJobSetupSpan).toHaveBeenCalledWith(expect.objectContaining({ traceId: undefined }));
    });

    it("should pass the trace ID as-is (already lowercase) when INPUT_TRACE_ID is lowercase", async () => {
      process.env.INPUT_TRACE_ID = "abcdef0123456789abcdef0123456789";

      await run();

      expect(mockSendJobSetupSpan).toHaveBeenCalledWith(expect.objectContaining({ traceId: "abcdef0123456789abcdef0123456789" }));
    });

    it("should normalize INPUT_TRACE_ID to lowercase", async () => {
      process.env.INPUT_TRACE_ID = "ABCDEF0123456789ABCDEF0123456789";

      await run();

      expect(mockSendJobSetupSpan).toHaveBeenCalledWith(expect.objectContaining({ traceId: "abcdef0123456789abcdef0123456789" }));
    });

    it("should log the INPUT_TRACE_ID value when set", async () => {
      process.env.INPUT_TRACE_ID = "abcdef0123456789abcdef0123456789";

      await run();

      expect(console.log).toHaveBeenCalledWith(expect.stringContaining("INPUT_TRACE_ID=abcdef0123456789abcdef0123456789"));
    });

    it("should log that INPUT_TRACE_ID is not set when absent", async () => {
      await run();

      expect(console.log).toHaveBeenCalledWith("[otlp] INPUT_TRACE_ID not set, a new trace ID will be generated");
    });

    it("should accept the hyphen form INPUT_TRACE-ID as a fallback", async () => {
      process.env["INPUT_TRACE-ID"] = "abcdef0123456789abcdef0123456789";

      await run();

      expect(mockSendJobSetupSpan).toHaveBeenCalledWith(expect.objectContaining({ traceId: "abcdef0123456789abcdef0123456789" }));
    });

    it("should prefer INPUT_TRACE_ID over the hyphen form when both are set", async () => {
      process.env.INPUT_TRACE_ID = "1111111111111111111111111111111a";
      process.env["INPUT_TRACE-ID"] = "2222222222222222222222222222222b";

      await run();

      expect(mockSendJobSetupSpan).toHaveBeenCalledWith(expect.objectContaining({ traceId: "1111111111111111111111111111111a" }));
    });
  });

  describe("INPUT_JOB_NAME normalization", () => {
    it("should set INPUT_JOB_NAME env var from the hyphen form when only hyphen form is present", async () => {
      delete process.env.INPUT_JOB_NAME;
      process.env["INPUT_JOB-NAME"] = "agent";

      await run();

      expect(process.env.INPUT_JOB_NAME).toBe("agent");
    });

    it("should set INPUT_JOB_NAME env var when INPUT_JOB_NAME is already in underscore form", async () => {
      process.env.INPUT_JOB_NAME = "setup";

      await run();

      expect(process.env.INPUT_JOB_NAME).toBe("setup");
    });
  });

  describe("GITHUB_OUTPUT file writing", () => {
    it("should write trace-id to GITHUB_OUTPUT when trace ID is valid", async () => {
      await run();

      const content = readFileSync(outputFile, "utf8");
      expect(content).toContain(`trace-id=${VALID_TRACE_ID}\n`);
    });

    it("should log that trace-id was written to GITHUB_OUTPUT", async () => {
      await run();

      expect(console.log).toHaveBeenCalledWith(`[otlp] trace-id=${VALID_TRACE_ID} written to GITHUB_OUTPUT`);
    });

    it("should not write to GITHUB_OUTPUT when GITHUB_OUTPUT is not set", async () => {
      delete process.env.GITHUB_OUTPUT;

      // Should not throw
      await expect(run()).resolves.toBeUndefined();
    });

    it("should not write trace-id to GITHUB_OUTPUT when returned trace ID is invalid", async () => {
      mockSendJobSetupSpan.mockResolvedValue({ traceId: "not-valid", spanId: VALID_SPAN_ID });

      await run();

      const content = readFileSync(outputFile, "utf8");
      expect(content).not.toContain("trace-id=");
    });
  });

  describe("GITHUB_ENV file writing", () => {
    it("should write GITHUB_AW_OTEL_TRACE_ID to GITHUB_ENV when trace ID is valid", async () => {
      await run();

      const content = readFileSync(envFile, "utf8");
      expect(content).toContain(`GITHUB_AW_OTEL_TRACE_ID=${VALID_TRACE_ID}\n`);
    });

    it("should write GITHUB_AW_OTEL_PARENT_SPAN_ID to GITHUB_ENV when span ID is valid", async () => {
      await run();

      const content = readFileSync(envFile, "utf8");
      expect(content).toContain(`GITHUB_AW_OTEL_PARENT_SPAN_ID=${VALID_SPAN_ID}\n`);
    });

    it("should write GITHUB_AW_OTEL_JOB_START_MS with a positive integer timestamp", async () => {
      await run();

      const content = readFileSync(envFile, "utf8");
      const match = content.match(/GITHUB_AW_OTEL_JOB_START_MS=(\d+)\n/);
      expect(match).not.toBeNull();
      const ts = parseInt(match?.[1] ?? "0", 10);
      expect(ts).toBeGreaterThan(0);
    });

    it("should always write GITHUB_AW_OTEL_JOB_START_MS even when trace ID is invalid", async () => {
      mockSendJobSetupSpan.mockResolvedValue({ traceId: "bad", spanId: "bad" });

      await run();

      const content = readFileSync(envFile, "utf8");
      expect(content).toContain("GITHUB_AW_OTEL_JOB_START_MS=");
    });

    it("should not write to GITHUB_ENV when GITHUB_ENV is not set", async () => {
      delete process.env.GITHUB_ENV;

      // Should not throw
      await expect(run()).resolves.toBeUndefined();
    });

    it("should not write GITHUB_AW_OTEL_TRACE_ID when returned trace ID is invalid", async () => {
      mockSendJobSetupSpan.mockResolvedValue({ traceId: "not-valid", spanId: VALID_SPAN_ID });

      await run();

      const content = readFileSync(envFile, "utf8");
      expect(content).not.toContain("GITHUB_AW_OTEL_TRACE_ID=");
    });

    it("should not write GITHUB_AW_OTEL_PARENT_SPAN_ID when returned span ID is invalid", async () => {
      mockSendJobSetupSpan.mockResolvedValue({ traceId: VALID_TRACE_ID, spanId: "not-valid" });

      await run();

      const content = readFileSync(envFile, "utf8");
      expect(content).not.toContain("GITHUB_AW_OTEL_PARENT_SPAN_ID=");
    });

    it("should log the GITHUB_AW_OTEL_TRACE_ID write", async () => {
      await run();

      expect(console.log).toHaveBeenCalledWith("[otlp] GITHUB_AW_OTEL_TRACE_ID written to GITHUB_ENV");
    });

    it("should log the GITHUB_AW_OTEL_PARENT_SPAN_ID write", async () => {
      await run();

      expect(console.log).toHaveBeenCalledWith("[otlp] GITHUB_AW_OTEL_PARENT_SPAN_ID written to GITHUB_ENV");
    });

    it("should log the GITHUB_AW_OTEL_JOB_START_MS write", async () => {
      await run();

      expect(console.log).toHaveBeenCalledWith("[otlp] GITHUB_AW_OTEL_JOB_START_MS written to GITHUB_ENV");
    });
  });

  describe("error handling", () => {
    it("should propagate errors from sendJobSetupSpan to the caller", async () => {
      mockSendJobSetupSpan.mockRejectedValueOnce(new Error("OTLP connection refused"));

      await expect(run()).rejects.toThrow("OTLP connection refused");
    });
  });
});
