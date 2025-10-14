function main() {
  const fs = require("fs");
  const path = require("path");

  try {
    const logPath = process.env.GITHUB_AW_AGENT_OUTPUT;
    if (!logPath) {
      core.info("No agent log file specified");
      return;
    }

    if (!fs.existsSync(logPath)) {
      core.info(`Log path not found: ${logPath}`);
      return;
    }

    let content = "";

    // Check if logPath is a directory or a file
    const stat = fs.statSync(logPath);
    if (stat.isDirectory()) {
      // Read all log files from the directory and concatenate them
      const files = fs.readdirSync(logPath);
      const logFiles = files.filter(file => file.endsWith(".log") || file.endsWith(".txt"));

      if (logFiles.length === 0) {
        core.info(`No log files found in directory: ${logPath}`);
        return;
      }

      // Sort log files by name to ensure consistent ordering
      logFiles.sort();

      // Concatenate all log files
      for (const file of logFiles) {
        const filePath = path.join(logPath, file);
        const fileContent = fs.readFileSync(filePath, "utf8");
        content += fileContent;
        // Add a newline between files if the previous file doesn't end with one
        if (content.length > 0 && !content.endsWith("\n")) {
          content += "\n";
        }
      }
    } else {
      // Read the single log file
      content = fs.readFileSync(logPath, "utf8");
    }

    const parsedLog = parseCopilotLog(content);

    if (parsedLog) {
      core.info(parsedLog);
      core.summary.addRaw(parsedLog).write();
      core.info("Copilot log parsed successfully");
    } else {
      core.error("Failed to parse Copilot log");
    }
  } catch (error) {
    core.setFailed(error instanceof Error ? error : String(error));
  }
}

/**
 * Parses Copilot CLI log content and converts it to markdown format
 * @param {string} logContent - The raw log content as a string
 * @returns {string} Formatted markdown content
 */
function parseCopilotLog(logContent) {
  try {
    let logEntries;

    // First, try to parse as JSON array (structured format)
    try {
      logEntries = JSON.parse(logContent);
      if (!Array.isArray(logEntries)) {
        throw new Error("Not a JSON array");
      }
    } catch (jsonArrayError) {
      // If that fails, try to parse as debug logs format
      const debugLogEntries = parseDebugLogFormat(logContent);
      if (debugLogEntries && debugLogEntries.length > 0) {
        logEntries = debugLogEntries;
      } else {
        // Try JSONL format
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
          if (!trimmedLine.startsWith("{")) {
            continue;
          }

          // Try to parse each line as JSON
          try {
            const jsonEntry = JSON.parse(trimmedLine);
            logEntries.push(jsonEntry);
          } catch (jsonLineError) {
            // Skip invalid JSON lines
            continue;
          }
        }
      }
    }

    if (!Array.isArray(logEntries) || logEntries.length === 0) {
      return "## Agent Log Summary\n\nLog format not recognized as Copilot JSON array or JSONL.\n";
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

    // Check for initialization data first
    const initEntry = logEntries.find(entry => entry.type === "system" && entry.subtype === "init");

    if (initEntry) {
      markdown += "## 🚀 Initialization\n\n";
      markdown += formatInitializationSummary(initEntry);
      markdown += "\n";
    }

    markdown += "\n## 🤖 Reasoning\n\n";

    // Second pass: process assistant messages in sequence
    for (const entry of logEntries) {
      if (entry.type === "assistant" && entry.message?.content) {
        for (const content of entry.message.content) {
          if (content.type === "text" && content.text) {
            // Add user message text directly as markdown
            const text = content.text.trim();
            if (text && text.length > 0) {
              markdown += text + "\n\n";
            }
          } else if (content.type === "tool_use") {
            // Process tool use with its result using HTML details
            const toolResult = toolUsePairs.get(content.id);
            const toolMarkdown = formatToolUseWithDetails(content, toolResult);
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
              continue;
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

    // Add Information section
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

      // Display premium request consumption if using a premium model
      const isPremiumModel =
        initEntry && initEntry.model_info && initEntry.model_info.billing && initEntry.model_info.billing.is_premium === true;
      if (isPremiumModel && lastEntry.num_turns) {
        markdown += `**Premium Requests Consumed:** ${lastEntry.num_turns}\n\n`;
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
    }

    return markdown;
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    return `## Agent Log Summary\n\nError parsing Copilot log (tried both JSON array and JSONL formats): ${errorMessage}\n`;
  }
}

/**
 * Parses Copilot CLI debug log format and reconstructs the conversation flow
 * @param {string} logContent - Raw debug log content
 * @returns {Array} Array of log entries in structured format
 */
function parseDebugLogFormat(logContent) {
  const entries = [];
  const lines = logContent.split("\n");

  // Extract model information from the start
  let model = "unknown";
  let sessionId = null;
  let modelInfo = null;
  const modelMatch = logContent.match(/Starting Copilot CLI: ([\d.]+)/);
  if (modelMatch) {
    sessionId = `copilot-${modelMatch[1]}-${Date.now()}`;
  }

  // Extract premium model info from "Got model info:" JSON block
  // Look for a multi-line JSON block that starts with "Got model info: {" and ends with "}"
  const gotModelInfoIndex = logContent.indexOf("[DEBUG] Got model info: {");
  if (gotModelInfoIndex !== -1) {
    // Find the start of the JSON (the opening brace)
    const jsonStart = logContent.indexOf("{", gotModelInfoIndex);
    if (jsonStart !== -1) {
      // Track braces to find the end of the JSON
      let braceCount = 0;
      let inString = false;
      let escapeNext = false;
      let jsonEnd = -1;

      for (let i = jsonStart; i < logContent.length; i++) {
        const char = logContent[i];

        if (escapeNext) {
          escapeNext = false;
          continue;
        }

        if (char === "\\") {
          escapeNext = true;
          continue;
        }

        if (char === '"' && !escapeNext) {
          inString = !inString;
          continue;
        }

        if (inString) continue;

        if (char === "{") {
          braceCount++;
        } else if (char === "}") {
          braceCount--;
          if (braceCount === 0) {
            jsonEnd = i + 1;
            break;
          }
        }
      }

      if (jsonEnd !== -1) {
        const modelInfoJson = logContent.substring(jsonStart, jsonEnd);
        try {
          modelInfo = JSON.parse(modelInfoJson);
        } catch (e) {
          // Failed to parse model info, continue without it
        }
      }
    }
  }

  // Find all JSON response blocks in the debug logs
  let inDataBlock = false;
  let currentJsonLines = [];
  let turnCount = 0;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    // Detect start of a JSON data block
    if (line.includes("[DEBUG] data:")) {
      inDataBlock = true;
      currentJsonLines = [];
      continue;
    }

    // While in a data block, accumulate lines
    if (inDataBlock) {
      // Check if this line starts with timestamp AND NOT [DEBUG] (new non-JSON log entry)
      const hasTimestamp = line.match(/^\d{4}-\d{2}-\d{2}T[\d:.]+Z /);
      const hasDebug = line.includes("[DEBUG]");

      if (hasTimestamp && !hasDebug) {
        // This is a new log line (not part of JSON) - end of JSON block, process what we have
        if (currentJsonLines.length > 0) {
          try {
            const jsonStr = currentJsonLines.join("\n");
            const jsonData = JSON.parse(jsonStr);

            // Extract model info
            if (jsonData.model) {
              model = jsonData.model;
            }

            // Process the choices in the response
            if (jsonData.choices && Array.isArray(jsonData.choices)) {
              for (const choice of jsonData.choices) {
                if (choice.message) {
                  const message = choice.message;

                  // Create an assistant entry
                  const content = [];
                  const toolResults = []; // Collect tool calls to create synthetic results (debug logs don't include actual results)

                  if (message.content && message.content.trim()) {
                    content.push({
                      type: "text",
                      text: message.content,
                    });
                  }

                  if (message.tool_calls && Array.isArray(message.tool_calls)) {
                    for (const toolCall of message.tool_calls) {
                      if (toolCall.function) {
                        let toolName = toolCall.function.name;
                        let args = {};

                        // Parse tool name (handle github- prefix and bash)
                        if (toolName.startsWith("github-")) {
                          toolName = "mcp__github__" + toolName.substring(7);
                        } else if (toolName === "bash") {
                          toolName = "Bash";
                        }

                        // Parse arguments
                        try {
                          args = JSON.parse(toolCall.function.arguments);
                        } catch (e) {
                          args = {};
                        }

                        const toolId = toolCall.id || `tool_${Date.now()}_${Math.random()}`;
                        content.push({
                          type: "tool_use",
                          id: toolId,
                          name: toolName,
                          input: args,
                        });

                        // Create a corresponding tool result (assume success since we don't have actual results in debug logs)
                        toolResults.push({
                          type: "tool_result",
                          tool_use_id: toolId,
                          content: "", // No actual output available in debug logs
                          is_error: false, // Assume success
                        });
                      }
                    }
                  }

                  if (content.length > 0) {
                    entries.push({
                      type: "assistant",
                      message: { content },
                    });
                    turnCount++;

                    // Add tool results as a user message if we have any
                    if (toolResults.length > 0) {
                      entries.push({
                        type: "user",
                        message: { content: toolResults },
                      });
                    }
                  }
                }
              }

              // Add usage/result entry if this is the last response
              if (jsonData.usage) {
                const resultEntry = {
                  type: "result",
                  num_turns: turnCount,
                  usage: jsonData.usage,
                };

                // Store for later (we'll add it at the end)
                entries._lastResult = resultEntry;
              }
            }
          } catch (e) {
            // Skip invalid JSON blocks
          }
        }

        inDataBlock = false;
        currentJsonLines = [];
      } else {
        // This line is part of the JSON - add it (remove [DEBUG] prefix if present)
        const cleanLine = line.replace(/^\d{4}-\d{2}-\d{2}T[\d:.]+Z \[DEBUG\] /, "");
        currentJsonLines.push(cleanLine);
      }
    }
  }

  // Process any remaining JSON block at the end of file
  if (inDataBlock && currentJsonLines.length > 0) {
    try {
      const jsonStr = currentJsonLines.join("\n");
      const jsonData = JSON.parse(jsonStr);

      if (jsonData.model) {
        model = jsonData.model;
      }

      if (jsonData.choices && Array.isArray(jsonData.choices)) {
        for (const choice of jsonData.choices) {
          if (choice.message) {
            const message = choice.message;
            const content = [];
            const toolResults = []; // Collect tool calls to create synthetic results (debug logs don't include actual results)

            if (message.content && message.content.trim()) {
              content.push({
                type: "text",
                text: message.content,
              });
            }

            if (message.tool_calls && Array.isArray(message.tool_calls)) {
              for (const toolCall of message.tool_calls) {
                if (toolCall.function) {
                  let toolName = toolCall.function.name;
                  let args = {};

                  if (toolName.startsWith("github-")) {
                    toolName = "mcp__github__" + toolName.substring(7);
                  } else if (toolName === "bash") {
                    toolName = "Bash";
                  }

                  try {
                    args = JSON.parse(toolCall.function.arguments);
                  } catch (e) {
                    args = {};
                  }

                  const toolId = toolCall.id || `tool_${Date.now()}_${Math.random()}`;
                  content.push({
                    type: "tool_use",
                    id: toolId,
                    name: toolName,
                    input: args,
                  });

                  // Create a corresponding tool result (assume success since we don't have actual results in debug logs)
                  toolResults.push({
                    type: "tool_result",
                    tool_use_id: toolId,
                    content: "", // No actual output available in debug logs
                    is_error: false, // Assume success
                  });
                }
              }
            }

            if (content.length > 0) {
              entries.push({
                type: "assistant",
                message: { content },
              });
              turnCount++;

              // Add tool results as a user message if we have any
              if (toolResults.length > 0) {
                entries.push({
                  type: "user",
                  message: { content: toolResults },
                });
              }
            }
          }
        }

        if (jsonData.usage) {
          const resultEntry = {
            type: "result",
            num_turns: turnCount,
            usage: jsonData.usage,
          };
          entries._lastResult = resultEntry;
        }
      }
    } catch (e) {
      // Skip invalid JSON
    }
  }

  // Add system init entry at the beginning if we have entries
  if (entries.length > 0) {
    const initEntry = {
      type: "system",
      subtype: "init",
      session_id: sessionId,
      model: model,
      tools: [], // We don't have tool info from debug logs
    };

    // Add model info if available
    if (modelInfo) {
      initEntry.model_info = modelInfo;
    }

    entries.unshift(initEntry);

    // Add the final result entry if we have it
    if (entries._lastResult) {
      entries.push(entries._lastResult);
      delete entries._lastResult;
    }
  }

  return entries;
}

/**
 * Formats initialization information from system init entry
 * @param {any} initEntry - The system init entry containing tools, mcp_servers, etc.
 * @returns {string} Formatted markdown string
 */
function formatInitializationSummary(initEntry) {
  let markdown = "";

  // Display model and session info
  if (initEntry.model) {
    markdown += `**Model:** ${initEntry.model}\n\n`;
  }

  // Display premium model information if available
  if (initEntry.model_info) {
    const modelInfo = initEntry.model_info;

    // Display model name and vendor
    if (modelInfo.name) {
      markdown += `**Model Name:** ${modelInfo.name}`;
      if (modelInfo.vendor) {
        markdown += ` (${modelInfo.vendor})`;
      }
      markdown += "\n\n";
    }

    // Display billing/premium information
    if (modelInfo.billing) {
      const billing = modelInfo.billing;
      if (billing.is_premium === true) {
        markdown += `**Premium Model:** Yes`;
        if (billing.multiplier && billing.multiplier !== 1) {
          markdown += ` (${billing.multiplier}x cost multiplier)`;
        }
        markdown += "\n";

        if (billing.restricted_to && Array.isArray(billing.restricted_to) && billing.restricted_to.length > 0) {
          markdown += `**Required Plans:** ${billing.restricted_to.join(", ")}\n`;
        }
        markdown += "\n";
      } else if (billing.is_premium === false) {
        markdown += `**Premium Model:** No\n\n`;
      }
    }
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

  return markdown;
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
 * Formats a tool use entry with its result using HTML details tags
 * @param {any} toolUse - The tool use object containing name, input, etc.
 * @param {any} toolResult - The corresponding tool result object
 * @returns {string} Formatted markdown string with HTML details
 */
function formatToolUseWithDetails(toolUse, toolResult) {
  const toolName = toolUse.name;
  const input = toolUse.input || {};

  // Skip TodoWrite
  if (toolName === "TodoWrite") {
    return "";
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
    metadata += ` \`${formatDuration(toolResult.duration_ms)}\``;
  }
  if (totalTokens > 0) {
    metadata += ` \`~${totalTokens}t\``;
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
      const relativePath = filePath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, "");
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
        // Generic tool formatting
        const keys = Object.keys(input);
        if (keys.length > 0) {
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
    // Build the details content with tool input parameters and response
    let detailsContent = "";

    // Add parameters section if we have input
    const inputKeys = Object.keys(input);
    if (inputKeys.length > 0) {
      detailsContent += "**Parameters:**\n\n";
      detailsContent += "``````json\n";
      detailsContent += JSON.stringify(input, null, 2);
      detailsContent += "\n``````\n\n";
    }

    // Add response section
    detailsContent += "**Response:**\n\n";
    detailsContent += "``````\n";
    detailsContent += details;
    detailsContent += "\n``````";

    return `<details>\n<summary>${summary}</summary>\n\n${detailsContent}\n</details>\n\n`;
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
      const provider = parts[1];
      const method = parts.slice(2).join("_");
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

  // Convert multi-line commands to single line
  let formatted = command.replace(/\n/g, " ").replace(/\r/g, " ").replace(/\t/g, " ").replace(/\s+/g, " ").trim();

  // Escape backticks to prevent markdown issues
  formatted = formatted.replace(/`/g, "\\`");

  // Truncate if too long
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
    parseCopilotLog,
    formatInitializationSummary,
    formatToolUseWithDetails,
    formatBashCommand,
    truncateString,
    formatMcpName,
    formatMcpParameters,
    estimateTokens,
    formatDuration,
  };
}

main();
