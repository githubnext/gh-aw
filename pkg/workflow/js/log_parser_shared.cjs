// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Shared utility functions for log parsers
 * Used by parse_claude_log.cjs and parse_copilot_log.cjs
 */

/**
 * Formats duration in milliseconds to human-readable string
 * @param {number} ms - Duration in milliseconds
 * @returns {string} Formatted duration string (e.g., "1s", "1m 30s")
 */
function formatDuration(ms) {
  if (!ms || ms <= 0) return "";

  const seconds = Math.round(ms / 1000);
  if (seconds < 60) {
    return `${seconds}s`;
  }

  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  if (remainingSeconds === 0) {
    return `${minutes}m`;
  }
  return `${minutes}m ${remainingSeconds}s`;
}

/**
 * Formats a bash command by normalizing whitespace and escaping
 * @param {string} command - The raw bash command string
 * @returns {string} Formatted and escaped command string
 */
function formatBashCommand(command) {
  if (!command) return "";

  // Convert multi-line commands to single line by replacing newlines with spaces
  // and collapsing multiple spaces
  let formatted = command
    .replace(/\n/g, " ") // Replace newlines with spaces
    .replace(/\r/g, " ") // Replace carriage returns with spaces
    .replace(/\t/g, " ") // Replace tabs with spaces
    .replace(/\s+/g, " ") // Collapse multiple spaces into one
    .trim(); // Remove leading/trailing whitespace

  // Escape backticks to prevent markdown issues
  formatted = formatted.replace(/`/g, "\\`");

  // Truncate if too long (keep reasonable length for summary)
  const maxLength = 300;
  if (formatted.length > maxLength) {
    formatted = formatted.substring(0, maxLength) + "...";
  }

  return formatted;
}

/**
 * Truncates a string to a maximum length with ellipsis
 * @param {string} str - The string to truncate
 * @param {number} maxLength - Maximum allowed length
 * @returns {string} Truncated string with ellipsis if needed
 */
function truncateString(str, maxLength) {
  if (!str) return "";
  if (str.length <= maxLength) return str;
  return str.substring(0, maxLength) + "...";
}

/**
 * Calculates approximate token count from text using 4 chars per token estimate
 * @param {string} text - The text to estimate tokens for
 * @returns {number} Approximate token count
 */
function estimateTokens(text) {
  if (!text) return 0;
  return Math.ceil(text.length / 4);
}

/**
 * Formats MCP tool name from internal format to display format
 * @param {string} toolName - The raw tool name (e.g., mcp__github__search_issues)
 * @returns {string} Formatted tool name (e.g., github::search_issues)
 */
function formatMcpName(toolName) {
  // Convert mcp__github__search_issues to github::search_issues
  if (toolName.startsWith("mcp__")) {
    const parts = toolName.split("__");
    if (parts.length >= 3) {
      const provider = parts[1]; // github, etc.
      const method = parts.slice(2).join("_"); // search_issues, etc.
      return `${provider}::${method}`;
    }
  }
  return toolName;
}

/**
 * Generates markdown summary from conversation log entries
 * This is the core shared logic between Claude and Copilot log parsers
 *
 * @param {Array} logEntries - Array of log entries with type, message, etc.
 * @param {Object} options - Configuration options
 * @param {Function} options.formatToolCallback - Callback function to format tool use (content, toolResult) => string
 * @param {Function} options.formatInitCallback - Callback function to format initialization (initEntry) => string or {markdown: string, mcpFailures: string[]}
 * @returns {{markdown: string, commandSummary: Array<string>}} Generated markdown and command summary
 */
function generateConversationMarkdown(logEntries, options) {
  const { formatToolCallback, formatInitCallback } = options;

  const toolUsePairs = new Map(); // Map tool_use_id to tool_result

  // First pass: collect tool results by tool_use_id
  for (const entry of logEntries) {
    if (entry.type === "user" && entry.message?.content) {
      for (const content of entry.message.content) {
        if (content.type === "tool_result" && content.tool_use_id) {
          toolUsePairs.set(content.tool_use_id, content);
        }
      }
    }
  }

  let markdown = "";

  // Check for initialization data first
  const initEntry = logEntries.find(entry => entry.type === "system" && entry.subtype === "init");

  if (initEntry && formatInitCallback) {
    markdown += "## 🚀 Initialization\n\n";
    const initResult = formatInitCallback(initEntry);
    // Handle both string and object returns (for backward compatibility)
    if (typeof initResult === "string") {
      markdown += initResult;
    } else if (initResult && initResult.markdown) {
      markdown += initResult.markdown;
    }
    markdown += "\n";
  }

  markdown += "\n## 🤖 Reasoning\n\n";

  // Second pass: process assistant messages in sequence
  for (const entry of logEntries) {
    if (entry.type === "assistant" && entry.message?.content) {
      for (const content of entry.message.content) {
        if (content.type === "text" && content.text) {
          // Add reasoning text directly
          const text = content.text.trim();
          if (text && text.length > 0) {
            markdown += text + "\n\n";
          }
        } else if (content.type === "tool_use") {
          // Process tool use with its result
          const toolResult = toolUsePairs.get(content.id);
          const toolMarkdown = formatToolCallback(content, toolResult);
          if (toolMarkdown) {
            markdown += toolMarkdown;
          }
        }
      }
    }
  }

  markdown += "## 🤖 Commands and Tools\n\n";
  const commandSummary = []; // For the succinct summary

  // Collect all tool uses for summary
  for (const entry of logEntries) {
    if (entry.type === "assistant" && entry.message?.content) {
      for (const content of entry.message.content) {
        if (content.type === "tool_use") {
          const toolName = content.name;
          const input = content.input || {};

          // Skip internal tools - only show external commands and API calls
          if (["Read", "Write", "Edit", "MultiEdit", "LS", "Grep", "Glob", "TodoWrite"].includes(toolName)) {
            continue; // Skip internal file operations and searches
          }

          // Find the corresponding tool result to get status
          const toolResult = toolUsePairs.get(content.id);
          let statusIcon = "❓";
          if (toolResult) {
            statusIcon = toolResult.is_error === true ? "❌" : "✅";
          }

          // Add to command summary (only external tools)
          if (toolName === "Bash") {
            const formattedCommand = formatBashCommand(input.command || "");
            commandSummary.push(`* ${statusIcon} \`${formattedCommand}\``);
          } else if (toolName.startsWith("mcp__")) {
            const mcpName = formatMcpName(toolName);
            commandSummary.push(`* ${statusIcon} \`${mcpName}(...)\``);
          } else {
            // Handle other external tools (if any)
            commandSummary.push(`* ${statusIcon} ${toolName}`);
          }
        }
      }
    }
  }

  // Add command summary
  if (commandSummary.length > 0) {
    for (const cmd of commandSummary) {
      markdown += `${cmd}\n`;
    }
  } else {
    markdown += "No commands or tools used.\n";
  }

  return { markdown, commandSummary };
}

/**
 * Generates information section markdown from the last log entry
 * @param {any} lastEntry - The last log entry with metadata (num_turns, duration_ms, etc.)
 * @param {Object} options - Configuration options
 * @param {Function} [options.additionalInfoCallback] - Optional callback for additional info (lastEntry) => string
 * @returns {string} Information section markdown
 */
function generateInformationSection(lastEntry, options = {}) {
  const { additionalInfoCallback } = options;

  let markdown = "\n## 📊 Information\n\n";

  if (!lastEntry) {
    return markdown;
  }

  if (lastEntry.num_turns) {
    markdown += `**Turns:** ${lastEntry.num_turns}\n\n`;
  }

  if (lastEntry.duration_ms) {
    const durationSec = Math.round(lastEntry.duration_ms / 1000);
    const minutes = Math.floor(durationSec / 60);
    const seconds = durationSec % 60;
    markdown += `**Duration:** ${minutes}m ${seconds}s\n\n`;
  }

  if (lastEntry.total_cost_usd) {
    markdown += `**Total Cost:** $${lastEntry.total_cost_usd.toFixed(4)}\n\n`;
  }

  // Call additional info callback if provided (for engine-specific info like premium requests)
  if (additionalInfoCallback) {
    const additionalInfo = additionalInfoCallback(lastEntry);
    if (additionalInfo) {
      markdown += additionalInfo;
    }
  }

  if (lastEntry.usage) {
    const usage = lastEntry.usage;
    if (usage.input_tokens || usage.output_tokens) {
      markdown += `**Token Usage:**\n`;
      if (usage.input_tokens) markdown += `- Input: ${usage.input_tokens.toLocaleString()}\n`;
      if (usage.cache_creation_input_tokens) markdown += `- Cache Creation: ${usage.cache_creation_input_tokens.toLocaleString()}\n`;
      if (usage.cache_read_input_tokens) markdown += `- Cache Read: ${usage.cache_read_input_tokens.toLocaleString()}\n`;
      if (usage.output_tokens) markdown += `- Output: ${usage.output_tokens.toLocaleString()}\n`;
      markdown += "\n";
    }
  }

  if (lastEntry.permission_denials && lastEntry.permission_denials.length > 0) {
    markdown += `**Permission Denials:** ${lastEntry.permission_denials.length}\n\n`;
  }

  return markdown;
}

// Export functions
module.exports = {
  formatDuration,
  formatBashCommand,
  truncateString,
  estimateTokens,
  formatMcpName,
  generateConversationMarkdown,
  generateInformationSection,
};
