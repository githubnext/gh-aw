// @ts-check
/// <reference types="@actions/github-script" />

const { createEngineLogParser, generateConversationMarkdown, generateInformationSection, formatInitializationSummary, formatToolUse } = require("./log_parser_shared.cjs");

const main = createEngineLogParser({
  parserName: "Gemini",
  parseFunction: parseGeminiLog,
  supportsDirectories: false,
});

/**
 * Parse Gemini CLI JSONL log output and format as markdown.
 * Gemini CLI outputs one JSON object per line (JSONL) with typed entries:
 * - type "init": session initialization with model and session_id
 * - type "message": user/assistant messages, assistant uses delta:true for streaming chunks
 * - type "tool_use": tool invocations with tool_name, tool_id, and parameters
 * - type "tool_result": tool responses with tool_id, status, and output
 * - type "result": final stats with token usage, duration, and tool call count
 * @param {string} logContent - The raw log content to parse
 * @returns {{markdown: string, logEntries: Array, mcpFailures: Array<string>, maxTurnsHit: boolean}} Parsed log data
 */
function parseGeminiLog(logContent) {
  if (!logContent) {
    return {
      markdown: "## 🤖 Gemini\n\nNo log content provided.\n\n",
      logEntries: [],
      mcpFailures: [],
      maxTurnsHit: false,
    };
  }

  // Parse JSONL lines
  /** @type {Array<any>} */
  const rawEntries = [];
  for (const line of logContent.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || !trimmed.startsWith("{")) {
      continue;
    }
    try {
      rawEntries.push(JSON.parse(trimmed));
    } catch (_e) {
      // Skip non-JSON lines
    }
  }

  if (rawEntries.length === 0) {
    return {
      markdown: "## 🤖 Gemini\n\nLog format not recognized as Gemini JSONL.\n\n",
      logEntries: [],
      mcpFailures: [],
      maxTurnsHit: false,
    };
  }

  // Transform Gemini JSONL entries into canonical logEntries format
  const logEntries = transformGeminiEntries(rawEntries);

  // Extract the final result entry for stats
  const resultEntry = rawEntries.find(e => e.type === "result");

  // Generate conversation markdown using shared function
  const conversationResult = generateConversationMarkdown(logEntries, {
    formatToolCallback: (toolUse, toolResult) => formatToolUse(toolUse, toolResult, { includeDetailedParameters: false }),
    formatInitCallback: initEntry => formatInitializationSummary(initEntry, { includeSlashCommands: false }),
  });

  let markdown = conversationResult.markdown;

  // Add Information section using Gemini-specific stats from the result entry
  if (resultEntry && resultEntry.stats) {
    const stats = resultEntry.stats;
    const syntheticEntry = {
      usage: {
        input_tokens: stats.input_tokens || 0,
        output_tokens: stats.output_tokens || 0,
        cache_read_input_tokens: stats.cached || 0,
      },
      duration_ms: stats.duration_ms || 0,
      num_turns: stats.tool_calls || 0,
    };
    markdown += generateInformationSection(syntheticEntry);
  } else {
    markdown += generateInformationSection(null);
  }

  return {
    markdown,
    logEntries,
    mcpFailures: [],
    maxTurnsHit: false,
  };
}

/**
 * Checks whether a canonical log entry is an assistant text entry eligible for merging
 * with a subsequent streaming delta chunk.
 * @param {any} entry - The candidate last entry
 * @returns {boolean} True when the entry is a mergeable assistant text entry
 */
function isConsecutiveDeltaEntry(entry) {
  return entry && entry.type === "assistant" && entry.message && Array.isArray(entry.message.content) && entry.message.content.length === 1 && entry.message.content[0].type === "text";
}

/**
 * Transforms raw Gemini JSONL entries into the canonical logEntries format
 * used by the shared generateConversationMarkdown function.
 *
 * Gemini entry types and their canonical mappings:
 * - "init" → {type:"system", subtype:"init", model, session_id}
 * - "message" (assistant, delta:true) → merged into {type:"assistant", message:{content:[{type:"text"}]}}
 * - "tool_use" → {type:"assistant", message:{content:[{type:"tool_use", id, name, input}]}}
 * - "tool_result" → {type:"user", message:{content:[{type:"tool_result", tool_use_id, content, is_error}]}}
 *
 * @param {Array<any>} rawEntries - Raw parsed JSONL entries
 * @returns {Array<any>} Canonical log entries for generateConversationMarkdown
 */
function transformGeminiEntries(rawEntries) {
  /** @type {Array<any>} */
  const entries = [];

  for (const raw of rawEntries) {
    if (raw.type === "init") {
      entries.push({
        type: "system",
        subtype: "init",
        model: raw.model,
        session_id: raw.session_id,
      });
    } else if (raw.type === "message" && raw.role === "assistant") {
      const text = raw.content || "";
      if (!text.trim()) {
        continue;
      }
      // Merge consecutive streaming delta chunks into one assistant text entry
      const last = entries[entries.length - 1];
      if (raw.delta === true && isConsecutiveDeltaEntry(last)) {
        last.message.content[0].text += text;
      } else {
        entries.push({
          type: "assistant",
          message: {
            content: [{ type: "text", text }],
          },
        });
      }
    } else if (raw.type === "tool_use") {
      entries.push({
        type: "assistant",
        message: {
          content: [
            {
              type: "tool_use",
              id: raw.tool_id,
              name: raw.tool_name,
              input: raw.parameters || {},
            },
          ],
        },
      });
    } else if (raw.type === "tool_result") {
      const output = typeof raw.output === "string" ? raw.output : JSON.stringify(raw.output || "");
      entries.push({
        type: "user",
        message: {
          content: [
            {
              type: "tool_result",
              tool_use_id: raw.tool_id,
              content: output,
              is_error: raw.status !== "success",
            },
          ],
        },
      });
    }
  }

  return entries;
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    main,
    parseGeminiLog,
    transformGeminiEntries,
  };
}
