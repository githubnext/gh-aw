// @ts-check

/**
 * Minimal dependency contract injected from log_parser_shared.cjs.
 * Keeping this explicit helps prevent silent drift between modules.
 *
 * @typedef {Object} LogParserFormatterDeps
 * @property {(command: string) => string} formatBashCommand
 * @property {(toolName: string) => string} formatMcpName
 * @property {(name: string, input: Object) => string} formatToolDisplayName
 * @property {(resultText: string, maxLineLength?: number) => string} formatResultPreview
 * @property {(options: {summary: string, statusIcon?: string, sections?: Array<{label: string, content: string, language?: string}>, metadata?: string, maxContentLength?: number}) => string} formatToolCallAsDetails
 * @property {(input: Record<string, any>) => string} formatMcpParameters
 * @property {(str: string, maxLength: number) => string} truncateString
 * @property {(text: string) => number} estimateTokens
 * @property {(ms: number) => string} formatDuration
 * @property {(text: string) => string} unfenceMarkdown
 * @property {number} MAX_AGENT_TEXT_LENGTH
 * @property {string} SIZE_LIMIT_WARNING
 */

/**
 * Public formatter API returned by createLogParserFormatters().
 *
 * @typedef {Object} LogParserFormatters
 * @property {(logEntries: Array<any>, options: {formatToolCallback: Function, formatInitCallback: Function, summaryTracker?: any}) => {markdown: string, commandSummary: Array<string>, sizeLimitReached: boolean}} generateConversationMarkdown
 * @property {(toolUse: any, toolResult: any, options?: {includeDetailedParameters?: boolean}) => string} formatToolUse
 * @property {(logEntries: Array<any>, options?: {model?: string, parserName?: string}) => string} generatePlainTextSummary
 * @property {(logEntries: Array<any>, options?: {model?: string, parserName?: string}) => string} generateCopilotCliStyleSummary
 */

/**
 * Creates formatter functions for log parsing summaries and rendering.
 * Dependencies are injected to avoid module cycles with log_parser_shared.cjs.
 *
 * @param {LogParserFormatterDeps} deps - Dependency injection container
 * @returns {LogParserFormatters} Formatter functions
 */
function createLogParserFormatters(deps) {
  const {
    formatBashCommand,
    formatMcpName,
    formatToolDisplayName,
    formatResultPreview,
    formatToolCallAsDetails,
    formatMcpParameters,
    truncateString,
    estimateTokens,
    formatDuration,
    unfenceMarkdown,
    MAX_AGENT_TEXT_LENGTH,
    SIZE_LIMIT_WARNING,
  } = deps;

  const INTERNAL_TOOLS = ["Read", "Write", "Edit", "MultiEdit", "LS", "Grep", "Glob", "TodoWrite"];

  /**
   * Generates markdown summary from conversation log entries
   * This is the core shared logic between Claude and Copilot log parsers
   *
   * When a summaryTracker is provided, the function tracks the accumulated size
   * and stops rendering additional content when approaching the step summary limit.
   *
   * @param {Array} logEntries - Array of log entries with type, message, etc.
   * @param {Object} options - Configuration options
   * @param {Function} options.formatToolCallback - Callback function to format tool use (content, toolResult) => string
   * @param {Function} options.formatInitCallback - Callback function to format initialization (initEntry) => string or {markdown: string, mcpFailures: string[]}
   * @param {any} [options.summaryTracker] - Optional tracker for step summary size limits
   * @returns {{markdown: string, commandSummary: Array<string>, sizeLimitReached: boolean}} Generated markdown, command summary, and size limit status
   */
  function generateConversationMarkdown(logEntries, options) {
    const { formatToolCallback, formatInitCallback, summaryTracker } = options;

    const toolUsePairs = collectToolUsePairs(logEntries);

    let markdown = "";
    let sizeLimitReached = false;

    function addContent(content) {
      if (summaryTracker && !summaryTracker.add(content)) {
        sizeLimitReached = true;
        return false;
      }
      markdown += content;
      return true;
    }

    const initEntry = logEntries.find(entry => entry.type === "system" && entry.subtype === "init");

    if (initEntry && formatInitCallback) {
      if (!addContent("## 🚀 Initialization\n\n")) {
        return { markdown, commandSummary: [], sizeLimitReached };
      }
      const initResult = formatInitCallback(initEntry);
      if (typeof initResult === "string") {
        if (!addContent(initResult)) {
          return { markdown, commandSummary: [], sizeLimitReached };
        }
      } else if (initResult && initResult.markdown) {
        if (!addContent(initResult.markdown)) {
          return { markdown, commandSummary: [], sizeLimitReached };
        }
      }
      if (!addContent("\n")) {
        return { markdown, commandSummary: [], sizeLimitReached };
      }
    }

    if (!addContent("\n## 🤖 Reasoning\n\n")) {
      return { markdown, commandSummary: [], sizeLimitReached };
    }

    for (const entry of logEntries) {
      if (sizeLimitReached) break;

      if (entry.type === "assistant" && entry.message?.content) {
        for (const content of entry.message.content) {
          if (sizeLimitReached) break;

          if (content.type === "text" && content.text) {
            let text = content.text.trim();
            text = unfenceMarkdown(text);
            if (text && text.length > 0) {
              if (!addContent(text + "\n\n")) {
                break;
              }
            }
          } else if (content.type === "tool_use") {
            const toolResult = toolUsePairs.get(content.id);
            const toolMarkdown = formatToolCallback(content, toolResult);
            if (toolMarkdown) {
              if (!addContent(toolMarkdown)) {
                break;
              }
            }
          }
        }
      }
    }

    if (sizeLimitReached) {
      markdown += SIZE_LIMIT_WARNING;
      return { markdown, commandSummary: [], sizeLimitReached };
    }

    if (!addContent("## 🤖 Commands and Tools\n\n")) {
      markdown += SIZE_LIMIT_WARNING;
      return { markdown, commandSummary: [], sizeLimitReached: true };
    }

    const commandSummary = [];

    for (const entry of logEntries) {
      if (entry.type === "assistant" && entry.message?.content) {
        for (const content of entry.message.content) {
          if (content.type === "tool_use") {
            const toolName = content.name;
            const input = content.input || {};

            if (INTERNAL_TOOLS.includes(toolName)) {
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
        if (!addContent(`${cmd}\n`)) {
          markdown += SIZE_LIMIT_WARNING;
          return { markdown, commandSummary, sizeLimitReached: true };
        }
      }
    } else if (!addContent("No commands or tools used.\n")) {
      markdown += SIZE_LIMIT_WARNING;
      return { markdown, commandSummary, sizeLimitReached: true };
    }

    return { markdown, commandSummary, sizeLimitReached };
  }

  /**
   * Formats a tool use entry with its result into markdown
   * @param {any} toolUse - The tool use object containing name, input, etc.
   * @param {any} toolResult - The corresponding tool result object
   * @param {Object} options - Configuration options
   * @param {boolean} [options.includeDetailedParameters] - Whether to include detailed parameter section (default: false)
   * @returns {string} Formatted markdown string
   */
  function formatToolUse(toolUse, toolResult, options = {}) {
    const { includeDetailedParameters = false } = options;
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

    const statusIcon = getStatusIcon();
    let summary = "";
    let details = "";

    if (toolResult && toolResult.content) {
      if (typeof toolResult.content === "string") {
        details = toolResult.content;
      } else if (Array.isArray(toolResult.content)) {
        details = toolResult.content.map(c => (typeof c === "string" ? c : c.text || "")).join("\n");
      }
    }

    const inputText = JSON.stringify(input);
    const outputText = details;
    const totalTokens = estimateTokens(inputText) + estimateTokens(outputText);

    let metadata = "";
    if (toolResult && toolResult.duration_ms) {
      metadata += `<code>${formatDuration(toolResult.duration_ms)}</code> `;
    }
    if (totalTokens > 0) {
      metadata += `<code>~${totalTokens}t</code>`;
    }
    metadata = metadata.trim();

    switch (toolName) {
      case "Bash": {
        const command = input.command || "";
        const description = input.description || "";
        const formattedCommand = formatBashCommand(command);

        if (description) {
          summary = `${description}: <code>${formattedCommand}</code>`;
        } else {
          summary = `<code>${formattedCommand}</code>`;
        }
        break;
      }

      case "Read": {
        const filePath = input.file_path || input.path || "";
        const relativePath = filePath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, "");
        summary = `Read <code>${relativePath}</code>`;
        break;
      }

      case "Write":
      case "Edit":
      case "MultiEdit": {
        const writeFilePath = input.file_path || input.path || "";
        const writeRelativePath = writeFilePath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, "");
        summary = `Write <code>${writeRelativePath}</code>`;
        break;
      }

      case "Grep":
      case "Glob": {
        const query = input.query || input.pattern || "";
        summary = `Search for <code>${truncateString(query, 80)}</code>`;
        break;
      }

      case "LS": {
        const lsPath = input.path || "";
        const lsRelativePath = lsPath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, "");
        summary = `LS: ${lsRelativePath || lsPath}`;
        break;
      }

      default:
        if (toolName.startsWith("mcp__")) {
          const mcpName = formatMcpName(toolName);
          const params = formatMcpParameters(input);
          summary = `${mcpName}(${params})`;
        } else {
          const keys = Object.keys(input);
          if (keys.length > 0) {
            const mainParam = keys.find(k => ["query", "command", "path", "file_path", "content"].includes(k)) || keys[0];
            const value = String(input[mainParam] || "");

            if (value) {
              summary = `${toolName}: ${truncateString(value, 100)}`;
            } else {
              summary = toolName;
            }
          } else {
            summary = toolName;
          }
        }
    }

    /** @type {Array<{label: string, content: string, language?: string}>} */
    const sections = [];

    if (includeDetailedParameters) {
      const inputKeys = Object.keys(input);
      if (inputKeys.length > 0) {
        sections.push({
          label: "Parameters",
          content: JSON.stringify(input, null, 2),
          language: "json",
        });
      }
    }

    if (details && details.trim()) {
      sections.push({
        label: includeDetailedParameters ? "Response" : "Output",
        content: details,
      });
    }

    return formatToolCallAsDetails({
      summary,
      statusIcon,
      sections,
      metadata: metadata || undefined,
    });
  }

  function collectToolUsePairs(logEntries) {
    const toolUsePairs = new Map();
    for (const entry of logEntries) {
      if (entry.type === "user" && entry.message?.content) {
        for (const content of entry.message.content) {
          if (content.type === "tool_result" && content.tool_use_id) {
            toolUsePairs.set(content.tool_use_id, content);
          }
        }
      }
    }
    return toolUsePairs;
  }

  function appendConversationLine(lines, line, state) {
    if (state.conversationLineCount >= state.maxConversationLines) {
      state.conversationTruncated = true;
      return false;
    }
    lines.push(line);
    state.conversationLineCount++;
    return true;
  }

  function appendAgentText(lines, text, state) {
    let displayText = text;
    if (displayText.length > MAX_AGENT_TEXT_LENGTH) {
      displayText = displayText.substring(0, MAX_AGENT_TEXT_LENGTH) + `... [truncated: showing first ${MAX_AGENT_TEXT_LENGTH} of ${text.length} chars]`;
    }

    const textLines = displayText.split("\n");
    for (let i = 0; i < textLines.length; i++) {
      const prefix = i === 0 ? "◆ " : "  ";
      if (!appendConversationLine(lines, `${prefix}${textLines[i]}`, state)) {
        return;
      }
    }
    appendConversationLine(lines, "", state);
  }

  function appendToolExecutionLine(lines, content, toolUsePairs, state) {
    const toolName = content.name;
    const input = content.input || {};

    if (INTERNAL_TOOLS.includes(toolName)) {
      return;
    }

    const toolResult = toolUsePairs.get(content.id);
    const isError = toolResult?.is_error === true;
    const statusIcon = isError ? "✗" : "✓";

    let displayName;
    let resultPreview = "";

    if (toolName === "Bash") {
      const cmd = formatBashCommand(input.command || "");
      displayName = `$ ${cmd}`;

      if (toolResult && toolResult.content) {
        const resultText = typeof toolResult.content === "string" ? toolResult.content : String(toolResult.content);
        resultPreview = formatResultPreview(resultText);
      }
    } else if (toolName.startsWith("mcp__")) {
      const formattedName = formatMcpName(toolName).replace("::", "-");
      displayName = formatToolDisplayName(formattedName, input);

      if (toolResult && toolResult.content) {
        const resultText = typeof toolResult.content === "string" ? toolResult.content : JSON.stringify(toolResult.content);
        resultPreview = formatResultPreview(resultText);
      }
    } else {
      displayName = formatToolDisplayName(toolName, input);

      if (toolResult && toolResult.content) {
        const resultText = typeof toolResult.content === "string" ? toolResult.content : String(toolResult.content);
        resultPreview = formatResultPreview(resultText);
      }
    }

    if (!appendConversationLine(lines, `${statusIcon} ${displayName}`, state)) {
      return;
    }

    if (resultPreview) {
      for (const previewLine of resultPreview.split("\n")) {
        if (!appendConversationLine(lines, previewLine, state)) {
          return;
        }
      }
    }

    appendConversationLine(lines, "", state);
  }

  function appendStatistics(lines, logEntries, toolUsePairs) {
    const lastEntry = logEntries[logEntries.length - 1];
    lines.push("Statistics:");
    if (lastEntry?.num_turns) {
      lines.push(`  Turns: ${lastEntry.num_turns}`);
    }
    if (lastEntry?.duration_ms) {
      const duration = formatDuration(lastEntry.duration_ms);
      if (duration) {
        lines.push(`  Duration: ${duration}`);
      }
    }

    let toolCounts = { total: 0, success: 0, error: 0 };
    for (const entry of logEntries) {
      if (entry.type === "assistant" && entry.message?.content) {
        for (const content of entry.message.content) {
          if (content.type === "tool_use") {
            const toolName = content.name;
            if (INTERNAL_TOOLS.includes(toolName)) {
              continue;
            }
            toolCounts.total++;
            const toolResult = toolUsePairs.get(content.id);
            const isError = toolResult?.is_error === true;
            if (isError) {
              toolCounts.error++;
            } else {
              toolCounts.success++;
            }
          }
        }
      }
    }

    if (toolCounts.total > 0) {
      lines.push(`  Tools: ${toolCounts.success}/${toolCounts.total} succeeded`);
    }
    if (lastEntry?.usage) {
      const usage = lastEntry.usage;
      if (usage.input_tokens || usage.output_tokens) {
        const inputTokens = usage.input_tokens || 0;
        const outputTokens = usage.output_tokens || 0;
        const cacheCreationTokens = usage.cache_creation_input_tokens || 0;
        const cacheReadTokens = usage.cache_read_input_tokens || 0;
        const totalTokens = inputTokens + outputTokens + cacheCreationTokens + cacheReadTokens;

        lines.push(`  Tokens: ${totalTokens.toLocaleString()} total (${inputTokens.toLocaleString()} in / ${outputTokens.toLocaleString()} out)`);
      }
    }
    if (lastEntry?.total_cost_usd) {
      lines.push(`  Cost: $${lastEntry.total_cost_usd.toFixed(4)}`);
    }
  }

  function generateSummaryLines(logEntries) {
    const lines = [];
    const toolUsePairs = collectToolUsePairs(logEntries);

    const state = {
      conversationLineCount: 0,
      maxConversationLines: 5000,
      conversationTruncated: false,
    };

    for (const entry of logEntries) {
      if (state.conversationLineCount >= state.maxConversationLines) {
        state.conversationTruncated = true;
        break;
      }

      if (entry.type === "assistant" && entry.message?.content) {
        for (const content of entry.message.content) {
          if (state.conversationLineCount >= state.maxConversationLines) {
            state.conversationTruncated = true;
            break;
          }

          if (content.type === "text" && content.text) {
            let text = content.text.trim();
            text = unfenceMarkdown(text);
            if (text && text.length > 0) {
              appendAgentText(lines, text, state);
            }
          } else if (content.type === "tool_use") {
            appendToolExecutionLine(lines, content, toolUsePairs, state);
          }
        }
      }
    }

    if (state.conversationTruncated) {
      lines.push("... (conversation truncated)");
      lines.push("");
    }

    appendStatistics(lines, logEntries, toolUsePairs);

    return lines;
  }

  /**
   * Generates plain-text Copilot CLI style summary for logs.
   * @param {Array} logEntries - Array of log entries with type, message, etc.
   * @param {Object} options - Configuration options
   * @param {string} [options.model] - Model name to include in the header
   * @param {string} [options.parserName] - Name of the parser (e.g., "Copilot", "Claude")
   * @returns {string} Plain text summary for console output
   */
  function generatePlainTextSummary(logEntries, options = {}) {
    const { model, parserName = "Agent" } = options;
    const lines = [];

    lines.push(`=== ${parserName} Execution Summary ===`);
    if (model) {
      lines.push(`Model: ${model}`);
    }
    lines.push("");

    lines.push("Conversation:");
    lines.push("");

    lines.push(...generateSummaryLines(logEntries));

    return lines.join("\n");
  }

  /**
   * Generates a markdown-formatted Copilot CLI style summary for step summaries.
   * @param {Array} logEntries - Array of log entries with type, message, etc.
   * @param {Object} options - Configuration options
   * @param {string} [options.model] - Model name to include in the header
   * @param {string} [options.parserName] - Name of the parser (e.g., "Copilot", "Claude")
   * @returns {string} Markdown-formatted summary for step summary rendering
   */
  function generateCopilotCliStyleSummary(logEntries, options = {}) {
    const lines = [];

    lines.push("```");
    lines.push("Conversation:");
    lines.push("");

    lines.push(...generateSummaryLines(logEntries));

    lines.push("```");

    return lines.join("\n");
  }

  return {
    generateConversationMarkdown,
    formatToolUse,
    generatePlainTextSummary,
    generateCopilotCliStyleSummary,
  };
}

module.exports = createLogParserFormatters;
