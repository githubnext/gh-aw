// @ts-check
/// <reference types="@actions/github-script" />

/**
 * emit_outcome_spans.cjs
 *
 * Reads per-item outcome evaluations from /tmp/gh-aw/outcome-evaluations.jsonl
 * and the fleet summary from /tmp/gh-aw/outcome-summary.json, then emits OTLP
 * spans for each evaluated outcome item plus one fleet summary span.
 *
 * Designed to run as a pre-agent step in the outcome-collector workflow so that
 * outcome data is available in OTEL dashboards (Grafana, Datadog, Honeycomb)
 * for per-workflow, per-type, and per-result drill-down.
 *
 * Span naming:
 *   - Per-item:  gh-aw.outcome.evaluation
 *   - Summary:   gh-aw.outcome.summary
 *
 * Errors are non-fatal: export failures must never break the workflow.
 */

const fs = require("fs");
const { nowMs } = require("./performance_now.cjs");

const {
  generateTraceId,
  generateSpanId,
  buildAttr,
  buildOTLPBatchPayload,
  buildOTLPSpan,
  buildGitHubActionsResourceAttributes,
  parseOTLPEndpoints,
  sendOTLPToAllEndpoints,
  appendToOTLPJSONL,
  readJSONIfExists,
} = require("./send_otlp_span.cjs");

const AW_INFO_PATH = "/tmp/gh-aw/aw_info.json";
const EVALUATIONS_PATH = "/tmp/gh-aw/outcome-evaluations.jsonl";
const SUMMARY_PATH = "/tmp/gh-aw/outcome-summary.json";

/**
 * Read a JSONL file, returning an array of parsed objects.
 * Silently skips invalid lines.
 * @param {string} filePath
 * @returns {object[]}
 */
function readJSONL(filePath) {
  try {
    const lines = fs.readFileSync(filePath, "utf8").split("\n");
    /** @type {object[]} */
    const items = [];
    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      try {
        items.push(JSON.parse(trimmed));
      } catch {
        // skip invalid lines
      }
    }
    return items;
  } catch {
    return [];
  }
}

async function main() {
  const evaluations = readJSONL(EVALUATIONS_PATH);
  const summary = readJSONIfExists(SUMMARY_PATH);
  const awInfo = readJSONIfExists(AW_INFO_PATH) || {};

  if (evaluations.length === 0 && (!summary || summary.total_outcomes === 0)) {
    console.log("[outcome-otel] No outcome evaluations to export");
    return;
  }

  const endpoints = parseOTLPEndpoints();
  const hasEndpoints = endpoints.length > 0;

  if (!hasEndpoints) {
    console.log("[outcome-otel] No OTLP endpoints configured, writing JSONL mirror only");
  }

  // Read aw_info.json first: GH_AW_INFO_VERSION and GH_AW_INFO_STAGED are only
  // present during setup, while aw_info.json is the authoritative runtime
  // source for later github-script steps. Prefer agent_version when available
  // to match the other OTEL helpers' service/scope version attribution.
  const staged = awInfo.staged === true || process.env.GH_AW_INFO_STAGED === "true";
  const scopeVersion = awInfo.agent_version || awInfo.version || process.env.GH_AW_INFO_VERSION || "unknown";
  const traceId = (process.env.GITHUB_AW_OTEL_TRACE_ID || "").trim().toLowerCase() || generateTraceId();
  const parentSpanId = (process.env.GITHUB_AW_OTEL_PARENT_SPAN_ID || "").trim().toLowerCase() || "";
  const summarySpanId = generateSpanId();
  const serviceName = process.env.OTEL_SERVICE_NAME || "gh-aw";
  const repository = process.env.GITHUB_REPOSITORY || "";
  const runId = process.env.GITHUB_RUN_ID || "";
  const runAttempt = process.env.GITHUB_RUN_ATTEMPT || "1";
  const eventName = process.env.GITHUB_EVENT_NAME || "";
  const ref = process.env.GITHUB_REF || "";
  const refName = process.env.GITHUB_REF_NAME || "";
  const sha = process.env.GITHUB_SHA || "";
  const job = process.env.GITHUB_JOB || "";
  const workflowRef = process.env.GITHUB_WORKFLOW_REF || "";

  const resourceAttributes = buildGitHubActionsResourceAttributes({
    repository,
    runId,
    eventName,
    ref,
    refName,
    sha,
    job,
    workflowRef,
    staged,
    runAttempt,
  });

  const evalEndMs = nowMs();

  // -------------------------------------------------------------------------
  // Per-item outcome spans
  // -------------------------------------------------------------------------
  const itemSpans = [];

  for (const eval_ of evaluations) {
    const type = typeof eval_.type === "string" ? eval_.type : "";
    const result = typeof eval_.result === "string" ? eval_.result : "unknown";
    const detail = typeof eval_.detail === "string" ? eval_.detail : "";
    const workflow = typeof eval_.workflow === "string" ? eval_.workflow : "";
    const sourceRunId = typeof eval_.run_id === "number" ? eval_.run_id : 0;
    const url = typeof eval_.url === "string" ? eval_.url : "";
    const repo = typeof eval_.repo === "string" ? eval_.repo : "";
    const timestamp = typeof eval_.timestamp === "string" ? eval_.timestamp : "";

    const attributes = [
      buildAttr("gh-aw.exporter.name", "outcome-collector"),
      buildAttr("gh-aw.outcome.type", type),
      buildAttr("gh-aw.outcome.result", result),
      buildAttr("gh-aw.outcome.workflow", workflow),
      buildAttr("gh-aw.outcome.run_id", sourceRunId),
      buildAttr("gh-aw.outcome.repo", repo),
    ];

    if (url) attributes.push(buildAttr("gh-aw.outcome.url", url));
    if (detail) attributes.push(buildAttr("gh-aw.outcome.detail", detail));
    if (timestamp) attributes.push(buildAttr("gh-aw.outcome.created_at", timestamp));

    // Map result to OTLP status: accepted=OK, rejected=ERROR, pending/ignored=UNSET
    const statusCode = result === "rejected" ? 2 : result === "accepted" ? 1 : 0;

    itemSpans.push(
      buildOTLPSpan({
        traceId,
        spanId: generateSpanId(),
        parentSpanId: summarySpanId,
        spanName: "gh-aw.outcome.evaluation",
        startMs: evalEndMs - 1, // point-in-time span
        endMs: evalEndMs,
        attributes,
        statusCode,
      })
    );
  }

  // -------------------------------------------------------------------------
  // Fleet summary span
  // -------------------------------------------------------------------------
  function getSummaryNumber(field, fallback) {
    return summary && typeof summary[field] === "number" ? summary[field] : fallback;
  }

  const summaryAttributes = [
    buildAttr("gh-aw.exporter.name", "outcome-collector"),
    buildAttr("gh-aw.outcome.runs_checked", getSummaryNumber("runs_checked", 0)),
    buildAttr("gh-aw.outcome.total", getSummaryNumber("total_outcomes", evaluations.length)),
    buildAttr("gh-aw.outcome.accepted", getSummaryNumber("accepted", 0)),
    buildAttr("gh-aw.outcome.rejected", getSummaryNumber("rejected", 0)),
    buildAttr("gh-aw.outcome.ignored", getSummaryNumber("ignored", 0)),
    buildAttr("gh-aw.outcome.pending", getSummaryNumber("pending", 0)),
    buildAttr("gh-aw.outcome.acceptance_rate", getSummaryNumber("acceptance_rate", 0)),
    buildAttr("gh-aw.outcome.waste_rate", getSummaryNumber("waste_rate", 0)),
    buildAttr("gh-aw.outcome.item_count", evaluations.length),
  ];

  if (summary && summary.date) {
    summaryAttributes.push(buildAttr("gh-aw.outcome.date", summary.date));
  }

  // Distinct workflows evaluated
  const workflows = [...new Set(evaluations.map(e => e.workflow).filter(Boolean))].sort();
  if (workflows.length > 0) {
    summaryAttributes.push(buildAttr("gh-aw.outcome.workflows", workflows.join(",")));
  }

  // Distinct outcome types seen
  const types = [...new Set(evaluations.map(e => e.type).filter(Boolean))].sort();
  if (types.length > 0) {
    summaryAttributes.push(buildAttr("gh-aw.outcome.types", types.join(",")));
  }

  const summarySpan = buildOTLPSpan({
    traceId,
    spanId: summarySpanId,
    parentSpanId: parentSpanId || undefined,
    spanName: "gh-aw.outcome.summary",
    startMs: evalEndMs - 100, // span covers the evaluation window
    endMs: evalEndMs,
    attributes: summaryAttributes,
    statusCode: 1, // OK
  });

  // -------------------------------------------------------------------------
  // Batch and send
  // -------------------------------------------------------------------------
  const allSpans = [summarySpan, ...itemSpans];

  const payload = buildOTLPBatchPayload({
    serviceName,
    scopeVersion,
    resourceAttributes,
    spans: allSpans,
  });

  // Always write to local JSONL mirror
  appendToOTLPJSONL(payload);

  console.log(`[outcome-otel] Emitting ${evaluations.length} outcome span(s) + 1 summary span`);

  if (hasEndpoints) {
    await sendOTLPToAllEndpoints(endpoints, payload, { skipJSONL: true });
    console.log(`[outcome-otel] Exported to ${endpoints.length} endpoint(s)`);
  }
}

if (require.main === module) {
  main().catch(err => {
    // Non-fatal: OTEL export failures must never break the workflow
    console.warn(`[outcome-otel] Export failed (non-fatal): ${err.message || err}`);
  });
}

module.exports = {
  main,
};
