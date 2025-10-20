function main() {
  const fs = require("fs");

  try {
    const logFile = process.env.GH_AW_AGENT_OUTPUT;
    if (!logFile) {
      core.info("No agent log file specified");
      return;
    }

    if (!fs.existsSync(logFile)) {
      core.info(`Log file not found: ${logFile}`);
      return;
    }

    const content = fs.readFileSync(logFile, "utf8");
    const parsedLog = parseCodexLog(content);

    if (parsedLog) {
      core.info(parsedLog);
      core.summary.addRaw(parsedLog).write();
      core.info("Codex log parsed successfully");
    } else {
      core.error("Failed to parse Codex log");
    }
  } catch (error) {
    core.setFailed(error instanceof Error ? error : String(error));
  }
}

/**
 * Parse codex log content and format as markdown
 * @param {string} logContent - The raw log content to parse
 * @returns {string} Formatted markdown content
 */
function parseCodexLog(logContent) {
  try {
    const lines = logContent.split("\n");

    // Look-ahead window size for finding tool results
    // New format has verbose debug logs, so requires larger window
    const LOOKAHEAD_WINDOW = 50;

    let markdown = "";

    markdown += "## 🤖 Reasoning\n\n";

    // Second pass: process full conversation flow with interleaved reasoning and tools
    let inThinkingSection = false;

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];

      // Skip metadata lines (including Rust debug lines)
      if (
        line.includes("OpenAI Codex") ||
        line.startsWith("--------") ||
        line.includes("workdir:") ||
        line.includes("model:") ||
        line.includes("provider:") ||
        line.includes("approval:") ||
        line.includes("sandbox:") ||
        line.includes("reasoning effort:") ||
        line.includes("reasoning summaries:") ||
        line.includes("tokens used:") ||
        line.includes("DEBUG codex") ||
        line.includes("INFO codex") ||
        line.match(/^\d{4}-\d{2}-\d{2}T[\d:.]+Z\s+(DEBUG|INFO|WARN|ERROR)/)
      ) {
        continue;
      }

      // Thinking section starts with standalone "thinking" line
      if (line.trim() === "thinking") {
        inThinkingSection = true;
        continue;
      }

      // Tool call line "tool github.list_pull_requests(...)"
      const toolMatch = line.match(/^tool\s+(\w+)\.(\w+)\(/);
      if (toolMatch) {
        inThinkingSection = false;
        const server = toolMatch[1];
        const toolName = toolMatch[2];

        // Look ahead to find the result status
        let statusIcon = "❓"; // Unknown by default
        for (let j = i + 1; j < Math.min(i + LOOKAHEAD_WINDOW, lines.length); j++) {
          const nextLine = lines[j];
          if (nextLine.includes(`${server}.${toolName}(`) && nextLine.includes("success in")) {
            statusIcon = "✅";
            break;
          } else if (nextLine.includes(`${server}.${toolName}(`) && (nextLine.includes("failed in") || nextLine.includes("error"))) {
            statusIcon = "❌";
            break;
          }
        }

        markdown += `${statusIcon} ${server}::${toolName}(...)\n\n`;
        continue;
      }

      // Process thinking content (filter out timestamp lines and very short lines)
      if (inThinkingSection && line.trim().length > 20 && !line.match(/^\d{4}-\d{2}-\d{2}T/)) {
        const trimmed = line.trim();
        // Add thinking content directly
        markdown += `${trimmed}\n\n`;
      }
    }

    markdown += "## 🤖 Commands and Tools\n\n";

    // First pass: collect tool calls with details
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];

      // Match: tool server.method(params) or ToolCall: server__method params
      const toolMatch = line.match(/^\[.*?\]\s+tool\s+(\w+)\.(\w+)\((.+)\)/) || line.match(/ToolCall:\s+(\w+)__(\w+)\s+(\{.+\})/);

      // Also match: exec bash -lc 'command' in /path
      const bashMatch = line.match(/^\[.*?\]\s+exec\s+bash\s+-lc\s+'([^']+)'/);

      if (toolMatch) {
        const server = toolMatch[1];
        const toolName = toolMatch[2];
        const params = toolMatch[3];

        // Look ahead to find the result
        let statusIcon = "❓";
        let response = "";
        let isError = false;

        for (let j = i + 1; j < Math.min(i + LOOKAHEAD_WINDOW, lines.length); j++) {
          const nextLine = lines[j];

          // Check for result line: server.method(...) success/failed in Xms:
          if (nextLine.includes(`${server}.${toolName}(`) && (nextLine.includes("success in") || nextLine.includes("failed in"))) {
            isError = nextLine.includes("failed in");
            statusIcon = isError ? "❌" : "✅";

            // Extract response - it's the JSON object following this line
            let jsonLines = [];
            let braceCount = 0;
            let inJson = false;

            for (let k = j + 1; k < Math.min(j + 30, lines.length); k++) {
              const respLine = lines[k];

              // Stop if we hit the next tool call or tokens used
              if (respLine.includes("tool ") || respLine.includes("ToolCall:") || respLine.includes("tokens used")) {
                break;
              }

              // Count braces to track JSON boundaries
              for (const char of respLine) {
                if (char === "{") {
                  braceCount++;
                  inJson = true;
                } else if (char === "}") {
                  braceCount--;
                }
              }

              if (inJson) {
                jsonLines.push(respLine);
              }

              if (inJson && braceCount === 0) {
                break;
              }
            }

            response = jsonLines.join("\n");
            break;
          }
        }

        // Format the tool call with HTML details
        markdown += formatCodexToolCall(server, toolName, params, response, statusIcon);
      } else if (bashMatch) {
        const command = bashMatch[1];

        // Look ahead to find the result
        let statusIcon = "❓";
        let response = "";
        let isError = false;

        for (let j = i + 1; j < Math.min(i + LOOKAHEAD_WINDOW, lines.length); j++) {
          const nextLine = lines[j];

          // Check for bash result line: bash -lc 'command' succeeded/failed in Xms:
          if (nextLine.includes("bash -lc") && (nextLine.includes("succeeded in") || nextLine.includes("failed in"))) {
            isError = nextLine.includes("failed in");
            statusIcon = isError ? "❌" : "✅";

            // Extract response - it's the plain text following this line
            let responseLines = [];

            for (let k = j + 1; k < Math.min(j + 20, lines.length); k++) {
              const respLine = lines[k];

              // Stop if we hit the next tool call, exec, or tokens used
              if (
                respLine.includes("tool ") ||
                respLine.includes("exec ") ||
                respLine.includes("ToolCall:") ||
                respLine.includes("tokens used") ||
                respLine.includes("thinking")
              ) {
                break;
              }

              responseLines.push(respLine);
            }

            response = responseLines.join("\n").trim();
            break;
          }
        }

        // Format the bash command with HTML details
        markdown += formatCodexBashCall(command, response, statusIcon);
      }
    }

    // Add Information section
    markdown += "\n## 📊 Information\n\n";

    // Extract metadata from Codex logs
    let totalTokens = 0;

    // TokenCount(TokenCountEvent { ... total_tokens: 13281 ...
    const tokenCountMatches = logContent.matchAll(/total_tokens:\s*(\d+)/g);
    for (const match of tokenCountMatches) {
      const tokens = parseInt(match[1]);
      totalTokens = Math.max(totalTokens, tokens); // Use the highest value (final total)
    }

    // Also check for "tokens used\n<number>" at the end (number may have commas)
    const finalTokensMatch = logContent.match(/tokens used\n([\d,]+)/);
    if (finalTokensMatch) {
      // Remove commas before parsing
      totalTokens = parseInt(finalTokensMatch[1].replace(/,/g, ""));
    }

    if (totalTokens > 0) {
      markdown += `**Total Tokens Used:** ${totalTokens.toLocaleString()}\n\n`;
    }

    // Count tool calls
    const toolCalls = (logContent.match(/ToolCall:\s+\w+__\w+/g) || []).length;

    if (toolCalls > 0) {
      markdown += `**Tool Calls:** ${toolCalls}\n\n`;
    }

    return markdown;
  } catch (error) {
    core.error(`Error parsing Codex log: ${error}`);
    return "## 🤖 Commands and Tools\n\nError parsing log content.\n\n## 🤖 Reasoning\n\nUnable to parse reasoning from log.\n\n";
  }
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
 * Format a Codex tool call with HTML details
 * @param {string} server - The server name (e.g., "github", "time")
 * @param {string} toolName - The tool name (e.g., "list_pull_requests")
 * @param {string} params - The parameters as JSON string
 * @param {string} response - The response as JSON string
 * @param {string} statusIcon - The status icon (✅, ❌, or ❓)
 * @returns {string} Formatted HTML details string
 */
function formatCodexToolCall(server, toolName, params, response, statusIcon) {
  // Calculate token estimate from params + response
  const totalTokens = estimateTokens(params) + estimateTokens(response);

  // Format metadata
  let metadata = "";
  if (totalTokens > 0) {
    metadata = ` <code>~${totalTokens}t</code>`;
  }

  const summary = `${statusIcon} <code>${server}::${toolName}</code>${metadata}`;

  // If no response, just show the summary
  if (!response || response.trim() === "") {
    return `${summary}\n\n`;
  }

  // Build the details content with parameters and response
  let details = "";

  // Add parameters section
  if (params && params.trim()) {
    details += "**Parameters:**\n\n";
    details += "``````json\n";
    details += params;
    details += "\n``````\n\n";
  }

  // Add response section
  if (response && response.trim()) {
    details += "**Response:**\n\n";
    details += "``````json\n";
    details += response;
    details += "\n``````";
  }

  // Return formatted HTML details
  return `<details>\n<summary>${summary}</summary>\n\n${details}\n</details>\n\n`;
}

/**
 * Format a Codex bash call with HTML details
 * @param {string} command - The bash command
 * @param {string} response - The response as plain text
 * @param {string} statusIcon - The status icon (✅, ❌, or ❓)
 * @returns {string} Formatted HTML details string
 */
function formatCodexBashCall(command, response, statusIcon) {
  // Calculate token estimate from command + response
  const totalTokens = estimateTokens(command) + estimateTokens(response);

  // Format metadata
  let metadata = "";
  if (totalTokens > 0) {
    metadata = ` <code>~${totalTokens}t</code>`;
  }

  const summary = `${statusIcon} <code>bash: ${truncateString(command, 60)}</code>${metadata}`;

  // If no response, just show the summary
  if (!response || response.trim() === "") {
    return `${summary}\n\n`;
  }

  // Build the details content with command and response
  let details = "";

  // Add command section
  details += "**Command:**\n\n";
  details += "``````bash\n";
  details += command;
  details += "\n``````\n\n";

  // Add response section
  if (response && response.trim()) {
    details += "**Output:**\n\n";
    details += "``````\n";
    details += response;
    details += "\n``````";
  }

  // Return formatted HTML details
  return `<details>\n<summary>${summary}</summary>\n\n${details}\n</details>\n\n`;
}

/**
 * Truncate string to maximum length
 * @param {string} str - The string to truncate
 * @param {number} maxLength - Maximum length allowed
 * @returns {string} Truncated string
 */
function truncateString(str, maxLength) {
  if (!str) return "";
  if (str.length <= maxLength) return str;
  return str.substring(0, maxLength) + "...";
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    parseCodexLog,
    formatCodexToolCall,
    formatCodexBashCall,
    truncateString,
    estimateTokens,
    formatDuration,
  };
}

main();
