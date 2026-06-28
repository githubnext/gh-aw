// @ts-check
/// <reference types="@actions/github-script" />

/**
 * eval_harness.cjs
 *
 * BinEval Evaluation Harness (experimental)
 *
 * Evaluates a set of binary questions about a completed agent workflow run.
 * Each question is evaluated independently by an LLM, producing a binary
 * pass/fail result with an optional rationale.
 *
 * Environment variables (set by the compiled workflow step):
 *   GH_AW_EVAL_SPEC       - JSON array of {id, question} evaluation definitions
 *   GH_AW_EVAL_WORK_DIR   - Working directory for evals (default: /tmp/gh-aw/evals)
 *   GH_AW_EVAL_MODEL      - LLM model to use (default: gpt-4o-mini)
 *
 * Input files (downloaded from agent artifact into GH_AW_EVAL_WORK_DIR):
 *   agent_output.json     - Structured agent output for context
 *   aw-prompts/prompt.txt - Original workflow prompt for context
 *
 * Output files (written to GH_AW_EVAL_WORK_DIR):
 *   eval_results.json     - Aggregated evaluation summary and per-question results
 *
 * Design principles:
 *   - Each question is evaluated independently (BinEval)
 *   - Partial failures are tolerated (a failed LLM call for one question does not
 *     abort evaluation of the remaining questions)
 *   - The evaluator is deterministic in aggregation: pass_rate = passed / total
 *   - No MCPs, no checkout: the harness only reads downloaded artifact files
 */

"use strict";

const fs = require("fs");
const path = require("path");
const https = require("https");

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const DEFAULT_WORK_DIR = "/tmp/gh-aw/evals";
const DEFAULT_MODEL = "gpt-4o-mini";

// GitHub Models API endpoint for chat completions (OpenAI-compatible).
// Uses GITHUB_TOKEN for authentication — no additional credentials required.
const GITHUB_MODELS_ENDPOINT = "models.github.com";
const GITHUB_MODELS_PATH = "/inference/chat/completions";
const EVAL_SYSTEM_PROMPT =
  "You are an objective evaluator. Answer binary (yes/no) questions about agentic workflow outputs. Always respond with a JSON object containing 'passed' (boolean), 'rationale' (string), and 'confidence' (number 0-1).";

// These caps keep the prompt comfortably within the context window of the
// default small eval model while still leaving room for the JSON answer.
const MAX_AGENT_OUTPUT_CHARS = 8000;
const MAX_PROMPT_CHARS = 4000;
const MAX_RATIONALE_CHARS = 500;

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
 * Reads and truncates a file for inclusion as LLM context.
 * Returns an empty string if the file does not exist.
 * @param {string} filePath
 * @param {number} maxChars
 * @returns {string}
 */
function readContextFile(filePath, maxChars) {
  if (!fs.existsSync(filePath)) {
    return "";
  }
  try {
    const content = fs.readFileSync(filePath, "utf-8");
    if (content.length <= maxChars) return content;
    return content.slice(0, maxChars) + "\n... (truncated)";
  } catch {
    return "";
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
 * Makes an HTTPS POST request and returns the response body as a string.
 * @param {object} options - https.request options
 * @param {string} body - Request body
 * @returns {Promise<string>}
 */
function httpsPost(options, body) {
  return new Promise((resolve, reject) => {
    const req = https.request(options, res => {
      const chunks = [];
      res.on("data", chunk => chunks.push(chunk));
      res.on("end", () => {
        const responseBody = Buffer.concat(chunks).toString("utf-8");
        if (res.statusCode && res.statusCode >= 400) {
          reject(new Error(`HTTP ${res.statusCode}: ${responseBody.slice(0, 200)}`));
        } else {
          resolve(responseBody);
        }
      });
    });
    req.on("error", reject);
    req.write(body);
    req.end();
  });
}

/**
 * Calls the GitHub Models API to evaluate a single binary question.
 * Returns an EvalResult. On failure, returns a failed result with an error rationale
 * so the harness can continue evaluating remaining questions (partial failure tolerance).
 * @param {string} token
 * @param {string} model
 * @param {string} question
 * @param {string} agentContext
 * @returns {Promise<{passed: boolean, rationale: string, confidence?: number}>}
 */
async function callLLMForQuestion(token, model, question, agentContext) {
  const prompt = buildEvalPrompt(question, agentContext);
  const requestBody = JSON.stringify({
    model,
    messages: [
      {
        role: "system",
        content: EVAL_SYSTEM_PROMPT,
      },
      { role: "user", content: prompt },
    ],
    response_format: { type: "json_object" },
    temperature: 0,
    // The response is a tiny JSON object with three fields, so 256 tokens leaves
    // ample headroom for rationale text without inflating per-question cost.
    max_tokens: 256,
  });

  const options = {
    hostname: GITHUB_MODELS_ENDPOINT,
    path: GITHUB_MODELS_PATH,
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: "Bearer " + token,
      "Content-Length": Buffer.byteLength(requestBody),
    },
  };

  const responseBody = await httpsPost(options, requestBody);
  const response = JSON.parse(responseBody);

  const content = response?.choices?.[0]?.message?.content;
  if (!content) {
    throw new Error("Empty response from LLM");
  }

  const parsed = JSON.parse(content);
  const rationale = typeof parsed.rationale === "string" ? parsed.rationale : "";
  if (rationale.length > MAX_RATIONALE_CHARS) {
    console.warn(`Truncating eval rationale from ${rationale.length} to ${MAX_RATIONALE_CHARS} characters`);
  }
  const result = {
    passed: Boolean(parsed.passed),
    rationale: rationale.slice(0, MAX_RATIONALE_CHARS),
  };

  if (typeof parsed.confidence === "number") {
    result.confidence = Math.max(0, Math.min(1, parsed.confidence));
  }

  return result;
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

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

/**
 * Main entry point for the eval harness.
 * @returns {Promise<void>}
 */
async function main() {
  const workDir = process.env.GH_AW_EVAL_WORK_DIR || DEFAULT_WORK_DIR;
  const model = process.env.GH_AW_EVAL_MODEL || DEFAULT_MODEL;
  const token = process.env.GITHUB_TOKEN || "";

  core.info(`BinEval harness starting (model: ${model}, workDir: ${workDir})`);

  // Parse eval definitions
  const evals = readEvalSpec();
  if (evals.length === 0) {
    core.warning("No eval definitions found in GH_AW_EVAL_SPEC — nothing to evaluate");
    return;
  }
  core.info(`Evaluating ${evals.length} question(s)`);

  // Load agent context files
  const agentOutputPath = path.join(workDir, "agent_output.json");
  const promptPath = path.join(workDir, "aw-prompts", "prompt.txt");

  const agentOutputRaw = readContextFile(agentOutputPath, MAX_AGENT_OUTPUT_CHARS);
  const promptRaw = readContextFile(promptPath, MAX_PROMPT_CHARS);

  // Build context string from available files
  const contextParts = [];
  if (promptRaw) contextParts.push(`### Workflow Prompt\n${promptRaw}`);
  if (agentOutputRaw) contextParts.push(`### Agent Output\n${agentOutputRaw}`);
  const agentContext = contextParts.join("\n\n");

  if (!agentContext) {
    core.warning("No agent context files found — evaluations will have limited context");
  }

  // Evaluate each question independently (partial failure tolerance)
  const results = [];
  for (const evalDef of evals) {
    core.info(`Evaluating: ${evalDef.id} — "${evalDef.question}"`);
    try {
      if (!token) {
        throw new Error("GITHUB_TOKEN is not set — cannot call GitHub Models API");
      }
      const result = await callLLMForQuestion(token, model, evalDef.question, agentContext);
      results.push({ id: evalDef.id, ...result });
      const icon = result.passed ? "✅" : "❌";
      core.info(`  ${icon} ${result.passed ? "pass" : "fail"} — ${result.rationale}`);
    } catch (err) {
      const sanitizedError = sanitizeEvalError(err);
      core.warning(`  ⚠️ Evaluation failed for "${evalDef.id}": ${sanitizedError}`);
      results.push({
        id: evalDef.id,
        passed: false,
        rationale: `Evaluation error: ${sanitizedError}`,
      });
    }
  }

  // Aggregate results
  const summary = aggregateResults(results);

  // Write JSON results
  fs.mkdirSync(workDir, { recursive: true });
  const resultsPath = path.join(workDir, "eval_results.json");
  fs.writeFileSync(resultsPath, JSON.stringify(summary, null, 2));
  core.info(`Results written to ${resultsPath}`);

  // Write step summary
  const markdownSummary = renderMarkdownSummary(summary);
  core.summary.addRaw(markdownSummary).write();

  // Set outputs
  core.setOutput("eval_passed", String(summary.passed));
  core.setOutput("eval_total", String(summary.total));
  core.setOutput("eval_pass_rate", summary.pass_rate.toFixed(4));

  core.info(`BinEval complete: ${summary.passed}/${summary.total} passed (${(summary.pass_rate * 100).toFixed(1)}%)`);
}

module.exports = { main, readEvalSpec, buildEvalPrompt, aggregateResults, renderMarkdownSummary, sanitizeEvalError };
