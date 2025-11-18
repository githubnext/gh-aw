// @ts-check
/// <reference types="@actions/github-script" />

async function main() {
  const skipQuery = process.env.GH_AW_SKIP_QUERY;
  const workflowName = process.env.GH_AW_WORKFLOW_NAME;

  if (!skipQuery) {
    core.setFailed("Configuration error: GH_AW_SKIP_QUERY not specified.");
    return;
  }

  if (!workflowName) {
    core.setFailed("Configuration error: GH_AW_WORKFLOW_NAME not specified.");
    return;
  }

  core.info(`Checking skip-if-match query: ${skipQuery}`);

  // Get repository information from context
  const { owner, repo } = github.context.repo;

  // Scope the query to the current repository
  const scopedQuery = `${skipQuery} repo:${owner}/${repo}`;

  core.info(`Scoped query: ${scopedQuery}`);

  try {
    // Search for issues and pull requests using the GitHub API
    const response = await github.rest.search.issuesAndPullRequests({
      q: scopedQuery,
      per_page: 1, // We only need to know if there are any matches
    });

    const totalCount = response.data.total_count;
    core.info(`Search found ${totalCount} matching items`);

    if (totalCount > 0) {
      core.warning(`🔍 Skip condition matched (${totalCount} items found). Workflow execution will be prevented by activation job.`);
      core.setOutput("skip_check_ok", "false");
      return;
    }

    core.info("✓ No matches found, workflow can proceed");
    core.setOutput("skip_check_ok", "true");
  } catch (error) {
    core.setFailed(`Failed to execute search query: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }
}
await main();
