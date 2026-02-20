// @ts-check
/// <reference types="@actions/github-script" />

const { createEngineLogParser } = require("./log_parser_shared.cjs");

const main = createEngineLogParser({
  parserName: "Gemini",
  parseFunction: parseGeminiLog,
  supportsDirectories: false,
});

/**
 * Parse Gemini CLI streaming JSON log output and format as markdown.
 * Gemini CLI outputs one JSON object per line when using --output-format stream-json (JSONL).
 * @param {string} logContent - The raw log content to parse
 * @returns {{markdown: string, logEntries: Array, mcpFailures: Array<string>, maxTurnsHit: boolean}} Parsed log data
 */
function parseGeminiLog(logContent) {
  if (!logContent) {
    return {
      markdown: "## 🤖 Gemini\n\nNo log content provided.\n\n",
      logEntries: [],
      mcpFailures: [],
      maxTurnsHit: false,
    };
  }

  let markdown = "";
  let totalInputTokens = 0;
  let totalOutputTokens = 0;
  let lastResponse = "";

  const lines = logContent.split("\n");
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }

    // Try to parse each line as a JSON object (Gemini --output-format json output)
    try {
      const parsed = JSON.parse(trimmed);

      if (parsed.response) {
        lastResponse = parsed.response;
      }

      // Aggregate token usage from stats
      if (parsed.stats && parsed.stats.models) {
        for (const modelStats of Object.values(parsed.stats.models)) {
          if (modelStats && typeof modelStats === "object") {
            if (typeof modelStats.input_tokens === "number") {
              totalInputTokens += modelStats.input_tokens;
            }
            if (typeof modelStats.output_tokens === "number") {
              totalOutputTokens += modelStats.output_tokens;
            }
          }
        }
      }
    } catch (_e) {
      // Not JSON - skip non-JSON lines
    }
  }

  // Build markdown output
  if (lastResponse) {
    markdown += "## 🤖 Reasoning\n\n";
    markdown += lastResponse + "\n\n";
  }

  markdown += "## 📊 Information\n\n";
  const totalTokens = totalInputTokens + totalOutputTokens;
  if (totalTokens > 0) {
    markdown += `**Total Tokens Used:** ${totalTokens.toLocaleString()}\n\n`;
    if (totalInputTokens > 0) {
      markdown += `**Input Tokens:** ${totalInputTokens.toLocaleString()}\n\n`;
    }
    if (totalOutputTokens > 0) {
      markdown += `**Output Tokens:** ${totalOutputTokens.toLocaleString()}\n\n`;
    }
  }

  return {
    markdown,
    logEntries: [],
    mcpFailures: [],
    maxTurnsHit: false,
  };
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    main,
    parseGeminiLog,
  };
}
