// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Generate workflow overview step that writes an agentic workflow run overview
 * to the GitHub step summary. This reads from aw_info.json that was created by
 * a previous step and uses HTML details/summary tags for collapsible output.
 *
 * @param {typeof import('@actions/core')} core - GitHub Actions core library
 * @returns {Promise<void>}
 */
async function generateWorkflowOverview(core) {
  const fs = require("fs");
  const awInfoPath = "/tmp/gh-aw/aw_info.json";

  // Load aw_info.json
  const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));

  let networkDetails = "";
  if (awInfo.allowed_domains && awInfo.allowed_domains.length > 0) {
    networkDetails = awInfo.allowed_domains
      .slice(0, 10)
      .map(d => `  - ${d}`)
      .join("\n");
    if (awInfo.allowed_domains.length > 10) {
      networkDetails += `\n  - ... and ${awInfo.allowed_domains.length - 10} more`;
    }
  }

  // Build summary using string concatenation to avoid YAML parsing issues with template literals

  // Workflow title (shown at top level, not inside a collapsible section)
  const title = awInfo.workflow_name || "Agentic Workflow";
  let summary = `## 🤖 ${title}\n\n`;

  // Workflow description (if available)
  if (awInfo.workflow_description) {
    summary += `${awInfo.workflow_description}\n\n`;
  }

  // Agent and model info (key info shown prominently, not hidden in collapsible)
  const engineName = awInfo.engine_name || awInfo.engine_id || "Unknown";
  const model = awInfo.model || "(default)";
  summary += "| | |\n" + "|:--|:--|\n" + `| **Agent** | ${engineName} |\n` + `| **Model** | ${model} |\n` + "\n";

  // Collapsible run details section for technical configuration
  summary +=
    "<details>\n" +
    "<summary>Run details</summary>\n\n" +
    "#### Engine Configuration\n" +
    "| Property | Value |\n" +
    "|----------|-------|\n" +
    `| Engine ID | ${awInfo.engine_id} |\n` +
    `| Engine Name | ${awInfo.engine_name} |\n` +
    `| Model | ${awInfo.model || "(default)"} |\n`;

  // Engine description (if available)
  if (awInfo.engine_description) {
    summary += "\n" + `_${awInfo.engine_description}_\n`;
  }

  summary +=
    "\n" +
    "#### Network Configuration\n" +
    "| Property | Value |\n" +
    "|----------|-------|\n" +
    `| Firewall | ${awInfo.firewall_enabled ? "✅ Enabled" : "❌ Disabled"} |\n` +
    `| Firewall Version | ${awInfo.awf_version || "(latest)"} |\n` +
    "\n" +
    (networkDetails ? `##### Allowed Domains\n${networkDetails}\n` : "") +
    "</details>";

  await core.summary.addRaw(summary).write();
  console.log("Generated workflow overview in step summary");
}

module.exports = {
  generateWorkflowOverview,
};
