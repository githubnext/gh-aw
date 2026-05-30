// @ts-check
/// <reference types="@actions/github-script" />

const { createEngineLogParser, generateInformationSection } = require("./log_parser_shared.cjs");

const main = createEngineLogParser({
  parserName: "Antigravity",
  parseFunction: parseAntigravityLog,
  supportsDirectories: false,
});

/**
 * Parse Antigravity CLI stream-json log output and format as markdown.
 * Antigravity CLI emits one JSON object per line (JSONL) with the following structure:
 * - Each line contains an accumulated response up to that point:
 *   {"response": "<accumulated text>", "stats": {"models": {...}, "tools": {...}}}
 * - Each new line supersedes the previous (the response field grows incrementally).
 * - The last valid JSON line contains the complete final response and final stats.
 *
 * Stats structure:
 * - stats.models: map of model name → {input_tokens, output_tokens}
 * - stats.tools: map of tool name → call count
 *
 * @param {string} logContent - The raw log content to parse
 * @returns {{markdown: string, logEntries: Array, mcpFailures: Array<string>, maxTurnsHit: boolean}} Parsed log data
 */
function parseAntigravityLog(logContent) {
  if (!logContent) {
    return {
      markdown: "## 🤖 Antigravity\n\nNo log content provided.\n\n",
      logEntries: [],
      mcpFailures: [],
      maxTurnsHit: false,
    };
  }

  /** @type {Array<{response: string, stats: any}>} */
  const parsedLines = [];
  for (const line of logContent.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || !trimmed.startsWith("{")) {
      continue;
    }
    try {
      const parsed = JSON.parse(trimmed);
      if (parsed && typeof parsed.response === "string") {
        parsedLines.push(parsed);
      }
    } catch (_e) {
      // Skip non-JSON lines
    }
  }

  if (parsedLines.length === 0) {
    return {
      markdown: "## 🤖 Antigravity\n\nLog format not recognized as Antigravity stream-json.\n\n",
      logEntries: [],
      mcpFailures: [],
      maxTurnsHit: false,
    };
  }

  // The last valid JSON line contains the complete final response and stats
  const lastEntry = parsedLines[parsedLines.length - 1];
  const finalResponse = lastEntry.response || "";
  const stats = lastEntry.stats || {};

  // Build markdown output
  let markdown = "## 🤖 Antigravity\n\n";

  if (finalResponse.trim()) {
    markdown += finalResponse.trim() + "\n\n";
  }

  // Compute aggregated token usage from all models
  let totalInputTokens = 0;
  let totalOutputTokens = 0;
  if (stats.models && typeof stats.models === "object") {
    for (const modelStats of Object.values(stats.models)) {
      if (modelStats && typeof modelStats === "object") {
        const { input_tokens = 0, output_tokens = 0 } = /** @type {any} */ modelStats;
        totalInputTokens += input_tokens;
        totalOutputTokens += output_tokens;
      }
    }
  }

  // Build a synthetic entry compatible with generateInformationSection
  const syntheticEntry =
    totalInputTokens > 0 || totalOutputTokens > 0
      ? {
          usage: {
            input_tokens: totalInputTokens,
            output_tokens: totalOutputTokens,
          },
          duration_ms: 0,
          num_turns: finalResponse.trim() ? 1 : 0,
        }
      : null;

  markdown += generateInformationSection(syntheticEntry);

  // Build logEntries for compatibility with createEngineLogParser contract
  /** @type {Array<any>} */
  const logEntries = [];
  if (finalResponse.trim()) {
    logEntries.push({
      type: "assistant",
      message: {
        content: [{ type: "text", text: finalResponse.trim() }],
      },
    });
  }

  return {
    markdown,
    logEntries,
    mcpFailures: [],
    maxTurnsHit: false,
  };
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    main,
    parseAntigravityLog,
  };
}
