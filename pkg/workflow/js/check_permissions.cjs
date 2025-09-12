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
  const { eventName } = context;

  // skip check for safe events
  const safeEvents = ["workflow_dispatch", "workflow_run", "schedule"];
  if (safeEvents.includes(eventName)) {
    console.log(`✅ Event ${eventName} does not require validation`);
    return;
  }

  const actor = context.actor;
  const { owner, repo } = context.repo;
  const requiredPermissionsEnv = process.env.GITHUB_AW_REQUIRED_ROLES;
  const requiredPermissions = requiredPermissionsEnv
    ? requiredPermissionsEnv.split(",").filter(p => p.trim() !== "")
    : [];

  if (!requiredPermissions || requiredPermissions.length === 0) {
    core.error(
      "❌ Configuration error: Required permissions not specified. Contact repository administrator."
    );
    await setCancelled(
      "Configuration error: Required permissions not specified"
    );
    return;
  }

  // Check if the actor has the required repository permissions
  try {
    console.log(
      `Checking if user '${actor}' has required permissions for ${owner}/${repo}`
    );
    console.log(`Required permissions: ${requiredPermissions.join(", ")}`);

    const repoPermission =
      await github.rest.repos.getCollaboratorPermissionLevel({
        owner: owner,
        repo: repo,
        username: actor,
      });

    const permission = repoPermission.data.permission;
    console.log(`Repository permission level: ${permission}`);

    // Check if user has one of the required permission levels
    for (const requiredPerm of requiredPermissions) {
      if (
        permission === requiredPerm ||
        (requiredPerm === "maintainer" && permission === "maintain")
      ) {
        console.log(`✅ User has ${permission} access to repository`);
        return;
      }
    }

    console.log(
      `User permission '${permission}' does not meet requirements: ${requiredPermissions.join(", ")}`
    );
  } catch (repoError) {
    const errorMessage =
      repoError instanceof Error ? repoError.message : String(repoError);
    core.error(`Repository permission check failed: ${errorMessage}`);
    await setCancelled(`Repository permission check failed: ${errorMessage}`);
    return;
  }

  // Cancel the job when permission check fails
  core.warning(
    `❌ Access denied: Only authorized users can trigger this workflow. User '${actor}' is not authorized. Required permissions: ${requiredPermissions.join(", ")}`
  );
  await setCancelled(
    `Access denied: User '${actor}' is not authorized. Required permissions: ${requiredPermissions.join(", ")}`
  );
  return;
}
await main();
