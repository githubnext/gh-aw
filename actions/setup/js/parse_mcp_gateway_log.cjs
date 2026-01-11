// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Parses MCP gateway logs and creates a step summary
 * Log file locations:
 *  - /tmp/gh-aw/mcp-logs/gateway.md (markdown summary from gateway, preferred)
 *  - /tmp/gh-aw/mcp-logs/gateway.log (main gateway log, fallback)
 *  - /tmp/gh-aw/mcp-logs/stderr.log (stderr output, fallback)
 */

/**
 * Main function to parse and display MCP gateway logs
 */
async function main() {
  try {
    const gatewayMdPath = "/tmp/gh-aw/mcp-logs/gateway.md";
    const gatewayLogPath = "/tmp/gh-aw/mcp-logs/gateway.log";
    const stderrLogPath = "/tmp/gh-aw/mcp-logs/stderr.log";

    // First, try to read gateway.md if it exists
    if (fs.existsSync(gatewayMdPath)) {
      const gatewayMdContent = fs.readFileSync(gatewayMdPath, "utf8");
      if (gatewayMdContent && gatewayMdContent.trim().length > 0) {
        core.info(`Found gateway.md (${gatewayMdContent.length} bytes)`);
        // Write the markdown directly to the step summary
        core.summary.addRaw(gatewayMdContent).write();
        core.info("MCP gateway markdown summary added to step summary");
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

    // If neither log file has content, nothing to do
    if ((!gatewayLogContent || gatewayLogContent.trim().length === 0) && (!stderrLogContent || stderrLogContent.trim().length === 0)) {
      core.info("MCP gateway log files are empty or missing");
      return;
    }

    // Generate step summary for both logs
    const summary = generateGatewayLogSummary(gatewayLogContent, stderrLogContent);
    core.summary.addRaw(summary).write();

    core.info("MCP gateway log summary added to step summary");
  } catch (error) {
    core.setFailed(getErrorMessage(error));
  }
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
  };
}

// Run main if called directly
if (require.main === module) {
  main();
}
