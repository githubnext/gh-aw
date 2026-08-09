// @ts-check

const { redactStepSummaryContent } = require("./redact_secrets.cjs");

/**
 * Render no-op messages using the same regular, collapsible structure as the
 * safe-output processing summary.
 *
 * @param {Array<string>} messages
 * @param {{runUrl?: string, staged?: boolean}} [options]
 * @returns {string}
 */
function buildNoopConclusionSummary(messages, options = {}) {
  const { runUrl, staged = false } = options;
  const emoji = staged ? "⚠️" : "✅";
  const label = staged ? "Staged No-Op Preview" : "Conclusion Summary";
  const count = messages.length;
  const target = typeof runUrl === "string" && /^https?:\/\//.test(runUrl) ? `**Target:** [Workflow run](${runUrl})\n\n` : "";

  let summary = `<details>\n<summary>${emoji} ${label} (${count} no-op message${count === 1 ? "" : "s"})</summary>\n\n`;
  summary += staged ? "The following messages would be logged if staged mode was disabled:\n\n" : "The following messages were logged for transparency:\n\n";
  summary += target;

  for (let index = 0; index < messages.length; index++) {
    summary += `<details>\n<summary>${emoji} No-Op - ${staged ? "Preview" : "Success"} (Message ${index + 1})</summary>\n\n`;
    summary += `### No-Op\n\n${messages[index]}\n\n`;
    summary += `</details>\n\n`;
  }

  summary += `</details>\n\n`;
  return redactStepSummaryContent(summary);
}

module.exports = { buildNoopConclusionSummary };
