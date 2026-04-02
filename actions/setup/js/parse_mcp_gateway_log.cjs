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
  lines.push("### 📊 Token Usage\n");
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

  return lines.join("\n") + "\n";
}

/**
 * Appends the token usage section to the step summary if data is present, then writes it.
 * Also exports GH_AW_EFFECTIVE_TOKENS as a GitHub Actions environment variable so
 * subsequent steps can display the ET value in generated footers.
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
      const markdown = generateTokenUsageSummary(parsedSummary);
      if (markdown.length > 0) {
        coreObj.summary.addRaw(markdown);
      }
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
  lines.push(`<summary><b>🔒 DIFC Filtered Events (${filteredEvents.length})</b></summary>\n`);
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

  // Tool calls / requests table
  if (requests.length > 0) {
    const blockedNote = blockedCount > 0 ? `, ${blockedCount} blocked` : "";
    const callLines = [];
    callLines.push("<details>");
    callLines.push(`<summary><b>MCP Gateway Activity (${requests.length} request${requests.length !== 1 ? "s" : ""}${blockedNote})</b></summary>\n`);
    callLines.push("");
    callLines.push("| Time | Server | Tool / Method |");
    callLines.push("|------|--------|---------------|");

    for (const req of requests) {
      const time = req.timestamp ? req.timestamp.replace("T", " ").replace(/\.\d+Z$/, "Z") : "-";
      const server = req.server_id || "-";
      const label = getRpcRequestLabel(req);
      callLines.push(`| ${time} | ${server} | \`${label}\` |`);
    }

    callLines.push("");
    callLines.push("</details>\n");
    parts.push(callLines.join("\n"));
  } else if (blockedCount > 0) {
    // No requests, but there are DIFC_FILTERED events — add a minimal header
    parts.push(`<details>\n<summary><b>MCP Gateway Activity (${blockedCount} blocked)</b></summary>\n\n*All tool calls were blocked by the integrity filter.*\n\n</details>\n`);
  }

  // Other message types (not REQUEST, RESPONSE, DIFC_FILTERED)
  if (other.length > 0) {
    /** @type {Record<string, number>} */
    const typeCounts = {};
    for (const entry of other) {
      typeCounts[entry.type] = (typeCounts[entry.type] || 0) + 1;
    }
    const otherLines = Object.entries(typeCounts).map(([type, count]) => `- **${type}**: ${count} message${count !== 1 ? "s" : ""}`);
    parts.push("<details>\n<summary><b>Other Gateway Messages</b></summary>\n\n" + otherLines.join("\n") + "\n\n</details>\n");
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
    summary.push("<summary><b>MCP Gateway Log (gateway.log)</b></summary>\n");
    summary.push("```");
    summary.push(gatewayLogContent.trim());
    summary.push("```");
    summary.push("\n</details>\n");
  }

  // Add stderr.log if it has content
  if (stderrLogContent && stderrLogContent.trim().length > 0) {
    summary.push("<details>");
    summary.push("<summary><b>MCP Gateway Log (stderr.log)</b></summary>\n");
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
