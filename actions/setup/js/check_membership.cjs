// @ts-check
/// <reference types="@actions/github-script" />

const { parseRequiredPermissions, parseAllowedBots, checkRepositoryPermission, checkBotStatus, isAllowedBot } = require("./check_permissions_utils.cjs");
const { writeDenialSummary } = require("./pre_activation_summary.cjs");

async function main() {
  const { eventName } = context;
  const actor = context.actor;
  const { owner, repo } = context.repo;
  const requiredPermissions = parseRequiredPermissions();
  const allowedBots = parseAllowedBots();

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
  // merge_group is safe because:
  // - Only triggered by GitHub's merge queue system (not user-initiated)
  // - Requires branch protection rules to be enabled
  // - Validates combined state of multiple PRs before merging
  const safeEvents = ["schedule", "merge_group"];
  if (safeEvents.includes(eventName)) {
    core.info(`✅ Event ${eventName} does not require validation`);
    core.setOutput("is_team_member", "true");
    core.setOutput("result", "safe_event");
    return;
  }

  if (requiredPermissions.length === 0) {
    core.warning("❌ Configuration error: Required permissions not specified. Contact repository administrator.");
    core.setOutput("is_team_member", "false");
    core.setOutput("result", "config_error");
    core.setOutput("error_message", "Configuration error: Required permissions not specified");
    await writeDenialSummary("Configuration error: Required permissions not specified.", "Contact the repository administrator to fix the workflow frontmatter configuration.");
    return;
  }

  // Check if the actor has the required repository permissions
  const result = await checkRepositoryPermission(actor, owner, repo, requiredPermissions);

  if (result.authorized) {
    core.setOutput("is_team_member", "true");
    core.setOutput("result", "authorized");
    core.setOutput("user_permission", result.permission);
  } else {
    // User doesn't have required permissions (or the permission check failed with an error).
    // Always attempt the bot allowlist fallback before giving up, so that GitHub Apps whose
    // actor is not a recognized GitHub user (e.g. "Copilot") are not silently denied.
    if (allowedBots.length > 0) {
      core.info(`Checking if actor '${actor}' is in allowed bots list: ${allowedBots.join(", ")}`);

      if (isAllowedBot(actor, allowedBots)) {
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
          const errorMessage = `Access denied: Bot '${actor}' is not active/installed on this repository`;
          core.warning(`Bot '${actor}' is in the allowed list but not active/installed on ${owner}/${repo}`);
          core.setOutput("is_team_member", "false");
          core.setOutput("result", "bot_not_active");
          core.setOutput("user_permission", result.permission ?? "bot");
          core.setOutput("error_message", errorMessage);
          await writeDenialSummary(errorMessage, "The bot is in the allowed list but is not installed or active on this repository. Install the GitHub App and try again.");
          return;
        } else {
          core.info(`Actor '${actor}' is in allowed bots list but bot status check failed`);
        }
      }
    }

    // Not authorized by role or bot
    if (result.error) {
      const errorMessage = `Repository permission check failed: ${result.error}`;
      core.setOutput("is_team_member", "false");
      core.setOutput("result", "api_error");
      core.setOutput("error_message", errorMessage);
      await writeDenialSummary(errorMessage, "The permission check failed with a GitHub API error. Check the `pre_activation` job log for details.");
    } else {
      const errorMessage =
        `Access denied: User '${actor}' is not authorized. Required permissions: ${requiredPermissions.join(", ")}. ` +
        `To allow this user to run the workflow, add their role to the frontmatter. Example: roles: [${requiredPermissions.join(", ")}, ${result.permission}]`;
      core.setOutput("is_team_member", "false");
      core.setOutput("result", "insufficient_permissions");
      core.setOutput("user_permission", result.permission);
      core.setOutput("error_message", errorMessage);
      await writeDenialSummary(errorMessage, `To allow a bot or GitHub App actor, add it to \`on.bots:\` in the workflow frontmatter. ` + `To change the required roles for human actors, update \`on.roles:\` in the workflow frontmatter.`);
    }
  }
}

module.exports = { main };
