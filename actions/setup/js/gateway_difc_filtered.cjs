// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Gateway DIFC Filtered Module
 *
 * This module handles reading MCP gateway logs and extracting DIFC_FILTERED events
 * for display in AI-generated footers.
 */

const fs = require("fs");
const { GATEWAY_JSONL_PATH, RPC_MESSAGES_PATH } = require("./constants.cjs");

/**
 * Parses JSONL content and extracts DIFC_FILTERED events
 * @param {string} jsonlContent - The JSONL file content
 * @returns {Array<Object>} Array of DIFC_FILTERED event objects
 */
function parseDifcFilteredEvents(jsonlContent) {
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
 * Reads DIFC_FILTERED events from MCP gateway logs.
 *
 * This function checks two possible locations for gateway logs:
 * 1. Path specified by gatewayJsonlPath (or /tmp/gh-aw/mcp-logs/gateway.jsonl by default)
 * 2. Path specified by rpcMessagesPath (or /tmp/gh-aw/mcp-logs/rpc-messages.jsonl as fallback)
 *
 * @param {string} [gatewayJsonlPath] - Path to gateway.jsonl. Defaults to /tmp/gh-aw/mcp-logs/gateway.jsonl
 * @param {string} [rpcMessagesPath] - Path to rpc-messages.jsonl fallback. Defaults to /tmp/gh-aw/mcp-logs/rpc-messages.jsonl
 * @returns {Array<Object>} Array of DIFC_FILTERED event objects
 */
function getDifcFilteredEvents(gatewayJsonlPath, rpcMessagesPath) {
  const jsonlPath = gatewayJsonlPath || GATEWAY_JSONL_PATH;
  const rpcPath = rpcMessagesPath || RPC_MESSAGES_PATH;

  if (fs.existsSync(jsonlPath)) {
    try {
      const content = fs.readFileSync(jsonlPath, "utf8");
      return parseDifcFilteredEvents(content);
    } catch {
      return [];
    }
  }

  if (fs.existsSync(rpcPath)) {
    try {
      const content = fs.readFileSync(rpcPath, "utf8");
      return parseDifcFilteredEvents(content);
    } catch {
      return [];
    }
  }

  return [];
}

/**
 * Generates HTML details/summary section for integrity-filtered items wrapped in a GitHub note alert.
 * @param {Array<Object>} filteredEvents - Array of DIFC_FILTERED event objects
 * @returns {string} GitHub note alert with details section, or empty string if no filtered events
 */
function generateDifcFilteredSection(filteredEvents) {
  if (!filteredEvents || filteredEvents.length === 0) {
    return "";
  }

  // Deduplicate events by their significant fields
  const seen = new Set();
  const uniqueEvents = filteredEvents.filter(event => {
    const key = [event.html_url || "", event.tool_name || "", event.description || "", event.reason || ""].join("|");
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });

  const count = uniqueEvents.length;
  const itemWord = count === 1 ? "item" : "items";

  let section = "\n\n> [!NOTE]\n";
  section += `> <details>\n`;
  section += `> <summary><b>🔒 Integrity filter blocked ${count} ${itemWord}</b></summary>\n`;
  section += `>\n`;
  section += `> The following ${itemWord} were blocked because they don't meet the GitHub integrity level.\n`;
  section += `>\n`;

  const maxItems = 16;
  const visibleEvents = uniqueEvents.slice(0, maxItems);
  const remainingCount = uniqueEvents.length - visibleEvents.length;

  for (const event of visibleEvents) {
    let reference;
    if (event.html_url) {
      const label = event.number ? `#${event.number}` : event.html_url;
      reference = `[${label}](${event.html_url})`;
    } else {
      const desc = event.description ? event.description.replace(/^[a-z-]+:(?!\/\/)/i, "") : null;
      const validDesc = desc && desc !== "#unknown" ? desc : null;
      reference = validDesc || (event.tool_name ? `\`${event.tool_name}\`` : "-");
    }
    const tool = event.tool_name ? `\`${event.tool_name}\`` : "-";
    const reason = (event.reason || "-").replace(/^Resource '[^']*' /, "").replace(/\n/g, " ");
    section += `> - ${reference} ${tool}: ${reason}\n`;
  }

  if (remainingCount > 0) {
    section += `> - ... and ${remainingCount} more ${remainingCount === 1 ? "item" : "items"}\n`;
  }

  section += `>\n`;
  const promptsDir = process.env.GH_AW_PROMPTS_DIR || `${process.env.RUNNER_TEMP}/gh-aw/prompts`;
  const remediationPath = `${promptsDir}/integrity_filter_remediation.md`;
  const remediationText = fs.readFileSync(remediationPath, "utf8");
  for (const line of remediationText.trimEnd().split("\n")) {
    section += line ? `> ${line}\n` : `>\n`;
  }
  section += `>\n`;
  section += `> </details>\n`;

  return section;
}

module.exports = {
  parseDifcFilteredEvents,
  getDifcFilteredEvents,
  generateDifcFilteredSection,
};
