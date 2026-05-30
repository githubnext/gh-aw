// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_API, ERR_CONFIG } = require("./error_codes.cjs");
const { buildSearchQuery } = require("./check_skip_if_helpers.cjs");
const { writeDenialSummary } = require("./pre_activation_summary.cjs");

async function main() {
  const { GH_AW_SKIP_QUERY: skipQuery, GH_AW_WORKFLOW_NAME: workflowName, GH_AW_SKIP_MIN_MATCHES: minMatchesStr = "1", GH_AW_SKIP_SCOPE: skipScope } = process.env;

  if (!skipQuery) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_SKIP_QUERY not specified.`);
    return;
  }

  if (!workflowName) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_WORKFLOW_NAME not specified.`);
    return;
  }

  const minMatches = parseInt(minMatchesStr, 10);
  if (isNaN(minMatches) || minMatches < 1) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_SKIP_MIN_MATCHES must be a positive integer, got "${minMatchesStr}".`);
    return;
  }

  core.info(`Checking skip-if-no-match query: ${skipQuery}`);
  core.info(`Minimum matches threshold: ${minMatches}`);

  const searchQuery = buildSearchQuery(skipQuery, skipScope);

  try {
    const {
      data: { total_count: totalCount },
    } = await github.rest.search.issuesAndPullRequests({
      q: searchQuery,
      per_page: 1,
    });

    core.info(`Search found ${totalCount} matching items`);

    if (totalCount < minMatches) {
      core.warning(`🔍 Skip condition matched (${totalCount} items found, minimum required: ${minMatches}). Workflow execution will be prevented by activation job.`);
      core.setOutput("skip_no_match_check_ok", "false");
      await writeDenialSummary(`Skip-if-no-match query returned too few results: ${totalCount} item(s) found (minimum required: ${minMatches}).`, "Update `on.skip-if-no-match:` in the workflow frontmatter if this skip was unexpected.");
      return;
    }

    core.info(`✓ Found ${totalCount} matches (meets or exceeds minimum of ${minMatches}), workflow can proceed`);
    core.setOutput("skip_no_match_check_ok", "true");
  } catch (error) {
    core.setFailed(`${ERR_API}: Failed to execute search query: ${getErrorMessage(error)}`);
  }
}

module.exports = { main };
