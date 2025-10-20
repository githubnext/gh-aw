async function main() {
  const { eventName } = context;

  // skip check for safe events
  const safeEvents = ["workflow_dispatch", "workflow_run", "schedule"];
  if (safeEvents.includes(eventName)) {
    core.info(`✅ Event ${eventName} does not require validation`);
    return;
  }

  const actor = context.actor;
  const { owner, repo } = context.repo;
  const requiredPermissionsEnv = process.env.GH_AW_REQUIRED_ROLES;
  const requiredPermissions = requiredPermissionsEnv ? requiredPermissionsEnv.split(",").filter(p => p.trim() !== "") : [];

  if (!requiredPermissions || requiredPermissions.length === 0) {
    core.error("❌ Configuration error: Required permissions not specified. Contact repository administrator.");
    core.setFailed("Configuration error: Required permissions not specified");
    return;
  }

  // Check if the actor has the required repository permissions
  try {
    core.info(`Checking if user '${actor}' has required permissions for ${owner}/${repo}`);
    core.info(`Required permissions: ${requiredPermissions.join(", ")}`);

    const repoPermission = await github.rest.repos.getCollaboratorPermissionLevel({
      owner: owner,
      repo: repo,
      username: actor,
    });

    const permission = repoPermission.data.permission;
    core.info(`Repository permission level: ${permission}`);

    // Check if user has one of the required permission levels
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
    core.setFailed(`Repository permission check failed: ${errorMessage}`);
    return;
  }

  // Fail the workflow when permission check fails (cancellation handled by activation job's if condition)
  core.warning(
    `Access denied: Only authorized users can trigger this workflow. User '${actor}' is not authorized. Required permissions: ${requiredPermissions.join(", ")}`
  );
  core.setFailed(`Access denied: User '${actor}' is not authorized. Required permissions: ${requiredPermissions.join(", ")}`);
}
await main();
