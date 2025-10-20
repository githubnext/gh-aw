async function main() {
  const { eventName } = context;
  const actor = context.actor;
  const { owner, repo } = context.repo;
  const requiredPermissionsEnv = process.env.GH_AW_REQUIRED_ROLES;
  const requiredPermissions = requiredPermissionsEnv ? requiredPermissionsEnv.split(",").filter(p => p.trim() !== "") : [];

  // For workflow_dispatch, only skip check if "write" is in the allowed roles
  // since workflow_dispatch can be triggered by users with write access
  if (eventName === "workflow_dispatch") {
    const hasWriteRole = requiredPermissions.includes("write");
    if (hasWriteRole) {
      core.info(`✅ Event ${eventName} does not require validation (write role allowed)`);
      core.setOutput("is_team_member", "true");
      core.setOutput("result", "safe_event");
      return;
    }
    // If write is not allowed, continue with permission check
    core.info(`Event ${eventName} requires validation (write role not allowed)`);
  }

  // skip check for other safe events
  const safeEvents = ["workflow_run", "schedule"];
  if (safeEvents.includes(eventName)) {
    core.info(`✅ Event ${eventName} does not require validation`);
    core.setOutput("is_team_member", "true");
    core.setOutput("result", "safe_event");
    return;
  }

  if (!requiredPermissions || requiredPermissions.length === 0) {
    core.warning("❌ Configuration error: Required permissions not specified. Contact repository administrator.");
    core.setOutput("is_team_member", "false");
    core.setOutput("result", "config_error");
    core.setOutput("error_message", "Configuration error: Required permissions not specified");
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
        core.setOutput("is_team_member", "true");
        core.setOutput("result", "authorized");
        core.setOutput("user_permission", permission);
        return;
      }
    }

    core.warning(`User permission '${permission}' does not meet requirements: ${requiredPermissions.join(", ")}`);
    core.setOutput("is_team_member", "false");
    core.setOutput("result", "insufficient_permissions");
    core.setOutput("user_permission", permission);
    core.setOutput(
      "error_message",
      `Access denied: User '${actor}' is not authorized. Required permissions: ${requiredPermissions.join(", ")}`
    );
  } catch (repoError) {
    const errorMessage = repoError instanceof Error ? repoError.message : String(repoError);
    core.warning(`Repository permission check failed: ${errorMessage}`);
    core.setOutput("is_team_member", "false");
    core.setOutput("result", "api_error");
    core.setOutput("error_message", `Repository permission check failed: ${errorMessage}`);
    return;
  }
}
await main();
