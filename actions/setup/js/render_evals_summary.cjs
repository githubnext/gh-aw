// @ts-check
/// <reference types="@actions/github-script" />

/**
 * render_evals_summary.cjs
 *
 * Renders BinEval evaluation results as a GitHub Actions step summary in Markdown.
 * Reads /tmp/gh-aw/evals/evals.jsonl and writes a formatted table to the step summary.
 */

"use strict";

const fs = require("fs");

/**
 * Render the BinEval evaluation results as a Markdown step summary.
 * @returns {Promise<void>}
 */
async function main() {
  const jsonlPath = "/tmp/gh-aw/evals/evals.jsonl";
  if (!fs.existsSync(jsonlPath)) {
    core.info("No evals results file found, skipping summary.");
    return;
  }

  const lines = fs
    .readFileSync(jsonlPath, "utf8")
    .split("\n")
    .filter(l => l.trim());
  const results = [];
  for (const l of lines) {
    try {
      results.push(JSON.parse(l));
    } catch {
      // Skip malformed lines.
    }
  }
  const passed = results.filter(r => r.passed).length;
  const total = results.length;
  const passRate = total > 0 ? Math.round((passed / total) * 100) : 0;

  let md = "## 🔬 BinEval Evaluation Results\n\n";
  md += `**${passed}/${total} passed** (${passRate}% pass rate)\n\n`;
  md += "| Question ID | Result | Rationale |\n";
  md += "|---|---|---|\n";
  for (const r of results) {
    const icon = r.passed ? "✅ YES" : "❌ NO";
    const rationale = (r.rationale || "").replace(/[|\n]/g, " ").slice(0, 200);
    md += `| ${r.id} | ${icon} | ${rationale} |\n`;
  }

  await core.summary.addRaw(md).write();
  core.info("Evaluation summary written.");
}

module.exports = { main };
