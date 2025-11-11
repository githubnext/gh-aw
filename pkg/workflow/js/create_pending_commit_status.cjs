// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Creates a pending commit status for the agentic workflow
 * This indicates to users that the workflow is in progress
 */
async function main() {
  // Check if commit SHA is available
  const commitSha = context.sha;
  if (!commitSha) {
    core.info("No commit SHA available in context - skipping commit status creation");
    return;
  }

  // Read configuration from environment variables
  const statusContext = process.env.GH_AW_COMMIT_STATUS_CONTEXT || "agentic-workflow";
  const runId = context.runId;
  const owner = context.repo.owner;
  const repo = context.repo.repo;

  // Build the target URL pointing to the workflow run
  const targetUrl = context.payload.repository
    ? `${context.payload.repository.html_url}/actions/runs/${runId}`
    : `${process.env.GITHUB_SERVER_URL || "https://github.com"}/${owner}/${repo}/actions/runs/${runId}`;

  core.info(`Creating pending commit status for commit: ${commitSha}`);
  core.info(`Status context: ${statusContext}`);
  core.info(`Target URL: ${targetUrl}`);

  try {
    // Create the pending commit status
    await github.rest.repos.createCommitStatus({
      owner,
      repo,
      sha: commitSha,
      state: "pending",
      context: statusContext,
      description: "Agentic workflow is running",
      target_url: targetUrl,
    });

    core.info("✓ Successfully created pending commit status");
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.setFailed(`Failed to create pending commit status: ${errorMessage}`);
  }
}

await main();
