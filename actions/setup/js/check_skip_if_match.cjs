// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_API, ERR_CONFIG } = require("./error_codes.cjs");
const { buildSearchQuery } = require("./check_skip_if_helpers.cjs");

async function main() {
  const { GH_AW_SKIP_QUERY: skipQuery, GH_AW_WORKFLOW_NAME: workflowName, GH_AW_SKIP_MAX_MATCHES: maxMatchesStr = "1", GH_AW_SKIP_SCOPE: skipScope } = process.env;

  if (!skipQuery) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_SKIP_QUERY not specified.`);
    return;
  }

  if (!workflowName) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_WORKFLOW_NAME not specified.`);
    return;
  }

  const maxMatches = parseInt(maxMatchesStr, 10);
  if (isNaN(maxMatches) || maxMatches < 1) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_SKIP_MAX_MATCHES must be a positive integer, got "${maxMatchesStr}".`);
    return;
  }

  core.info(`Checking skip-if-match query: ${skipQuery}`);
  core.info(`Maximum matches threshold: ${maxMatches}`);

  const searchQuery = buildSearchQuery(skipQuery, skipScope);

  try {
    const {
      data: { total_count: totalCount },
    } = await github.rest.search.issuesAndPullRequests({
      q: searchQuery,
      per_page: 1,
    });

    core.info(`Search found ${totalCount} matching items`);

    if (totalCount >= maxMatches) {
      core.warning(`🔍 Skip condition matched (${totalCount} items found, threshold: ${maxMatches}). Workflow execution will be prevented by activation job.`);
      core.setOutput("skip_check_ok", "false");
      return;
    }

    core.info(`✓ Found ${totalCount} matches (below threshold of ${maxMatches}), workflow can proceed`);
    core.setOutput("skip_check_ok", "true");
  } catch (error) {
    core.setFailed(`${ERR_API}: Failed to execute search query: ${getErrorMessage(error)}`);
  }
}

module.exports = { main };
