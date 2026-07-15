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
    core.setFailed(`${ERR_VALIDATION}: GH_AW_EVALS_QUESTIONS is not valid JSON: ` + e.message);
    return;
  }

  if (!Array.isArray(questions) || questions.length === 0) {
    core.setFailed(`${ERR_VALIDATION}: GH_AW_EVALS_QUESTIONS must be a non-empty JSON array`);
    return;
  }

  fs.mkdirSync(EVALS_DIR, { recursive: true });

  // Load agent output for evaluation context
  const agentOutputPath = path.join(EVALS_DIR, AGENT_OUTPUT_FILENAME);
  let agentOutputContent = "";
  if (fs.existsSync(agentOutputPath)) {
    const stats = fs.statSync(agentOutputPath);
    agentOutputContent = fs.readFileSync(agentOutputPath, "utf-8");
    core.info(`Agent output loaded: ${agentOutputPath} (${stats.size} bytes)`);
  } else {
    core.warning(`Agent output not found at ${agentOutputPath}. ` + "Ensure the agent artifact includes agent_output.json. " + "Evaluation will proceed without agent context.");
  }

  const prompt = buildEvalPrompt(questions, agentOutputContent);

  fs.mkdirSync("/tmp/gh-aw/aw-prompts", { recursive: true });
  fs.writeFileSync("/tmp/gh-aw/aw-prompts/prompt.txt", prompt);
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
    fs.writeFileSync(EVALS_OUTPUT_PATH, "");
    return;
  }

  const logContent = fs.readFileSync(EVALS_LOG_PATH, "utf-8");
  core.info(`Parsing evals log: ${EVALS_LOG_PATH} (${logContent.length} bytes)`);

  // Collect all positional Q1/Q2/... answers from the log for fallback lookup
  const positionalAnswers = extractAllPositionalAnswers(logContent);

  const timestamp = new Date().toISOString();
  const results = [];

  for (let i = 0; i < questions.length; i++) {
    const q = questions[i];

    // Try ID-specific match first (e.g. "builds: YES"), then positional (Q1: YES)
    let answer = extractAnswerByID(logContent, q.id);
    if (answer === "UNKNOWN" && i < positionalAnswers.length && positionalAnswers[i]) {
      answer = positionalAnswers[i];
    }

    const record = {
      id: q.id,
      question: q.question,
      answer,
      model,
      timestamp,
    };
    results.push(record);
    core.info(`Q[${q.id}]: ${answer}`);
  }

  // Write JSONL — one JSON object per line
  const jsonlLines = results.map(r => JSON.stringify(r));
  fs.writeFileSync(EVALS_OUTPUT_PATH, jsonlLines.join("\n") + (jsonlLines.length > 0 ? "\n" : ""));
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

module.exports = { main, setupMain, parseMain };
