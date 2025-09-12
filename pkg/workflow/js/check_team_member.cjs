/**
 * Custom setCancelled function that uses self-cancellation
 * @param {string} message - The cancellation message
 */
async function setCancelled(message) {
  try {
    // Cancel the current workflow run using GitHub Actions API
    await github.rest.actions.cancelWorkflowRun({
      owner: context.repo.owner,
      repo: context.repo.repo,
      run_id: context.runId,
    });

    core.info(`Cancellation requested for this workflow run: ${message}`);
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.warning(`Failed to cancel workflow run: ${errorMessage}`);
    // Fallback to core.setFailed if API call fails (since core.setCancelled doesn't exist in types)
    core.setFailed(message);
  }
}

async function main() {
  const actor = context.actor;
  const { owner, repo } = context.repo;

  // Check if the actor has repository access (admin, maintain permissions)
  try {
    console.log(
      `Checking if user '${actor}' is admin or maintainer of ${owner}/${repo}`
    );

    const repoPermission =
      await github.rest.repos.getCollaboratorPermissionLevel({
        owner: owner,
        repo: repo,
        username: actor,
      });

    const permission = repoPermission.data.permission;
    console.log(`Repository permission level: ${permission}`);

    if (permission === "admin" || permission === "maintain") {
      console.log(`User has ${permission} access to repository`);
      core.setOutput("is_team_member", "true");
      return;
    }
  } catch (repoError) {
    const errorMessage =
      repoError instanceof Error ? repoError.message : String(repoError);
    core.warning(`Repository permission check failed: ${errorMessage}`);
  }

  // Team membership check failed - use self-cancellation
  const failureMessage = `Access denied: User '${actor}' is not authorized to trigger this workflow. Only admin or maintainer users can run this workflow.`;
  core.warning(`❌ ${failureMessage}`);

  await setCancelled(failureMessage);

  // Set output for any dependent steps that might check before cancellation takes effect
  core.setOutput("is_team_member", "false");

  // Return to finish the script
  return;
}
await main();
