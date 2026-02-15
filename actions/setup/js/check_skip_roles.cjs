// @ts-check
/// <reference types="@actions/github-script" />

const { checkRepositoryPermission } = require("./check_permissions_utils.cjs");

/**
 * Parse skip-roles from environment variable
 * @returns {string[]} Array of roles that should be skipped
 */
function parseSkipRoles() {
  return (
    process.env.GH_AW_SKIP_ROLES?.split(",")
      .map(r => r.trim())
      .filter(r => r) ?? []
  );
}

async function main() {
  const { eventName } = context;
  const actor = context.actor;
  const { owner, repo } = context.repo;
  const skipRoles = parseSkipRoles();

  // If no skip-roles configured, workflow should proceed
  if (!skipRoles || skipRoles.length === 0) {
    core.info("✅ No skip-roles configured - workflow can proceed");
    core.setOutput("skip_roles_ok", "true");
    core.setOutput("result", "no_skip_roles");
    return;
  }

  core.info(`Checking if user '${actor}' has any of the skip-roles: ${skipRoles.join(", ")}`);

  // For safe events that don't require permission checks, always proceed
  const safeEvents = ["schedule", "merge_group"];
  if (safeEvents.includes(eventName)) {
    core.info(`✅ Event ${eventName} is a safe event - skip-roles check not applicable`);
    core.setOutput("skip_roles_ok", "true");
    core.setOutput("result", "safe_event");
    return;
  }

  // Check user's repository permission
  const result = await checkRepositoryPermission(actor, owner, repo, skipRoles);

  if (result.error) {
    // If there's an API error, we'll allow the workflow to proceed to avoid blocking on transient issues
    core.warning(`⚠️ Could not verify user permissions: ${result.error}`);
    core.info("Allowing workflow to proceed due to API error");
    core.setOutput("skip_roles_ok", "true");
    core.setOutput("result", "api_error");
    core.setOutput("error_message", `Skip-roles check failed: ${result.error}`);
    return;
  }

  if (result.authorized) {
    // User has one of the skip-roles, workflow should be skipped
    core.info(`❌ User '${actor}' has role '${result.permission}' which is in skip-roles list`);
    core.info(`Workflow will be cancelled for this user`);
    core.setOutput("skip_roles_ok", "false");
    core.setOutput("result", "user_should_be_skipped");
    core.setOutput("user_permission", result.permission);
    core.setOutput("error_message", `Workflow skipped: User '${actor}' has role '${result.permission}' which is configured to skip this workflow (skip-roles: ${skipRoles.join(", ")})`);
  } else {
    // User does NOT have any of the skip-roles, workflow can proceed
    core.info(`✅ User '${actor}' does not have any skip-roles - workflow can proceed`);
    core.setOutput("skip_roles_ok", "true");
    core.setOutput("result", "user_not_in_skip_roles");
    core.setOutput("user_permission", result.permission);
  }
}

module.exports = { main };
