interface ParseResult {
  markdown: string;
  mcpFailures: string[];
}

interface LogEntry {
  level?: string;
  msg?: string;
  time?: string;
  caller?: string;
  [key: string]: any;
}

function parseClaudeLogMain(): void {
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
 * @param logContent - The raw log content as a string
 * @returns Result with formatted markdown content and MCP failure list
 */
function parseClaudeLog(logContent: string): ParseResult {
  try {
    let logEntries: LogEntry[];

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

        // Try to parse as JSON object (JSONL format)
        try {
          const entry = JSON.parse(trimmedLine);
          if (typeof entry === "object" && entry !== null) {
            logEntries.push(entry);
          }
        } catch (jsonError) {
          // If it's not JSON, treat it as a plain text debug log
          // Check if it looks like a structured log entry
          if (
            trimmedLine.includes("level=") ||
            trimmedLine.includes("msg=") ||
            trimmedLine.includes("time=")
          ) {
            // Parse structured log entry (key=value format)
            const entry: LogEntry = {};
            const parts = trimmedLine.split(/\s+/);
            
            for (const part of parts) {
              const [key, ...valueParts] = part.split("=");
              if (valueParts.length > 0) {
                const value = valueParts.join("=").replace(/^"(.*)"$/, "$1"); // Remove quotes
                entry[key] = value;
              }
            }
            
            logEntries.push(entry);
          } else {
            // Plain text debug log
            logEntries.push({
              level: "debug",
              msg: trimmedLine,
              time: new Date().toISOString(),
            });
          }
        }
      }
    }

    // Generate markdown from log entries
    let markdown = "## 🤖 Claude Agent Execution Log\n\n";
    const mcpFailures: string[] = [];
    const toolCalls: { [tool: string]: number } = {};
    const errorMessages: string[] = [];
    const mcpServers: string[] = [];

    // Count tool usage and track errors
    for (const entry of logEntries) {
      const msg = entry.msg || "";
      const level = entry.level || "info";

      // Track MCP server launches
      if (msg.includes("launching MCP server") || msg.includes("MCP server started")) {
        const serverMatch = msg.match(/server[:\s]+([^\s,]+)/i);
        if (serverMatch && !mcpServers.includes(serverMatch[1])) {
          mcpServers.push(serverMatch[1]);
        }
      }

      // Track MCP failures
      if (
        level === "error" &&
        (msg.includes("MCP server") || msg.includes("MCP") || msg.includes("failed to launch"))
      ) {
        const serverMatch = msg.match(/server[:\s]+([^\s,]+)/i);
        if (serverMatch) {
          mcpFailures.push(serverMatch[1]);
        } else {
          mcpFailures.push("unknown server");
        }
      }

      // Track tool calls
      if (msg.includes("calling tool") || msg.includes("tool call")) {
        const toolMatch = msg.match(/tool[:\s]+([^\s,()]+)/i);
        if (toolMatch) {
          const tool = toolMatch[1];
          toolCalls[tool] = (toolCalls[tool] || 0) + 1;
        }
      }

      // Collect error messages
      if (level === "error" && !errorMessages.includes(msg)) {
        errorMessages.push(msg);
      }
    }

    // Add summary section
    if (mcpServers.length > 0) {
      markdown += "### 🔧 MCP Servers\n\n";
      for (const server of mcpServers) {
        const status = mcpFailures.includes(server) ? "❌ Failed" : "✅ Running";
        markdown += `- **${server}**: ${status}\n`;
      }
      markdown += "\n";
    }

    // Add tool usage summary
    if (Object.keys(toolCalls).length > 0) {
      markdown += "### 🛠️ Tool Usage Summary\n\n";
      const sortedTools = Object.entries(toolCalls).sort((a, b) => b[1] - a[1]);
      for (const [tool, count] of sortedTools) {
        markdown += `- **${tool}**: ${count} call${count !== 1 ? 's' : ''}\n`;
      }
      markdown += "\n";
    }

    // Add errors section if any
    if (errorMessages.length > 0) {
      markdown += "### ❌ Errors\n\n";
      for (const error of errorMessages) {
        markdown += `- ${error}\n`;
      }
      markdown += "\n";
    }

    // Add detailed execution log
    markdown += "### 📋 Execution Details\n\n";
    
    let currentSection = "";
    let sectionCount = 0;
    
    for (let i = 0; i < logEntries.length; i++) {
      const entry = logEntries[i];
      const msg = entry.msg || "";
      const level = entry.level || "info";
      const time = entry.time || "";

      // Group related log entries
      if (msg.includes("calling tool") || msg.includes("tool call")) {
        const toolMatch = msg.match(/tool[:\s]+([^\s,()]+)/i);
        if (toolMatch) {
          const tool = toolMatch[1];
          if (currentSection !== tool) {
            currentSection = tool;
            sectionCount++;
            markdown += `#### ${sectionCount}. 🔧 Tool: ${tool}\n\n`;
          }
        }
      } else if (msg.includes("executing") || msg.includes("running")) {
        if (currentSection !== "execution") {
          currentSection = "execution";
          sectionCount++;
          markdown += `#### ${sectionCount}. ⚡ Execution\n\n`;
        }
      }

      // Format log entry
      const levelEmoji = getLevelEmoji(level);
      const timeStr = time ? ` \`${time}\`` : "";
      
      if (level === "error") {
        markdown += `${levelEmoji}${timeStr} **ERROR**: ${msg}\n\n`;
      } else if (level === "warn") {
        markdown += `${levelEmoji}${timeStr} **WARNING**: ${msg}\n\n`;
      } else if (msg.length > 200) {
        // Truncate very long messages
        markdown += `${levelEmoji}${timeStr} ${msg.substring(0, 200)}...\n\n`;
      } else {
        markdown += `${levelEmoji}${timeStr} ${msg}\n\n`;
      }
    }

    // Add footer
    markdown += "---\n";
    markdown += `*Log parsed at ${new Date().toISOString()}*\n`;
    markdown += `*Total entries: ${logEntries.length}*\n`;

    return {
      markdown,
      mcpFailures,
    };

  } catch (error) {
    core.error(`Error parsing log: ${error instanceof Error ? error.message : String(error)}`);
    return {
      markdown: `## ❌ Log Parsing Error\n\nFailed to parse Claude log: ${error instanceof Error ? error.message : String(error)}\n`,
      mcpFailures: [],
    };
  }
}

function getLevelEmoji(level: string): string {
  switch (level.toLowerCase()) {
    case "error":
      return "❌ ";
    case "warn":
    case "warning":
      return "⚠️ ";
    case "info":
      return "ℹ️ ";
    case "debug":
      return "🔍 ";
    default:
      return "📝 ";
  }
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    parseClaudeLog,
    getLevelEmoji,
  };
}

(async () => {
  parseClaudeLogMain();
})();