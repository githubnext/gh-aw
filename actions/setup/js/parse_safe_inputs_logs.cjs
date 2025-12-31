// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Parses safe-inputs MCP server logs and creates a step summary
 * Log format: [timestamp] [server-name] message
 */

/**
 * Main function to parse and display safe-inputs logs
 */
async function main() {
  try {
    // Get the safe-inputs logs directory path
    const safeInputsLogsDir = `/tmp/gh-aw/safe-inputs/logs/`;

    if (!fs.existsSync(safeInputsLogsDir)) {
      core.info(`No safe-inputs logs directory found at: ${safeInputsLogsDir}`);
      return;
    }

    // Find all log files
    const files = fs.readdirSync(safeInputsLogsDir).filter(file => file.endsWith(".log"));

    if (files.length === 0) {
      core.info(`No safe-inputs log files found in: ${safeInputsLogsDir}`);
      return;
    }

    core.info(`Found ${files.length} safe-inputs log file(s)`);

    // Parse all log files and aggregate results
    const allLogEntries = [];

    for (const file of files) {
      const filePath = path.join(safeInputsLogsDir, file);
      core.info(`Parsing safe-inputs log: ${file}`);

      const content = fs.readFileSync(filePath, "utf8");
      const lines = content.split("\n").filter(line => line.trim());

      for (const line of lines) {
        const entry = parseSafeInputsLogLine(line);
        if (entry) {
          allLogEntries.push(entry);
        }
      }
    }

    if (allLogEntries.length === 0) {
      core.info("No parseable log entries found in safe-inputs logs");
      return;
    }

    // Generate plain text summary for core.info (Copilot CLI style)
    const plainTextSummary = generatePlainTextSummary(allLogEntries);
    core.info(plainTextSummary);

    // Generate step summary
    const summary = generateSafeInputsSummary(allLogEntries);
    core.summary.addRaw(summary).write();
  } catch (error) {
    core.setFailed(getErrorMessage(error));
  }
}

/**
 * Parses a single safe-inputs log line
 * Expected format: [timestamp] [server-name] message
 * @param {string} line - Log line to parse
 * @returns {Object|null} Parsed log entry or null if invalid
 */
function parseSafeInputsLogLine(line) {
  // Match format: [timestamp] [server-name] message
  const match = line.match(/^\[([^\]]+)\]\s+\[([^\]]+)\]\s+(.+)$/);

  if (!match) {
    // Return unparsed line as-is for display
    return {
      timestamp: null,
      serverName: null,
      message: line.trim(),
      raw: true,
    };
  }

  const [, timestamp, serverName, message] = match;

  return {
    timestamp: timestamp.trim(),
    serverName: serverName.trim(),
    message: message.trim(),
    raw: false,
  };
}

/**
 * Generates a lightweight plain text summary optimized for console output.
 * This is designed for core.info output, similar to agent logs style.
 *
 * @param {Array<Object>} logEntries - Parsed log entries
 * @returns {string} Plain text summary for console output
 */
function generatePlainTextSummary(logEntries) {
  const lines = [];

  // Header
  lines.push("=== Safe Inputs MCP Server Logs ===");
  lines.push("");

  // Count events by type
  const eventCounts = {
    startup: 0,
    toolRegistration: 0,
    toolExecution: 0,
    errors: 0,
    other: 0,
  };

  const errors = [];
  const toolCalls = [];

  for (const entry of logEntries) {
    const msg = entry.message.toLowerCase();

    // Categorize log entries
    if (msg.includes("starting safe inputs") || msg.includes("server started")) {
      eventCounts.startup++;
    } else if (msg.includes("registering tool") || msg.includes("tool registration")) {
      eventCounts.toolRegistration++;
    } else if (msg.includes("calling handler") || msg.includes("handler returned")) {
      eventCounts.toolExecution++;
      if (msg.includes("calling handler")) {
        // Extract tool name from message like "Calling handler for tool: my-tool"
        const toolMatch = entry.message.match(/tool:\s*(\S+)/i);
        if (toolMatch) {
          toolCalls.push({
            tool: toolMatch[1],
            timestamp: entry.timestamp,
          });
        }
      }
    } else if (msg.includes("error") || msg.includes("failed")) {
      eventCounts.errors++;
      errors.push(entry);
    } else {
      eventCounts.other++;
    }
  }

  // Log events summary
  lines.push("Log Events:");
  lines.push(`  Total entries: ${logEntries.length}`);
  lines.push(`  Startup events: ${eventCounts.startup}`);
  lines.push(`  Tool registrations: ${eventCounts.toolRegistration}`);
  lines.push(`  Tool executions: ${eventCounts.toolExecution}`);
  if (eventCounts.errors > 0) {
    lines.push(`  Errors: ${eventCounts.errors}`);
  }
  lines.push("");

  // Tool execution details
  if (toolCalls.length > 0) {
    lines.push("Tool Executions:");
    for (const call of toolCalls) {
      const time = call.timestamp ? new Date(call.timestamp).toLocaleTimeString() : "N/A";
      lines.push(`  ✓ ${time} - ${call.tool}`);
    }
    lines.push("");
  }

  // Errors (if any)
  if (errors.length > 0) {
    lines.push("Errors:");
    for (const error of errors) {
      const time = error.timestamp ? `[${error.timestamp}]` : "";
      const server = error.serverName ? `[${error.serverName}]` : "";
      lines.push(`  ✗ ${time} ${server} ${error.message}`);
    }
    lines.push("");
  }

  // Full logs section (limited to first 5000 lines)
  lines.push("Full Logs (first 5000 lines):");
  lines.push("");

  let lineCount = 0;
  for (const entry of logEntries) {
    if (lineCount >= 5000) {
      lines.push(`... (truncated, showing first 5000 lines of ${logEntries.length} total entries)`);
      break;
    }

    if (entry.raw) {
      // Display unparsed lines as-is
      lines.push(entry.message);
    } else {
      const server = entry.serverName ? `[${entry.serverName}]` : "";
      lines.push(`${server} ${entry.message}`.trim());
    }
    lineCount++;
  }

  return lines.join("\n");
}

/**
 * Generates a markdown summary of safe-inputs logs
 * @param {Array<Object>} logEntries - Parsed log entries
 * @returns {string} Markdown summary
 */
function generateSafeInputsSummary(logEntries) {
  const summary = [];

  // Count events by type
  const eventCounts = {
    startup: 0,
    toolRegistration: 0,
    toolExecution: 0,
    errors: 0,
    other: 0,
  };

  const errors = [];
  const toolCalls = [];

  for (const entry of logEntries) {
    const msg = entry.message.toLowerCase();

    // Categorize log entries
    if (msg.includes("starting safe inputs") || msg.includes("server started")) {
      eventCounts.startup++;
    } else if (msg.includes("registering tool") || msg.includes("tool registration")) {
      eventCounts.toolRegistration++;
    } else if (msg.includes("calling handler") || msg.includes("handler returned")) {
      eventCounts.toolExecution++;
      if (msg.includes("calling handler")) {
        // Extract tool name from message like "Calling handler for tool: my-tool"
        const toolMatch = entry.message.match(/tool:\s*(\S+)/i);
        if (toolMatch) {
          toolCalls.push({
            tool: toolMatch[1],
            timestamp: entry.timestamp,
          });
        }
      }
    } else if (msg.includes("error") || msg.includes("failed")) {
      eventCounts.errors++;
      errors.push(entry);
    } else {
      eventCounts.other++;
    }
  }

  // Wrap entire section in a details tag
  summary.push("<details>");
  summary.push("<summary>Safe Inputs MCP Server Logs</summary>\n");

  // Statistics
  summary.push("**Statistics**\n");
  summary.push("| Metric | Count |");
  summary.push("|--------|-------|");
  summary.push(`| Total Log Entries | ${logEntries.length} |`);
  summary.push(`| Startup Events | ${eventCounts.startup} |`);
  summary.push(`| Tool Registrations | ${eventCounts.toolRegistration} |`);
  summary.push(`| Tool Executions | ${eventCounts.toolExecution} |`);
  summary.push(`| Errors | ${eventCounts.errors} |`);
  summary.push(`| Other Events | ${eventCounts.other} |`);
  summary.push("");

  // Tool execution details (if any)
  if (toolCalls.length > 0) {
    summary.push("**Tool Executions**\n");
    summary.push("<details>");
    summary.push("<summary>View tool execution details</summary>\n");
    summary.push("| Time | Tool Name |");
    summary.push("|------|-----------|");
    for (const call of toolCalls) {
      const time = call.timestamp ? new Date(call.timestamp).toLocaleTimeString() : "N/A";
      summary.push(`| ${time} | \`${call.tool}\` |`);
    }
    summary.push("\n</details>\n");
  }

  // Errors (if any)
  if (errors.length > 0) {
    summary.push("**Errors**\n");
    summary.push("<details>");
    summary.push("<summary>View error details</summary>\n");
    summary.push("```");
    for (const error of errors) {
      const time = error.timestamp ? `[${error.timestamp}]` : "";
      const server = error.serverName ? `[${error.serverName}]` : "";
      summary.push(`${time} ${server} ${error.message}`);
    }
    summary.push("```");
    summary.push("\n</details>\n");
  }

  // Full log details (collapsed by default)
  summary.push("**Full Logs**\n");
  summary.push("<details>");
  summary.push("<summary>View full safe-inputs logs</summary>\n");
  summary.push("```");
  for (const entry of logEntries) {
    if (entry.raw) {
      // Display unparsed lines as-is
      summary.push(entry.message);
    } else {
      const time = entry.timestamp ? `[${entry.timestamp}]` : "";
      const server = entry.serverName ? `[${entry.serverName}]` : "";
      summary.push(`${time} ${server} ${entry.message}`);
    }
  }
  summary.push("```");
  summary.push("</details>");

  // Close the outer details tag
  summary.push("\n</details>");

  return summary.join("\n");
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    main,
    parseSafeInputsLogLine,
    generateSafeInputsSummary,
    generatePlainTextSummary,
  };
}

// Run main if called directly
if (require.main === module) {
  main();
}
