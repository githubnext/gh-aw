// @ts-check
/// <reference types="@actions/github-script" />

/**
 * run_evals_harness.cjs
 *
 * BinEval evaluation harness for GitHub Agentic Workflows.
 *
 * Evaluates binary YES/NO questions about an agent's execution by calling the
 * GitHub Models inference API for each declared question. Results are written
 * to /tmp/gh-aw/evals/evals.jsonl for downstream steps and artifacts.
 *
 * Environment variables:
 *   GH_AW_EVALS_QUESTIONS  - JSON array of {id, question, model?} objects
 *   GH_AW_EVALS_MODEL      - Default model to use (e.g. "small", "openai/gpt-4o-mini")
 *   GITHUB_TOKEN           - Token for GitHub Models API authentication
 */

"use strict";

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");

/** GitHub Models inference endpoint. */
const GITHUB_MODELS_ENDPOINT = "https://models.github.ai/inference/chat/completions";

/** Fallback model when none is specified. */
const FALLBACK_MODEL = "openai/gpt-4o-mini";

/**
 * Read the agent context files from the evaluation working directory.
 * @returns {string} Combined context text for LLM evaluation.
 */
function readAgentContext() {
  const agentOutputPath = "/tmp/gh-aw/evals/agent_output.json";
  const promptPath = "/tmp/gh-aw/evals/prompt.txt";
  let context = "";

  try {
    const agentOutput = JSON.parse(fs.readFileSync(agentOutputPath, "utf8") || "{}");
    context += "Agent output:\n" + JSON.stringify(agentOutput, null, 2) + "\n\n";
  } catch (_) {
    // Agent output unavailable; continue without it.
  }
  try {
    context += "Agent prompt:\n" + fs.readFileSync(promptPath, "utf8") + "\n\n";
  } catch (_) {
    // Prompt unavailable; continue without it.
  }

  return context;
}

/**
 * Call the GitHub Models inference API for a single evaluation question.
 * @param {string} endpoint - Inference API endpoint URL.
 * @param {string} model - Model ID to use.
 * @param {string} userMessage - Full user message to send.
 * @param {string} token - GitHub token for authentication.
 * @returns {Promise<string>} Raw response content from the model.
 */
async function callInferenceAPI(endpoint, model, userMessage, token) {
  const response = await fetch(endpoint, {
    method: "POST",
    headers: {
      Authorization: "Bearer " + token,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      model,
      messages: [{ role: "user", content: userMessage }],
      temperature: 0,
      max_tokens: 512,
    }),
  });

  if (!response.ok) {
    const errText = await response.text();
    throw new Error("API error " + response.status + ": " + errText);
  }

  const data = await response.json();
  return (data.choices?.[0]?.message?.content || "").trim();
}

/**
 * Parse a YES/NO answer from the model's response.
 * The last non-empty line is the authoritative answer; everything before is rationale.
 * @param {string} content - Raw model response.
 * @returns {{ passed: boolean, rationale: string }}
 */
function parseAnswer(content) {
  const lines = content.split("\n").filter(l => l.trim() !== "");
  const lastLine = lines[lines.length - 1]?.trim().toUpperCase() || "";

  let passed;
  if (lastLine === "YES" || lastLine.endsWith("YES")) {
    passed = true;
  } else if (lastLine === "NO" || lastLine.endsWith("NO")) {
    passed = false;
  } else {
    passed = null;
  }

  const rationale = lines.slice(0, -1).join(" ").trim() || content;
  return { passed: passed ?? false, rationale, indeterminate: passed === null };
}

/**
 * Main entry point for the BinEval evaluation harness.
 * Reads questions from GH_AW_EVALS_QUESTIONS, evaluates each, and writes results to evals.jsonl.
 * @returns {Promise<void>}
 */
async function main() {
  const questionsEnv = process.env.GH_AW_EVALS_QUESTIONS || "[]";
  const defaultModel = process.env.GH_AW_EVALS_MODEL || FALLBACK_MODEL;
  const token = process.env.GITHUB_TOKEN;

  let questions;
  try {
    questions = JSON.parse(questionsEnv);
  } catch (e) {
    core.setFailed("Failed to parse GH_AW_EVALS_QUESTIONS: " + getErrorMessage(e));
    return;
  }

  const agentContext = readAgentContext();
  const results = [];
  let anyError = false;

  for (const q of questions) {
    // Per-question model override takes precedence over the default.
    const model = q.model || defaultModel;
    core.info("Evaluating: " + q.id + " — " + q.question + " (model: " + model + ")");

    const userMessage =
      agentContext +
      "Evaluate the following question about the agent's execution:\n" +
      q.question +
      "\n\n" +
      "Instructions:\n" +
      "- You may include brief reasoning before your final answer.\n" +
      "- Your final line must be exactly YES or NO (case-insensitive).\n" +
      "- YES means the statement is true / the condition is met.\n" +
      "- NO means the statement is false / the condition is not met.";

    let passed = false;
    let rationale = "";

    try {
      const content = await callInferenceAPI(GITHUB_MODELS_ENDPOINT, model, userMessage, token);
      const answer = parseAnswer(content);
      passed = answer.passed;
      rationale = answer.rationale;

      if (answer.indeterminate) {
        core.warning('Could not determine YES/NO answer for eval "' + q.id + '". Raw response: ' + content);
      }

      core.info("  Result: " + (passed ? "YES" : "NO") + (rationale ? " | " + rationale.slice(0, 120) : ""));
    } catch (err) {
      core.error('Failed to evaluate "' + q.id + '": ' + getErrorMessage(err));
      rationale = "Error: " + getErrorMessage(err);
      anyError = true;
    }

    results.push({ id: q.id, passed, rationale, model });
  }

  // Write JSONL results.
  const outDir = "/tmp/gh-aw/evals";
  fs.mkdirSync(outDir, { recursive: true });
  const jsonlPath = path.join(outDir, "evals.jsonl");
  const jsonlContent = results.map(r => JSON.stringify(r)).join("\n") + "\n";
  fs.writeFileSync(jsonlPath, jsonlContent, "utf8");
  core.info("Wrote " + results.length + " eval results to " + jsonlPath);

  // Set outputs.
  const passedCount = results.filter(r => r.passed).length;
  const failedCount = results.filter(r => !r.passed).length;
  core.setOutput("total", String(results.length));
  core.setOutput("passed", String(passedCount));
  core.setOutput("failed", String(failedCount));
  core.setOutput("pass_rate", results.length > 0 ? (passedCount / results.length).toFixed(2) : "0.00");

  if (anyError) {
    core.warning("One or more evaluations encountered errors. Check the log for details.");
  }
}

module.exports = { main, readAgentContext, callInferenceAPI, parseAnswer };
