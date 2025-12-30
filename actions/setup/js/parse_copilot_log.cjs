// @ts-check
/// <reference types="@actions/github-script" />

const { createEngineLogParser, generateConversationMarkdown, generateInformationSection, formatInitializationSummary, formatToolUse, parseLogEntries } = require("./log_parser_shared.cjs");

const main = createEngineLogParser({
  parserName: "Copilot",
  parseFunction: parseCopilotLog,
  supportsDirectories: true,
});

/**
 * Extracts the premium request count from the log content using regex
 * @param {string} logContent - The raw log content as a string
 * @returns {number} The number of premium requests consumed (defaults to 1 if not found)
 */
function extractPremiumRequestCount(logContent) {
  // Try various patterns that might appear in the Copilot CLI output
  const patterns = [/premium\s+requests?\s+consumed:?\s*(\d+)/i, /(\d+)\s+premium\s+requests?\s+consumed/i, /consumed\s+(\d+)\s+premium\s+requests?/i];

  for (const pattern of patterns) {
    const match = logContent.match(pattern);
    if (match && match[1]) {
      const count = parseInt(match[1], 10);
      if (!isNaN(count) && count > 0) {
        return count;
      }
    }
  }

  // Default to 1 if no match found
  // For agentic workflows, 1 premium request is consumed per workflow run
  return 1;
}

/**
 * Parses Copilot CLI log content and converts it to markdown format
 * @param {string} logContent - The raw log content as a string
 * @returns {{markdown: string, logEntries: Array, mcpFailures?: string[], maxTurnsHit?: boolean}} Formatted result with markdown and metadata
 */
function parseCopilotLog(logContent) {
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
      // Try JSONL format using shared function
      logEntries = parseLogEntries(logContent);
    }
  }

  if (!logEntries || logEntries.length === 0) {
    return { markdown: "## Agent Log Summary\n\nLog format not recognized as Copilot JSON array or JSONL.\n", logEntries: [] };
  }

  // Generate conversation markdown using shared function
  const conversationResult = generateConversationMarkdown(logEntries, {
    formatToolCallback: (toolUse, toolResult) => formatToolUse(toolUse, toolResult, { includeDetailedParameters: true }),
    formatInitCallback: initEntry =>
      formatInitializationSummary(initEntry, {
        includeSlashCommands: false,
        modelInfoCallback: entry => {
          // Display premium model information if available (Copilot-specific)
          if (!entry.model_info) return "";

          const modelInfo = entry.model_info;
          let markdown = "";

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

          return markdown;
        },
      }),
  });

  let markdown = conversationResult.markdown;

  // Add Information section
  const lastEntry = logEntries[logEntries.length - 1];
  const initEntry = logEntries.find(entry => entry.type === "system" && entry.subtype === "init");

  markdown += generateInformationSection(lastEntry, {
    additionalInfoCallback: entry => {
      // Display premium request consumption if using a premium model
      const isPremiumModel = initEntry && initEntry.model_info && initEntry.model_info.billing && initEntry.model_info.billing.is_premium === true;
      if (isPremiumModel) {
        const premiumRequestCount = extractPremiumRequestCount(logContent);
        return `**Premium Requests Consumed:** ${premiumRequestCount}\n\n`;
      }
      return "";
    },
  });

  return { markdown, logEntries };
}

/**
 * Scans log content for tool execution errors and builds a map of failed tools
 * @param {string} logContent - Raw debug log content
 * @returns {Map<string, boolean>} Map of tool IDs/names to error status
 */
function scanForToolErrors(logContent) {
  const toolErrors = new Map();
  const lines = logContent.split("\n");

  // Track recent tool calls to associate errors with them
  const recentToolCalls = [];
  const MAX_RECENT_TOOLS = 10;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    // Look for tool_calls in data blocks (not in JSON arguments)
    // Only match if it's in a choices/message context
    if (line.includes('"tool_calls":') && !line.includes('\\"tool_calls\\"')) {
      // Next few lines should contain tool call details
      for (let j = i + 1; j < Math.min(i + 30, lines.length); j++) {
        const nextLine = lines[j];

        // Extract tool call ID
        const idMatch = nextLine.match(/"id":\s*"([^"]+)"/);
        // Extract function name (not arguments with escaped quotes)
        const nameMatch = nextLine.match(/"name":\s*"([^"]+)"/) && !nextLine.includes('\\"name\\"');

        if (idMatch) {
          const toolId = idMatch[1];
          // Keep looking for the name
          for (let k = j; k < Math.min(j + 10, lines.length); k++) {
            const nameLine = lines[k];
            const funcNameMatch = nameLine.match(/"name":\s*"([^"]+)"/);
            if (funcNameMatch && !nameLine.includes('\\"name\\"')) {
              const toolName = funcNameMatch[1];
              recentToolCalls.unshift({ id: toolId, name: toolName });
              if (recentToolCalls.length > MAX_RECENT_TOOLS) {
                recentToolCalls.pop();
              }
              break;
            }
          }
        }
      }
    }

    // Look for error messages
    const errorMatch = line.match(/\[ERROR\].*(?:Tool execution failed|Permission denied|Resource not accessible|Error executing tool)/i);
    if (errorMatch) {
      // Try to extract tool name from error line
      const toolNameMatch = line.match(/Tool execution failed:\s*([^\s]+)/i);
      const toolIdMatch = line.match(/tool_call_id:\s*([^\s]+)/i);

      if (toolNameMatch) {
        const toolName = toolNameMatch[1];
        toolErrors.set(toolName, true);
        // Also mark by ID if we can find it in recent calls
        const matchingTool = recentToolCalls.find(t => t.name === toolName);
        if (matchingTool) {
          toolErrors.set(matchingTool.id, true);
        }
      } else if (toolIdMatch) {
        toolErrors.set(toolIdMatch[1], true);
      } else if (recentToolCalls.length > 0) {
        // Mark the most recent tool call as failed
        const lastTool = recentToolCalls[0];
        toolErrors.set(lastTool.id, true);
        toolErrors.set(lastTool.name, true);
      }
    }
  }

  return toolErrors;
}

/**
 * Parses Copilot CLI debug log format and reconstructs the conversation flow
 * @param {string} logContent - Raw debug log content
 * @returns {Array} Array of log entries in structured format
 */
function parseDebugLogFormat(logContent) {
  const entries = [];
  const lines = logContent.split("\n");

  // First pass: scan for tool errors
  const toolErrors = scanForToolErrors(logContent);

  // Extract model information from the start
  let model = "unknown";
  let sessionId = null;
  let modelInfo = null;
  let tools = [];
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

  // Extract tools from "[DEBUG] Tools:" section
  // The format is: [DEBUG] Tools: \n[DEBUG] [\n  { "type": "function", "function": { "name": "..." } }\n]
  const toolsIndex = logContent.indexOf("[DEBUG] Tools:");
  if (toolsIndex !== -1) {
    // Find the start of the JSON array - look for a line that starts with [DEBUG] [
    // Skip past the "Tools:" line first
    const afterToolsLine = logContent.indexOf("\n", toolsIndex);
    let toolsStart = logContent.indexOf("[DEBUG] [", afterToolsLine);
    if (toolsStart !== -1) {
      // Find the actual '[' character after '[DEBUG] '
      toolsStart = logContent.indexOf("[", toolsStart + 7); // Skip '[DEBUG] ' which is 8 chars
    }
    if (toolsStart !== -1) {
      // Track brackets to find the end of the JSON array
      let bracketCount = 0;
      let inString = false;
      let escapeNext = false;
      let toolsEnd = -1;

      for (let i = toolsStart; i < logContent.length; i++) {
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

        if (char === "[") {
          bracketCount++;
        } else if (char === "]") {
          bracketCount--;
          if (bracketCount === 0) {
            toolsEnd = i + 1;
            break;
          }
        }
      }

      if (toolsEnd !== -1) {
        // Remove [DEBUG] prefixes from each line in the JSON
        let toolsJson = logContent.substring(toolsStart, toolsEnd);
        toolsJson = toolsJson.replace(/^\d{4}-\d{2}-\d{2}T[\d:.]+Z \[DEBUG\] /gm, "");

        try {
          const toolsArray = JSON.parse(toolsJson);
          // Extract tool names from the OpenAI function format
          // Format: [{ "type": "function", "function": { "name": "bash", ... } }, ...]
          if (Array.isArray(toolsArray)) {
            tools = toolsArray
              .map(tool => {
                if (tool.type === "function" && tool.function && tool.function.name) {
                  // Convert github-* names to mcp__github__* format for consistency
                  let name = tool.function.name;
                  if (name.startsWith("github-")) {
                    name = "mcp__github__" + name.substring(7);
                  } else if (name.startsWith("safe_outputs-")) {
                    name = name; // Keep safe_outputs names as-is
                  }
                  return name;
                }
                return null;
              })
              .filter(name => name !== null);
          }
        } catch (e) {
          // Failed to parse tools, continue without them
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
      // Check if this line starts with timestamp
      const hasTimestamp = line.match(/^\d{4}-\d{2}-\d{2}T[\d:.]+Z /);

      if (hasTimestamp) {
        // Strip the timestamp and [DEBUG] prefix to see what remains
        const cleanLine = line.replace(/^\d{4}-\d{2}-\d{2}T[\d:.]+Z \[DEBUG\] /, "");

        // If after stripping, the line starts with JSON characters, it's part of JSON
        // Otherwise, it's a new log entry and we should end the block
        const isJsonContent = /^[{\[}\]"]/.test(cleanLine) || cleanLine.trim().startsWith('"');

        if (!isJsonContent) {
          // This is a new log line (not JSON content) - end of JSON block, process what we have
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
                          const originalToolName = toolName; // Keep original for error matching
                          const toolId = toolCall.id || `tool_${Date.now()}_${Math.random()}`;
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

                          content.push({
                            type: "tool_use",
                            id: toolId,
                            name: toolName,
                            input: args,
                          });

                          // Check if this tool had an error (by ID or by name)
                          const hasError = toolErrors.has(toolId) || toolErrors.has(originalToolName);

                          // Create a corresponding tool result
                          toolResults.push({
                            type: "tool_result",
                            tool_use_id: toolId,
                            content: hasError ? "Permission denied or tool execution failed" : "", // Set error message if failed
                            is_error: hasError, // Mark as error if we detected failure
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

                // Accumulate usage/result entry from each response
                if (jsonData.usage) {
                  // Initialize accumulator if needed
                  // @ts-ignore - Dynamic property for accumulating usage data
                  if (!entries._accumulatedUsage) {
                    // @ts-ignore
                    entries._accumulatedUsage = {
                      input_tokens: 0,
                      output_tokens: 0,
                    };
                  }

                  // Accumulate token counts from this response
                  // OpenAI uses prompt_tokens/completion_tokens, normalize to input_tokens/output_tokens
                  if (jsonData.usage.prompt_tokens) {
                    // @ts-ignore
                    entries._accumulatedUsage.input_tokens += jsonData.usage.prompt_tokens;
                  }
                  if (jsonData.usage.completion_tokens) {
                    // @ts-ignore
                    entries._accumulatedUsage.output_tokens += jsonData.usage.completion_tokens;
                  }

                  // Store result entry with accumulated usage
                  // @ts-ignore - Dynamic property for storing last result
                  entries._lastResult = {
                    type: "result",
                    num_turns: turnCount,
                    // @ts-ignore
                    usage: entries._accumulatedUsage,
                  };
                }
              }
            } catch (e) {
              // Skip invalid JSON blocks
            }
          }

          inDataBlock = false;
          currentJsonLines = [];
          continue; // Don't add this line to JSON
        } else if (hasTimestamp && isJsonContent) {
          // This line has a timestamp but is JSON content - strip prefix and add
          currentJsonLines.push(cleanLine);
        }
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
                  const originalToolName = toolName;
                  const toolId = toolCall.id || `tool_${Date.now()}_${Math.random()}`;
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

                  content.push({
                    type: "tool_use",
                    id: toolId,
                    name: toolName,
                    input: args,
                  });

                  // Check if this tool had an error (by ID or by name)
                  const hasError = toolErrors.has(toolId) || toolErrors.has(originalToolName);

                  // Create a corresponding tool result
                  toolResults.push({
                    type: "tool_result",
                    tool_use_id: toolId,
                    content: hasError ? "Permission denied or tool execution failed" : "",
                    is_error: hasError,
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
          // Initialize accumulator if needed
          // @ts-ignore - Dynamic property for accumulating usage data
          if (!entries._accumulatedUsage) {
            // @ts-ignore
            entries._accumulatedUsage = {
              input_tokens: 0,
              output_tokens: 0,
            };
          }

          // Accumulate token counts from this response
          // OpenAI uses prompt_tokens/completion_tokens, normalize to input_tokens/output_tokens
          if (jsonData.usage.prompt_tokens) {
            // @ts-ignore
            entries._accumulatedUsage.input_tokens += jsonData.usage.prompt_tokens;
          }
          if (jsonData.usage.completion_tokens) {
            // @ts-ignore
            entries._accumulatedUsage.output_tokens += jsonData.usage.completion_tokens;
          }

          // Store result entry with accumulated usage
          // @ts-ignore - Dynamic property for storing last result
          entries._lastResult = {
            type: "result",
            num_turns: turnCount,
            // @ts-ignore
            usage: entries._accumulatedUsage,
          };
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
      tools: tools, // Tools extracted from [DEBUG] Tools: section
    };

    // Add model info if available
    if (modelInfo) {
      initEntry.model_info = modelInfo;
    }

    entries.unshift(initEntry);

    // Add the final result entry if we have it
    // @ts-ignore - Dynamic property for last result
    if (entries._lastResult) {
      // @ts-ignore
      entries.push(entries._lastResult);
      // @ts-ignore
      delete entries._lastResult;
    }
  }

  return entries;
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    main,
    parseCopilotLog,
    extractPremiumRequestCount,
  };
}
