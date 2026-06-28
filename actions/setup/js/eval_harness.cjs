// @ts-check
/// <reference types="@actions/github-script" />

/**
 * eval_harness.cjs
 *
 * BinEval Evaluation Harness utilities (experimental)
 *
 * Shared utility functions used by the BinEval evaluation pipeline:
 *   - setup_eval.cjs      (prompt setup — writes /tmp/gh-aw/aw-prompts/prompt.txt)
 *   - parse_eval_results.cjs (result parsing — reads engine log, writes eval_results.json)
 *
 * Inference is performed by the configured agentic engine running inside AWF,
 * not by direct API calls from this module.
 */

"use strict";

// ---------------------------------------------------------------------------
// Types (JSDoc)
// ---------------------------------------------------------------------------

/**
 * @typedef {Object} EvalDefinition
 * @property {string} id
 * @property {string} question
 */

/**
 * @typedef {Object} EvalResult
 * @property {string} id
 * @property {boolean} passed
 * @property {string} [rationale]
 * @property {number} [confidence]
 */

/**
 * @typedef {Object} EvalSummary
 * @property {number} total
 * @property {number} passed
 * @property {number} failed
 * @property {number} pass_rate
 * @property {EvalResult[]} results
 */

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Reads the eval specification from the GH_AW_EVAL_SPEC environment variable.
 * @returns {EvalDefinition[]}
 */
function readEvalSpec() {
  const raw = process.env.GH_AW_EVAL_SPEC || "[]";
  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      throw new Error("GH_AW_EVAL_SPEC must be a JSON array");
    }
    return parsed.filter(e => e && typeof e.id === "string" && e.id && typeof e.question === "string" && e.question);
  } catch (err) {
    throw new Error(`Failed to parse GH_AW_EVAL_SPEC: ${err.message}`);
  }
}

/**
 * Sanitizes an error message before including it in eval artifacts or logs.
 * Redacts tokens, URLs, and control characters to prevent credential leaks.
 * @param {unknown} err
 * @returns {string}
 */
function sanitizeEvalError(err) {
  const raw = err instanceof Error ? err.message : String(err ?? "unknown error");
  const sanitized = raw
    .replace(/Bearer\s+[A-Za-z0-9._-]+/gi, "[REDACTED_TOKEN]")
    .replace(/\*{4,}/g, "[REDACTED_TOKEN]")
    .replace(/\b[A-Za-z0-9._-]*token[A-Za-z0-9._-]*\b/gi, "[REDACTED_TOKEN]")
    .replace(/\b(gh[pousr]_[A-Za-z0-9_]+)\b/g, "[REDACTED_TOKEN]")
    .replace(/https?:\/\/\S+/gi, "[REDACTED_URL]")
    .replace(/[\r\n\t]+/g, " ")
    .trim();
  return sanitized.slice(0, 200) || "unknown error";
}

/**
 * Builds an evaluation prompt for a single binary question given the agent context.
 * Used by tests and by setup_eval.cjs when constructing the engine prompt.
 * @param {string} question
 * @param {string} agentContext
 * @returns {string}
 */
function buildEvalPrompt(question, agentContext) {
  const contextSection = agentContext ? `## Agent Output Context\n\n${agentContext}\n\n` : "";
  return (
    `${contextSection}` +
    `## Evaluation Question\n\n${question}\n\n` +
    `## Instructions\n\n` +
    `Answer the evaluation question above based solely on the agent output context provided.\n` +
    `Respond with a JSON object containing exactly these fields:\n` +
    `- "passed": true if the answer is yes, false if the answer is no\n` +
    `- "rationale": a brief one-sentence explanation (max 100 words)\n` +
    `- "confidence": a number between 0 and 1 indicating your confidence\n\n` +
    `Respond only with the JSON object, no other text.`
  );
}

/**
 * Aggregates an array of EvalResult into an EvalSummary.
 * Aggregation is deterministic: pass_rate = passed / total.
 * @param {EvalResult[]} results
 * @returns {EvalSummary}
 */
function aggregateResults(results) {
  const total = results.length;
  const passed = results.filter(r => r.passed).length;
  const failed = total - passed;
  const pass_rate = total > 0 ? passed / total : 0;
  return { total, passed, failed, pass_rate, results };
}

/**
 * Renders a markdown summary table from an EvalSummary.
 * @param {EvalSummary} summary
 * @returns {string}
 */
function renderMarkdownSummary(summary) {
  const passRatePercent = (summary.pass_rate * 100).toFixed(1);
  const lines = ["## 🧪 BinEval Results (experimental)\n", `**${summary.passed}/${summary.total} passed** (${passRatePercent}%)\n\n`, "| Question ID | Result | Rationale |\n", "| --- | --- | --- |\n"];
  for (const r of summary.results) {
    const icon = r.passed ? "✅ pass" : "❌ fail";
    const rationale = (r.rationale || "").replace(/\|/g, "\\|");
    lines.push(`| \`${r.id}\` | ${icon} | ${rationale} |\n`);
  }
  return lines.join("");
}

module.exports = { readEvalSpec, buildEvalPrompt, aggregateResults, renderMarkdownSummary, sanitizeEvalError };
