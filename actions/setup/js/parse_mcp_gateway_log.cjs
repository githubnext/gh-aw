// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { displayDirectories } = require("./display_file_helpers.cjs");
const { ERR_PARSE } = require("./error_codes.cjs");

/**
 * Parses MCP gateway logs and creates a step summary
 * Log file locations:
 *  - /tmp/gh-aw/mcp-logs/gateway.jsonl (structured JSONL log, parsed for DIFC_FILTERED events)
 *  - /tmp/gh-aw/mcp-logs/gateway.md (markdown summary from gateway, preferred for general content)
 *  - /tmp/gh-aw/mcp-logs/gateway.log (main gateway log, fallback)
 *  - /tmp/gh-aw/mcp-logs/stderr.log (stderr output, fallback)
 */

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
      resource = event.description || "-";
    }
    lines.push(`| ${time} | ${server} | ${tool} | ${reason} | ${user} | ${resource} |`);
  }

  lines.push("");
  lines.push("</details>\n");
  return lines.join("\n");
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
    if (fs.existsSync(gatewayJsonlPath)) {
      const jsonlContent = fs.readFileSync(gatewayJsonlPath, "utf8");
      core.info(`Found gateway.jsonl (${jsonlContent.length} bytes)`);
      difcFilteredEvents = parseGatewayJsonlForDifcFiltered(jsonlContent);
      if (difcFilteredEvents.length > 0) {
        core.info(`Found ${difcFilteredEvents.length} DIFC_FILTERED event(s) in gateway.jsonl`);
      }
    } else if (fs.existsSync(rpcMessagesPath)) {
      const jsonlContent = fs.readFileSync(rpcMessagesPath, "utf8");
      core.info(`No gateway.jsonl found; scanning rpc-messages.jsonl (${jsonlContent.length} bytes) for DIFC_FILTERED events`);
      difcFilteredEvents = parseGatewayJsonlForDifcFiltered(jsonlContent);
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

        core.summary.write();
        return;
      }
    } else {
      core.info(`No gateway.md found at: ${gatewayMdPath}, falling back to log files`);
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

    // If no legacy log content and no DIFC events, nothing to do
    if ((!gatewayLogContent || gatewayLogContent.trim().length === 0) && (!stderrLogContent || stderrLogContent.trim().length === 0) && difcFilteredEvents.length === 0) {
      core.info("MCP gateway log files are empty or missing");
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
      core.summary.addRaw(fullSummary).write();
    }
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
    printAllGatewayFiles,
  };
}

// Run main if called directly
if (require.main === module) {
  main();
}
