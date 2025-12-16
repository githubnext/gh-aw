// @ts-check
/// <reference types="@actions/github-script" />

const { parseRequiredPermissions, parseAllowedBots, checkRepositoryPermission, checkBotStatus } = require("./check_permissions_utils.cjs");

async function main() {
  const { eventName } = context;
  const actor = context.actor;
  const { owner, repo } = context.repo;
  const requiredPermissions = parseRequiredPermissions();
  const allowedBots = parseAllowedBots();

  // For workflow_dispatch, always skip the external user check since it's a
  // manually triggered workflow that should run for any user with write access
  // (GitHub's native workflow_dispatch permission model already handles this)
  if (eventName === "workflow_dispatch") {
    core.info(`✅ Event ${eventName} does not require validation (manually triggered workflow)`);
    core.setOutput("is_team_member", "true");
    core.setOutput("result", "safe_event");
    return;
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
    // User doesn't have required permissions, check if they're an allowed bot
    if (allowedBots && allowedBots.length > 0) {
      core.info(`Checking if actor '${actor}' is in allowed bots list: ${allowedBots.join(", ")}`);

      if (allowedBots.includes(actor)) {
        core.info(`Actor '${actor}' is in the allowed bots list`);

        // Verify the bot is active/installed on the repository
        const botStatus = await checkBotStatus(actor, owner, repo);

        if (botStatus.isBot && botStatus.isActive) {
          core.info(`✅ Bot '${actor}' is active on the repository and authorized`);
          core.setOutput("is_team_member", "true");
          core.setOutput("result", "authorized_bot");
          core.setOutput("user_permission", "bot");
          return;
        } else if (botStatus.isBot && !botStatus.isActive) {
          core.warning(`Bot '${actor}' is in the allowed list but not active/installed on ${owner}/${repo}`);
          core.setOutput("is_team_member", "false");
          core.setOutput("result", "bot_not_active");
          core.setOutput("user_permission", result.permission);
          core.setOutput("error_message", `Access denied: Bot '${actor}' is not active/installed on this repository`);
          return;
        } else {
          core.info(`Actor '${actor}' is in allowed bots list but bot status check failed`);
        }
      }
    }

    // Not authorized by role or bot
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
