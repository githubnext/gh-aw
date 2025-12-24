// @ts-check
/// <reference types="@actions/github-script" />

async function main() {
  const skipQuery = process.env.GH_AW_SKIP_QUERY;
  const workflowName = process.env.GH_AW_WORKFLOW_NAME;
  const maxMatchesStr = process.env.GH_AW_SKIP_MAX_MATCHES || "1";

  if (!skipQuery) {
    core.setFailed("Configuration error: GH_AW_SKIP_QUERY not specified.");
    return;
  }

  if (!workflowName) {
    core.setFailed("Configuration error: GH_AW_WORKFLOW_NAME not specified.");
    return;
  }

  const maxMatches = parseInt(maxMatchesStr, 10);
  if (isNaN(maxMatches) || maxMatches < 1) {
    core.setFailed(`Configuration error: GH_AW_SKIP_MAX_MATCHES must be a positive integer, got "${maxMatchesStr}".`);
    return;
  }

  core.info(`Checking skip-if-match query: ${skipQuery}`);
  core.info(`Maximum matches threshold: ${maxMatches}`);

  // Get repository information from context
  const { owner, repo } = context.repo;

  // Scope the query to the current repository
  const scopedQuery = `${skipQuery} repo:${owner}/${repo}`;

  core.info(`Scoped query: ${scopedQuery}`);

  try {
    // Search for issues and pull requests using the GitHub API
    // We only need to know if the count reaches the threshold
    const response = await github.rest.search.issuesAndPullRequests({
      q: scopedQuery,
      per_page: 1, // We only need the count, not the items
    });

    const totalCount = response.data.total_count;
    core.info(`Search found ${totalCount} matching items`);

    if (totalCount >= maxMatches) {
      core.warning(`🔍 Skip condition matched (${totalCount} items found, threshold: ${maxMatches}). Workflow execution will be prevented by activation job.`);
      core.setOutput("skip_check_ok", "false");
      return;
    }

    core.info(`✓ Found ${totalCount} matches (below threshold of ${maxMatches}), workflow can proceed`);
    core.setOutput("skip_check_ok", "true");
  } catch (error) {
    core.setFailed(`Failed to execute search query: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }
}

module.exports = { main };
