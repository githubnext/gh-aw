// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");

const AW_INFO_PATH = "/tmp/gh-aw/aw_info.json";
const AGENT_OUTPUT_PATH = "/tmp/gh-aw/agent_output.json";
const OTLP_EXPORT_ERRORS_PATH = "/tmp/gh-aw/otlp-export-errors.count";
const OTLP_EXPORT_ERROR_DETAILS_PATH = "/tmp/gh-aw/otlp-export-errors.jsonl";
const gatewayEventPaths = ["/tmp/gh-aw/mcp-logs/gateway.jsonl", "/tmp/gh-aw/mcp-logs/rpc-messages.jsonl"];

function readJSONIfExists(path) {
  if (!fs.existsSync(path)) {
    return null;
  }

  try {
    return JSON.parse(fs.readFileSync(path, "utf8"));
  } catch {
    return null;
  }
}

function countBlockedRequests() {
  let total = 0;

  for (const path of gatewayEventPaths) {
    if (!fs.existsSync(path)) {
      continue;
    }

    const lines = fs.readFileSync(path, "utf8").split("\n");
    for (const raw of lines) {
      const line = raw.trim();
      if (!line) continue;
      try {
        const entry = JSON.parse(line);
        if (entry && entry.type === "DIFC_FILTERED") total++;
      } catch {
        // skip malformed lines
      }
    }
  }

  return total;
}

function uniqueCreatedItemTypes(items) {
  const types = new Set();

  for (const item of items) {
    if (item && typeof item.type === "string" && item.type.trim() !== "") {
      types.add(item.type);
    }
  }

  return [...types].sort();
}

function readOTLPExportErrorCount() {
  if (!fs.existsSync(OTLP_EXPORT_ERRORS_PATH)) {
    return 0;
  }

  try {
    const raw = fs.readFileSync(OTLP_EXPORT_ERRORS_PATH, "utf8").trim();
    const parsed = parseInt(raw, 10);
    return Number.isInteger(parsed) && parsed > 0 ? parsed : 0;
  } catch {
    return 0;
  }
}

function readOTLPExportErrorDetails() {
  if (!fs.existsSync(OTLP_EXPORT_ERROR_DETAILS_PATH)) {
    return [];
  }

  const details = [];
  try {
    for (const rawLine of fs.readFileSync(OTLP_EXPORT_ERROR_DETAILS_PATH, "utf8").split("\n")) {
      const line = rawLine.trim();
      if (!line) continue;
      try {
        const entry = JSON.parse(line);
        if (!entry || typeof entry !== "object") continue;
        const host = typeof entry.host === "string" ? entry.host.trim() : "";
        const reason = typeof entry.reason === "string" ? entry.reason.trim() : "";
        const status = Number.isInteger(entry.status) && entry.status > 0 ? entry.status : undefined;
        if (!host || !reason) continue;
        details.push(`${host}${status ? ` status=${status}` : ""} reason=${reason}`);
      } catch {
        // Ignore malformed JSONL lines.
      }
    }
  } catch {
    return [];
  }

  return details;
}

function collectObservabilityData() {
  const awInfo = readJSONIfExists(AW_INFO_PATH) || {};
  const agentOutput = readJSONIfExists(AGENT_OUTPUT_PATH) || { items: [], errors: [] };
  const items = Array.isArray(agentOutput.items) ? agentOutput.items : [];
  const errors = Array.isArray(agentOutput.errors) ? agentOutput.errors : [];
  // Prefer GITHUB_AW_OTEL_TRACE_ID (written to GITHUB_ENV by action_setup_otlp.cjs)
  // so the summary always shows the trace ID that is actually present in the OTLP backend.
  // Fall back to context.otel_trace_id for cross-workflow traces propagated from a parent.
  // Do NOT fall back to workflow_call_id — it is not a valid OTLP trace ID.
  const traceId = process.env.GITHUB_AW_OTEL_TRACE_ID || (awInfo.context ? awInfo.context.otel_trace_id || "" : "");

  return {
    workflowName: awInfo.workflow_name || "",
    engineId: awInfo.engine_id || "",
    traceId,
    staged: awInfo.staged === true,
    firewallEnabled: awInfo.firewall_enabled === true,
    createdItemCount: items.length,
    createdItemTypes: uniqueCreatedItemTypes(items),
    outputErrorCount: errors.length,
    blockedRequests: countBlockedRequests(),
    otlpExportErrors: readOTLPExportErrorCount(),
    otlpExportErrorDetails: readOTLPExportErrorDetails(),
  };
}

function buildObservabilitySummary(data) {
  const posture = data.createdItemCount > 0 ? "write-capable" : "read-only";
  const lines = [];

  lines.push("<details>");
  lines.push("<summary>Observability</summary>");
  lines.push("");

  if (data.workflowName) {
    lines.push(`- **workflow**: ${data.workflowName}`);
  }
  if (data.engineId) {
    lines.push(`- **engine**: ${data.engineId}`);
  }
  if (data.traceId) {
    lines.push(`- **trace id**: ${data.traceId}`);
  }

  lines.push(`- **posture**: ${posture}`);
  lines.push(`- **created items**: ${data.createdItemCount}`);
  lines.push(`- **blocked requests**: ${data.blockedRequests}`);
  lines.push(`- **agent output errors**: ${data.outputErrorCount}`);
  lines.push(`- **otlp export errors**: ${data.otlpExportErrors}`);
  lines.push(`- **firewall enabled**: ${data.firewallEnabled}`);
  lines.push(`- **staged**: ${data.staged}`);

  if (data.otlpExportErrors > 0) {
    lines.push("- ⚠️ OTLP export failures detected; telemetry may not be visible in the backend.");
  }

  if (data.createdItemTypes.length > 0) {
    lines.push("- **item types**:");
    for (const itemType of data.createdItemTypes) {
      lines.push(`  - ${itemType}`);
    }
  }

  if (data.otlpExportErrorDetails.length > 0) {
    lines.push("- **otlp export failure details**:");
    for (const detail of data.otlpExportErrorDetails) {
      lines.push(`  - ${detail}`);
    }
  }

  lines.push("");
  lines.push("</details>");

  return lines.join("\n") + "\n";
}

async function main(core) {
  const data = collectObservabilityData();
  const markdown = buildObservabilitySummary(data);
  await core.summary.addRaw(markdown).write();
  core.info("Generated observability summary in step summary");
}

module.exports = {
  buildObservabilitySummary,
  collectObservabilityData,
  main,
};
