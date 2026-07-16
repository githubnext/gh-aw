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
    fs.writeFileSync(EVALS_OUTPUT_PATH, "");
    return;
  }

  const logContent = fs.readFileSync(EVALS_LOG_PATH, "utf-8");
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
  fs.writeFileSync(EVALS_OUTPUT_PATH, jsonlLines.join("\n") + (jsonlLines.length > 0 ? "\n" : ""));
  core.info(`BinEval results written to ${EVALS_OUTPUT_PATH} (${results.length} record(s))`);
  // Step summary rendering is handled by the dedicated render_evals_summary.cjs step
  // that runs after secret redaction, so the published summary is always redacted.
}

// ---------------------------------------------------------------------------
// Main entry point
// ---------------------------------------------------------------------------

/**
 * Dispatches to setupMain, parseMain, or runScriptsMain based on GH_AW_EVALS_PHASE.
 * @returns {Promise<void>}
 */
async function main() {
  const phase = process.env.GH_AW_EVALS_PHASE || "setup";
  if (phase === "parse") {
    await parseMain();
  } else if (phase === "run-scripts") {
    await runScriptsMain();
  } else {
    await setupMain();
  }
}

// ---------------------------------------------------------------------------
// Phase 3 – run-scripts: execute deterministic script evals and write results
// ---------------------------------------------------------------------------

/**
 * Reads script eval definitions from GH_AW_EVALS_SCRIPT_DEFS, runs each script
 * with the appropriate environment variables, captures stdout YES/NO output, and
 * appends JSONL records to the evals output file.
 *
 * Environment variables passed to each script:
 *   GH_AW_AGENT_OUTPUT       – path to the agent_output.json file
 *   GH_AW_SAFE_OUTPUT_ITEMS  – path to the safe-output-items.jsonl file
 *   GITHUB_RUN_ID             – GitHub Actions run ID
 *
 * @returns {Promise<void>}
 */
async function runScriptsMain() {
  const scriptDefsRaw = process.env.GH_AW_EVALS_SCRIPT_DEFS;
  const model = process.env.GH_AW_EVALS_MODEL || "";
  const runID = process.env.GITHUB_RUN_ID || "unknown";

  if (!scriptDefsRaw) {
    core.setFailed(`${ERR_VALIDATION}: GH_AW_EVALS_SCRIPT_DEFS is not set`);
    return;
  }

  /** @type {Array<{id: string, run: string}>} */
  let scriptDefs;
  try {
    scriptDefs = JSON.parse(scriptDefsRaw);
  } catch (e) {
    core.setFailed(`${ERR_VALIDATION}: GH_AW_EVALS_SCRIPT_DEFS is not valid JSON: ` + e.message);
    return;
  }

  if (!Array.isArray(scriptDefs) || scriptDefs.length === 0) {
    core.info("No script evals to run");
    return;
  }

  const timestamp = new Date().toISOString();
  const results = [];

  const scriptEnv = {
    ...process.env,
    GH_AW_AGENT_OUTPUT: process.env.GH_AW_AGENT_OUTPUT || "/tmp/gh-aw/agent_output.json",
    GH_AW_SAFE_OUTPUT_ITEMS: process.env.GH_AW_SAFE_OUTPUT_ITEMS || "/tmp/gh-aw/safe-output-items.jsonl",
    GITHUB_RUN_ID: runID,
  };

  for (const def of scriptDefs) {
    const { id, run: script } = def;
    core.info(`Running script eval [${id}]: ${script}`);

    let answer = "UNKNOWN";
    try {
      let stdout = "";
      const exitCode = await exec.exec("bash", ["-c", script], {
        env: scriptEnv,
        silent: true,
        listeners: {
          stdout: data => {
            stdout += data.toString();
          },
        },
        ignoreReturnCode: true,
      });

      const trimmed = stdout.trim();
      const firstToken = trimmed.split(/\s/)[0].toUpperCase();
      if (firstToken === "YES" || firstToken === "NO") {
        answer = firstToken;
      } else {
        core.warning(`Script eval [${id}] stdout did not start with YES or NO (exit code ${exitCode}): ${trimmed.slice(0, 200)}`);
      }
    } catch (e) {
      core.warning(`Script eval [${id}] failed to execute: ${e.message}`);
    }

    core.info(`Script eval [${id}]: ${answer}`);
    results.push({ id, run: script, answer, model, timestamp, runid: runID });
  }

  if (results.length === 0) {
    return;
  }

  // Append results to the evals output file (creating it if it does not exist).
  fs.mkdirSync(path.dirname(EVALS_OUTPUT_PATH), { recursive: true });
  const existingContent = fs.existsSync(EVALS_OUTPUT_PATH) ? fs.readFileSync(EVALS_OUTPUT_PATH, "utf-8") : "";
  const newLines = results.map(r => JSON.stringify(r)).join("\n");
  const separator = existingContent.length > 0 && !existingContent.endsWith("\n") ? "\n" : "";
  fs.writeFileSync(EVALS_OUTPUT_PATH, existingContent + separator + newLines + "\n");
  core.info(`Script eval results appended to ${EVALS_OUTPUT_PATH} (${results.length} record(s))`);
}

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

module.exports = { main, setupMain, parseMain, runScriptsMain, extractAssistantTextFromJsonlLog };
