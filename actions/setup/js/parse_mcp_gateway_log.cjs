// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { displayDirectories } = require("./display_file_helpers.cjs");
const { ERR_PARSE } = require("./error_codes.cjs");
const { computeEffectiveTokens, getTokenClassWeights, formatET } = require("./effective_tokens.cjs");

/**
 * Parses MCP gateway logs and creates a step summary
 * Log file locations:
 *  - /tmp/gh-aw/mcp-logs/gateway.jsonl (structured JSONL log, parsed for DIFC_FILTERED events)
 *  - /tmp/gh-aw/mcp-logs/gateway.md (markdown summary from gateway, preferred for general content)
 *  - /tmp/gh-aw/mcp-logs/gateway.log (main gateway log, fallback)
 *  - /tmp/gh-aw/mcp-logs/stderr.log (stderr output, fallback)
 *  - /tmp/gh-aw/sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl (token usage from firewall proxy)
 */

const TOKEN_USAGE_PATH = "/tmp/gh-aw/sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl";
const MAX_RPC_SUMMARY_DETAILS_LENGTH = 120;
const MAX_RPC_SUMMARY_GENERIC_LENGTH = 160;
const MAX_RPC_MESSAGE_LABEL_LENGTH = 80;
const TOP_LEVEL_RPC_IGNORED_KEYS = new Set(["timestamp", "direction", "type", "server_id", "payload"]);

/**
 * Formats milliseconds as a human-readable duration string.
 * @param {number} ms - Duration in milliseconds
 * @returns {string} Formatted duration (e.g. "500ms", "2.5s", "1m30s")
 */
function formatDurationMs(ms) {
  if (ms < 1000) return `${ms}ms`;
  const seconds = ms / 1000;
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const minutes = Math.floor(seconds / 60);
  const secs = Math.round(seconds % 60);
  return `${minutes}m${secs}s`;
}

/**
 * Parses token-usage.jsonl content and returns an aggregated summary.
 * Computes effective tokens (ET) per model using the GH_AW_MODEL_MULTIPLIERS env var.
 * @param {string} jsonlContent - The token-usage.jsonl file content
 * @returns {{totalInputTokens: number, totalOutputTokens: number, totalCacheReadTokens: number, totalCacheWriteTokens: number, totalRequests: number, totalDurationMs: number, cacheEfficiency: number, totalEffectiveTokens: number, byModel: Object} | null}
 */
function parseTokenUsageJsonl(jsonlContent) {
  const summary = {
    totalInputTokens: 0,
    totalOutputTokens: 0,
    totalCacheReadTokens: 0,
    totalCacheWriteTokens: 0,
    totalRequests: 0,
    totalDurationMs: 0,
    cacheEfficiency: 0,
    totalEffectiveTokens: 0,
    byModel: {},
  };

  const lines = jsonlContent.split("\n");
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    try {
      const entry = JSON.parse(trimmed);
      if (!entry || typeof entry !== "object") continue;

      const inputTokens = entry.input_tokens || 0;
      const outputTokens = entry.output_tokens || 0;
      const cacheReadTokens = entry.cache_read_tokens || 0;
      const cacheWriteTokens = entry.cache_write_tokens || 0;

      summary.totalInputTokens += inputTokens;
      summary.totalOutputTokens += outputTokens;
      summary.totalCacheReadTokens += cacheReadTokens;
      summary.totalCacheWriteTokens += cacheWriteTokens;
      summary.totalRequests++;
      summary.totalDurationMs += entry.duration_ms || 0;

      const model = entry.model || "unknown";
      summary.byModel[model] ??= {
        provider: entry.provider || "",
        inputTokens: 0,
        outputTokens: 0,
        cacheReadTokens: 0,
        cacheWriteTokens: 0,
        requests: 0,
        durationMs: 0,
        effectiveTokens: 0,
      };
      const m = summary.byModel[model];
      m.inputTokens += inputTokens;
      m.outputTokens += outputTokens;
      m.cacheReadTokens += cacheReadTokens;
      m.cacheWriteTokens += cacheWriteTokens;
      m.requests++;
      m.durationMs += entry.duration_ms || 0;
    } catch {
      // skip malformed lines
    }
  }

  if (summary.totalRequests === 0) return null;

  const totalInputPlusCacheRead = summary.totalInputTokens + summary.totalCacheReadTokens;
  if (totalInputPlusCacheRead > 0) {
    summary.cacheEfficiency = summary.totalCacheReadTokens / totalInputPlusCacheRead;
  }

  // Compute effective tokens per model and aggregate total
  let totalEffectiveTokens = 0;
  for (const [model, usage] of Object.entries(summary.byModel)) {
    const et = computeEffectiveTokens(model, usage.inputTokens, usage.outputTokens, usage.cacheReadTokens, usage.cacheWriteTokens);
    usage.effectiveTokens = et;
    totalEffectiveTokens += et;
  }
  summary.totalEffectiveTokens = totalEffectiveTokens;

  return summary;
}

/**
 * Generates a markdown summary section for token usage data.
 * Includes an Effective Tokens (ET) column per model and a ● ET summary line.
 * @param {{totalInputTokens: number, totalOutputTokens: number, totalCacheReadTokens: number, totalCacheWriteTokens: number, totalRequests: number, totalDurationMs: number, cacheEfficiency: number, totalEffectiveTokens: number, byModel: Object} | null} summary
 * @returns {string} Markdown section, or empty string if no data
 */
function generateTokenUsageSummary(summary) {
  if (!summary || summary.totalRequests === 0) return "";

  const lines = [];
  lines.push("| Model | Input | Output | Cache Read | Cache Write | ET | Requests | Duration |");
  lines.push("|-------|------:|-------:|-----------:|------------:|---:|---------:|---------:|");

  // Sort models by total tokens descending
  const models = Object.entries(summary.byModel).sort(([, a], [, b]) => {
    const aTotal = a.inputTokens + a.outputTokens + a.cacheReadTokens + a.cacheWriteTokens;
    const bTotal = b.inputTokens + b.outputTokens + b.cacheReadTokens + b.cacheWriteTokens;
    return bTotal - aTotal;
  });

  for (const [model, usage] of models) {
    const et = formatET(Math.round(usage.effectiveTokens || 0));
    lines.push(
      `| ${model} | ${usage.inputTokens.toLocaleString()} | ${usage.outputTokens.toLocaleString()} | ${usage.cacheReadTokens.toLocaleString()} | ${usage.cacheWriteTokens.toLocaleString()} | ${et} | ${usage.requests} | ${formatDurationMs(usage.durationMs)} |`
    );
  }

  const totalET = formatET(Math.round(summary.totalEffectiveTokens || 0));
  lines.push(
    `| **Total** | **${summary.totalInputTokens.toLocaleString()}** | **${summary.totalOutputTokens.toLocaleString()}** | **${summary.totalCacheReadTokens.toLocaleString()}** | **${summary.totalCacheWriteTokens.toLocaleString()}** | **${totalET}** | **${summary.totalRequests}** | **${formatDurationMs(summary.totalDurationMs)}** |`
  );

  // Footer line with ET summary using ● symbol and optional cache efficiency
  const footerParts = [];
  if (summary.totalEffectiveTokens > 0) {
    footerParts.push(`● ${formatET(Math.round(summary.totalEffectiveTokens))}`);
  }
  if (summary.cacheEfficiency > 0) {
    footerParts.push(`Cache efficiency: ${(summary.cacheEfficiency * 100).toFixed(1)}%`);
  }
  if (footerParts.length > 0) {
    lines.push(`\n_${footerParts.join(" · ")}_`);
    // Disclose the token class weights used to compute ET (required by the ET spec)
    const w = getTokenClassWeights();
    lines.push(`<sub>ET weights: input=${w.input} · cached_input=${w.cached_input} · output=${w.output} · reasoning=${w.reasoning} · cache_write=${w.cache_write}</sub>`);
  }

  lines.push("");

  return lines.join("\n") + "\n";
}

/**
 * Writes the step summary and exports GH_AW_EFFECTIVE_TOKENS when token usage data exists.
 * Token Usage rendering is handled by parse_token_usage.cjs to avoid duplicate sections.
 * This is the final call in each main() exit path — it consolidates the summary write
 * so callers don't need to chain addRaw() + write() themselves.
 * @param {typeof import('@actions/core')} coreObj - The GitHub Actions core object
 */
function writeStepSummaryWithTokenUsage(coreObj) {
  if (!fs.existsSync(TOKEN_USAGE_PATH)) {
    coreObj.debug(`No token-usage.jsonl found at: ${TOKEN_USAGE_PATH}`);
  } else {
    const content = fs.readFileSync(TOKEN_USAGE_PATH, "utf8");
    if (content?.trim()) {
      coreObj.info(`Found token-usage.jsonl (${content.length} bytes)`);
      const parsedSummary = parseTokenUsageJsonl(content);
      // Export total effective tokens as a GitHub Actions env var for use in
      // generated footers (GH_AW_EFFECTIVE_TOKENS is read by messages_footer.cjs)
      if (parsedSummary && parsedSummary.totalEffectiveTokens > 0) {
        const roundedET = Math.round(parsedSummary.totalEffectiveTokens);
        coreObj.exportVariable("GH_AW_EFFECTIVE_TOKENS", String(roundedET));
        // Also set as a step output so the value can flow to the safe_outputs job
        // via the agent job's effective_tokens output (job-level env vars are not
        // inherited by downstream jobs — only job outputs are).
        coreObj.setOutput("effective_tokens", String(roundedET));
        coreObj.info(`Effective tokens: ${roundedET}`);
      }
    }
  }
  coreObj.summary.write();
}

/**
 * Prints all gateway-related files to core.info for debugging
 */
function printAllGatewayFiles() {
  const gatewayDirs = ["/tmp/gh-aw/mcp-logs"];
  displayDirectories(gatewayDirs, 64 * 1024);
}

/**
 * Parses gateway.jsonl content and extracts DIFC_FILTERED events
 * @param {string} jsonlContent - The gateway.jsonl file content
 * @returns {Array<Object>} Array of DIFC_FILTERED event objects
 */
function parseGatewayJsonlForDifcFiltered(jsonlContent) {
  const filteredEvents = [];
  const lines = jsonlContent.split("\n");
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || !trimmed.includes("DIFC_FILTERED")) continue;
    try {
      const entry = JSON.parse(trimmed);
      if (entry.type === "DIFC_FILTERED") {
        filteredEvents.push(entry);
      }
    } catch {
      // skip malformed lines
    }
  }
  return filteredEvents;
}

/**
 * Generates a markdown summary section for DIFC_FILTERED events
 * @param {Array<Object>} filteredEvents - Array of DIFC_FILTERED event objects
 * @returns {string} Markdown section, or empty string if no events
 */
function generateDifcFilteredSummary(filteredEvents) {
  if (!filteredEvents || filteredEvents.length === 0) return "";

  const lines = [];
  lines.push("<details>");
  lines.push(`<summary>🔒 DIFC Filtered Events (${filteredEvents.length})</summary>\n`);
  lines.push("");
  lines.push("The following tool calls were blocked by DIFC integrity or secrecy checks:\n");
  lines.push("");
  lines.push("| Time | Server | Tool | Reason | User | Resource |");
  lines.push("|------|--------|------|--------|------|----------|");

  for (const event of filteredEvents) {
    const time = event.timestamp ? event.timestamp.replace("T", " ").replace(/\.\d+Z$/, "Z") : "-";
    const server = event.server_id || "-";
    const tool = event.tool_name ? `\`${event.tool_name}\`` : "-";
    const reason = (event.reason || "-").replace(/\n/g, " ").replace(/\|/g, "\\|");
    const user = event.author_login ? `${event.author_login} (${event.author_association || "NONE"})` : "-";
    let resource;
    if (event.html_url) {
      const lastSegment = event.html_url.split("/").filter(Boolean).pop();
      const label = event.number ? `#${event.number}` : lastSegment || event.html_url;
      resource = `[${label}](${event.html_url})`;
    } else {
      const rawDesc = event.description ? event.description.replace(/^[a-z-]+:(?!\/\/)/i, "") : null;
      resource = rawDesc && rawDesc !== "#unknown" ? event.description : "-";
    }
    lines.push(`| ${time} | ${server} | ${tool} | ${reason} | ${user} | ${resource} |`);
  }

  lines.push("");
  lines.push("</details>\n");
  return lines.join("\n");
}

/**
 * Parses rpc-messages.jsonl content and returns entries categorized by type.
 * DIFC_FILTERED entries are excluded here because they are handled separately
 * by parseGatewayJsonlForDifcFiltered.
 * @param {string} jsonlContent - The rpc-messages.jsonl file content
 * @returns {{requests: Array<Object>, responses: Array<Object>, other: Array<Object>}}
 */
function parseRpcMessagesJsonl(jsonlContent) {
  const requests = [];
  const responses = [];
  const other = [];

  const lines = jsonlContent.split("\n");
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    try {
      const entry = JSON.parse(trimmed);
      if (!entry || typeof entry !== "object" || !entry.type) continue;

      if (entry.type === "REQUEST") {
        requests.push(entry);
      } else if (entry.type === "RESPONSE") {
        responses.push(entry);
      } else if (entry.type !== "DIFC_FILTERED") {
        other.push(entry);
      }
    } catch {
      // skip malformed lines
    }
  }

  return { requests, responses, other };
}

/**
 * Extracts a human-readable label for an MCP REQUEST entry.
 * For tools/call requests, returns the tool name; for other methods, returns the method name.
 * @param {Object} entry - REQUEST entry from rpc-messages.jsonl
 * @returns {string} Display label for the request
 */
function getRpcRequestLabel(entry) {
  const payload = entry.payload;
  if (!payload) return "unknown";
  const method = payload.method;
  if (method === "tools/call") {
    const toolName = payload.params && payload.params.name;
    return toolName || method;
  }
  return method || "unknown";
}

/**
 * Formats an rpc-messages timestamp for display in the step summary.
 * @param {string|undefined} timestamp
 * @returns {string}
 */
function formatRpcMessageTime(timestamp) {
  return timestamp ? timestamp.replace("T", " ").replace(/\.\d+Z$/, "Z") : "-";
}

/**
 * Escapes text for safe display inside a markdown table cell.
 * @param {unknown} value
 * @returns {string}
 */
function escapeMarkdownTableCell(value) {
  return String(value ?? "-")
    .replace(/\n/g, " ")
    .replace(/\|/g, "\\|")
    .trim();
}

/**
 * Escapes text for safe use in HTML fragments embedded in markdown.
 * @param {unknown} value
 * @returns {string}
 */
function escapeHtml(value) {
  return String(value ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

/**
 * Truncates a string to a maximum length, appending an ellipsis when needed.
 * @param {unknown} value
 * @param {number} maxLength
 * @returns {string}
 */
function truncateSummaryValue(value, maxLength) {
  const text = String(value);
  if (maxLength <= 0) return "";
  if (text.length <= maxLength) return text;
  if (maxLength < 4) return text.slice(0, maxLength);
  return `${text.slice(0, maxLength - 3)}...`;
}

/**
 * Normalizes an RPC summary label sourced from logs.
 * @param {unknown} value
 * @param {number} maxLength
 * @returns {string}
 */
function normalizeRpcSummaryLabel(value, maxLength = MAX_RPC_MESSAGE_LABEL_LENGTH) {
  return truncateSummaryValue(
    String(value ?? "-")
      .replace(/\s+/g, " ")
      .trim() || "-",
    maxLength
  );
}

/**
 * Formats an RPC label as HTML code for safe use inside markdown tables.
 * @param {unknown} value
 * @returns {string}
 */
function formatRpcInlineCodeLabel(value) {
  return `<code>${escapeHtml(normalizeRpcSummaryLabel(value))}</code>`;
}

/**
 * Summarizes an MCP RESPONSE entry for table rendering.
 * @param {Object} entry
 * @returns {{status: string, details: string}}
 */
function summarizeRpcResponseEntry(entry) {
  const payload = entry.payload && typeof entry.payload === "object" ? entry.payload : {};
  const error = payload.error && typeof payload.error === "object" ? payload.error : null;
  if (error) {
    const code = error.code !== null && error.code !== undefined ? ` ${error.code}` : "";
    const message = truncateSummaryValue(String(error.message || "Unknown error"), MAX_RPC_SUMMARY_DETAILS_LENGTH);
    return {
      status: "error",
      details: `error${code}: ${message}`,
    };
  }

  const result = payload.result;
  if (result && typeof result === "object") {
    if (Array.isArray(result.tools)) {
      return {
        status: "ok",
        details: `${result.tools.length} tool${result.tools.length !== 1 ? "s" : ""}`,
      };
    }

    const keys = Object.keys(result);
    if (keys.length > 0) {
      const shownKeys = keys.slice(0, 3);
      const moreCount = keys.length - shownKeys.length;
      return {
        status: "ok",
        details: `result keys: ${shownKeys.join(", ")}${moreCount > 0 ? ` +${moreCount} more` : ""}`,
      };
    }
  }

  if (result !== undefined) {
    return {
      status: "ok",
      details: truncateSummaryValue(JSON.stringify(result), MAX_RPC_SUMMARY_DETAILS_LENGTH),
    };
  }

  return {
    status: "ok",
    details: "response received",
  };
}

/**
 * Summarizes a non-REQUEST rpc-messages entry for table rendering.
 * @param {Object} entry
 * @returns {string}
 */
function summarizeGenericRpcEntry(entry) {
  const parts = [];
  const pushPart = (key, value) => {
    parts.push(`${key}=${truncateSummaryValue(String(value), MAX_RPC_SUMMARY_GENERIC_LENGTH)}`);
  };

  for (const [key, value] of Object.entries(entry)) {
    if (TOP_LEVEL_RPC_IGNORED_KEYS.has(key) || value === null || value === undefined || typeof value === "object") continue;
    pushPart(key, value);
  }

  const payload = entry.payload && typeof entry.payload === "object" ? entry.payload : null;
  if (payload) {
    if (payload.method) {
      pushPart("method", payload.method);
    }
    if (payload.params && typeof payload.params === "object" && payload.params.name) {
      pushPart("tool", payload.params.name);
    }
    if (payload.id !== null && payload.id !== undefined) {
      pushPart("id", payload.id);
    }
    if (payload.error && typeof payload.error === "object" && payload.error.message) {
      pushPart("error", payload.error.message);
    }
    if (parts.length === 0) {
      const payloadKeys = Object.keys(payload);
      if (payloadKeys.length > 0) {
        pushPart("payload keys", payloadKeys.join(", "));
      }
    }
  }

  if (parts.length === 0) {
    return "-";
  }

  return truncateSummaryValue(parts.join(" · "), MAX_RPC_SUMMARY_GENERIC_LENGTH);
}

/**
 * Builds a markdown table row for the RPC message summary.
 * @param {Array<unknown>} cells
 * @returns {string}
 */
function buildRpcSummaryRow(cells) {
  return `| ${cells.map(cell => escapeMarkdownTableCell(cell)).join(" | ")} |`;
}

/**
 * Generates a markdown step summary for rpc-messages.jsonl entries (mcpg v0.2.0+ format).
 * Shows a table of REQUEST entries (tool calls), a count of RESPONSE entries, any other
 * message types, and the DIFC_FILTERED section if there are blocked events.
 * @param {{requests: Array<Object>, responses: Array<Object>, other: Array<Object>}} entries
 * @param {Array<Object>} difcFilteredEvents - DIFC_FILTERED events parsed separately
 * @returns {string} Markdown summary, or empty string if nothing to show
 */
function generateRpcMessagesSummary(entries, difcFilteredEvents) {
  const { requests, responses, other } = entries;
  const blockedCount = difcFilteredEvents ? difcFilteredEvents.length : 0;
  const totalMessages = requests.length + responses.length + other.length + blockedCount;

  if (totalMessages === 0) return "";

  const parts = [];
  /** @type {Map<string, Array<Object>>} */
  const otherByType = new Map();
  for (const entry of other) {
    const entriesForType = otherByType.get(entry.type) || [];
    entriesForType.push(entry);
    otherByType.set(entry.type, entriesForType);
  }
  const renderedOtherTypes = Array.from(otherByType.keys());

  if (requests.length === 0 && responses.length === 0 && other.length === 0 && blockedCount > 0) {
    // No requests, but there are DIFC_FILTERED events — add a minimal header
    parts.push(`<details>\n<summary>MCP Gateway Activity (${blockedCount} blocked)</summary>\n\n*All tool calls were blocked by the integrity filter.*\n\n</details>\n`);
  } else {
    const summaryParts = [];
    if (requests.length > 0) {
      summaryParts.push(`${requests.length} request${requests.length !== 1 ? "s" : ""}`);
    }
    if (responses.length > 0) {
      summaryParts.push(`${responses.length} response${responses.length !== 1 ? "s" : ""}`);
    }
    for (const type of renderedOtherTypes) {
      const count = otherByType.get(type)?.length || 0;
      summaryParts.push(`${count} ${escapeHtml(normalizeRpcSummaryLabel(type))}`);
    }
    if (blockedCount > 0) {
      summaryParts.push(`${blockedCount} blocked`);
    }

    const callLines = [];
    callLines.push("<details>");
    callLines.push(`<summary>MCP Gateway Activity (${summaryParts.join(", ")})</summary>\n`);
    callLines.push("");

    if (requests.length > 0) {
      callLines.push("#### REQUEST");
      callLines.push("");
      callLines.push("| Time | Server | Tool / Method |");
      callLines.push("|------|--------|---------------|");

      for (const req of requests) {
        const time = formatRpcMessageTime(req.timestamp);
        const server = escapeMarkdownTableCell(req.server_id || "-");
        const label = formatRpcInlineCodeLabel(getRpcRequestLabel(req));
        callLines.push(`| ${time} | ${server} | ${label} |`);
      }

      callLines.push("");
    }

    if (responses.length > 0) {
      callLines.push("#### RESPONSE");
      callLines.push("");
      callLines.push("| Time | Server | Direction | Status | Details |");
      callLines.push("|------|--------|-----------|--------|---------|");

      for (const response of responses) {
        const { status, details } = summarizeRpcResponseEntry(response);
        callLines.push(buildRpcSummaryRow([formatRpcMessageTime(response.timestamp), response.server_id || "-", response.direction || "-", status, details]));
      }

      callLines.push("");
    }

    for (const type of renderedOtherTypes) {
      callLines.push(`#### ${escapeHtml(normalizeRpcSummaryLabel(type))}`);
      callLines.push("");
      callLines.push("| Time | Server | Direction | Details |");
      callLines.push("|------|--------|-----------|---------|");

      for (const entry of otherByType.get(type) || []) {
        callLines.push(buildRpcSummaryRow([formatRpcMessageTime(entry.timestamp), entry.server_id || "-", entry.direction || "-", summarizeGenericRpcEntry(entry)]));
      }

      callLines.push("");
    }

    callLines.push("</details>\n");
    parts.push(callLines.join("\n"));
  }

  // DIFC_FILTERED section (re-uses existing table renderer)
  if (blockedCount > 0) {
    parts.push(generateDifcFilteredSummary(difcFilteredEvents));
  }

  return parts.join("\n");
}

/**
 * Main function to parse and display MCP gateway logs
 */
async function main() {
  try {
    // First, print all gateway-related files for debugging
    printAllGatewayFiles();

    const gatewayJsonlPath = "/tmp/gh-aw/mcp-logs/gateway.jsonl";
    const rpcMessagesPath = "/tmp/gh-aw/mcp-logs/rpc-messages.jsonl";
    const gatewayMdPath = "/tmp/gh-aw/mcp-logs/gateway.md";
    const gatewayLogPath = "/tmp/gh-aw/mcp-logs/gateway.log";
    const stderrLogPath = "/tmp/gh-aw/mcp-logs/stderr.log";

    // Parse DIFC_FILTERED events from gateway.jsonl (preferred) or rpc-messages.jsonl (fallback).
    // Both files use the same JSONL format with DIFC_FILTERED entries interleaved.
    let difcFilteredEvents = [];
    let rpcMessagesContent = null;
    if (fs.existsSync(gatewayJsonlPath)) {
      const jsonlContent = fs.readFileSync(gatewayJsonlPath, "utf8");
      core.info(`Found gateway.jsonl (${jsonlContent.length} bytes)`);
      difcFilteredEvents = parseGatewayJsonlForDifcFiltered(jsonlContent);
      if (difcFilteredEvents.length > 0) {
        core.info(`Found ${difcFilteredEvents.length} DIFC_FILTERED event(s) in gateway.jsonl`);
      }
    } else if (fs.existsSync(rpcMessagesPath)) {
      rpcMessagesContent = fs.readFileSync(rpcMessagesPath, "utf8");
      core.info(`Found rpc-messages.jsonl (${rpcMessagesContent.length} bytes)`);
      difcFilteredEvents = parseGatewayJsonlForDifcFiltered(rpcMessagesContent);
      if (difcFilteredEvents.length > 0) {
        core.info(`Found ${difcFilteredEvents.length} DIFC_FILTERED event(s) in rpc-messages.jsonl`);
      }
    } else {
      core.info(`No gateway.jsonl or rpc-messages.jsonl found for DIFC_FILTERED scanning`);
    }

    // Try to read gateway.md if it exists (preferred for general gateway summary)
    if (fs.existsSync(gatewayMdPath)) {
      const gatewayMdContent = fs.readFileSync(gatewayMdPath, "utf8");
      if (gatewayMdContent && gatewayMdContent.trim().length > 0) {
        core.info(`Found gateway.md (${gatewayMdContent.length} bytes)`);

        // Write the markdown directly to the step summary
        core.summary.addRaw(gatewayMdContent.endsWith("\n") ? gatewayMdContent : gatewayMdContent + "\n");

        // Append DIFC_FILTERED section if any events found
        if (difcFilteredEvents.length > 0) {
          const difcSummary = generateDifcFilteredSummary(difcFilteredEvents);
          core.summary.addRaw(difcSummary);
        }

        writeStepSummaryWithTokenUsage(core);
        return;
      }
    } else {
      core.info(`No gateway.md found at: ${gatewayMdPath}, falling back to log files`);
    }

    // When no gateway.md exists, check if rpc-messages.jsonl is available (mcpg v0.2.0+ unified format).
    // In this format, all message types (REQUEST, RESPONSE, DIFC_FILTERED, etc.) are written to a
    // single rpc-messages.jsonl file instead of separate gateway.md / gateway.log streams.
    if (rpcMessagesContent !== null) {
      const rpcEntries = parseRpcMessagesJsonl(rpcMessagesContent);
      const totalMessages = rpcEntries.requests.length + rpcEntries.responses.length + rpcEntries.other.length;
      core.info(`rpc-messages.jsonl: ${rpcEntries.requests.length} request(s), ${rpcEntries.responses.length} response(s), ${rpcEntries.other.length} other, ${difcFilteredEvents.length} DIFC_FILTERED`);

      if (totalMessages > 0 || difcFilteredEvents.length > 0) {
        const rpcSummary = generateRpcMessagesSummary(rpcEntries, difcFilteredEvents);
        if (rpcSummary.length > 0) {
          core.summary.addRaw(rpcSummary);
        }
      } else {
        core.info("rpc-messages.jsonl is present but contains no renderable messages");
      }
      writeStepSummaryWithTokenUsage(core);
      return;
    }

    // Fallback to legacy log files
    let gatewayLogContent = "";
    let stderrLogContent = "";

    // Read gateway.log if it exists
    if (fs.existsSync(gatewayLogPath)) {
      gatewayLogContent = fs.readFileSync(gatewayLogPath, "utf8");
      core.info(`Found gateway.log (${gatewayLogContent.length} bytes)`);
    } else {
      core.info(`No gateway.log found at: ${gatewayLogPath}`);
    }

    // Read stderr.log if it exists
    if (fs.existsSync(stderrLogPath)) {
      stderrLogContent = fs.readFileSync(stderrLogPath, "utf8");
      core.info(`Found stderr.log (${stderrLogContent.length} bytes)`);
    } else {
      core.info(`No stderr.log found at: ${stderrLogPath}`);
    }

    // If no legacy log content and no DIFC events, check if token usage is available
    if ((!gatewayLogContent || gatewayLogContent.trim().length === 0) && (!stderrLogContent || stderrLogContent.trim().length === 0) && difcFilteredEvents.length === 0) {
      core.info("MCP gateway log files are empty or missing");
      writeStepSummaryWithTokenUsage(core);
      return;
    }

    // Generate plain text summary for core.info
    if ((gatewayLogContent && gatewayLogContent.trim().length > 0) || (stderrLogContent && stderrLogContent.trim().length > 0)) {
      const plainTextSummary = generatePlainTextLegacySummary(gatewayLogContent, stderrLogContent);
      core.info(plainTextSummary);
    }

    // Generate step summary: legacy logs + DIFC filtered section
    const legacySummary = generateGatewayLogSummary(gatewayLogContent, stderrLogContent);
    const difcSummary = generateDifcFilteredSummary(difcFilteredEvents);
    const fullSummary = [legacySummary, difcSummary].filter(s => s.length > 0).join("\n");

    if (fullSummary.length > 0) {
      core.summary.addRaw(fullSummary);
    }
    writeStepSummaryWithTokenUsage(core);
  } catch (error) {
    core.setFailed(`${ERR_PARSE}: ${getErrorMessage(error)}`);
  }
}

/**
 * Generates a plain text summary from gateway.md content for console output
 * @param {string} gatewayMdContent - The gateway.md markdown content
 * @returns {string} Plain text summary for console output
 */
function generatePlainTextGatewaySummary(gatewayMdContent) {
  const lines = [];

  // Header
  lines.push("=== MCP Gateway Logs ===");
  lines.push("");

  // Strip markdown formatting for plain text display
  const plainText = gatewayMdContent
    .replace(/<details>/g, "")
    .replace(/<\/details>/g, "")
    .replace(/<summary>(.*?)<\/summary>/g, "$1")
    .replace(/```[\s\S]*?```/g, match => {
      // Extract content from code blocks
      return match.replace(/```[a-z]*\n?/g, "").replace(/```$/g, "");
    })
    .replace(/\*\*(.*?)\*\*/g, "$1") // Remove bold
    .replace(/\*(.*?)\*/g, "$1") // Remove italic
    .replace(/`(.*?)`/g, "$1") // Remove inline code
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1") // Remove links, keep text
    .replace(/^#+\s+/gm, "") // Remove heading markers
    .replace(/^\|-+.*-+\|$/gm, "") // Remove table separator lines
    .replace(/^\|/gm, "") // Remove leading pipe from table rows
    .replace(/\|$/gm, "") // Remove trailing pipe from table rows
    .replace(/\s*\|\s*/g, " ") // Replace remaining pipes with spaces
    .trim();

  lines.push(plainText);
  lines.push("");

  return lines.join("\n");
}

/**
 * Generates a plain text summary from legacy log files for console output
 * @param {string} gatewayLogContent - The gateway.log content
 * @param {string} stderrLogContent - The stderr.log content
 * @returns {string} Plain text summary for console output
 */
function generatePlainTextLegacySummary(gatewayLogContent, stderrLogContent) {
  const lines = [];

  // Header
  lines.push("=== MCP Gateway Logs ===");
  lines.push("");

  // Add gateway.log if it has content
  if (gatewayLogContent && gatewayLogContent.trim().length > 0) {
    lines.push("Gateway Log (gateway.log):");
    lines.push("");
    lines.push(gatewayLogContent.trim());
    lines.push("");
  }

  // Add stderr.log if it has content
  if (stderrLogContent && stderrLogContent.trim().length > 0) {
    lines.push("Gateway Log (stderr.log):");
    lines.push("");
    lines.push(stderrLogContent.trim());
    lines.push("");
  }

  return lines.join("\n");
}

/**
 * Generates a markdown summary of MCP gateway logs
 * @param {string} gatewayLogContent - The gateway.log content
 * @param {string} stderrLogContent - The stderr.log content
 * @returns {string} Markdown summary
 */
function generateGatewayLogSummary(gatewayLogContent, stderrLogContent) {
  const summary = [];

  // Add gateway.log if it has content
  if (gatewayLogContent && gatewayLogContent.trim().length > 0) {
    summary.push("<details>");
    summary.push("<summary>MCP Gateway Log (gateway.log)</summary>\n");
    summary.push("```");
    summary.push(gatewayLogContent.trim());
    summary.push("```");
    summary.push("\n</details>\n");
  }

  // Add stderr.log if it has content
  if (stderrLogContent && stderrLogContent.trim().length > 0) {
    summary.push("<details>");
    summary.push("<summary>MCP Gateway Log (stderr.log)</summary>\n");
    summary.push("```");
    summary.push(stderrLogContent.trim());
    summary.push("```");
    summary.push("\n</details>");
  }

  return summary.join("\n");
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    main,
    generateGatewayLogSummary,
    generatePlainTextGatewaySummary,
    generatePlainTextLegacySummary,
    parseGatewayJsonlForDifcFiltered,
    generateDifcFilteredSummary,
    parseRpcMessagesJsonl,
    getRpcRequestLabel,
    generateRpcMessagesSummary,
    printAllGatewayFiles,
    parseTokenUsageJsonl,
    generateTokenUsageSummary,
    formatDurationMs,
  };
}

// Run main if called directly
if (require.main === module) {
  main();
}
