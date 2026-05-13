import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const sendOtlpModule = req("./send_otlp_span.cjs");

const mockGenerateTraceId = vi.fn();
const mockGenerateSpanId = vi.fn();
const mockBuildAttr = vi.fn();
const mockBuildOTLPSpan = vi.fn();
const mockBuildOTLPBatchPayload = vi.fn();
const mockBuildGitHubActionsResourceAttributes = vi.fn();
const mockParseOTLPEndpoints = vi.fn();
const mockSendOTLPToAllEndpoints = vi.fn();
const mockAppendToOTLPJSONL = vi.fn();
const mockReadJSONIfExists = vi.fn();

const PATCHED_KEYS = [
  "generateTraceId",
  "generateSpanId",
  "buildAttr",
  "buildOTLPSpan",
  "buildOTLPBatchPayload",
  "buildGitHubActionsResourceAttributes",
  "parseOTLPEndpoints",
  "sendOTLPToAllEndpoints",
  "appendToOTLPJSONL",
  "readJSONIfExists",
];

const originals = Object.fromEntries(PATCHED_KEYS.map(key => [key, sendOtlpModule[key]]));

const EVALUATIONS_PATH = "/tmp/gh-aw/outcome-evaluations.jsonl";
const AW_INFO_PATH = "/tmp/gh-aw/aw_info.json";
const SUMMARY_PATH = "/tmp/gh-aw/outcome-summary.json";

describe("emit_outcome_spans.cjs", () => {
  /** @type {Record<string, string | undefined>} */
  let savedEnv;
  /** @type {unknown} */
  let currentAwInfo;
  /** @type {unknown} */
  let currentSummary;
  /** @type {{ main: () => Promise<void> }} */
  let moduleUnderTest;
  let spanCounter;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, "log").mockImplementation(() => {});
    vi.spyOn(console, "warn").mockImplementation(() => {});

    fs.mkdirSync("/tmp/gh-aw", { recursive: true });
    for (const path of [EVALUATIONS_PATH, AW_INFO_PATH, SUMMARY_PATH]) {
      if (fs.existsSync(path)) {
        fs.unlinkSync(path);
      }
    }

    currentAwInfo = null;
    currentSummary = null;
    spanCounter = 0;

    mockGenerateTraceId.mockReturnValue("trace-generated-1234567890abcdef");
    mockGenerateSpanId.mockImplementation(() => `span-id-${++spanCounter}`);
    mockBuildAttr.mockImplementation((key, value) => ({ key, value }));
    mockBuildOTLPSpan.mockImplementation(opts => ({ ...opts }));
    mockBuildOTLPBatchPayload.mockImplementation(opts => ({ payload: true, ...opts }));
    mockBuildGitHubActionsResourceAttributes.mockImplementation(({ repository, staged }) => [
      { key: "github.repository", value: repository },
      { key: "deployment.environment", value: staged ? "staging" : "production" },
    ]);
    mockParseOTLPEndpoints.mockReturnValue([]);
    mockSendOTLPToAllEndpoints.mockResolvedValue(undefined);
    mockAppendToOTLPJSONL.mockReturnValue(undefined);
    mockReadJSONIfExists.mockImplementation(filePath => {
      if (filePath === AW_INFO_PATH) return currentAwInfo;
      if (filePath === SUMMARY_PATH) return currentSummary;
      return null;
    });

    sendOtlpModule.generateTraceId = mockGenerateTraceId;
    sendOtlpModule.generateSpanId = mockGenerateSpanId;
    sendOtlpModule.buildAttr = mockBuildAttr;
    sendOtlpModule.buildOTLPSpan = mockBuildOTLPSpan;
    sendOtlpModule.buildOTLPBatchPayload = mockBuildOTLPBatchPayload;
    sendOtlpModule.buildGitHubActionsResourceAttributes = mockBuildGitHubActionsResourceAttributes;
    sendOtlpModule.parseOTLPEndpoints = mockParseOTLPEndpoints;
    sendOtlpModule.sendOTLPToAllEndpoints = mockSendOTLPToAllEndpoints;
    sendOtlpModule.appendToOTLPJSONL = mockAppendToOTLPJSONL;
    sendOtlpModule.readJSONIfExists = mockReadJSONIfExists;

    savedEnv = {
      GITHUB_AW_OTEL_TRACE_ID: process.env.GITHUB_AW_OTEL_TRACE_ID,
      GITHUB_AW_OTEL_PARENT_SPAN_ID: process.env.GITHUB_AW_OTEL_PARENT_SPAN_ID,
      GITHUB_REPOSITORY: process.env.GITHUB_REPOSITORY,
      GITHUB_RUN_ID: process.env.GITHUB_RUN_ID,
      GITHUB_RUN_ATTEMPT: process.env.GITHUB_RUN_ATTEMPT,
      GITHUB_EVENT_NAME: process.env.GITHUB_EVENT_NAME,
      GITHUB_REF: process.env.GITHUB_REF,
      GITHUB_REF_NAME: process.env.GITHUB_REF_NAME,
      GITHUB_SHA: process.env.GITHUB_SHA,
      GITHUB_JOB: process.env.GITHUB_JOB,
      GITHUB_WORKFLOW_REF: process.env.GITHUB_WORKFLOW_REF,
      GH_AW_INFO_STAGED: process.env.GH_AW_INFO_STAGED,
      GH_AW_INFO_VERSION: process.env.GH_AW_INFO_VERSION,
      OTEL_SERVICE_NAME: process.env.OTEL_SERVICE_NAME,
    };

    process.env.GITHUB_AW_OTEL_TRACE_ID = "feedfacefeedfacefeedfacefeedface";
    process.env.GITHUB_AW_OTEL_PARENT_SPAN_ID = "cafebabecafebabe";
    process.env.GITHUB_REPOSITORY = "github/gh-aw";
    process.env.GITHUB_RUN_ID = "12345";
    process.env.GITHUB_RUN_ATTEMPT = "2";
    process.env.GITHUB_EVENT_NAME = "workflow_dispatch";
    process.env.GITHUB_REF = "refs/heads/main";
    process.env.GITHUB_REF_NAME = "main";
    process.env.GITHUB_SHA = "abc123";
    process.env.GITHUB_JOB = "outcome-collector";
    process.env.GITHUB_WORKFLOW_REF = "github/gh-aw/.github/workflows/outcome.yml@refs/heads/main";
    delete process.env.GH_AW_INFO_STAGED;
    delete process.env.GH_AW_INFO_VERSION;
    delete process.env.OTEL_SERVICE_NAME;

    const emitModulePath = req.resolve("./emit_outcome_spans.cjs");
    delete req.cache[emitModulePath];
    moduleUnderTest = req("./emit_outcome_spans.cjs");
  });

  afterEach(() => {
    for (const key of PATCHED_KEYS) {
      sendOtlpModule[key] = originals[key];
    }

    const emitModulePath = req.resolve("./emit_outcome_spans.cjs");
    delete req.cache[emitModulePath];

    for (const [key, value] of Object.entries(savedEnv)) {
      if (value === undefined) {
        delete process.env[key];
      } else {
        process.env[key] = value;
      }
    }

    for (const path of [EVALUATIONS_PATH, AW_INFO_PATH, SUMMARY_PATH]) {
      if (fs.existsSync(path)) {
        fs.unlinkSync(path);
      }
    }

    vi.restoreAllMocks();
  });

  it("no-ops when there are no evaluations and no summary data", async () => {
    await moduleUnderTest.main();

    expect(console.log).toHaveBeenCalledWith("[outcome-otel] No outcome evaluations to export");
    expect(mockBuildGitHubActionsResourceAttributes).not.toHaveBeenCalled();
    expect(mockBuildOTLPBatchPayload).not.toHaveBeenCalled();
    expect(mockAppendToOTLPJSONL).not.toHaveBeenCalled();
    expect(mockSendOTLPToAllEndpoints).not.toHaveBeenCalled();
  });

  it("builds item and summary spans using aw_info staged/version metadata and mirrors locally without endpoints", async () => {
    currentAwInfo = { staged: true, agent_version: "v9.9.9" };
    currentSummary = {
      runs_checked: 3,
      total_outcomes: 2,
      accepted: 1,
      rejected: 1,
      ignored: 0,
      pending: 0,
      acceptance_rate: 0.5,
      waste_rate: 0.5,
      date: "2026-05-13",
    };
    fs.writeFileSync(
      EVALUATIONS_PATH,
      [
        JSON.stringify({
          type: "issue",
          result: "accepted",
          detail: "created item",
          workflow: "triage",
          run_id: 101,
          url: "https://github.com/github/gh-aw/issues/1",
          repo: "github/gh-aw",
          timestamp: "2026-05-13T09:00:00Z",
        }),
        JSON.stringify({
          type: "comment",
          result: "rejected",
          workflow: "triage",
          run_id: 102,
          repo: "github/gh-aw",
        }),
      ].join("\n")
    );

    await moduleUnderTest.main();

    expect(mockBuildGitHubActionsResourceAttributes).toHaveBeenCalledWith(
      expect.objectContaining({
        repository: "github/gh-aw",
        runId: "12345",
        runAttempt: "2",
        staged: true,
      })
    );

    expect(mockBuildOTLPBatchPayload).toHaveBeenCalledWith(
      expect.objectContaining({
        serviceName: "gh-aw",
        scopeVersion: "v9.9.9",
        resourceAttributes: [
          { key: "github.repository", value: "github/gh-aw" },
          { key: "deployment.environment", value: "staging" },
        ],
      })
    );

    const { spans } = mockBuildOTLPBatchPayload.mock.calls[0][0];
    const summarySpan = spans[0];
    expect(spans).toHaveLength(3);
    expect(summarySpan).toEqual(
      expect.objectContaining({
        spanName: "gh-aw.outcome.summary",
        parentSpanId: "cafebabecafebabe",
        statusCode: 1,
      })
    );
    expect(spans[1]).toEqual(
      expect.objectContaining({
        spanName: "gh-aw.outcome.evaluation",
        parentSpanId: summarySpan.spanId,
        statusCode: 1,
      })
    );
    expect(spans[2]).toEqual(
      expect.objectContaining({
        spanName: "gh-aw.outcome.evaluation",
        parentSpanId: summarySpan.spanId,
        statusCode: 2,
      })
    );

    expect(summarySpan.attributes).toContainEqual({ key: "gh-aw.exporter.name", value: "outcome-collector" });
    expect(summarySpan.attributes).toContainEqual({ key: "gh-aw.outcome.date", value: "2026-05-13" });
    expect(spans[1].attributes).toContainEqual({ key: "gh-aw.exporter.name", value: "outcome-collector" });
    expect(spans[1].attributes).toContainEqual({ key: "gh-aw.outcome.url", value: "https://github.com/github/gh-aw/issues/1" });
    expect(spans[1].attributes).toContainEqual({ key: "gh-aw.outcome.detail", value: "created item" });
    expect(spans[1].attributes).toContainEqual({ key: "gh-aw.outcome.created_at", value: "2026-05-13T09:00:00Z" });

    expect(mockAppendToOTLPJSONL).toHaveBeenCalledOnce();
    expect(mockSendOTLPToAllEndpoints).not.toHaveBeenCalled();
    expect(console.log).toHaveBeenCalledWith("[outcome-otel] No OTLP endpoints configured, writing JSONL mirror only");
  });

  it("falls back to GH_AW_INFO_* env vars and exports to configured endpoints", async () => {
    currentAwInfo = { staged: false };
    currentSummary = { total_outcomes: 1 };
    process.env.GH_AW_INFO_STAGED = "true";
    process.env.GH_AW_INFO_VERSION = "v2.0.0";
    process.env.OTEL_SERVICE_NAME = "custom-gh-aw";
    mockParseOTLPEndpoints.mockReturnValue([{ url: "https://otel.example.com" }]);

    fs.writeFileSync(
      EVALUATIONS_PATH,
      JSON.stringify({
        type: "issue",
        result: "pending",
        workflow: "nightly",
        run_id: 500,
        repo: "github/gh-aw",
      }) + "\n"
    );

    await moduleUnderTest.main();

    expect(mockBuildGitHubActionsResourceAttributes).toHaveBeenCalledWith(expect.objectContaining({ staged: true }));
    expect(mockBuildOTLPBatchPayload).toHaveBeenCalledWith(
      expect.objectContaining({
        serviceName: "custom-gh-aw",
        scopeVersion: "v2.0.0",
      })
    );
    expect(mockAppendToOTLPJSONL).toHaveBeenCalledOnce();
    expect(mockSendOTLPToAllEndpoints).toHaveBeenCalledWith([{ url: "https://otel.example.com" }], expect.any(Object), { skipJSONL: true });
    expect(console.log).toHaveBeenCalledWith("[outcome-otel] Exported to 1 endpoint(s)");
  });
});
