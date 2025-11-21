// @ts-check
/// <reference types="@actions/github-script" />

const { runLogParser } = require("./log_parser_bootstrap.cjs");
const {
  formatDuration,
  formatBashCommand,
  truncateString,
  estimateTokens,
  formatMcpName,
  generateConversationMarkdown,
  generateInformationSection,
  formatMcpParameters,
  formatInitializationSummary,
  formatToolUse,
} = require("./log_parser_shared.cjs");

function main() {
  runLogParser({
    parseLog: parseClaudeLog,
    parserName: "Claude",
    supportsDirectories: false,
  });
}

/**
 * Parses Claude log content and converts it to markdown format
 * @param {string} logContent - The raw log content as a string
 * @returns {{markdown: string, mcpFailures: string[], maxTurnsHit: boolean}} Result with formatted markdown content, MCP failure list, and max-turns status
 */
function parseClaudeLog(logContent) {
  try {
    let logEntries;

    // First, try to parse as JSON array (old format)
    try {
      logEntries = JSON.parse(logContent);
      if (!Array.isArray(logEntries)) {
        throw new Error("Not a JSON array");
      }
    } catch (jsonArrayError) {
      // If that fails, try to parse as mixed format (debug logs + JSONL)
      logEntries = [];
      const lines = logContent.split("\n");

      for (const line of lines) {
        const trimmedLine = line.trim();
        if (trimmedLine === "") {
          continue; // Skip empty lines
        }

        // Handle lines that start with [ (JSON array format)
        if (trimmedLine.startsWith("[{")) {
          try {
            const arrayEntries = JSON.parse(trimmedLine);
            if (Array.isArray(arrayEntries)) {
              logEntries.push(...arrayEntries);
              continue;
            }
          } catch (arrayParseError) {
            // Skip invalid array lines
            continue;
          }
        }

        // Skip debug log lines that don't start with {
        // (these are typically timestamped debug messages)
        if (!trimmedLine.startsWith("{")) {
          continue;
        }

        // Try to parse each line as JSON
        try {
          const jsonEntry = JSON.parse(trimmedLine);
          logEntries.push(jsonEntry);
        } catch (jsonLineError) {
          // Skip invalid JSON lines (could be partial debug output)
          continue;
        }
      }
    }

    if (!Array.isArray(logEntries) || logEntries.length === 0) {
      return {
        markdown: "## Agent Log Summary\n\nLog format not recognized as Claude JSON array or JSONL.\n",
        mcpFailures: [],
        maxTurnsHit: false,
      };
    }

    const mcpFailures = [];

    // Generate conversation markdown using shared function
    const conversationResult = generateConversationMarkdown(logEntries, {
      formatToolCallback: (toolUse, toolResult) => formatToolUse(toolUse, toolResult, { includeDetailedParameters: false }),
      formatInitCallback: initEntry => {
        const result = formatInitializationSummary(initEntry, {
          includeSlashCommands: true,
          mcpFailureCallback: server => {
            // Display detailed error information for failed MCP servers (Claude-specific)
            const errorDetails = [];

            if (server.error) {
              errorDetails.push(`**Error:** ${server.error}`);
            }

            if (server.stderr) {
              // Truncate stderr if too long
              const maxStderrLength = 500;
              const stderr = server.stderr.length > maxStderrLength ? server.stderr.substring(0, maxStderrLength) + "..." : server.stderr;
              errorDetails.push(`**Stderr:** \`${stderr}\``);
            }

            if (server.exitCode !== undefined && server.exitCode !== null) {
              errorDetails.push(`**Exit Code:** ${server.exitCode}`);
            }

            if (server.command) {
              errorDetails.push(`**Command:** \`${server.command}\``);
            }

            if (server.message) {
              errorDetails.push(`**Message:** ${server.message}`);
            }

            if (server.reason) {
              errorDetails.push(`**Reason:** ${server.reason}`);
            }

            // Return formatted error details with proper indentation
            if (errorDetails.length > 0) {
              return errorDetails.map(detail => `  - ${detail}\n`).join("");
            }
            return "";
          },
        });

        // Track MCP failures
        if (result.mcpFailures) {
          mcpFailures.push(...result.mcpFailures);
        }
        return result;
      },
    });

    let markdown = conversationResult.markdown;

    // Add Information section from the last entry with result metadata
    const lastEntry = logEntries[logEntries.length - 1];
    markdown += generateInformationSection(lastEntry);

    // Check if max-turns limit was hit
    let maxTurnsHit = false;
    const maxTurns = process.env.GH_AW_MAX_TURNS;
    if (maxTurns && lastEntry && lastEntry.num_turns) {
      const configuredMaxTurns = parseInt(maxTurns, 10);
      if (!isNaN(configuredMaxTurns) && lastEntry.num_turns >= configuredMaxTurns) {
        maxTurnsHit = true;
      }
    }

    return { markdown, mcpFailures, maxTurnsHit };
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    return {
      markdown: `## Agent Log Summary\n\nError parsing Claude log (tried both JSON array and JSONL formats): ${errorMessage}\n`,
      mcpFailures: [],
      maxTurnsHit: false,
    };
  }
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    parseClaudeLog,
  };
}

main();
