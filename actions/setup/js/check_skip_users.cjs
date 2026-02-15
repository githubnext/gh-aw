// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Check if the workflow should be skipped based on user identity
 * Reads skip-users from GH_AW_SKIP_USERS environment variable
 * If the github.actor is in the skip-users list, set skip_users_ok to false (skip the workflow)
 * Otherwise, set skip_users_ok to true (allow the workflow to proceed)
 */
async function main() {
  const { eventName } = context;
  const actor = context.actor;

  // Parse skip-users from environment variable
  const skipUsersEnv = process.env.GH_AW_SKIP_USERS;
  if (!skipUsersEnv || skipUsersEnv.trim() === "") {
    // No skip-users configured, workflow should proceed
    core.info("✅ No skip-users configured, workflow will proceed");
    core.setOutput("skip_users_ok", "true");
    core.setOutput("result", "no_skip_users");
    return;
  }

  const skipUsers = skipUsersEnv
    .split(",")
    .map(u => u.trim())
    .filter(u => u);
  core.info(`Checking if user '${actor}' is in skip-users: ${skipUsers.join(", ")}`);

  // Check if the actor is in the skip-users list
  if (skipUsers.includes(actor)) {
    // User is in skip-users, skip the workflow
    core.info(`❌ User '${actor}' is in skip-users [${skipUsers.join(", ")}]. Workflow will be skipped.`);
    core.setOutput("skip_users_ok", "false");
    core.setOutput("result", "skipped");
    core.setOutput("error_message", `Workflow skipped: User '${actor}' is in skip-users: [${skipUsers.join(", ")}]`);
  } else {
    // User is NOT in skip-users, allow workflow to proceed
    core.info(`✅ User '${actor}' is NOT in skip-users [${skipUsers.join(", ")}]. Workflow will proceed.`);
    core.setOutput("skip_users_ok", "true");
    core.setOutput("result", "not_skipped");
  }
}

module.exports = { main };
