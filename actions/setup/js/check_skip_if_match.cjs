// @ts-check
/// <reference types="@actions/github-script" />

const { runSkipQueryGate } = require("./check_skip_if_helpers.cjs");

async function main() {
  const { GH_AW_SKIP_QUERY: skipQuery, GH_AW_WORKFLOW_NAME: workflowName, GH_AW_SKIP_MAX_MATCHES: maxMatchesStr = "1", GH_AW_SKIP_SCOPE: skipScope } = process.env;

  await runSkipQueryGate({
    skipQuery,
    workflowName,
    thresholdStr: maxMatchesStr,
    thresholdEnvVar: "GH_AW_SKIP_MAX_MATCHES",
    thresholdLabel: "Maximum matches threshold",
    checkLabel: "skip-if-match",
    outputName: "skip_check_ok",
    skipScope,
    shouldSkip: (totalCount, threshold) => totalCount >= threshold,
    warningMessage: (totalCount, threshold) => `🔍 Skip condition matched (${totalCount} items found, threshold: ${threshold}). Workflow execution will be prevented by activation job.`,
    successMessage: (totalCount, threshold) => `✓ Found ${totalCount} matches (below threshold of ${threshold}), workflow can proceed`,
    denialSummaryMessage: (totalCount, threshold) => `Skip-if-match query matched: ${totalCount} item(s) found (threshold: ${threshold}).`,
    denialSummaryNextStep: "Update `on.skip-if-match:` in the workflow frontmatter if this skip was unexpected.",
  });
}

module.exports = { main };
