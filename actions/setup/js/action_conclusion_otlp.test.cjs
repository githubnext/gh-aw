// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";

// Use CJS require so we share the same module cache as action_conclusion_otlp.cjs
const req = createRequire(import.meta.url);

// Load the real send_otlp_span module and capture the original function
const sendOtlpModule = req("./send_otlp_span.cjs");
const originalSendJobConclusionSpan = sendOtlpModule.sendJobConclusionSpan;

// Load the module under test — it holds a reference to the same sendOtlpModule object
const { run } = req("./action_conclusion_otlp.cjs");

// Shared mock function — patched onto the module exports in beforeEach
const mockSendJobConclusionSpan = vi.fn();

describe("action_conclusion_otlp.cjs", () => {
  let originalEnv;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, "log").mockImplementation(() => {});
    mockSendJobConclusionSpan.mockResolvedValue(undefined);
    // Patch the shared CJS exports object — run() accesses this at call time
    sendOtlpModule.sendJobConclusionSpan = mockSendJobConclusionSpan;

    originalEnv = {
      OTEL_EXPORTER_OTLP_ENDPOINT: process.env.OTEL_EXPORTER_OTLP_ENDPOINT,
      INPUT_JOB_NAME: process.env.INPUT_JOB_NAME,
      "INPUT_JOB-NAME": process.env["INPUT_JOB-NAME"],
      GITHUB_AW_OTEL_JOB_START_MS: process.env.GITHUB_AW_OTEL_JOB_START_MS,
    };
    delete process.env.OTEL_EXPORTER_OTLP_ENDPOINT;
    delete process.env.INPUT_JOB_NAME;
    delete process.env["INPUT_JOB-NAME"];
    delete process.env.GITHUB_AW_OTEL_JOB_START_MS;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    sendOtlpModule.sendJobConclusionSpan = originalSendJobConclusionSpan;

    if (originalEnv.OTEL_EXPORTER_OTLP_ENDPOINT !== undefined) {
      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = originalEnv.OTEL_EXPORTER_OTLP_ENDPOINT;
    } else {
      delete process.env.OTEL_EXPORTER_OTLP_ENDPOINT;
    }
    if (originalEnv.INPUT_JOB_NAME !== undefined) {
      process.env.INPUT_JOB_NAME = originalEnv.INPUT_JOB_NAME;
    } else {
      delete process.env.INPUT_JOB_NAME;
    }
    if (originalEnv["INPUT_JOB-NAME"] !== undefined) {
      process.env["INPUT_JOB-NAME"] = originalEnv["INPUT_JOB-NAME"];
    } else {
      delete process.env["INPUT_JOB-NAME"];
    }
    if (originalEnv.GITHUB_AW_OTEL_JOB_START_MS !== undefined) {
      process.env.GITHUB_AW_OTEL_JOB_START_MS = originalEnv.GITHUB_AW_OTEL_JOB_START_MS;
    } else {
      delete process.env.GITHUB_AW_OTEL_JOB_START_MS;
    }
  });

  it("should export run as a function", () => {
    expect(typeof run).toBe("function");
  });

  describe("when OTEL_EXPORTER_OTLP_ENDPOINT is not set", () => {
    it("should log that OTLP export is skipped and JSONL mirror will be attempted", async () => {
      await run();

      expect(console.log).toHaveBeenCalledWith("[otlp] OTEL_EXPORTER_OTLP_ENDPOINT not set, skipping OTLP export (will attempt JSONL mirror)");
    });

    it("should still call sendJobConclusionSpan for JSONL mirror", async () => {
      await run();

      expect(mockSendJobConclusionSpan).toHaveBeenCalledOnce();
      expect(mockSendJobConclusionSpan).toHaveBeenCalledWith("gh-aw.job.conclusion", { startMs: undefined });
    });
  });

  describe("when OTEL_EXPORTER_OTLP_ENDPOINT is set", () => {
    beforeEach(() => {
      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:4318";
    });

    it("should call sendJobConclusionSpan once", async () => {
      await run();

      expect(mockSendJobConclusionSpan).toHaveBeenCalledOnce();
    });

    it("should log the conclusion span export as attempted", async () => {
      await run();

      expect(console.log).toHaveBeenCalledWith("[otlp] conclusion span export attempted");
    });

    it("should log the endpoint URL in the sending message", async () => {
      await run();

      expect(console.log).toHaveBeenCalledWith(expect.stringContaining("http://localhost:4318"));
    });

    describe("span name construction", () => {
      it("should use default span name 'gh-aw.job.conclusion' when INPUT_JOB_NAME is not set", async () => {
        await run();

        expect(mockSendJobConclusionSpan).toHaveBeenCalledWith("gh-aw.job.conclusion", { startMs: undefined });
      });

      it("should use job name from INPUT_JOB_NAME when set", async () => {
        process.env.INPUT_JOB_NAME = "agent";

        await run();

        expect(mockSendJobConclusionSpan).toHaveBeenCalledWith("gh-aw.agent.conclusion", { startMs: undefined });
      });

      it("should use job name from INPUT_JOB-NAME (hyphen form) when INPUT_JOB_NAME is not set", async () => {
        delete process.env.INPUT_JOB_NAME;
        process.env["INPUT_JOB-NAME"] = "agent";

        await run();

        expect(mockSendJobConclusionSpan).toHaveBeenCalledWith("gh-aw.agent.conclusion", { startMs: undefined });
      });

      it("should log the full span name in the sending message", async () => {
        process.env.INPUT_JOB_NAME = "setup";

        await run();

        expect(console.log).toHaveBeenCalledWith('[otlp] sending conclusion span "gh-aw.setup.conclusion" to http://localhost:4318');
      });

      it("should handle different job names correctly", async () => {
        process.env.INPUT_JOB_NAME = "activation";

        await run();

        expect(mockSendJobConclusionSpan).toHaveBeenCalledWith("gh-aw.activation.conclusion", { startMs: undefined });
      });
    });

    describe("startMs propagation from GITHUB_AW_OTEL_JOB_START_MS", () => {
      it("should pass startMs when GITHUB_AW_OTEL_JOB_START_MS is set to a valid timestamp", async () => {
        const jobStartMs = Date.now() - 60_000; // 1 minute ago
        process.env.GITHUB_AW_OTEL_JOB_START_MS = String(jobStartMs);

        await run();

        expect(mockSendJobConclusionSpan).toHaveBeenCalledWith("gh-aw.job.conclusion", { startMs: jobStartMs });
      });

      it("should pass startMs: undefined when GITHUB_AW_OTEL_JOB_START_MS is not set", async () => {
        await run();

        expect(mockSendJobConclusionSpan).toHaveBeenCalledWith("gh-aw.job.conclusion", { startMs: undefined });
      });

      it("should pass startMs: undefined when GITHUB_AW_OTEL_JOB_START_MS is '0'", async () => {
        process.env.GITHUB_AW_OTEL_JOB_START_MS = "0";

        await run();

        expect(mockSendJobConclusionSpan).toHaveBeenCalledWith("gh-aw.job.conclusion", { startMs: undefined });
      });

      it("should pass startMs: undefined when GITHUB_AW_OTEL_JOB_START_MS is not a number", async () => {
        process.env.GITHUB_AW_OTEL_JOB_START_MS = "not-a-number";

        await run();

        expect(mockSendJobConclusionSpan).toHaveBeenCalledWith("gh-aw.job.conclusion", { startMs: undefined });
      });
    });
  });

  describe("error handling", () => {
    it("should propagate errors from sendJobConclusionSpan", async () => {
      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:4318";
      mockSendJobConclusionSpan.mockRejectedValueOnce(new Error("Network error"));

      // run() propagates the error; callers swallow it via .catch(() => {})
      await expect(run()).rejects.toThrow("Network error");
    });
  });
});
