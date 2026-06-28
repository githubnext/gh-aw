// @ts-check
/// <reference types="@actions/github-script" />

/**
 * setup_eval.cjs
 *
 * BinEval Evaluation Setup (experimental)
 *
 * Prepares the evaluation prompt file that the agentic engine reads to answer
 * binary (yes/no) questions about a completed workflow run. This script is the
 * counterpart to setup_threat_detection.cjs — it creates /tmp/gh-aw/aw-prompts/prompt.txt
 * with the rendered eval prompt so the engine execution step can pick it up via
 * GH_AW_PROMPT (set as an Actions environment variable by this script).
 *
 * Environment variables (set by the compiled workflow step):
 *   GH_AW_EVAL_SPEC     - JSON array of {id, question} evaluation definitions
 *   GH_AW_EVAL_WORK_DIR - Working directory where artifact was downloaded (default: /tmp/gh-aw/eval)
 *
 * Input files (downloaded from agent artifact into GH_AW_EVAL_WORK_DIR):
 *   agent_output.json     - Structured agent output for context
 *   aw-prompts/prompt.txt - Original workflow prompt for context
 *
 * Output:
 *   /tmp/gh-aw/aw-prompts/prompt.txt - Rendered eval prompt for the engine
 */

"use strict";

const fs = require("fs");
const path = require("path");
const { getPromptPath } = require("./messages_core.cjs");
const { ERR_VALIDATION } = require("./error_codes.cjs");

const DEFAULT_WORK_DIR = "/tmp/gh-aw/eval";

/**
 * Reads the eval specification from the GH_AW_EVAL_SPEC environment variable.
 * @returns {{ id: string, question: string }[]}
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
 * Formats the eval questions for embedding in the prompt template.
 * Each question is rendered as a numbered list item with its ID label.
 * @param {{ id: string, question: string }[]} evals
 * @returns {string}
 */
function formatEvalQuestions(evals) {
  return evals.map((e, i) => `${i + 1}. **${e.id}**: ${e.question}`).join("\n");
}

/**
 * Main entry point for eval setup.
 * @returns {Promise<void>}
 */
async function main() {
  const workDir = process.env.GH_AW_EVAL_WORK_DIR || DEFAULT_WORK_DIR;

  // Parse eval definitions
  const evals = readEvalSpec();
  if (evals.length === 0) {
    core.warning(`⚠️ ${ERR_VALIDATION}: No eval definitions found in GH_AW_EVAL_SPEC — skipping eval setup`);
    return;
  }
  core.info(`Setting up ${evals.length} BinEval question(s)`);

  // Read the eval prompt template
  const templatePath = getPromptPath("eval.md");
  if (!fs.existsSync(templatePath)) {
    core.setFailed(`${ERR_VALIDATION}: Eval prompt template not found at: ${templatePath}`);
    return;
  }
  const templateContent = fs.readFileSync(templatePath, "utf-8");

  // Locate agent context files (downloaded from the agent artifact)
  const promptPath = path.join(workDir, "aw-prompts", "prompt.txt");
  let promptFileInfo;
  if (!fs.existsSync(promptPath)) {
    promptFileInfo = `${promptPath} (unavailable)`;
    core.warning(`⚠️ ${ERR_VALIDATION}: Missing workflow prompt at ${promptPath}. Eval will proceed with reduced context.`);
  } else {
    const stats = fs.statSync(promptPath);
    promptFileInfo = stats.size > 0 ? `${promptPath} (${stats.size} bytes)` : `${promptPath} (unavailable)`;
    if (stats.size === 0) {
      core.warning(`⚠️ ${ERR_VALIDATION}: Workflow prompt is empty at ${promptPath}. Eval will proceed with reduced context.`);
    } else {
      core.info(`Prompt file found: ${promptPath} (${stats.size} bytes)`);
    }
  }

  const agentOutputPath = path.join(workDir, "agent_output.json");
  let agentOutputFileInfo;
  if (!fs.existsSync(agentOutputPath)) {
    agentOutputFileInfo = `${agentOutputPath} (unavailable)`;
    core.warning(`⚠️ ${ERR_VALIDATION}: Missing agent output at ${agentOutputPath}. Eval will proceed with reduced context.`);
  } else {
    const stats = fs.statSync(agentOutputPath);
    agentOutputFileInfo = `${agentOutputPath} (${stats.size} bytes)`;
    core.info(`Agent output found: ${agentOutputPath} (${stats.size} bytes)`);
  }

  // Render the prompt template
  const evalQuestions = formatEvalQuestions(evals);
  const promptContent = templateContent
    .replace(/{WORKFLOW_PROMPT_FILE}/g, promptFileInfo)
    .replace(/{AGENT_OUTPUT_FILE}/g, agentOutputFileInfo)
    .replace(/{EVAL_QUESTIONS}/g, evalQuestions);

  // Write prompt file
  fs.mkdirSync("/tmp/gh-aw/aw-prompts", { recursive: true });
  fs.writeFileSync("/tmp/gh-aw/aw-prompts/prompt.txt", promptContent);
  core.exportVariable("GH_AW_PROMPT", "/tmp/gh-aw/aw-prompts/prompt.txt");
  core.info(`Eval prompt written to /tmp/gh-aw/aw-prompts/prompt.txt`);

  // Write rendered prompt to step summary
  await core.summary.addRaw("<details>\n<summary>BinEval Prompt</summary>\n\n" + "``````markdown\n" + promptContent + "\n" + "``````\n\n</details>\n").write();

  core.info("BinEval setup completed");
}

module.exports = { main, readEvalSpec, formatEvalQuestions };
