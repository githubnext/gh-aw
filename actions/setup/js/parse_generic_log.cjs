// @ts-check
/// <reference types="@actions/github-script" />

const { createEngineLogParser } = require("./log_parser_shared.cjs");

const main = createEngineLogParser({
  parserName: "Generic",
  parseFunction: parseGenericLog,
  supportsDirectories: false,
});

/**
 * Parses generic engine log content and converts it to markdown format
 * @param {string} logContent - The raw log content as a string
 * @returns {{markdown: string, mcpFailures: string[], maxTurnsHit: boolean, logEntries: Array}} Result with formatted markdown content, MCP failure list, max-turns status, and parsed log entries
 */
function parseGenericLog(logContent) {
  // For generic logs that don't have a specific format, just return the content as a code block
  const markdown = `## Agent Log

\`\`\`
${logContent}
\`\`\`
`;

  return {
    markdown,
    mcpFailures: [],
    maxTurnsHit: false,
    logEntries: [],
  };
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    main,
    parseGenericLog,
  };
}
