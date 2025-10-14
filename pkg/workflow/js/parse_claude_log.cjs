function main() {
  const fs = require("fs");

  try {
    const logFile = process.env.GITHUB_AW_AGENT_OUTPUT;
    if (!logFile) {
      core.info("No agent log file specified");
      return;
    }
    if (!fs.existsSync(logFile)) {
      core.info(`Log file not found: ${logFile}`);
      return;
    }
    const logContent = fs.readFileSync(logFile, "utf8");
    const result = parseClaudeLog(logContent);
    core.info(result.markdown);
    core.summary.addRaw(result.markdown).write();
    if (result.mcpFailures && result.mcpFailures.length > 0) {
      const failedServers = result.mcpFailures.join(", ");
      core.setFailed(`MCP server(s) failed to launch: ${failedServers}`);
    }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.setFailed(errorMessage);
  }
}

/**
 * Parses Claude log content and converts it to markdown format
 * @param {string} logContent - The raw log content as a string
 * @returns {{markdown: string, mcpFailures: string[]}} Result with formatted markdown content and MCP failure list
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
      };
    }

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
    const mcpFailures = [];

    // Check for initialization data first
    const initEntry = logEntries.find(entry => entry.type === "system" && entry.subtype === "init");

    if (initEntry) {
      markdown += "## 🚀 Initialization\n\n";
      const initResult = formatInitializationSummary(initEntry);
      markdown += initResult.markdown;
      mcpFailures.push(...initResult.mcpFailures);
      markdown += "\n";
    }

    markdown += "\n## 🤖 Reasoning\n\n";

    // Second pass: process assistant messages in sequence
    for (const entry of logEntries) {
      if (entry.type === "assistant" && entry.message?.content) {
        for (const content of entry.message.content) {
          if (content.type === "text" && content.text) {
            // Add reasoning text directly (no header)
            const text = content.text.trim();
            if (text && text.length > 0) {
              markdown += text + "\n\n";
            }
          } else if (content.type === "tool_use") {
            // Process tool use with its result
            const toolResult = toolUsePairs.get(content.id);
            const toolMarkdown = formatToolUse(content, toolResult);
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

    // Add Information section from the last entry with result metadata
    markdown += "\n## 📊 Information\n\n";

    // Find the last entry with metadata
    const lastEntry = logEntries[logEntries.length - 1];
    if (lastEntry && (lastEntry.num_turns || lastEntry.duration_ms || lastEntry.total_cost_usd || lastEntry.usage)) {
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
    }

    return { markdown, mcpFailures };
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    return {
      markdown: `## Agent Log Summary\n\nError parsing Claude log (tried both JSON array and JSONL formats): ${errorMessage}\n`,
      mcpFailures: [],
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

      // Track failed MCP servers
      if (server.status === "failed") {
        mcpFailures.push(server.name);
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
        if (tools.length <= 5) {
          // Show all tools if 5 or fewer
          markdown += `  - ${tools.join(", ")}\n`;
        } else {
          // Show first few and count
          markdown += `  - ${tools.slice(0, 3).join(", ")}, and ${tools.length - 3} more\n`;
        }
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
 * Calculates approximate token count from text using 4 chars per token estimate
 * @param {string} text - The text to estimate tokens for
 * @returns {number} Approximate token count
 */
function estimateTokens(text) {
  if (!text) return 0;
  return Math.ceil(text.length / 4);
}

/**
 * Formats duration in seconds
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
  const maxLength = 80;
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
