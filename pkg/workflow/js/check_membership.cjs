async function main() {
  const { eventName } = context;

  // skip check for safe events
  const safeEvents = ["workflow_dispatch", "workflow_run", "schedule"];
  if (safeEvents.includes(eventName)) {
    core.info(`✅ Event ${eventName} does not require validation`);
    core.setOutput("is_team_member", "true");
    core.setOutput("membership_check_result", "safe_event");
    return;
  }

  const actor = context.actor;
  const { owner, repo } = context.repo;
  const requiredPermissionsEnv = process.env.GITHUB_AW_REQUIRED_ROLES;
  const requiredPermissions = requiredPermissionsEnv ? requiredPermissionsEnv.split(",").filter(p => p.trim() !== "") : [];

  if (!requiredPermissions || requiredPermissions.length === 0) {
    core.warning("❌ Configuration error: Required permissions not specified. Contact repository administrator.");
    core.setOutput("is_team_member", "false");
    core.setOutput("membership_check_result", "config_error");
    core.setOutput("error_message", "Configuration error: Required permissions not specified");
    return;
  }

  // Check if the actor has the required repository permissions
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

    // Check if user has one of the required permission levels
    for (const requiredPerm of requiredPermissions) {
      if (permission === requiredPerm || (requiredPerm === "maintainer" && permission === "maintain")) {
        core.info(`✅ User has ${permission} access to repository`);
        core.setOutput("is_team_member", "true");
        core.setOutput("membership_check_result", "authorized");
        core.setOutput("user_permission", permission);
        return;
      }
    }

    core.warning(`User permission '${permission}' does not meet requirements: ${requiredPermissions.join(", ")}`);
    core.setOutput("is_team_member", "false");
    core.setOutput("membership_check_result", "insufficient_permissions");
    core.setOutput("user_permission", permission);
    core.setOutput(
      "error_message",
      `Access denied: User '${actor}' is not authorized. Required permissions: ${requiredPermissions.join(", ")}`
    );
  } catch (repoError) {
    const errorMessage = repoError instanceof Error ? repoError.message : String(repoError);
    core.warning(`Repository permission check failed: ${errorMessage}`);
    core.setOutput("is_team_member", "false");
    core.setOutput("membership_check_result", "api_error");
    core.setOutput("error_message", `Repository permission check failed: ${errorMessage}`);
    return;
  }
}
await main();
