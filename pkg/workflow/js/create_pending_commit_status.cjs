// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Creates a pending commit status at the start of workflow execution
 * This runs in the activation job before the main agent job starts
 */

async function main() {
  try {
    const context = process.env.GH_AW_COMMIT_STATUS_CONTEXT || "agentic-workflow";
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Agentic Workflow";
    
    // Determine the SHA from the GitHub context
    let sha;
    
    // Try to get SHA from pull request head
    if (context.payload.pull_request?.head?.sha) {
      sha = context.payload.pull_request.head.sha;
      core.info(`Using SHA from PR head: ${sha}`);
    }
    // Try to get SHA from push event
    else if (context.payload.after && context.payload.after !== "0000000000000000000000000000000000000000") {
      sha = context.payload.after;
      core.info(`Using SHA from push event: ${sha}`);
    }
    // Fallback to workflow context SHA
    else if (context.sha) {
      sha = context.sha;
      core.info(`Using SHA from workflow context: ${sha}`);
    }
    else {
      core.setFailed("Could not determine commit SHA for status creation");
      return;
    }

    core.info(`Creating pending commit status with context: ${context}`);
    core.info(`SHA: ${sha}`);
    core.info(`Workflow: ${workflowName}`);

    // Create the pending commit status
    await github.rest.repos.createCommitStatus({
      owner: context.repo.owner,
      repo: context.repo.repo,
      sha: sha,
      state: "pending",
      context: context,
      description: `${workflowName} is running...`,
      target_url: `${context.serverUrl}/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`,
    });

    core.info("✓ Successfully created pending commit status");
    core.setOutput("status_context", context);
    core.setOutput("status_sha", sha);
  } catch (error) {
    core.error(`Failed to create pending commit status: ${error instanceof Error ? error.message : String(error)}`);
    // Don't fail the workflow if pending status creation fails
    // This is a non-critical operation
  }
}

await main();
