// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");

const AW_INFO_PATH = "/tmp/gh-aw/aw_info.json";
const AGENT_OUTPUT_PATH = "/tmp/gh-aw/agent_output.json";
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

function readContextString(value) {
  return typeof value === "string" ? value.trim() : "";
}

function collectObservabilityData() {
  const awInfo = readJSONIfExists(AW_INFO_PATH) || {};
  const context = awInfo.context && typeof awInfo.context === "object" ? awInfo.context : {};
  const agentOutput = readJSONIfExists(AGENT_OUTPUT_PATH) || { items: [], errors: [] };
  const items = Array.isArray(agentOutput.items) ? agentOutput.items : [];
  const errors = Array.isArray(agentOutput.errors) ? agentOutput.errors : [];
  // Prefer GITHUB_AW_OTEL_TRACE_ID (written to GITHUB_ENV by action_setup_otlp.cjs)
  // so the summary always shows the trace ID that is actually present in the OTLP backend.
  // Fall back to context.otel_trace_id for cross-workflow traces propagated from a parent.
  // Do NOT fall back to workflow_call_id — it is not a valid OTLP trace ID.
  const traceId = process.env.GITHUB_AW_OTEL_TRACE_ID || readContextString(context.otel_trace_id);

  return {
    workflowName: awInfo.workflow_name || "",
    engineId: awInfo.engine_id || "",
    traceId,
    episodeId: readContextString(context.episode_id) || readContextString(context.workflow_call_id),
    hopId: readContextString(context.hop_id) || readContextString(context.workflow_call_id),
    parentHopId: readContextString(context.parent_hop_id),
    originEvent: readContextString(context.origin_event) || readContextString(context.event_type),
    rootRepo: readContextString(context.root_repo) || readContextString(context.repo),
    rootWorkflowId: readContextString(context.root_workflow_id) || readContextString(context.workflow_id),
    staged: awInfo.staged === true,
    firewallEnabled: awInfo.firewall_enabled === true,
    createdItemCount: items.length,
    createdItemTypes: uniqueCreatedItemTypes(items),
    outputErrorCount: errors.length,
    blockedRequests: countBlockedRequests(),
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
  if (data.episodeId) {
    lines.push(`- **episode**: ${data.episodeId}`);
  }
  if (data.hopId) {
    lines.push(`- **hop**: ${data.hopId}`);
  }
  if (data.parentHopId) {
    lines.push(`- **parent hop**: ${data.parentHopId}`);
  }
  if (data.originEvent) {
    lines.push(`- **origin event**: ${data.originEvent}`);
  }
  if (data.rootRepo) {
    lines.push(`- **root repo**: ${data.rootRepo}`);
  }
  if (data.rootWorkflowId) {
    lines.push(`- **root workflow**: ${data.rootWorkflowId}`);
  }

  lines.push(`- **posture**: ${posture}`);
  lines.push(`- **created items**: ${data.createdItemCount}`);
  lines.push(`- **blocked requests**: ${data.blockedRequests}`);
  lines.push(`- **agent output errors**: ${data.outputErrorCount}`);
  lines.push(`- **firewall enabled**: ${data.firewallEnabled}`);
  lines.push(`- **staged**: ${data.staged}`);

  if (data.createdItemTypes.length > 0) {
    lines.push("- **item types**:");
    for (const itemType of data.createdItemTypes) {
      lines.push(`  - ${itemType}`);
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
