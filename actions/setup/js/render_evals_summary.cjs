// @ts-check
/// <reference types="@actions/github-script" />

/**
 * render_evals_summary — BinEval results step-summary renderer.
 *
 * Reads the redacted evals.jsonl written by run_evals (parse phase) and
 * renders the results as a collapsible <details> section in the GitHub
 * Actions step summary.
 *
 * This step runs AFTER redact_secrets so that no credentials appear in the
 * published summary. It is a no-op when the file does not exist or is empty.
 */

"use strict";

const fs = require("fs");
const { EVALS_OUTPUT_PATH } = require("./evals_constants.cjs");
const { buildStepSummaryDetailsSection } = require("./log_parser_step_summary_builder.cjs");

/**
 * Reads and parses evals.jsonl records.
 * Returns an empty array when the file is absent or unparseable.
 * @returns {Array<{id: string, question: string, answer: string, model: string, timestamp: string}>}
 */
function readEvalsResults() {
  if (!fs.existsSync(EVALS_OUTPUT_PATH)) {
    return [];
  }

  const results = [];
  let raw;
  try {
    raw = fs.readFileSync(EVALS_OUTPUT_PATH, "utf8");
  } catch {
    return [];
  }

  for (const rawLine of raw.split("\n")) {
    const line = rawLine.trim();
    if (!line) continue;
    try {
      const record = JSON.parse(line);
      if (record && typeof record === "object") {
        results.push({
          id: String(record.id ?? ""),
          question: String(record.question ?? ""),
          answer: String(record.answer ?? "UNKNOWN")
            .trim()
            .toUpperCase(),
          model: String(record.model ?? ""),
          timestamp: String(record.timestamp ?? ""),
        });
      }
    } catch {
      // skip malformed lines
    }
  }

  return results;
}

/**
 * Builds the markdown body for the evals <details> section.
 * @param {Array<{id: string, question: string, answer: string, model: string, timestamp: string}>} results
 * @returns {string}
 */
function buildEvalsBody(results) {
  if (results.length === 0) {
    return "";
  }

  const yesCount = results.filter(r => r.answer === "YES").length;
  const noCount = results.filter(r => r.answer === "NO").length;
  const unknownCount = results.filter(r => r.answer === "UNKNOWN").length;

  const lines = [];
  lines.push(`| ID | Question | Answer |`);
  lines.push(`| --- | --- | --- |`);
  for (const r of results) {
    const answerEmoji = r.answer === "YES" ? "✅ YES" : r.answer === "NO" ? "❌ NO" : "❓ UNKNOWN";
    lines.push(`| ${escapeMarkdownCell(r.id)} | ${escapeMarkdownCell(r.question)} | ${answerEmoji} |`);
  }
  lines.push("");
  lines.push(`**YES**: ${yesCount} | **NO**: ${noCount} | **UNKNOWN**: ${unknownCount}`);

  const model = results[0]?.model;
  if (model) {
    lines.push(`**model**: ${escapeMarkdownCell(model)}`);
  }

  return lines.join("\n");
}

/**
 * Escapes a string for use inside a Markdown table cell.
 * @param {string} text
 * @returns {string}
 */
function escapeMarkdownCell(text) {
  return text
    .replace(/[\r\n]/g, " ")
    .replace(/\|/g, "\\|")
    .replace(/`/g, "\\`");
}

/**
 * Main entry point: reads evals results and writes the step summary section.
 * @returns {Promise<void>}
 */
async function main() {
  const results = readEvalsResults();
  if (results.length === 0) {
    core.info("No evals results found; skipping step summary section");
    return;
  }

  core.info(`Rendering evals summary: ${results.length} result(s)`);

  const body = buildEvalsBody(results);
  const markdown = buildStepSummaryDetailsSection("BinEval Results", body);

  await core.summary.addRaw(markdown).write();
  core.info("BinEval results section written to step summary");
}

module.exports = { main, readEvalsResults, buildEvalsBody };
