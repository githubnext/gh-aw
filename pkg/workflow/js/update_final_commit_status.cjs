// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Updates the commit status at the end of workflow execution
 * This runs in the update_reaction job after all other jobs complete
 * Maps the agent job result to commit status state
 */

async function main() {
  try {
    const statusContext = process.env.GH_AW_COMMIT_STATUS_CONTEXT || process.env.GH_AW_STATUS_CONTEXT || "agentic-workflow";
    const sha = process.env.GH_AW_STATUS_SHA;
    const agentConclusion = process.env.GH_AW_AGENT_CONCLUSION || "unknown";
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Agentic Workflow";
    const runUrl = process.env.GH_AW_RUN_URL;

    if (!sha) {
      core.setFailed("Missing status SHA - cannot update commit status");
      return;
    }

    core.info(`Updating commit status with context: ${statusContext}`);
    core.info(`SHA: ${sha}`);
    core.info(`Agent conclusion: ${agentConclusion}`);

    // Map agent job conclusion to commit status state
    let state;
    let description;
    
    switch (agentConclusion) {
      case "success":
        state = "success";
        description = `✓ ${workflowName} completed successfully`;
        break;
      case "failure":
        state = "failure";
        description = `✗ ${workflowName} failed`;
        break;
      case "cancelled":
        state = "error";
        description = `⊘ ${workflowName} was cancelled`;
        break;
      case "skipped":
        state = "error";
        description = `⊘ ${workflowName} was skipped`;
        break;
      case "timed_out":
        state = "error";
        description = `⊘ ${workflowName} timed out`;
        break;
      default:
        // Unknown status - mark as failed
        state = "failure";
        description = `✗ ${workflowName} completed with unknown status`;
        core.warning(`Unknown agent conclusion: ${agentConclusion}, marking as failure`);
        break;
    }

    core.info(`Setting status to: ${state}`);
    core.info(`Description: ${description}`);

    // Update the commit status
    await github.rest.repos.createCommitStatus({
      owner: github.context.repo.owner,
      repo: github.context.repo.name,
      sha: sha,
      state: state,
      context: statusContext,
      description: description,
      target_url: runUrl,
    });

    core.info("✓ Successfully updated commit status");
  } catch (error) {
    core.setFailed(`Failed to update commit status: ${error instanceof Error ? error.message : String(error)}`);
  }
}

await main();
