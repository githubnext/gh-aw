// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

async function main() {
  const skipQuery = process.env.GH_AW_SKIP_QUERY;
  const workflowName = process.env.GH_AW_WORKFLOW_NAME;
  const minMatchesStr = process.env.GH_AW_SKIP_MIN_MATCHES ?? "1";

  if (!skipQuery) {
    core.setFailed("Configuration error: GH_AW_SKIP_QUERY not specified.");
    return;
  }

  if (!workflowName) {
    core.setFailed("Configuration error: GH_AW_WORKFLOW_NAME not specified.");
    return;
  }

  const minMatches = parseInt(minMatchesStr, 10);
  if (isNaN(minMatches) || minMatches < 1) {
    core.setFailed(`Configuration error: GH_AW_SKIP_MIN_MATCHES must be a positive integer, got "${minMatchesStr}".`);
    return;
  }

  core.info(`Checking skip-if-no-match query: ${skipQuery}`);
  core.info(`Minimum matches threshold: ${minMatches}`);

  const { owner, repo } = context.repo;
  const scopedQuery = `${skipQuery} repo:${owner}/${repo}`;

  core.info(`Scoped query: ${scopedQuery}`);

  try {
    const response = await github.rest.search.issuesAndPullRequests({
      q: scopedQuery,
      per_page: 1,
    });

    const totalCount = response.data.total_count;
    core.info(`Search found ${totalCount} matching items`);

    if (totalCount < minMatches) {
      core.warning(`🔍 Skip condition matched (${totalCount} items found, minimum required: ${minMatches}). Workflow execution will be prevented by activation job.`);
      core.setOutput("skip_no_match_check_ok", "false");
      return;
    }

    core.info(`✓ Found ${totalCount} matches (meets or exceeds minimum of ${minMatches}), workflow can proceed`);
    core.setOutput("skip_no_match_check_ok", "true");
  } catch (error) {
    core.setFailed(`Failed to execute search query: ${getErrorMessage(error)}`);
  }
}

module.exports = { main };
