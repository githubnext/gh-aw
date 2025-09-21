async function setCheckPermissionsCancelled(message) {
  try {
    await github.rest.actions.cancelWorkflowRun({
      owner: context.repo.owner,
      repo: context.repo.repo,
      run_id: context.runId,
    });
    core.info(`Cancellation requested for this workflow run: ${message}`);
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.warning(`Failed to cancel workflow run: ${errorMessage}`);
    core.setFailed(message);
  }
}
async function checkPermissionsMain() {
  const { eventName } = context;
  const safeEvents = ["workflow_dispatch", "workflow_run", "schedule"];
  if (safeEvents.includes(eventName)) {
    core.info(`✅ Event ${eventName} does not require validation`);
    return;
  }
  const actor = context.actor;
  const { owner, repo } = context.repo;
  const requiredPermissionsEnv = process.env.GITHUB_AW_REQUIRED_ROLES;
  const requiredPermissions = requiredPermissionsEnv ? requiredPermissionsEnv.split(",").filter(p => p.trim() !== "") : [];
  if (!requiredPermissions || requiredPermissions.length === 0) {
    core.error("❌ Configuration error: Required permissions not specified. Contact repository administrator.");
    await setCheckPermissionsCancelled("Configuration error: Required permissions not specified");
    return;
  }
  try {
    core.debug(`Checking if user '${actor}' has required permissions for ${owner}/${repo}`);
    core.debug(`Required permissions: ${requiredPermissions.join(", ")}`);
    const repoPermission = await github.rest.repos.getCollaboratorPermissionLevel({
      owner: owner,
      repo: repo,
      username: actor,
    });
    const permission = repoPermission.data.permission;
    core.debug(`Repository permission level: ${permission}`);
    for (const requiredPerm of requiredPermissions) {
      if (permission === requiredPerm || (requiredPerm === "maintainer" && permission === "maintain")) {
        core.info(`✅ User has ${permission} access to repository`);
        return;
      }
    }
    core.warning(`User permission '${permission}' does not meet requirements: ${requiredPermissions.join(", ")}`);
  } catch (repoError) {
    const errorMessage = repoError instanceof Error ? repoError.message : String(repoError);
    core.warning(`Repository permission check failed: ${errorMessage}`);
    await setCheckPermissionsCancelled(`Repository permission check failed: ${errorMessage}`);
    return;
  }
  core.warning(
    `Access denied: Only authorized users can trigger this workflow. User '${actor}' is not authorized. Required permissions: ${requiredPermissions.join(", ")}`
  );
  await setCheckPermissionsCancelled(
    `Access denied: User '${actor}' is not authorized. Required permissions: ${requiredPermissions.join(", ")}`
  );
}
(async () => {
  await checkPermissionsMain();
})();
