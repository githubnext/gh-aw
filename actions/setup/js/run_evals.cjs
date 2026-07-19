// @ts-check
/// <reference types="@actions/github-script" />

/**
 * run_evals — BinEval binary evaluation harness.
 *
 * This module operates in two phases selected by GH_AW_EVALS_PHASE:
 *
 * Phase "setup" (default, runs BEFORE the agentic engine):
 *   - Reads configured eval questions from GH_AW_EVALS_QUESTIONS (JSON array)
 *   - Reads the agent output from /tmp/gh-aw/evals/agent_output.json
 *   - Builds a multi-question binary evaluation prompt
 *   - Writes the prompt to /tmp/gh-aw/aw-prompts/prompt.txt for the engine
 *
 * Phase "parse" (runs AFTER the agentic engine):
 *   - Reads the engine output log from /tmp/gh-aw/evals/evals.log
 *   - Extracts YES/NO answer for each question by ID or by position
 *   - Writes structured results to /tmp/gh-aw/evals.jsonl
 *
 * Environment variables:
 *   GH_AW_EVALS_QUESTIONS   JSON array of { id, question } objects
 *   GH_AW_EVALS_PHASE       "setup" (default) or "parse"
 *   GH_AW_EVALS_MODEL       LLM model name recorded in output metadata
 *
 * Design note: this file is intentionally engine-agnostic. The engine is
 * installed and executed by separate Go-generated GitHub Actions steps that
 * call engine.GetInstallationSteps / engine.GetExecutionSteps; this module
 * only handles prompt construction and result parsing.
 */

"use strict";

const fs = require("fs");
const path = require("path");

const { ERR_VALIDATION } = require("./error_codes.cjs");
const { EVALS_OUTPUT_PATH } = require("./evals_constants.cjs");

const EVALS_DIR = "/tmp/gh-aw/evals";
const EVALS_LOG_PATH = "/tmp/gh-aw/evals/evals.log";
const AGENT_OUTPUT_FILENAME = "agent_output.json";

// ---------------------------------------------------------------------------
// Phase 1 – setup: write multi-question evaluation prompt
// ---------------------------------------------------------------------------

/**
 * Reads eval questions and agent output, constructs a BinEval prompt, and
 * writes it to the standard GH_AW_PROMPT path for the agentic engine.
 * @returns {Promise<void>}
 */
async function setupMain() {
  const questionsRaw = process.env.GH_AW_EVALS_QUESTIONS;
  if (!questionsRaw) {
    core.setFailed(`${ERR_VALIDATION}: GH_AW_EVALS_QUESTIONS is not set`);
    return;
  }

  let questions;
  try {
    questions = JSON.parse(questionsRaw);
  } catch (e) {
    const eAny = /** @type {any} */ e;
    core.setFailed(`${ERR_VALIDATION}: GH_AW_EVALS_QUESTIONS is not valid JSON: ` + (eAny?.message ?? String(e)));
    return;
  }

  if (!Array.isArray(questions) || questions.length === 0) {
    core.setFailed(`${ERR_VALIDATION}: GH_AW_EVALS_QUESTIONS must be a non-empty JSON array`);
    return;
  }

  try {
    fs.mkdirSync(EVALS_DIR, { recursive: true });
  } catch (err) {
    throw new Error(`Failed to create directory ${EVALS_DIR}: ${String(err)}`, { cause: err });
  }

  // Load agent output for evaluation context
  const agentOutputPath = path.join(EVALS_DIR, AGENT_OUTPUT_FILENAME);
  let agentOutputContent = "";
  if (fs.existsSync(agentOutputPath)) {
    const stats = fs.statSync(agentOutputPath);
    try {
      agentOutputContent = fs.readFileSync(agentOutputPath, "utf-8");
    } catch (err) {
      throw new Error(`Failed to read file ${agentOutputPath}: ${String(err)}`, { cause: err });
    }
    core.info(`Agent output loaded: ${agentOutputPath} (${stats.size} bytes)`);
  } else {
    core.warning(`Agent output not found at ${agentOutputPath}. ` + "Ensure the agent artifact includes agent_output.json. " + "Evaluation will proceed without agent context.");
  }

  const prompt = buildEvalPrompt(questions, agentOutputContent);

  try {
    fs.mkdirSync("/tmp/gh-aw/aw-prompts", { recursive: true });
    fs.writeFileSync("/tmp/gh-aw/aw-prompts/prompt.txt", prompt);
  } catch (err) {
    throw new Error(`Failed to prepare eval prompt file: ${String(err)}`, { cause: err });
  }
  core.exportVariable("GH_AW_PROMPT", "/tmp/gh-aw/aw-prompts/prompt.txt");

  core.info(`BinEval setup complete: wrote prompt with ${questions.length} question(s)`);

  core.summary.addDetails("BinEval Evaluation Prompt", "\n\n``````markdown\n" + prompt + "\n``````\n\n");
  await core.summary.write();
}

// ---------------------------------------------------------------------------
// Phase 2 – parse: extract answers and write evals.jsonl
// ---------------------------------------------------------------------------

/**
 * Reads the engine log, extracts per-question YES/NO answers, and writes
 * structured JSONL records to the evals output file.
 * @returns {Promise<void>}
 */
async function parseMain() {
  const questionsRaw = process.env.GH_AW_EVALS_QUESTIONS;
  const model = process.env.GH_AW_EVALS_MODEL || "";
  const runID = process.env.GITHUB_RUN_ID || "unknown";

  /** @type {Array<{id: string, question: string}>} */
  let questions = [];
  if (questionsRaw) {
    try {
      questions = JSON.parse(questionsRaw);
    } catch {
      core.warning("GH_AW_EVALS_QUESTIONS is not valid JSON; result IDs will be positional");
    }
  }

  if (!fs.existsSync(EVALS_LOG_PATH)) {
    core.warning(`Evals log not found at ${EVALS_LOG_PATH}; no results written`);
    try {
      fs.writeFileSync(EVALS_OUTPUT_PATH, "");
    } catch (err) {
      throw new Error(`Failed to write file ${EVALS_OUTPUT_PATH}: ${String(err)}`, { cause: err });
    }
    return;
  }

  let logContent;
  try {
    logContent = fs.readFileSync(EVALS_LOG_PATH, "utf-8");
  } catch (err) {
    throw new Error(`Failed to read file ${EVALS_LOG_PATH}: ${String(err)}`, { cause: err });
  }
  core.info(`Parsing evals log: ${EVALS_LOG_PATH} (${logContent.length} bytes)`);

  // Build a search corpus that includes both raw log lines AND any assistant text
  // extracted from JSONL log entries (e.g. Pi engine turn_end events).  The engine
  // may emit answers inside JSON-encoded strings where newlines are represented as
  // the escape sequence "\n", so the line-based regex patterns below would miss them
  // unless the JSON content is decoded first.
  const extractedText = extractAssistantTextFromJsonlLog(logContent);
  const searchContent = extractedText ? logContent + "\n" + extractedText : logContent;

  // Collect all positional Q1/Q2/... answers from the log for fallback lookup
  const positionalAnswers = extractAllPositionalAnswers(searchContent);

  const timestamp = new Date().toISOString();
  const results = [];

  for (let i = 0; i < questions.length; i++) {
    const q = questions[i];

    // Try ID-specific match first (e.g. "builds: YES"), then positional (Q1: YES)
    let answer = extractAnswerByID(searchContent, q.id);
    if (answer === "UNKNOWN" && i < positionalAnswers.length && positionalAnswers[i]) {
      answer = positionalAnswers[i];
    }

    const record = {
      id: q.id,
      question: q.question,
      answer,
      model,
      timestamp,
      runid: runID,
    };
    results.push(record);
    core.info(`Q[${q.id}]: ${answer}`);
  }

  // Write JSONL — one JSON object per line
  const jsonlLines = results.map(r => JSON.stringify(r));
  try {
    fs.writeFileSync(EVALS_OUTPUT_PATH, jsonlLines.join("\n") + (jsonlLines.length > 0 ? "\n" : ""));
  } catch (err) {
    throw new Error(`Failed to write file ${EVALS_OUTPUT_PATH}: ${String(err)}`, { cause: err });
  }
  core.info(`BinEval results written to ${EVALS_OUTPUT_PATH} (${results.length} record(s))`);
  // Step summary rendering is handled by the dedicated render_evals_summary.cjs step
  // that runs after secret redaction, so the published summary is always redacted.
}

// ---------------------------------------------------------------------------
// Main entry point
// ---------------------------------------------------------------------------

/**
 * Dispatches to setupMain or parseMain based on GH_AW_EVALS_PHASE.
 * @returns {Promise<void>}
 */
async function main() {
  const phase = process.env.GH_AW_EVALS_PHASE || "setup";
  if (phase === "parse") {
    await parseMain();
  } else {
    await setupMain();
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Builds a multi-question binary evaluation prompt.
 * @param {Array<{id: string, question: string}>} questions
 * @param {string} agentOutput
 * @returns {string}
 */
function buildEvalPrompt(questions, agentOutput) {
  const questionList = questions.map((q, i) => `<question number="${i + 1}" id="${q.id}">${q.question}</question>`).join("\n");

  const agentSection = agentOutput ? `<agent_output>\n${agentOutput}\n</agent_output>` : "<agent_output>\n(no agent output available)\n</agent_output>";

  return `# BinEval: Binary Evaluation

You are evaluating the output of an AI agentic workflow using BinEval (binary evaluation).
For each question below, answer with exactly YES or NO based on the agent output provided.

<questions>
${questionList}
</questions>

${agentSection}

<instructions>
Answer each question on a separate line using EXACTLY this format:
Q1: YES
Q2: NO

Use only YES or NO. Do not provide explanations or reasoning.
Evaluate each question solely based on the agent output shown above.
</instructions>`;
}

/**
 * Extracts all positional Q1/Q2/... answers from log content.
 * Returns a 0-indexed array where index 0 = Q1's answer.
 * @param {string} logContent
 * @returns {string[]}
 */
function extractAllPositionalAnswers(logContent) {
  /** @type {string[]} */
  const answers = [];
  for (const line of logContent.split("\n")) {
    const match = line.trim().match(/^Q(\d+):\s+(YES|NO)\b/i);
    if (match) {
      const idx = parseInt(match[1], 10) - 1; // Convert 1-indexed to 0-indexed
      if (idx >= 0) {
        answers[idx] = match[2].toUpperCase();
      }
    }
  }
  return answers;
}

/**
 * Tries to find an answer for a question by its id using flexible pattern matching.
 * Returns "YES", "NO", or "UNKNOWN".
 * @param {string} logContent
 * @param {string} id
 * @returns {string}
 */
function extractAnswerByID(logContent, id) {
  const escaped = id.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const yesPattern = new RegExp(`\\b${escaped}\\b[:\\s]+(YES)\\b`, "i");
  const noPattern = new RegExp(`\\b${escaped}\\b[:\\s]+(NO)\\b`, "i");
  if (yesPattern.test(logContent)) return "YES";
  if (noPattern.test(logContent)) return "NO";
  return "UNKNOWN";
}

/**
 * Extracts all assistant text content from a JSONL engine log.
 * Engines such as Pi emit one JSON object per line (JSONL). The assistant's
 * final answer lives in a `turn_end` event (v3 schema) or `assistant` events
 * (v1 legacy schema) where newlines are JSON-encoded as the two-character
 * sequence `\n`.  Parsing those JSON objects restores the actual newlines so
 * the positional regex patterns can match correctly.
 *
 * Returns a single string with all extracted text joined by newlines, or an
 * empty string when no JSONL content is found.
 * @param {string} logContent
 * @returns {string}
 */
function extractAssistantTextFromJsonlLog(logContent) {
  const texts = [];
  for (const line of logContent.split("\n")) {
    const trimmed = line.trim();
    // Find the first '{' which starts the JSON object.  Some runner environments
    // prefix log lines with a timestamp (e.g. "2026-07-16T07:21:45Z {...}");
    // stripping that prefix lets us parse the JSON regardless.
    const jsonStart = trimmed.indexOf("{");
    if (jsonStart === -1) continue;
    let obj;
    try {
      obj = JSON.parse(trimmed.slice(jsonStart));
    } catch {
      continue;
    }
    // v3 schema: turn_end carries the complete assistant message
    if (obj.type === "turn_end" && obj.message && Array.isArray(obj.message.content)) {
      for (const part of obj.message.content) {
        if (part && typeof part.text === "string") {
          texts.push(part.text);
        }
      }
    }
    // v1 legacy schema: assistant event carries raw text content
    if (obj.type === "assistant" && typeof obj.content === "string" && obj.content) {
      texts.push(obj.content);
    }
  }
  return texts.join("\n");
}

module.exports = { main, setupMain, parseMain, extractAssistantTextFromJsonlLog };
