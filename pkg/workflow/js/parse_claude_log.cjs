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
      formatToolCallback: formatToolUse,
      formatInitCallback: initEntry => {
        const result = formatInitializationSummary(initEntry);
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

/**
 * Formats initialization information from system init entry
 * @param {any} initEntry - The system init entry containing tools, mcp_servers, etc.
 * @returns {{markdown: string, mcpFailures: string[]}} Result with formatted markdown string and MCP failure list
 */
function formatInitializationSummary(initEntry) {
  let markdown = "";
  const mcpFailures = [];

  // Display model and session info
  if (initEntry.model) {
    markdown += `**Model:** ${initEntry.model}\n\n`;
  }

  if (initEntry.session_id) {
    markdown += `**Session ID:** ${initEntry.session_id}\n\n`;
  }

  if (initEntry.cwd) {
    // Show a cleaner path by removing common prefixes
    const cleanCwd = initEntry.cwd.replace(/^\/home\/runner\/work\/[^\/]+\/[^\/]+/, ".");
    markdown += `**Working Directory:** ${cleanCwd}\n\n`;
  }

  // Display MCP servers status
  if (initEntry.mcp_servers && Array.isArray(initEntry.mcp_servers)) {
    markdown += "**MCP Servers:**\n";
    for (const server of initEntry.mcp_servers) {
      const statusIcon = server.status === "connected" ? "✅" : server.status === "failed" ? "❌" : "❓";
      markdown += `- ${statusIcon} ${server.name} (${server.status})\n`;

      // Track failed MCP servers and display detailed error information
      if (server.status === "failed") {
        mcpFailures.push(server.name);

        // Display error details if available
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

        // Display error details with proper indentation
        if (errorDetails.length > 0) {
          for (const detail of errorDetails) {
            markdown += `  - ${detail}\n`;
          }
        }
      }
    }
    markdown += "\n";
  }

  // Display tools by category
  if (initEntry.tools && Array.isArray(initEntry.tools)) {
    markdown += "**Available Tools:**\n";

    // Categorize tools
    /** @type {{ [key: string]: string[] }} */
    const categories = {
      Core: [],
      "File Operations": [],
      "Git/GitHub": [],
      MCP: [],
      Other: [],
    };

    for (const tool of initEntry.tools) {
      if (["Task", "Bash", "BashOutput", "KillBash", "ExitPlanMode"].includes(tool)) {
        categories["Core"].push(tool);
      } else if (["Read", "Edit", "MultiEdit", "Write", "LS", "Grep", "Glob", "NotebookEdit"].includes(tool)) {
        categories["File Operations"].push(tool);
      } else if (tool.startsWith("mcp__github__")) {
        categories["Git/GitHub"].push(formatMcpName(tool));
      } else if (tool.startsWith("mcp__") || ["ListMcpResourcesTool", "ReadMcpResourceTool"].includes(tool)) {
        categories["MCP"].push(tool.startsWith("mcp__") ? formatMcpName(tool) : tool);
      } else {
        categories["Other"].push(tool);
      }
    }

    // Display categories with tools
    for (const [category, tools] of Object.entries(categories)) {
      if (tools.length > 0) {
        markdown += `- **${category}:** ${tools.length} tools\n`;
        // Show all tools for complete visibility
        markdown += `  - ${tools.join(", ")}\n`;
      }
    }
    markdown += "\n";
  }

  // Display slash commands if available
  if (initEntry.slash_commands && Array.isArray(initEntry.slash_commands)) {
    const commandCount = initEntry.slash_commands.length;
    markdown += `**Slash Commands:** ${commandCount} available\n`;
    if (commandCount <= 10) {
      markdown += `- ${initEntry.slash_commands.join(", ")}\n`;
    } else {
      markdown += `- ${initEntry.slash_commands.slice(0, 5).join(", ")}, and ${commandCount - 5} more\n`;
    }
    markdown += "\n";
  }

  return { markdown, mcpFailures };
}

/**
 * Formats a tool use entry with its result into markdown
 * @param {any} toolUse - The tool use object containing name, input, etc.
 * @param {any} toolResult - The corresponding tool result object
 * @returns {string} Formatted markdown string
 */
function formatToolUse(toolUse, toolResult) {
  const toolName = toolUse.name;
  const input = toolUse.input || {};

  // Skip TodoWrite except the very last one (we'll handle this separately)
  if (toolName === "TodoWrite") {
    return ""; // Skip for now, would need global context to find the last one
  }

  // Helper function to determine status icon
  function getStatusIcon() {
    if (toolResult) {
      return toolResult.is_error === true ? "❌" : "✅";
    }
    return "❓"; // Unknown by default
  }

  const statusIcon = getStatusIcon();
  let summary = "";
  let details = "";

  // Get tool output from result
  if (toolResult && toolResult.content) {
    if (typeof toolResult.content === "string") {
      details = toolResult.content;
    } else if (Array.isArray(toolResult.content)) {
      details = toolResult.content.map(c => (typeof c === "string" ? c : c.text || "")).join("\n");
    }
  }

  // Calculate token estimate from input + output
  const inputText = JSON.stringify(input);
  const outputText = details;
  const totalTokens = estimateTokens(inputText) + estimateTokens(outputText);

  // Format metadata (duration and tokens)
  let metadata = "";
  if (toolResult && toolResult.duration_ms) {
    metadata += ` <code>${formatDuration(toolResult.duration_ms)}</code>`;
  }
  if (totalTokens > 0) {
    metadata += ` <code>~${totalTokens}t</code>`;
  }

  switch (toolName) {
    case "Bash":
      const command = input.command || "";
      const description = input.description || "";

      // Format the command to be single line
      const formattedCommand = formatBashCommand(command);

      if (description) {
        summary = `${statusIcon} ${description}: <code>${formattedCommand}</code>${metadata}`;
      } else {
        summary = `${statusIcon} <code>${formattedCommand}</code>${metadata}`;
      }
      break;

    case "Read":
      const filePath = input.file_path || input.path || "";
      const relativePath = filePath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, ""); // Remove /home/runner/work/repo/repo/ prefix
      summary = `${statusIcon} Read <code>${relativePath}</code>${metadata}`;
      break;

    case "Write":
    case "Edit":
    case "MultiEdit":
      const writeFilePath = input.file_path || input.path || "";
      const writeRelativePath = writeFilePath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, "");
      summary = `${statusIcon} Write <code>${writeRelativePath}</code>${metadata}`;
      break;

    case "Grep":
    case "Glob":
      const query = input.query || input.pattern || "";
      summary = `${statusIcon} Search for <code>${truncateString(query, 80)}</code>${metadata}`;
      break;

    case "LS":
      const lsPath = input.path || "";
      const lsRelativePath = lsPath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, "");
      summary = `${statusIcon} LS: ${lsRelativePath || lsPath}${metadata}`;
      break;

    default:
      // Handle MCP calls and other tools
      if (toolName.startsWith("mcp__")) {
        const mcpName = formatMcpName(toolName);
        const params = formatMcpParameters(input);
        summary = `${statusIcon} ${mcpName}(${params})${metadata}`;
      } else {
        // Generic tool formatting - show the tool name and main parameters
        const keys = Object.keys(input);
        if (keys.length > 0) {
          // Try to find the most important parameter
          const mainParam = keys.find(k => ["query", "command", "path", "file_path", "content"].includes(k)) || keys[0];
          const value = String(input[mainParam] || "");

          if (value) {
            summary = `${statusIcon} ${toolName}: ${truncateString(value, 100)}${metadata}`;
          } else {
            summary = `${statusIcon} ${toolName}${metadata}`;
          }
        } else {
          summary = `${statusIcon} ${toolName}${metadata}`;
        }
      }
  }

  // Format with HTML details tag if we have output
  if (details && details.trim()) {
    // Truncate details if too long
    const maxDetailsLength = 500;
    const truncatedDetails = details.length > maxDetailsLength ? details.substring(0, maxDetailsLength) + "..." : details;
    return `<details>\n<summary>${summary}</summary>\n\n\`\`\`\`\`\n${truncatedDetails}\n\`\`\`\`\`\n</details>\n\n`;
  } else {
    // No details, just show summary
    return `${summary}\n\n`;
  }
}

/**
 * Formats MCP parameters into a human-readable string
 * @param {Record<string, any>} input - The input object containing parameters
 * @returns {string} Formatted parameters string
 */
function formatMcpParameters(input) {
  const keys = Object.keys(input);
  if (keys.length === 0) return "";

  const paramStrs = [];
  for (const key of keys.slice(0, 4)) {
    // Show up to 4 parameters
    const value = String(input[key] || "");
    paramStrs.push(`${key}: ${truncateString(value, 40)}`);
  }

  if (keys.length > 4) {
    paramStrs.push("...");
  }

  return paramStrs.join(", ");
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    parseClaudeLog,
    formatToolUse,
    formatInitializationSummary,
    formatBashCommand,
    truncateString,
    estimateTokens,
    formatDuration,
  };
}

main();
