// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_PARSE } = require("./error_codes.cjs");
const { parseTokenUsageJsonl, generateTokenUsageSummary } = require("./parse_mcp_gateway_log.cjs");

/**
 * Parses the firewall proxy token-usage.jsonl and appends a collapsible markdown
 * table to $GITHUB_STEP_SUMMARY via core.summary.addDetails.
 *
 * Also writes aggregated token totals to /tmp/gh-aw/agent_usage.json so the data
 * is bundled in the agent artifact and accessible to third-party tools.
 */

const TOKEN_USAGE_AUDIT_PATH = "/tmp/gh-aw/sandbox/firewall-audit-logs/api-proxy-logs/token-usage.jsonl";
const TOKEN_USAGE_PATH = "/tmp/gh-aw/sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl";
const TOKEN_USAGE_PATHS = [TOKEN_USAGE_AUDIT_PATH, TOKEN_USAGE_PATH];
const AGENT_USAGE_PATH = "/tmp/gh-aw/agent_usage.json";

/**
 * Returns readable, non-empty token usage files, skipping paths that error.
 * @param {string[]} paths
 * @returns {string[]}
 */
function getReadableTokenUsagePaths(paths) {
  const readablePaths = [];
  for (const path of paths) {
    try {
      if (!fs.existsSync(path)) continue;
      const stat = fs.statSync(path);
      if (!stat || stat.size <= 0) continue;
      readablePaths.push(path);
    } catch (error) {
      core.warning(`Skipping token usage path ${path}: ${getErrorMessage(error)}`);
    }
  }
  return readablePaths;
}

/**
 * Extracts request_id with lightweight matching (no full JSON parse).
 * @param {string} line
 * @returns {string}
 */
function extractRequestId(line) {
  const match = line.match(/"request_id"\s*:\s*"((?:\\.|[^"\\])*)"/);
  return match ? match[1] : "";
}

/**
 * Reads token usage files and deduplicates overlapping lines by request_id.
 * Falls back to raw line dedupe when request_id is absent.
 * @param {string[]} paths
 * @returns {string}
 */
function readDedupedTokenUsage(paths) {
  const uniqueLineKeys = new Set();
  const dedupedLines = [];

  for (const path of paths) {
    let fileContent = "";
    try {
      fileContent = fs.readFileSync(path, "utf8");
    } catch (error) {
      core.warning(`Skipping unreadable token usage file ${path}: ${getErrorMessage(error)}`);
      continue;
    }

    for (const line of fileContent.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      const requestId = extractRequestId(trimmed);
      const dedupeKey = requestId ? `request_id:${requestId}` : trimmed;
      if (uniqueLineKeys.has(dedupeKey)) continue;
      uniqueLineKeys.add(dedupeKey);
      dedupedLines.push(trimmed);
    }
  }

  return dedupedLines.join("\n");
}

/**
 * Main function to parse token usage and write the step summary.
 */
async function main() {
  try {
    const tokenUsagePaths = getReadableTokenUsagePaths(TOKEN_USAGE_PATHS);
    if (tokenUsagePaths.length === 0) {
      core.info("No token usage data found, skipping summary");
      return;
    }

    const content = readDedupedTokenUsage(tokenUsagePaths);
    core.info(`Parsing token usage from ${tokenUsagePaths.length} file(s): ${tokenUsagePaths.join(", ")} (${content.length} bytes)`);

    const summary = parseTokenUsageJsonl(content);
    if (!summary || summary.totalRequests === 0) {
      core.info("Token usage file contained no valid entries");
      return;
    }

    const markdown = generateTokenUsageSummary(summary);
    if (markdown.length > 0) {
      core.summary.addDetails("Token Usage", "\n\n" + markdown);
    }

    await core.summary.write();
    core.info("Token usage summary appended to step summary");

    // Write agent_usage.json so the aggregated totals are bundled in the agent
    // artifact and accessible to third-party tools without parsing the step summary.
    const effectiveTokens = Math.round(summary.totalEffectiveTokens || 0);

    // Determine the primary model: the one with the highest effective tokens.
    // This is the actual model name from the API call logs, which may differ from
    // GH_AW_ENGINE_MODEL when the user specified a model alias (e.g. "agent").
    let primaryModel = "";
    let primaryModelET = -1;
    for (const [model, usage] of Object.entries(summary.byModel || {})) {
      if (model !== "unknown" && usage && typeof usage.effectiveTokens === "number" && usage.effectiveTokens > primaryModelET) {
        primaryModelET = usage.effectiveTokens;
        primaryModel = model;
      }
    }

    const agentUsage = {
      input_tokens: summary.totalInputTokens,
      output_tokens: summary.totalOutputTokens,
      cache_read_tokens: summary.totalCacheReadTokens,
      cache_write_tokens: summary.totalCacheWriteTokens,
      effective_tokens: effectiveTokens,
      ...(primaryModel ? { primary_model: primaryModel } : {}),
    };
    fs.writeFileSync(AGENT_USAGE_PATH, JSON.stringify(agentUsage) + "\n");

    if (effectiveTokens > 0) {
      // Export as env var so messages_footer.cjs can read GH_AW_EFFECTIVE_TOKENS,
      // and as a step output so it can flow to downstream jobs.
      core.exportVariable("GH_AW_EFFECTIVE_TOKENS", String(effectiveTokens));
      core.setOutput("effective_tokens", String(effectiveTokens));
      core.info(`Effective tokens: ${effectiveTokens}`);
    }
  } catch (error) {
    core.setFailed(`${ERR_PARSE}: ${getErrorMessage(error)}`);
  }
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    main,
    getReadableTokenUsagePaths,
    extractRequestId,
    readDedupedTokenUsage,
    TOKEN_USAGE_AUDIT_PATH,
    TOKEN_USAGE_PATH,
    TOKEN_USAGE_PATHS,
    AGENT_USAGE_PATH,
  };
}

// Run main if called directly
if (require.main === module) {
  main();
}
