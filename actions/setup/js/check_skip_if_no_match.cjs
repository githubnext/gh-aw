// @ts-check
/// <reference types="@actions/github-script" />

const { runSkipQueryGate } = require("./check_skip_if_helpers.cjs");

async function main() {
  const { GH_AW_SKIP_QUERY: skipQuery, GH_AW_WORKFLOW_NAME: workflowName, GH_AW_SKIP_MIN_MATCHES: minMatchesStr = "1", GH_AW_SKIP_SCOPE: skipScope } = process.env;

  await runSkipQueryGate({
    skipQuery,
    workflowName,
    thresholdStr: minMatchesStr,
    thresholdEnvVar: "GH_AW_SKIP_MIN_MATCHES",
    thresholdLabel: "Minimum matches threshold",
    checkLabel: "skip-if-no-match",
    outputName: "skip_no_match_check_ok",
    skipScope,
    shouldSkip: (totalCount, threshold) => totalCount < threshold,
    warningMessage: (totalCount, threshold) => `🔍 Skip condition matched (${totalCount} items found, minimum required: ${threshold}). Workflow execution will be prevented by activation job.`,
    successMessage: (totalCount, threshold) => `✓ Found ${totalCount} matches (meets or exceeds minimum of ${threshold}), workflow can proceed`,
    denialSummaryMessage: (totalCount, threshold) => `Skip-if-no-match query returned too few results: ${totalCount} item(s) found (minimum required: ${threshold}).`,
    denialSummaryNextStep: "Update `on.skip-if-no-match:` in the workflow frontmatter if this skip was unexpected.",
  });
}

module.exports = { main };
