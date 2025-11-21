// @ts-check
/// <reference types="@actions/github-script" />

const { parseRequiredPermissions, checkRepositoryPermission } = require("./check_permissions_utils.cjs");

async function main() {
  const { eventName } = context;
  const actor = context.actor;
  const { owner, repo } = context.repo;
  const requiredPermissions = parseRequiredPermissions();

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
  // workflow_run is intentionally excluded due to HIGH security risks:
  // - Privilege escalation (inherits permissions from triggering workflow)
  // - Branch protection bypass (can execute on protected branches)
  // - Secret exposure (secrets available from untrusted code)
  const safeEvents = ["schedule"];
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
  const result = await checkRepositoryPermission(actor, owner, repo, requiredPermissions);

  if (result.error) {
    core.setOutput("is_team_member", "false");
    core.setOutput("result", "api_error");
    core.setOutput("error_message", `Repository permission check failed: ${result.error}`);
    return;
  }

  if (result.authorized) {
    core.setOutput("is_team_member", "true");
    core.setOutput("result", "authorized");
    core.setOutput("user_permission", result.permission);
  } else {
    core.setOutput("is_team_member", "false");
    core.setOutput("result", "insufficient_permissions");
    core.setOutput("user_permission", result.permission);
    core.setOutput(
      "error_message",
      `Access denied: User '${actor}' is not authorized. Required permissions: ${requiredPermissions.join(", ")}`
    );
  }
}
await main();
