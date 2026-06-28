// @ts-check
/// <reference types="@actions/github-script" />

/**
 * parse_eval_results.cjs
 *
 * BinEval Evaluation Result Parser (experimental)
 *
 * Parses the agentic engine's log file for the structured EVAL_RESULT marker
 * written by the engine during a BinEval evaluation run, then aggregates the
 * per-question results into a JSON summary and a markdown step summary.
 *
 * The engine writes its verdict to stdout which is piped through `tee -a` to
 * eval.log. This parser reads that file to extract EVAL_RESULT:{...json...}.
 *
 * Output files (written to GH_AW_EVAL_WORK_DIR):
 *   eval_results.json - Aggregated evaluation summary and per-question results
 */

"use strict";

const fs = require("fs");
const path = require("path");
const { aggregateResults, renderMarkdownSummary } = require("./eval_harness.cjs");
const { ERR_PARSE, ERR_SYSTEM } = require("./error_codes.cjs");

const DEFAULT_WORK_DIR = "/tmp/gh-aw/eval";
const EVAL_LOG_FILENAME = "eval.log";
const RESULT_PREFIX = "EVAL_RESULT:";

// ---------------------------------------------------------------------------
// Log parsing
// ---------------------------------------------------------------------------

/**
 * Extracts a complete JSON object from a string that begins with RESULT_PREFIX,
 * using brace counting to find the matching closing brace. Handles string contexts
 * and escape sequences correctly.
 *
 * @param {string} text - Text starting with RESULT_PREFIX
 * @returns {string|null} RESULT_PREFIX + complete JSON, or null
 */
function extractResultFromText(text) {
  const jsonStartPos = text.indexOf("{", RESULT_PREFIX.length);
  if (jsonStartPos === -1) return null;

  let depth = 0;
  let inString = false;
  let escaped = false;
  let jsonEndPos = -1;

  for (let i = jsonStartPos; i < text.length; i++) {
    const ch = text[i];
    if (escaped) {
      escaped = false;
      continue;
    }
    if (ch === "\\" && inString) {
      escaped = true;
      continue;
    }
    if (ch === '"') {
      inString = !inString;
      continue;
    }
    if (!inString) {
      if (ch === "{") depth++;
      else if (ch === "}") {
        depth--;
        if (depth === 0) {
          jsonEndPos = i;
          break;
        }
      }
    }
  }

  if (jsonEndPos === -1) return null;
  return text.slice(0, jsonEndPos + 1);
}

/**
 * Unwrap a stream-json encoded line: if the text is a JSON object with a
 * "result" string field that contains RESULT_PREFIX, extract that inner string.
 *
 * @param {string} line
 * @returns {string} The unwrapped line, or the original if not stream-json
 */
function extractFromStreamJson(line) {
  if (!line.includes(RESULT_PREFIX)) return line;
  try {
    const outer = JSON.parse(line);
    if (outer && typeof outer.result === "string" && outer.result.includes(RESULT_PREFIX)) {
      return outer.result;
    }
  } catch {
    // Not valid JSON — use the line as-is
  }
  return line;
}

/**
 * Parse the eval log file for the EVAL_RESULT marker.
 * Returns the parsed results array or an error string.
 *
 * @param {string} logContent - Contents of eval.log
 * @returns {{ results: Array<{id: string, passed: boolean, rationale?: string}> | null, error: string | null }}
 */
function parseEvalLog(logContent) {
  const lines = logContent.split("\n");
  for (const rawLine of lines) {
    const line = extractFromStreamJson(rawLine.trim());
    const idx = line.indexOf(RESULT_PREFIX);
    if (idx === -1) continue;
    const candidate = line.slice(idx);
    const extracted = extractResultFromText(candidate);
    if (!extracted) continue;
    const jsonStr = extracted.slice(RESULT_PREFIX.length);
    try {
      const escapedJson = jsonStr.replace(/\n/g, "\\n");
      const parsed = JSON.parse(escapedJson);
      if (parsed && Array.isArray(parsed.results)) {
        return { results: parsed.results, error: null };
      }
      return { results: null, error: "EVAL_RESULT JSON did not contain a 'results' array" };
    } catch (e) {
      return { results: null, error: `Failed to parse EVAL_RESULT JSON: ${e instanceof Error ? e.message : String(e)}` };
    }
  }
  return { results: null, error: "No EVAL_RESULT found in eval log" };
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

/**
 * Main entry point.
 * @returns {Promise<void>}
 */
async function main() {
  const workDir = process.env.GH_AW_EVAL_WORK_DIR || DEFAULT_WORK_DIR;
  const logPath = path.join(workDir, EVAL_LOG_FILENAME);

  core.info(`Parsing BinEval results from: ${logPath}`);

  // Verify log file exists
  if (!fs.existsSync(logPath)) {
    const msg = `${ERR_SYSTEM}: Eval log not found at ${logPath}`;
    core.error(msg);
    core.setFailed(msg);
    return;
  }

  let logContent;
  try {
    logContent = fs.readFileSync(logPath, "utf-8");
  } catch (/** @type {any} */ err) {
    const msg = `${ERR_SYSTEM}: Failed to read eval log: ${err.message}`;
    core.error(msg);
    core.setFailed(msg);
    return;
  }

  core.info(`Eval log: ${logContent.split("\n").length} lines, ${logContent.length} bytes`);

  // Parse the log
  const { results: rawResults, error } = parseEvalLog(logContent);
  if (error || !rawResults) {
    const msg = `${ERR_PARSE}: ${error || "No eval results found"}`;
    core.error(msg);
    core.info('Expected format: EVAL_RESULT:{"results":[{"id":"<id>","passed":true|false,"rationale":"..."},...]}');
    core.setFailed(msg);
    return;
  }

  // Normalise: ensure each entry has id (string) and passed (boolean)
  const normalised = rawResults
    .filter(r => r && typeof r.id === "string" && r.id)
    .map(r => ({
      id: r.id,
      passed: Boolean(r.passed),
      rationale: typeof r.rationale === "string" ? r.rationale.slice(0, 500) : undefined,
    }));

  // Aggregate
  const summary = aggregateResults(normalised);

  // Write JSON results
  fs.mkdirSync(workDir, { recursive: true });
  const resultsPath = path.join(workDir, "eval_results.json");
  fs.writeFileSync(resultsPath, JSON.stringify(summary, null, 2));
  core.info(`Eval results written to ${resultsPath}`);

  // Write step summary
  const markdownSummary = renderMarkdownSummary(summary);
  await core.summary.addRaw(markdownSummary).write();

  // Set outputs
  core.setOutput("eval_passed", String(summary.passed));
  core.setOutput("eval_total", String(summary.total));
  core.setOutput("eval_pass_rate", summary.pass_rate.toFixed(4));

  core.info(`BinEval complete: ${summary.passed}/${summary.total} passed (${(summary.pass_rate * 100).toFixed(1)}%)`);
}

module.exports = { main, parseEvalLog, extractResultFromText, extractFromStreamJson };
