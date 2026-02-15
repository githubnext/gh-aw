// @ts-check
/// <reference types="@actions/github-script" />

const { checkRepositoryPermission } = require("./check_permissions_utils.cjs");

/**
 * Check if the workflow should be skipped based on user's role
 * Reads skip-roles from GH_AW_SKIP_ROLES environment variable
 * If the user has one of the skip-roles, set skip_roles_ok to false (skip the workflow)
 * Otherwise, set skip_roles_ok to true (allow the workflow to proceed)
 */
async function main() {
  const { eventName } = context;
  const actor = context.actor;
  const { owner, repo } = context.repo;

  // Parse skip-roles from environment variable
  const skipRolesEnv = process.env.GH_AW_SKIP_ROLES;
  if (!skipRolesEnv || skipRolesEnv.trim() === "") {
    // No skip-roles configured, workflow should proceed
    core.info("✅ No skip-roles configured, workflow will proceed");
    core.setOutput("skip_roles_ok", "true");
    core.setOutput("result", "no_skip_roles");
    return;
  }

  const skipRoles = skipRolesEnv
    .split(",")
    .map(r => r.trim())
    .filter(r => r);
  core.info(`Checking if user '${actor}' has any of the skip-roles: ${skipRoles.join(", ")}`);

  // Check the user's repository permission
  const result = await checkRepositoryPermission(actor, owner, repo, skipRoles);

  if (result.error) {
    // API error - fail safe and allow the workflow to proceed
    core.warning(`⚠️ Repository permission check failed: ${result.error}`);
    core.setOutput("skip_roles_ok", "true");
    core.setOutput("result", "api_error");
    core.setOutput("error_message", `Repository permission check failed: ${result.error}`);
    return;
  }

  if (result.authorized) {
    // User has one of the skip-roles, skip the workflow
    core.info(`❌ User '${actor}' has role '${result.permission}' which is in skip-roles [${skipRoles.join(", ")}]. Workflow will be skipped.`);
    core.setOutput("skip_roles_ok", "false");
    core.setOutput("result", "skipped");
    core.setOutput("user_permission", result.permission);
    core.setOutput("error_message", `Workflow skipped: User '${actor}' has role '${result.permission}' which is in skip-roles: [${skipRoles.join(", ")}]`);
  } else {
    // User does NOT have any of the skip-roles, allow workflow to proceed
    core.info(`✅ User '${actor}' has role '${result.permission}' which is NOT in skip-roles [${skipRoles.join(", ")}]. Workflow will proceed.`);
    core.setOutput("skip_roles_ok", "true");
    core.setOutput("result", "not_skipped");
    core.setOutput("user_permission", result.permission);
  }
}

module.exports = { main };
