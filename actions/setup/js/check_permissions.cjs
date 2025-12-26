// @ts-check
/// <reference types="@actions/github-script" />

const { parseRequiredPermissions, checkRepositoryPermission } = require("./check_permissions_utils.cjs");

async function main() {
  const { eventName } = context;

  // skip check for safe events
  // workflow_run is intentionally excluded due to HIGH security risks:
  // - Privilege escalation (inherits permissions from triggering workflow)
  // - Branch protection bypass (can execute on protected branches)
  // - Secret exposure (secrets available from untrusted code)
  const safeEvents = ["workflow_dispatch", "schedule"];
  if (safeEvents.includes(eventName)) {
    core.info(`✅ Event ${eventName} does not require validation`);
    return;
  }

  const actor = context.actor;
  const { owner, repo } = context.repo;
  const requiredPermissions = parseRequiredPermissions();

  if (!requiredPermissions || requiredPermissions.length === 0) {
    core.error("❌ Configuration error: Required permissions not specified. Contact repository administrator.");
    core.setFailed("Configuration error: Required permissions not specified");
    return;
  }

  // Check if the actor has the required repository permissions
  const result = await checkRepositoryPermission(actor, owner, repo, requiredPermissions);

  if (result.error) {
    core.setFailed(`Repository permission check failed: ${result.error}`);
    return;
  }

  if (!result.authorized) {
    // Fail the workflow when permission check fails (cancellation handled by activation job's if condition)
    core.warning(`Access denied: Only authorized users can trigger this workflow. User '${actor}' is not authorized. Required permissions: ${requiredPermissions.join(", ")}`);
    core.setFailed(`Access denied: User '${actor}' is not authorized. Required permissions: ${requiredPermissions.join(", ")}`);
  }
}

module.exports = { main };
