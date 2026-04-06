// @ts-check
/// <reference types="@actions/github-script" />

const path = require("path");
const { renderTemplateFromFile } = require("./messages_core.cjs");

/**
 * Writes a pre-activation skip denial summary to the GitHub Actions job summary.
 * Uses the pre_activation_skip.md template from the prompts directory when available,
 * falling back to a hardcoded format when the template cannot be loaded (e.g. in tests).
 *
 * @param {string} reason - The denial reason message
 * @param {string} remediation - Remediation hint for the operator
 */
async function writeDenialSummary(reason, remediation) {
  let content;

  const runnerTemp = process.env.RUNNER_TEMP;
  if (runnerTemp) {
    const templatePath = path.join(runnerTemp, "gh-aw", "prompts", "pre_activation_skip.md");
    try {
      content = renderTemplateFromFile(templatePath, { reason, remediation });
    } catch (err) {
      // Log unexpected errors but still fall through to the hardcoded fallback
      if (err && typeof err === "object" && "code" in err && err.code !== "ENOENT") {
        core.warning(`pre_activation_summary: could not read template ${templatePath}: ${err instanceof Error ? err.message : String(err)}`);
      }
    }
  }

  if (!content) {
    content = `## ⏭️ Workflow Activation Skipped\n\n> ${reason}\n\n**Remediation:** ${remediation}\n\n---\n_See the \`pre_activation\` job log for full details._`;
  }

  await core.summary.addRaw(content).write();
}

module.exports = { writeDenialSummary };
