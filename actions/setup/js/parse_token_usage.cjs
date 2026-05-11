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
 * Main function to parse token usage and write the step summary.
 */
async function main() {
  try {
    const tokenUsagePaths = TOKEN_USAGE_PATHS.filter(path => fs.existsSync(path) && fs.statSync(path).size > 0);
    if (tokenUsagePaths.length === 0) {
      core.info("No token usage data found, skipping summary");
      return;
    }

    const uniqueLineKeys = new Set();
    const dedupedLines = [];
    for (const path of tokenUsagePaths) {
      const fileContent = fs.readFileSync(path, "utf8");
      const lines = fileContent.split("\n");
      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed) continue;
        let dedupeKey = trimmed;
        let normalizedLine = trimmed;
        try {
          const parsed = JSON.parse(trimmed);
          if (parsed && typeof parsed === "object") {
            dedupeKey = typeof parsed.request_id === "string" && parsed.request_id.length > 0 ? `request_id:${parsed.request_id}` : trimmed;
            normalizedLine = JSON.stringify(parsed);
          }
        } catch {
          // Keep raw line as-is; parseTokenUsageJsonl will skip malformed entries.
        }
        if (uniqueLineKeys.has(dedupeKey)) continue;
        uniqueLineKeys.add(dedupeKey);
        dedupedLines.push(normalizedLine);
      }
    }
    const content = dedupedLines.join("\n");
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
    const agentUsage = {
      input_tokens: summary.totalInputTokens,
      output_tokens: summary.totalOutputTokens,
      cache_read_tokens: summary.totalCacheReadTokens,
      cache_write_tokens: summary.totalCacheWriteTokens,
      effective_tokens: effectiveTokens,
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
