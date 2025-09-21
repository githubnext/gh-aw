async function parseClaudeLogMain() {
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
    await core.summary.addRaw(result.markdown).write();
    if (result.mcpFailures && result.mcpFailures.length > 0) {
      const failedServers = result.mcpFailures.join(", ");
      core.setFailed(`MCP server(s) failed to launch: ${failedServers}`);
    }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.setFailed(errorMessage);
  }
}
function parseClaudeLog(logContent) {
  try {
    let logEntries;
    try {
      logEntries = JSON.parse(logContent);
      if (!Array.isArray(logEntries)) {
        throw new Error("Not a JSON array");
      }
    } catch (jsonArrayError) {
      logEntries = [];
      const lines = logContent.split("\n");
      for (const line of lines) {
        const trimmedLine = line.trim();
        if (trimmedLine === "") {
          continue;
        }
        if (trimmedLine.startsWith("[{")) {
          try {
            const arrayEntries = JSON.parse(trimmedLine);
            if (Array.isArray(arrayEntries)) {
              logEntries.push(...arrayEntries);
              continue;
            }
          } catch (arrayParseError) {
            continue;
          }
        }
        if (!trimmedLine.startsWith("{")) {
          continue;
        }
        try {
          const jsonEntry = JSON.parse(trimmedLine);
          logEntries.push(jsonEntry);
        } catch (jsonLineError) {
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
    let markdown = "";
    const mcpFailures = [];
    const initEntry = logEntries.find(entry => entry.type === "system" && entry.subtype === "init");
    if (initEntry) {
      markdown += "## 🚀 Initialization\n\n";
      const initResult = formatInitializationSummary(initEntry);
      markdown += initResult.markdown;
      mcpFailures.push(...initResult.mcpFailures);
      markdown += "\n";
    }
    markdown += "## 🤖 Commands and Tools\n\n";
    const toolUsePairs = new Map();
    const commandSummary = [];
    for (const entry of logEntries) {
      if (entry.type === "user" && entry.message?.content) {
        for (const content of entry.message.content) {
          if (content.type === "tool_result" && content.tool_use_id) {
            toolUsePairs.set(content.tool_use_id, content);
          }
        }
      }
    }
    for (const entry of logEntries) {
      if (entry.type === "assistant" && entry.message?.content) {
        for (const content of entry.message.content) {
          if (content.type === "tool_use") {
            const toolName = content.name;
            const input = content.input || {};
            if (["Read", "Write", "Edit", "MultiEdit", "LS", "Grep", "Glob", "TodoWrite"].includes(toolName)) {
              continue;
            }
            const toolResult = toolUsePairs.get(content.id);
            let statusIcon = "❓";
            if (toolResult) {
              statusIcon = toolResult.is_error === true ? "❌" : "✅";
            }
            if (toolName === "Bash") {
              const formattedCommand = formatBashCommand(input.command || "");
              commandSummary.push(`* ${statusIcon} \`${formattedCommand}\``);
            } else if (toolName.startsWith("mcp__")) {
              const mcpName = formatMcpName(toolName);
              commandSummary.push(`* ${statusIcon} \`${mcpName}(...)\``);
            } else {
              commandSummary.push(`* ${statusIcon} ${toolName}`);
            }
          }
        }
      }
    }
    if (commandSummary.length > 0) {
      for (const cmd of commandSummary) {
        markdown += `${cmd}\n`;
      }
    } else {
      markdown += "No commands or tools used.\n";
    }
    markdown += "\n## 📊 Information\n\n";
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
    markdown += "\n## 🤖 Reasoning\n\n";
    for (const entry of logEntries) {
      if (entry.type === "assistant" && entry.message?.content) {
        for (const content of entry.message.content) {
          if (content.type === "text" && content.text) {
            const text = content.text.trim();
            if (text && text.length > 0) {
              markdown += text + "\n\n";
            }
          } else if (content.type === "tool_use") {
            const toolResult = toolUsePairs.get(content.id);
            const toolMarkdown = formatToolUse(content, toolResult);
            if (toolMarkdown) {
              markdown += toolMarkdown;
            }
          }
        }
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
function formatInitializationSummary(initEntry) {
  let markdown = "";
  const mcpFailures = [];
  if (initEntry.model) {
    markdown += `**Model:** ${initEntry.model}\n\n`;
  }
  if (initEntry.session_id) {
    markdown += `**Session ID:** ${initEntry.session_id}\n\n`;
  }
  if (initEntry.cwd) {
    const cleanCwd = initEntry.cwd.replace(/^\/home\/runner\/work\/[^\/]+\/[^\/]+/, ".");
    markdown += `**Working Directory:** ${cleanCwd}\n\n`;
  }
  if (initEntry.mcp_servers && Array.isArray(initEntry.mcp_servers)) {
    markdown += "**MCP Servers:**\n";
    for (const server of initEntry.mcp_servers) {
      const statusIcon = server.status === "connected" ? "✅" : server.status === "failed" ? "❌" : "❓";
      markdown += `- ${statusIcon} ${server.name} (${server.status})\n`;
      if (server.status === "failed") {
        mcpFailures.push(server.name);
      }
    }
    markdown += "\n";
  }
  if (initEntry.tools && Array.isArray(initEntry.tools)) {
    markdown += "**Available Tools:**\n";
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
    for (const [category, tools] of Object.entries(categories)) {
      if (tools.length > 0) {
        markdown += `- **${category}:** ${tools.length} tools\n`;
        if (tools.length <= 5) {
          markdown += `  - ${tools.join(", ")}\n`;
        } else {
          markdown += `  - ${tools.slice(0, 3).join(", ")}, and ${tools.length - 3} more\n`;
        }
      }
    }
    markdown += "\n";
  }
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
function formatToolUse(toolUse, toolResult) {
  const toolName = toolUse.name;
  const input = toolUse.input || {};
  if (toolName === "TodoWrite") {
    return "";
  }
  function getStatusIcon() {
    if (toolResult) {
      return toolResult.is_error === true ? "❌" : "✅";
    }
    return "❓";
  }
  let markdown = "";
  const statusIcon = getStatusIcon();
  switch (toolName) {
    case "Bash":
      const command = input.command || "";
      const description = input.description || "";
      const formattedCommand = formatBashCommand(command);
      if (description) {
        markdown += `${description}:\n\n`;
      }
      markdown += `${statusIcon} \`${formattedCommand}\`\n\n`;
      break;
    case "Read":
      const filePath = input.file_path || input.path || "";
      const relativePath = filePath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, "");
      markdown += `${statusIcon} Read \`${relativePath}\`\n\n`;
      break;
    case "Write":
    case "Edit":
    case "MultiEdit":
      const writeFilePath = input.file_path || input.path || "";
      const writeRelativePath = writeFilePath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, "");
      markdown += `${statusIcon} Write \`${writeRelativePath}\`\n\n`;
      break;
    case "Grep":
    case "Glob":
      const query = input.query || input.pattern || "";
      markdown += `${statusIcon} Search for \`${truncateString(query, 80)}\`\n\n`;
      break;
    case "LS":
      const lsPath = input.path || "";
      const lsRelativePath = lsPath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, "");
      markdown += `${statusIcon} LS: ${lsRelativePath || lsPath}\n\n`;
      break;
    default:
      if (toolName.startsWith("mcp__")) {
        const mcpName = formatMcpName(toolName);
        const params = formatMcpParameters(input);
        markdown += `${statusIcon} ${mcpName}(${params})\n\n`;
      } else {
        const keys = Object.keys(input);
        if (keys.length > 0) {
          const mainParam = keys.find(k => ["query", "command", "path", "file_path", "content"].includes(k)) || keys[0];
          const value = String(input[mainParam] || "");
          if (value) {
            markdown += `${statusIcon} ${toolName}: ${truncateString(value, 100)}\n\n`;
          } else {
            markdown += `${statusIcon} ${toolName}\n\n`;
          }
        } else {
          markdown += `${statusIcon} ${toolName}\n\n`;
        }
      }
  }
  return markdown;
}
function formatMcpName(toolName) {
  if (toolName.startsWith("mcp__")) {
    const parts = toolName.split("__");
    if (parts.length >= 3) {
      const provider = parts[1];
      const method = parts.slice(2).join("_");
      return `${provider}::${method}`;
    }
  }
  return toolName;
}
function formatMcpParameters(input) {
  const keys = Object.keys(input);
  if (keys.length === 0) return "";
  const paramStrs = [];
  for (const key of keys.slice(0, 4)) {
    const value = String(input[key] || "");
    paramStrs.push(`${key}: ${truncateString(value, 40)}`);
  }
  if (keys.length > 4) {
    paramStrs.push("...");
  }
  return paramStrs.join(", ");
}
function formatBashCommand(command) {
  if (!command) return "";
  let formatted = command.replace(/\n/g, " ").replace(/\r/g, " ").replace(/\t/g, " ").replace(/\s+/g, " ").trim();
  formatted = formatted.replace(/`/g, "\\`");
  const maxLength = 80;
  if (formatted.length > maxLength) {
    formatted = formatted.substring(0, maxLength) + "...";
  }
  return formatted;
}
function truncateString(str, maxLength) {
  if (!str) return "";
  if (str.length <= maxLength) return str;
  return str.substring(0, maxLength) + "...";
}
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    parseClaudeLog,
    formatToolUse,
    formatInitializationSummary,
    formatBashCommand,
    truncateString,
  };
}
(async () => {
  await parseClaudeLogMain();
})();
