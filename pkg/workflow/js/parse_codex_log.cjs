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

      // New Rust format: tool github.list_pull_requests(...)
      // Match lines like: [2025-08-31T12:37:33] tool time.get_current_time({"timezone":"UTC"})
      const toolCallMatch = line.match(/\[\d{4}-\d{2}-\d{2}T[\d:.]+\]\s+tool\s+(\w+)\.(\w+)\(/);
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
      }

      // Bash command execution: [2025-08-31T12:37:55] exec bash -lc 'git remote -v' in /path
      const bashExecMatch = line.match(/\[\d{4}-\d{2}-\d{2}T[\d:.]+\]\s+exec\s+(bash\s+-lc\s+'[^']+')/);
      if (bashExecMatch) {
        const command = bashExecMatch[1];

        // Look ahead to find the result status
        let statusIcon = "❓";
        for (let j = i + 1; j < Math.min(i + LOOKAHEAD_WINDOW, lines.length); j++) {
          const nextLine = lines[j];
          if (nextLine.includes("bash -lc") && nextLine.includes("succeeded in")) {
            statusIcon = "✅";
            break;
          } else if (nextLine.includes("bash -lc") && (nextLine.includes("failed in") || nextLine.includes("error"))) {
            statusIcon = "❌";
            break;
          }
        }

        commandSummary.push(`* ${statusIcon} \`${command}\``);
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

    // New Rust format: [2025-08-31T12:37:33] tokens used: 14582
    // Sum all token usage entries
    const tokenMatches = logContent.matchAll(/\[\d{4}-\d{2}-\d{2}T[\d:.]+\]\s+tokens used:\s*(\d+)/g);
    for (const match of tokenMatches) {
      const tokens = parseInt(match[1]);
      totalTokens += tokens; // Sum all token entries
    }

    if (totalTokens > 0) {
      markdown += `**Total Tokens Used:** ${totalTokens.toLocaleString()}\n\n`;
    }

    // Count tool calls (new Rust format)
    const toolCalls = (logContent.match(/\[\d{4}-\d{2}-\d{2}T[\d:.]+\]\s+tool\s+\w+\.\w+\(/g) || []).length;

    if (toolCalls > 0) {
      markdown += `**Tool Calls:** ${toolCalls}\n\n`;
    }

    // Count bash exec commands
    const bashCommands = (logContent.match(/\[\d{4}-\d{2}-\d{2}T[\d:.]+\]\s+exec\s+bash\s+-lc/g) || []).length;

    if (bashCommands > 0) {
      markdown += `**Commands Executed:** ${bashCommands}\n\n`;
    }

    markdown += "\n## 🤖 Reasoning\n\n";

    // Second pass: process full conversation flow with interleaved reasoning and tools
    let inThinkingSection = false;

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];

      // Skip metadata lines (including Rust debug lines and user instructions)
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
        line.includes("User instructions:") ||
        line.match(/^\[\d{4}-\d{2}-\d{2}T[\d:.]+\]\s+(DEBUG|INFO|WARN|ERROR)/)
      ) {
        continue;
      }

      // Thinking section starts with timestamped "thinking" line: [2025-08-31T12:38:35] thinking
      if (line.match(/^\[\d{4}-\d{2}-\d{2}T[\d:.]+\]\s+thinking$/)) {
        inThinkingSection = true;
        continue;
      }

      // Tool call line: [2025-08-31T12:37:33] tool github.list_pull_requests(...)
      const toolMatch = line.match(/^\[\d{4}-\d{2}-\d{2}T[\d:.]+\]\s+tool\s+(\w+)\.(\w+)\(/);
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

      // Bash execution line: [2025-08-31T12:37:55] exec bash -lc 'git remote -v' in /path
      const bashMatch = line.match(/^\[\d{4}-\d{2}-\d{2}T[\d:.]+\]\s+exec\s+(bash\s+-lc\s+'[^']+')/);
      if (bashMatch) {
        inThinkingSection = false;
        const command = bashMatch[1];

        // Look ahead to find the result status
        let statusIcon = "❓";
        for (let j = i + 1; j < Math.min(i + LOOKAHEAD_WINDOW, lines.length); j++) {
          const nextLine = lines[j];
          if (nextLine.includes("bash -lc") && nextLine.includes("succeeded in")) {
            statusIcon = "✅";
            break;
          } else if (nextLine.includes("bash -lc") && (nextLine.includes("failed in") || nextLine.includes("error"))) {
            statusIcon = "❌";
            break;
          }
        }

        markdown += `${statusIcon} \`${command}\`\n\n`;
        continue;
      }

      // Process thinking content (filter out JSON content and very short lines)
      if (inThinkingSection && line.trim().length > 20 && !line.match(/^\[\d{4}-\d{2}-\d{2}T/) && !line.match(/^\s*[\{\[]/)) {
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
