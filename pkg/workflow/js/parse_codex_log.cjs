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
    let markdown = "## 🤖 Commands and Tools\n\n";

    const commandSummary = [];

    // Look-ahead window size for finding tool results
    // New format has verbose debug logs, so requires larger window
    const LOOKAHEAD_WINDOW = 50;

    // First pass: collect commands for summary
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];

      // Old TypeScript format: ToolCall: github__list_pull_requests {...}
      const toolCallMatch = line.match(/ToolCall:\s+(\w+)__(\w+)\s+(\{.+\})/);
      if (toolCallMatch) {
        const server = toolCallMatch[1];
        const toolName = toolCallMatch[2];

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

        commandSummary.push(`* ${statusIcon} \`${server}::${toolName}(...)\``);
        continue;
      }

      // New Rust format: tool github.list_pull_requests(...)
      const trimmedLine = line.trim();
      const rustToolMatch = trimmedLine.match(/^tool\s+(\w+)\.(\w+)\(/);
      if (rustToolMatch) {
        const server = rustToolMatch[1];
        const toolName = rustToolMatch[2];

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

        commandSummary.push(`* ${statusIcon} \`${server}::${toolName}(...)\``);
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

    // Add Information section
    markdown += "\n## 📊 Information\n\n";

    // Extract metadata from Codex logs
    let totalTokens = 0;

    // Old TypeScript format: TokenCount(TokenCountEvent { ... total_tokens: 13281 ...
    const tokenCountMatches = logContent.matchAll(/total_tokens:\s*(\d+)/g);
    for (const match of tokenCountMatches) {
      const tokens = parseInt(match[1]);
      totalTokens = Math.max(totalTokens, tokens); // Use the highest value (final total)
    }

    // New Rust format: "tokens used: <number>" or "tokens used\n<number>"
    // Check for inline format first: "tokens used: 15234"
    const inlineTokensMatch = logContent.match(/tokens used:\s*([\d,]+)/);
    if (inlineTokensMatch) {
      totalTokens = parseInt(inlineTokensMatch[1].replace(/,/g, ""));
    } else {
      // Fallback to newline format: "tokens used\n<number>"
      const finalTokensMatch = logContent.match(/tokens used\n([\d,]+)/);
      if (finalTokensMatch) {
        // Remove commas before parsing
        totalTokens = parseInt(finalTokensMatch[1].replace(/,/g, ""));
      }
    }

    if (totalTokens > 0) {
      markdown += `**Total Tokens Used:** ${totalTokens.toLocaleString()}\n\n`;
    }

    // Count tool calls (support both old and new formats)
    const oldFormatToolCalls = (logContent.match(/ToolCall:\s+\w+__\w+/g) || []).length;
    const newFormatToolCalls = (logContent.match(/^tool\s+\w+\.\w+\(/gm) || []).length;
    const toolCalls = oldFormatToolCalls + newFormatToolCalls;

    if (toolCalls > 0) {
      markdown += `**Tool Calls:** ${toolCalls}\n\n`;
    }

    markdown += "\n## 🤖 Reasoning\n\n";

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

    return markdown;
  } catch (error) {
    core.error(`Error parsing Codex log: ${error}`);
    return "## 🤖 Commands and Tools\n\nError parsing log content.\n\n## 🤖 Reasoning\n\nUnable to parse reasoning from log.\n\n";
  }
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
  module.exports = { parseCodexLog, truncateString };
}

main();
