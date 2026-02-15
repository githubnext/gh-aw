// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Check if the workflow should be skipped based on bot/user identity
 * Reads skip-bots from GH_AW_SKIP_BOTS environment variable
 * If the github.actor is in the skip-bots list, set skip_bots_ok to false (skip the workflow)
 * Otherwise, set skip_bots_ok to true (allow the workflow to proceed)
 */
async function main() {
  const { eventName } = context;
  const actor = context.actor;

  // Parse skip-bots from environment variable
  const skipBotsEnv = process.env.GH_AW_SKIP_BOTS;
  if (!skipBotsEnv || skipBotsEnv.trim() === "") {
    // No skip-bots configured, workflow should proceed
    core.info("✅ No skip-bots configured, workflow will proceed");
    core.setOutput("skip_bots_ok", "true");
    core.setOutput("result", "no_skip_bots");
    return;
  }

  const skipBots = skipBotsEnv
    .split(",")
    .map(u => u.trim())
    .filter(u => u);
  core.info(`Checking if user '${actor}' is in skip-bots: ${skipBots.join(", ")}`);

  // Check if the actor is in the skip-bots list
  // Match both exact username and username with [bot] suffix
  // e.g., "github-actions" matches both "github-actions" and "github-actions[bot]"
  const isSkipped = skipBots.some(skipBot => {
    // Exact match
    if (actor === skipBot) {
      return true;
    }
    // Match with [bot] suffix
    if (actor === `${skipBot}[bot]`) {
      return true;
    }
    // Match if skip-bot has [bot] suffix and actor matches base name
    if (skipBot.endsWith("[bot]") && actor === skipBot.slice(0, -5)) {
      return true;
    }
    return false;
  });

  if (isSkipped) {
    // User is in skip-bots, skip the workflow
    core.info(`❌ User '${actor}' is in skip-bots [${skipBots.join(", ")}]. Workflow will be skipped.`);
    core.setOutput("skip_bots_ok", "false");
    core.setOutput("result", "skipped");
    core.setOutput("error_message", `Workflow skipped: User '${actor}' is in skip-bots: [${skipBots.join(", ")}]`);
  } else {
    // User is NOT in skip-bots, allow workflow to proceed
    core.info(`✅ User '${actor}' is NOT in skip-bots [${skipBots.join(", ")}]. Workflow will proceed.`);
    core.setOutput("skip_bots_ok", "true");
    core.setOutput("result", "not_skipped");
  }
}

module.exports = { main };
