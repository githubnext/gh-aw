// @ts-check
/// <reference types="@actions/github-script" />

const { generatePlainTextSummary, generateCopilotCliStyleSummary, wrapAgentLogInSection, formatSafeOutputsPreview } = require("./log_parser_shared.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_API, ERR_CONFIG, ERR_VALIDATION } = require("./error_codes.cjs");

/**
 * Bootstrap helper for log parser entry points.
 * Handles common logic for environment variable lookup, file existence checks,
 * content reading (file or directory), and summary emission.
 *
 * @param {Object} options - Configuration options
 * @param {function(string): string|{markdown: string, mcpFailures?: string[], maxTurnsHit?: boolean, logEntries?: Array}} options.parseLog - Parser function that takes log content and returns markdown or result object
 * @param {string} options.parserName - Name of the parser (e.g., "Codex", "Claude", "Copilot")
 * @param {boolean} [options.supportsDirectories=false] - Whether the parser supports reading from directories
 * @returns {Promise<void>}
 */
async function runLogParser(options) {
  const fs = require("fs");
  const path = require("path");
  const { parseLog, parserName, supportsDirectories = false } = options;

  /**
   * Recursively searches a directory tree for the first events.jsonl file.
   * This file is written by the Copilot CLI and contains structured session events.
   * @param {string} dirPath - Directory to search
   * @returns {string|null} Absolute path to events.jsonl, or null if not found
   */
  function findEventsJsonlRecursive(dirPath) {
    try {
      const entries = fs.readdirSync(dirPath, { withFileTypes: true });
      for (const entry of entries) {
        const fullPath = path.join(dirPath, entry.name);
        if (entry.isDirectory()) {
          const found = findEventsJsonlRecursive(fullPath);
          if (found) return found;
        } else if (entry.name === "events.jsonl") {
          return fullPath;
        }
      }
    } catch (e) {
      // Ignore read errors (e.g. permission denied on subdirectories)
    }
    return null;
  }

  /**
   * Count valid JSONL entries from a safe outputs file.
   * @param {string} content - Raw safe outputs JSONL content
   * @returns {number} Number of valid entries
   */
  function countSafeOutputEntries(content) {
    if (!content || content.trim().length === 0) {
      return 0;
    }

    let count = 0;
    const lines = content.trim().split(/\r?\n/);
    for (const line of lines) {
      const trimmedLine = line.trim();
      if (!trimmedLine) {
        continue;
      }
      try {
        JSON.parse(trimmedLine);
        count++;
      } catch (e) {
        // Ignore invalid JSONL lines
      }
    }
    return count;
  }

  /**
   * Returns true if the log entries show the agent ran at least one turn.
   *
   * "At least one turn" is used (rather than "all work finished") because the
   * log only records the turn count, not whether every intended task succeeded.
   * The check is sufficient to distinguish a post-completion MCP relaunch
   * failure (the agent was already executing) from a startup failure where the
   * MCP never launched and the agent ran zero turns.
   *
   * Handles both log formats:
   *   - Legacy format (Codex, Copilot, etc.): { type: "result", num_turns: N }
   *   - Copilot event format (Claude): { type: "session.result", data: { numTurns: N } }
   *
   * @param {Array|null|undefined} entries
   * @returns {boolean}
   */
  function agentRanToCompletion(entries) {
    if (!entries || !Array.isArray(entries) || entries.length === 0) {
      return false;
    }
    return entries.some(e => {
      if (!e || typeof e !== "object") return false;
      // Legacy format
      if (e.type === "result" && typeof e.num_turns === "number" && e.num_turns > 0) return true;
      // Copilot event format (Claude)
      if (e.type === "session.result" && e.data && typeof e.data.numTurns === "number" && e.data.numTurns > 0) return true;
      return false;
    });
  }

  try {
    const logPath = process.env.GH_AW_AGENT_OUTPUT;
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
      if (!supportsDirectories) {
        core.info(`Log path is a directory but ${parserName} parser does not support directories: ${logPath}`);
        return;
      }

      // Prefer events.jsonl (structured Copilot session format) over debug .log files
      const eventsJsonlPath = findEventsJsonlRecursive(logPath);
      if (eventsJsonlPath) {
        core.info(`Using Copilot session events from: ${eventsJsonlPath}`);
        content = fs.readFileSync(eventsJsonlPath, "utf8");
      } else {
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

          // Add a newline before this file if the previous content doesn't end with one
          if (content.length > 0 && !content.endsWith("\n")) {
            content += "\n";
          }

          content += fileContent;
        }
      }
    } else {
      // Read the single log file
      content = fs.readFileSync(logPath, "utf8");
    }

    const result = parseLog(content);

    // Handle result that may be a simple string or an object with metadata
    let markdown = "";
    let mcpFailures = [];
    let maxTurnsHit = false;
    let logEntries = null;

    if (typeof result === "string") {
      markdown = result;
    } else if (result && typeof result === "object") {
      markdown = result.markdown || "";
      mcpFailures = result.mcpFailures || [];
      maxTurnsHit = result.maxTurnsHit || false;
      logEntries = result.logEntries || null;
    }

    // Enrich agent-stdio.log with a normalized result entry when the engine does not
    // write one directly (e.g. Copilot, Pi).  The OTEL conclusion span
    // (send_otlp_span.cjs → readAgentRuntimeMetrics) reads agent-stdio.log for the
    // gh-aw.turns attribute and token usage; without this entry those fields are zero
    // for every engine except Claude Code, leaving 80 % of fleet runs un-triageable.
    //
    // Safety rules:
    //  1. Only append when agent-stdio.log does NOT already contain a result entry
    //     (avoids double-counting on Claude Code runs where the entry is written by
    //     the --debug-file flag).
    //  2. The appended line must be a standalone JSON object on its own line so that
    //     the existing line-oriented parser in readAgentRuntimeMetrics can find it.
    //  3. All errors are non-fatal – telemetry enrichment must never break workflows.
    if (logEntries && Array.isArray(logEntries)) {
      const resultEntry = logEntries.find(e => e && typeof e === "object" && e.type === "result" && (typeof e.num_turns === "number" || e.usage));
      if (resultEntry) {
        const normalizedResultEntry = {
          type: "result",
          num_turns: typeof resultEntry.num_turns === "number" && Number.isFinite(resultEntry.num_turns) && resultEntry.num_turns >= 0 ? resultEntry.num_turns : 0,
          usage: {
            input_tokens: typeof resultEntry.usage?.input_tokens === "number" && Number.isFinite(resultEntry.usage.input_tokens) && resultEntry.usage.input_tokens >= 0 ? resultEntry.usage.input_tokens : 0,
            output_tokens: typeof resultEntry.usage?.output_tokens === "number" && Number.isFinite(resultEntry.usage.output_tokens) && resultEntry.usage.output_tokens >= 0 ? resultEntry.usage.output_tokens : 0,
          },
        };
        const stdioLogPath = "/tmp/gh-aw/agent-stdio.log";
        try {
          let alreadyHasResult = false;
          if (fs.existsSync(stdioLogPath)) {
            const stdioContent = fs.readFileSync(stdioLogPath, "utf8");
            alreadyHasResult = stdioContent.split("\n").some(line => {
              const objectStart = line.indexOf("{");
              const arrayStart = line.indexOf("[");
              let start = -1;
              if (objectStart >= 0 && arrayStart >= 0) {
                start = Math.min(objectStart, arrayStart);
              } else if (objectStart >= 0) {
                start = objectStart;
              } else {
                start = arrayStart;
              }
              if (start < 0) return false;
              try {
                const parsed = JSON.parse(line.slice(start));
                if (Array.isArray(parsed)) {
                  return parsed.some(entry => entry && typeof entry === "object" && entry.type === "result");
                }
                return parsed && parsed.type === "result";
              } catch {
                return false;
              }
            });
          }
          if (!alreadyHasResult) {
            fs.mkdirSync(path.dirname(stdioLogPath), { recursive: true });
            fs.appendFileSync(stdioLogPath, JSON.stringify(normalizedResultEntry) + "\n");
            core.info(`[log-parser] Wrote ${parserName} result entry to agent-stdio.log: num_turns=${normalizedResultEntry.num_turns ?? "n/a"}`);
          }
        } catch (err) {
          core.warning(`[log-parser] Failed to enrich agent-stdio.log with result entry: ${getErrorMessage(err)}`);
        }
      }
    }

    // Read safe outputs file if available
    let safeOutputsContent = "";
    let safeOutputEntriesCount = 0;
    const safeOutputsPath = process.env.GH_AW_SAFE_OUTPUTS;
    if (safeOutputsPath && fs.existsSync(safeOutputsPath)) {
      try {
        safeOutputsContent = fs.readFileSync(safeOutputsPath, "utf8");
        safeOutputEntriesCount = countSafeOutputEntries(safeOutputsContent);
      } catch (error) {
        core.warning(`Failed to read safe outputs file: ${getErrorMessage(error)}`);
      }
    }

    if (markdown) {
      // Generate lightweight plain text summary for core.info and Copilot CLI style for step summary
      if (logEntries && Array.isArray(logEntries) && logEntries.length > 0) {
        // Extract model from init entry if available
        const initEntry = logEntries.find(entry => (entry.type === "system" && entry.subtype === "init") || entry.type === "session.init");
        const model = initEntry?.model || initEntry?.data?.model || null;

        const plainTextSummary = generatePlainTextSummary(logEntries, {
          model,
          parserName,
        });
        core.info(plainTextSummary);

        // Add safe outputs preview to core.info
        if (safeOutputsContent) {
          const safeOutputsPlainText = formatSafeOutputsPreview(safeOutputsContent, { isPlainText: true });
          if (safeOutputsPlainText) {
            core.info(safeOutputsPlainText);
          }
        }

        // Generate Copilot CLI style markdown for step summary
        const copilotCliStyleMarkdown = generateCopilotCliStyleSummary(logEntries, {
          model,
          parserName,
        });

        // Wrap the agent log in a details/summary section (open by default)
        const wrappedAgentLog = wrapAgentLogInSection(copilotCliStyleMarkdown, {
          parserName,
          open: true,
        });

        // Add safe outputs preview to step summary
        let fullMarkdown = wrappedAgentLog;
        if (safeOutputsContent) {
          const safeOutputsMarkdown = formatSafeOutputsPreview(safeOutputsContent, { isPlainText: false });
          if (safeOutputsMarkdown) {
            fullMarkdown += "\n" + safeOutputsMarkdown;
          }
        }

        await core.summary.addRaw(fullMarkdown).write();
      } else {
        // Fallback path: markdown exists but no structured log entries were parsed.
        // Suppress the "parsed successfully" message for Claude since it always produces
        // logEntries when healthy — absence of entries means the parse fell back and is
        // about to emit a guardrail warning/failure below.
        if (parserName !== "Claude") {
          core.info(`${parserName} log parsed successfully`);
        }

        // Add safe outputs preview to core.info (fallback path)
        if (safeOutputsContent) {
          const safeOutputsPlainText = formatSafeOutputsPreview(safeOutputsContent, { isPlainText: true });
          if (safeOutputsPlainText) {
            core.info(safeOutputsPlainText);
          }
        }

        // Wrap the original markdown in a details/summary section (open by default)
        const wrappedAgentLog = wrapAgentLogInSection(markdown, {
          parserName,
          open: true,
        });

        // Write wrapped markdown to step summary if available
        let fullMarkdown = wrappedAgentLog;
        if (safeOutputsContent) {
          const safeOutputsMarkdown = formatSafeOutputsPreview(safeOutputsContent, { isPlainText: false });
          if (safeOutputsMarkdown) {
            fullMarkdown += "\n" + safeOutputsMarkdown;
          }
        }
        await core.summary.addRaw(fullMarkdown).write();
      }
    } else {
      core.error(`Failed to parse ${parserName} log`);
    }

    // Claude-specific guardrail: if no structured log entries were parsed, treat as execution failure.
    // This catches silent startup failures where Claude exits before producing JSON tool activity.
    // Exception: when safeOutputEntriesCount > 0 the agent demonstrably completed and emitted
    // safe outputs — treat as a non-fatal post-completion infrastructure failure (e.g. sandbox
    // teardown race leaving agent-stdio.log unreadable) and downgrade to a warning.
    if (parserName === "Claude" && (!logEntries || logEntries.length === 0)) {
      if (safeOutputEntriesCount > 0) {
        core.warning(
          `Claude produced no structured log entries, but agent completed with ${safeOutputEntriesCount} safe output ${safeOutputEntriesCount === 1 ? "entry" : "entries"} — treating as non-fatal post-completion infrastructure failure`
        );
      } else {
        core.setFailed(`${ERR_CONFIG}: Claude execution failed: no structured log entries were produced. This usually indicates a startup or configuration error before tool execution.`);
      }
    }

    // Handle MCP server failures if present
    if (mcpFailures && mcpFailures.length > 0) {
      const failedServers = mcpFailures.join(", ");
      if (safeOutputEntriesCount > 0) {
        core.warning(`MCP server(s) failed to launch (${failedServers}), but agent completed with ${safeOutputEntriesCount} safe output ${safeOutputEntriesCount === 1 ? "entry" : "entries"}`);
      } else if (agentRanToCompletion(logEntries)) {
        // The agent ran turns to completion even though an MCP server failed to launch.
        // This is a post-completion relaunch/health-probe failure — the MCP server was
        // healthy during execution (the agent used it throughout the run) and the failure
        // occurred after the work was done.  Treat as non-fatal so genuine task success
        // is not masked by a transient infrastructure event.
        core.warning(`MCP server(s) failed to launch (${failedServers}), but agent completed turns — treating as non-fatal post-completion relaunch`);
      } else {
        core.setFailed(`${ERR_API}: MCP server(s) failed to launch: ${failedServers}`);
      }
    }

    // Handle max-turns limit if hit
    if (maxTurnsHit) {
      core.setFailed(`${ERR_VALIDATION}: Agent execution stopped: max-turns limit reached. The agent did not complete its task successfully.`);
    }
  } catch (error) {
    core.setFailed(`${ERR_API}: ${error instanceof Error ? error.message : String(error)}`);
  }
}

// Export for testing and usage
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    runLogParser,
  };
}
