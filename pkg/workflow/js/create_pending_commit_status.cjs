// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Creates a pending commit status at the start of workflow execution
 * This runs in the activation job before the main agent job starts
 */

async function main() {
  try {
    const statusContext = process.env.GH_AW_COMMIT_STATUS_CONTEXT || "agentic-workflow";
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Agentic Workflow";
    
    // Determine the SHA from the GitHub context
    let sha;
    
    // Try to get SHA from pull request head (for pull_request and pull_request_target events)
    if (github.context.payload.pull_request && github.context.payload.pull_request.head && github.context.payload.pull_request.head.sha) {
      sha = github.context.payload.pull_request.head.sha;
      core.info(`Using SHA from PR head: ${sha}`);
    }
    // Try to get SHA from issue events that have associated pull request
    else if (github.context.payload.issue && github.context.payload.issue.pull_request) {
      // For issue events on PRs, we need to fetch the PR to get the head SHA
      const prNumber = github.context.payload.issue.number;
      core.info(`Fetching PR #${prNumber} to get head SHA...`);
      try {
        const { data: pr } = await github.rest.pulls.get({
          owner: github.context.repo.owner,
          repo: github.context.repo.repo,
          pull_number: prNumber,
        });
        sha = pr.head.sha;
        core.info(`Using SHA from PR #${prNumber} head: ${sha}`);
      } catch (prError) {
        core.warning(`Failed to fetch PR #${prNumber}: ${prError instanceof Error ? prError.message : String(prError)}`);
        // Continue to fallback options
      }
    }
    // Try to get SHA from push event
    if (!sha && github.context.payload.after && github.context.payload.after !== "0000000000000000000000000000000000000000") {
      sha = github.context.payload.after;
      core.info(`Using SHA from push event: ${sha}`);
    }
    // Fallback to workflow context SHA
    if (!sha && github.context.sha) {
      sha = github.context.sha;
      core.info(`Using SHA from workflow context: ${sha}`);
    }
    
    if (!sha) {
      core.setFailed("Could not determine commit SHA for status creation");
      return;
    }

    core.info(`Creating pending commit status with context: ${statusContext}`);
    core.info(`SHA: ${sha}`);
    core.info(`Workflow: ${workflowName}`);

    // Create the pending commit status
    await github.rest.repos.createCommitStatus({
      owner: github.context.repo.owner,
      repo: github.context.repo.repo,
      sha: sha,
      state: "pending",
      context: statusContext,
      description: `${workflowName} is running...`,
      target_url: `${github.context.serverUrl}/${github.context.repo.owner}/${github.context.repo.repo}/actions/runs/${github.context.runId}`,
    });

    core.info("✓ Successfully created pending commit status");
    core.setOutput("status_context", statusContext);
    core.setOutput("status_sha", sha);
  } catch (error) {
    core.error(`Failed to create pending commit status: ${error instanceof Error ? error.message : String(error)}`);
    // Don't fail the workflow if pending status creation fails
    // This is a non-critical operation
  }
}

await main();
