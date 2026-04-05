import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

// ---------------------------------------------------------------------------
// Module imports
// ---------------------------------------------------------------------------

const { run: runSetup } = await import("./action_setup_otlp.cjs");
const { run: runConclusion } = await import("./action_conclusion_otlp.cjs");

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// ---------------------------------------------------------------------------
// action_setup_otlp — run()
// ---------------------------------------------------------------------------

describe("action_setup_otlp run()", () => {
  let originalEnv;

  beforeEach(() => {
    originalEnv = { ...process.env };
    // Clear any OTLP endpoint so send_otlp_span.cjs is a no-op
    delete process.env.OTEL_EXPORTER_OTLP_ENDPOINT;
    delete process.env.GITHUB_OUTPUT;
    delete process.env.GITHUB_ENV;
    delete process.env.SETUP_START_MS;
    delete process.env.INPUT_TRACE_ID;
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  it("resolves without throwing when OTLP endpoint is not configured", async () => {
    await expect(runSetup()).resolves.toBeUndefined();
  });

  it("writes trace-id to GITHUB_OUTPUT even when endpoint is not configured", async () => {
    const tmpOut = path.join(path.dirname(__dirname), `action_setup_otlp_test_no_endpoint_${Date.now()}.txt`);
    try {
      // No OTEL endpoint — span must NOT be sent but trace-id must still be written.
      process.env.GITHUB_OUTPUT = tmpOut;
      process.env.GITHUB_ENV = tmpOut;

      await runSetup();

      const contents = fs.readFileSync(tmpOut, "utf8");
      expect(contents).toMatch(/^trace-id=[0-9a-f]{32}$/m);
      expect(contents).toMatch(/^GITHUB_AW_OTEL_TRACE_ID=[0-9a-f]{32}$/m);
    } finally {
      fs.rmSync(tmpOut, { force: true });
    }
  });

  it("uses INPUT_TRACE_ID as trace ID when provided", async () => {
    const inputTraceId = "a".repeat(32);
    const tmpOut = path.join(path.dirname(__dirname), `action_setup_otlp_test_input_tid_${Date.now()}.txt`);
    try {
      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:14317";
      process.env.INPUT_TRACE_ID = inputTraceId;
      process.env.GITHUB_OUTPUT = tmpOut;
      process.env.GITHUB_ENV = tmpOut;

      const fetchSpy = vi.spyOn(global, "fetch").mockResolvedValue(new Response(null, { status: 200 }));

      await runSetup();

      const contents = fs.readFileSync(tmpOut, "utf8");
      expect(contents).toContain(`trace-id=${inputTraceId}`);
      expect(contents).toContain(`GITHUB_AW_OTEL_TRACE_ID=${inputTraceId}`);

      fetchSpy.mockRestore();
    } finally {
      fs.rmSync(tmpOut, { force: true });
    }
  });

  it("writes trace-id to GITHUB_OUTPUT when endpoint is configured", async () => {
    const tmpOut = path.join(path.dirname(__dirname), `action_setup_otlp_test_output_${Date.now()}.txt`);
    try {
      // Provide a fake endpoint (fetch will fail gracefully)
      process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:14317";
      process.env.SETUP_START_MS = String(Date.now() - 1000);
      process.env.GITHUB_OUTPUT = tmpOut;
      process.env.GITHUB_ENV = tmpOut;

      // Mock fetch so no real network call is made
      const fetchSpy = vi.spyOn(global, "fetch").mockResolvedValue(new Response(null, { status: 200 }));

      await runSetup();

      const contents = fs.readFileSync(tmpOut, "utf8");
      expect(contents).toMatch(/^trace-id=[0-9a-f]{32}$/m);
      expect(contents).toMatch(/^GITHUB_AW_OTEL_TRACE_ID=[0-9a-f]{32}$/m);
      expect(contents).toMatch(/^GITHUB_AW_OTEL_PARENT_SPAN_ID=[0-9a-f]{16}$/m);

      fetchSpy.mockRestore();
    } finally {
      fs.rmSync(tmpOut, { force: true });
    }
  });

  it("generates a new trace-id when INPUT_TRACE_ID is absent", async () => {
    const tmpOut = path.join(path.dirname(__dirname), `action_setup_otlp_test_no_input_tid_${Date.now()}.txt`);
    try {
      // INPUT_TRACE_ID is not set — a fresh trace ID must be generated.
      process.env.GITHUB_OUTPUT = tmpOut;
      process.env.GITHUB_ENV = tmpOut;

      await runSetup();

      const contents = fs.readFileSync(tmpOut, "utf8");
      // A generated 32-char hex trace-id must always be written.
      expect(contents).toMatch(/^trace-id=[0-9a-f]{32}$/m);
      expect(contents).toMatch(/^GITHUB_AW_OTEL_TRACE_ID=[0-9a-f]{32}$/m);
    } finally {
      fs.rmSync(tmpOut, { force: true });
    }
  });

  it("does not throw when GITHUB_OUTPUT is not set", async () => {
    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:14317";
    const fetchSpy = vi.spyOn(global, "fetch").mockResolvedValue(new Response(null, { status: 200 }));
    await expect(runSetup()).resolves.toBeUndefined();
    fetchSpy.mockRestore();
  });
});

// ---------------------------------------------------------------------------
// action_conclusion_otlp — run()
// ---------------------------------------------------------------------------

describe("action_conclusion_otlp run()", () => {
  let originalEnv;

  beforeEach(() => {
    originalEnv = { ...process.env };
    delete process.env.OTEL_EXPORTER_OTLP_ENDPOINT;
    delete process.env.INPUT_JOB_NAME;
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  it("resolves without throwing when OTLP endpoint is not configured", async () => {
    await expect(runConclusion()).resolves.toBeUndefined();
  });

  it("resolves without throwing when endpoint is configured", async () => {
    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:14317";
    const fetchSpy = vi.spyOn(global, "fetch").mockResolvedValue(new Response(null, { status: 200 }));
    await expect(runConclusion()).resolves.toBeUndefined();
    fetchSpy.mockRestore();
  });

  it("uses job name from INPUT_JOB_NAME in span name", async () => {
    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:14317";
    process.env.INPUT_JOB_NAME = "agent";
    let capturedBody;
    const fetchSpy = vi.spyOn(global, "fetch").mockImplementation((_url, opts) => {
      capturedBody = opts?.body;
      return Promise.resolve(new Response(null, { status: 200 }));
    });

    await runConclusion();

    const payload = JSON.parse(capturedBody);
    const spanName = payload?.resourceSpans?.[0]?.scopeSpans?.[0]?.spans?.[0]?.name;
    expect(spanName).toBe("gh-aw.job.agent");
    fetchSpy.mockRestore();
  });

  it("uses default span name when INPUT_JOB_NAME is not set", async () => {
    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:14317";
    let capturedBody;
    const fetchSpy = vi.spyOn(global, "fetch").mockImplementation((_url, opts) => {
      capturedBody = opts?.body;
      return Promise.resolve(new Response(null, { status: 200 }));
    });

    await runConclusion();

    const payload = JSON.parse(capturedBody);
    const spanName = payload?.resourceSpans?.[0]?.scopeSpans?.[0]?.spans?.[0]?.name;
    expect(spanName).toBe("gh-aw.job.conclusion");
    fetchSpy.mockRestore();
  });

  it("records agent failure conclusion as STATUS_CODE_ERROR when GH_AW_AGENT_CONCLUSION is 'failure'", async () => {
    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:14317";
    process.env.GH_AW_AGENT_CONCLUSION = "failure";
    let capturedBody;
    const fetchSpy = vi.spyOn(global, "fetch").mockImplementation((_url, opts) => {
      capturedBody = opts?.body;
      return Promise.resolve(new Response(null, { status: 200 }));
    });

    await runConclusion();

    const payload = JSON.parse(capturedBody);
    const span = payload?.resourceSpans?.[0]?.scopeSpans?.[0]?.spans?.[0];
    expect(span?.status?.code).toBe(2); // STATUS_CODE_ERROR
    expect(span?.status?.message).toBe("agent failure");
    const conclusionAttr = span?.attributes?.find(a => a.key === "gh-aw.agent.conclusion");
    expect(conclusionAttr?.value?.stringValue).toBe("failure");
    fetchSpy.mockRestore();
    delete process.env.GH_AW_AGENT_CONCLUSION;
  });

  it("records timed_out conclusion as STATUS_CODE_ERROR when GH_AW_AGENT_CONCLUSION is 'timed_out'", async () => {
    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:14317";
    process.env.GH_AW_AGENT_CONCLUSION = "timed_out";
    let capturedBody;
    const fetchSpy = vi.spyOn(global, "fetch").mockImplementation((_url, opts) => {
      capturedBody = opts?.body;
      return Promise.resolve(new Response(null, { status: 200 }));
    });

    await runConclusion();

    const payload = JSON.parse(capturedBody);
    const span = payload?.resourceSpans?.[0]?.scopeSpans?.[0]?.spans?.[0];
    expect(span?.status?.code).toBe(2); // STATUS_CODE_ERROR
    expect(span?.status?.message).toBe("agent timed_out");
    const conclusionAttr = span?.attributes?.find(a => a.key === "gh-aw.agent.conclusion");
    expect(conclusionAttr?.value?.stringValue).toBe("timed_out");
    fetchSpy.mockRestore();
    delete process.env.GH_AW_AGENT_CONCLUSION;
  });

  it("records success conclusion as STATUS_CODE_OK when GH_AW_AGENT_CONCLUSION is 'success'", async () => {
    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:14317";
    process.env.GH_AW_AGENT_CONCLUSION = "success";
    let capturedBody;
    const fetchSpy = vi.spyOn(global, "fetch").mockImplementation((_url, opts) => {
      capturedBody = opts?.body;
      return Promise.resolve(new Response(null, { status: 200 }));
    });

    await runConclusion();

    const payload = JSON.parse(capturedBody);
    const span = payload?.resourceSpans?.[0]?.scopeSpans?.[0]?.spans?.[0];
    expect(span?.status?.code).toBe(1); // STATUS_CODE_OK
    expect(span?.status?.message).toBeUndefined();
    const conclusionAttr = span?.attributes?.find(a => a.key === "gh-aw.agent.conclusion");
    expect(conclusionAttr?.value?.stringValue).toBe("success");
    fetchSpy.mockRestore();
    delete process.env.GH_AW_AGENT_CONCLUSION;
  });

  it("records cancelled conclusion as STATUS_CODE_OK when GH_AW_AGENT_CONCLUSION is 'cancelled'", async () => {
    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:14317";
    process.env.GH_AW_AGENT_CONCLUSION = "cancelled";
    let capturedBody;
    const fetchSpy = vi.spyOn(global, "fetch").mockImplementation((_url, opts) => {
      capturedBody = opts?.body;
      return Promise.resolve(new Response(null, { status: 200 }));
    });

    await runConclusion();

    const payload = JSON.parse(capturedBody);
    const span = payload?.resourceSpans?.[0]?.scopeSpans?.[0]?.spans?.[0];
    expect(span?.status?.code).toBe(1); // STATUS_CODE_OK (cancelled is not an error)
    const conclusionAttr = span?.attributes?.find(a => a.key === "gh-aw.agent.conclusion");
    expect(conclusionAttr?.value?.stringValue).toBe("cancelled");
    fetchSpy.mockRestore();
    delete process.env.GH_AW_AGENT_CONCLUSION;
  });

  it("omits gh-aw.agent.conclusion attribute when GH_AW_AGENT_CONCLUSION is not set", async () => {
    process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:14317";
    delete process.env.GH_AW_AGENT_CONCLUSION;
    let capturedBody;
    const fetchSpy = vi.spyOn(global, "fetch").mockImplementation((_url, opts) => {
      capturedBody = opts?.body;
      return Promise.resolve(new Response(null, { status: 200 }));
    });

    await runConclusion();

    const payload = JSON.parse(capturedBody);
    const span = payload?.resourceSpans?.[0]?.scopeSpans?.[0]?.spans?.[0];
    expect(span?.status?.code).toBe(1); // STATUS_CODE_OK
    const conclusionAttr = span?.attributes?.find(a => a.key === "gh-aw.agent.conclusion");
    expect(conclusionAttr).toBeUndefined();
    fetchSpy.mockRestore();
  });
});
