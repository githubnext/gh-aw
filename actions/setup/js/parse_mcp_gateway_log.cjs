// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Parses MCP gateway logs and creates a step summary
 * Log file location: /tmp/gh-aw/mcp-logs/gateway/stderr.log
 */

/**
 * Main function to parse and display MCP gateway logs
 */
async function main() {
  try {
    // Get the MCP gateway log file path
    const gatewayLogPath = "/tmp/gh-aw/mcp-logs/gateway/stderr.log";

    if (!fs.existsSync(gatewayLogPath)) {
      core.info(`No MCP gateway log found at: ${gatewayLogPath}`);
      return;
    }

    // Read the log file
    const logContent = fs.readFileSync(gatewayLogPath, "utf8");

    if (!logContent || logContent.trim().length === 0) {
      core.info("MCP gateway log file is empty");
      return;
    }

    core.info(`Found MCP gateway log (${logContent.length} bytes)`);

    // Generate step summary
    const summary = generateGatewayLogSummary(logContent);
    core.summary.addRaw(summary).write();

    core.info("MCP gateway log summary added to step summary");
  } catch (error) {
    core.setFailed(getErrorMessage(error));
  }
}

/**
 * Generates a markdown summary of MCP gateway logs
 * @param {string} logContent - The raw log content
 * @returns {string} Markdown summary
 */
function generateGatewayLogSummary(logContent) {
  const summary = [];

  // Wrap entire section in a details tag
  summary.push("<details>");
  summary.push("<summary>MCP Gateway Log</summary>\n");

  // Add the log content in a code fence
  summary.push("```");
  summary.push(logContent.trim());
  summary.push("```");

  // Close the details tag
  summary.push("\n</details>");

  return summary.join("\n");
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    main,
    generateGatewayLogSummary,
  };
}

// Run main if called directly
if (require.main === module) {
  main();
}
