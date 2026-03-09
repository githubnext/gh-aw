// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Shared utility for repository permission validation
 * Used by both check_permissions.cjs and check_membership.cjs
 */

/**
 * Parse required permissions from environment variable
 * @returns {string[]} Array of required permission levels
 */
function parseRequiredPermissions() {
  return process.env.GH_AW_REQUIRED_ROLES?.split(",").filter(p => p.trim()) ?? [];
}

/**
 * Parse allowed bot identifiers from environment variable
 * @returns {string[]} Array of allowed bot identifiers
 */
function parseAllowedBots() {
  return process.env.GH_AW_ALLOWED_BOTS?.split(",").filter(b => b.trim()) ?? [];
}

/**
 * Canonicalize a bot/App identifier by stripping the [bot] suffix.
 * Both "my-app" and "my-app[bot]" normalize to "my-app".
 * @param {string} name - Bot identifier (with or without [bot] suffix)
 * @returns {string} The base slug without [bot] suffix
 */
function canonicalizeBotIdentifier(name) {
  return name.endsWith("[bot]") ? name.slice(0, -5) : name;
}

/**
 * Check if an actor matches any entry in the allowed bots list,
 * treating <slug> and <slug>[bot] as equivalent App identities.
 * @param {string} actor - The runtime actor name
 * @param {string[]} allowedBots - Array of allowed bot identifiers
 * @returns {boolean}
 */
function isAllowedBot(actor, allowedBots) {
  const canonicalActor = canonicalizeBotIdentifier(actor);
  return allowedBots.some(bot => canonicalizeBotIdentifier(bot) === canonicalActor);
}

/**
 * Check if the actor is a bot and if it's active on the repository.
 * Accepts both <slug> and <slug>[bot] actor forms, since GitHub Apps
 * may appear either way depending on the event context.
 * @param {string} actor - GitHub username to check
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @returns {Promise<{isBot: boolean, isActive: boolean, error?: string}>}
 */
async function checkBotStatus(actor, owner, repo) {
  try {
    // GitHub Apps can appear as either <slug> or <slug>[bot].
    // Treat both forms as a bot identity; always query the API with the [bot] form.
    const actorHasBotSuffix = actor.endsWith("[bot]");
    const actorForApi = actorHasBotSuffix ? actor : `${actor}[bot]`;

    core.info(`Checking if bot '${actor}' is active on ${owner}/${repo}`);

    // Try to get the bot's permission level to verify it's installed/active on the repo.
    // GitHub Apps/bots that are installed on a repository show up in the collaborators.
    // Use the [bot]-suffixed form since that is how GitHub App identities are listed.
    try {
      const botPermission = await github.rest.repos.getCollaboratorPermissionLevel({
        owner,
        repo,
        username: actorForApi,
      });

      core.info(`Bot '${actor}' is active with permission level: ${botPermission.data.permission}`);
      return { isBot: true, isActive: true };
    } catch (botError) {
      // If we get a 404, the bot is not installed/active on this repository
      // @ts-expect-error - Error handling with optional chaining
      if (botError?.status === 404) {
        core.warning(`Bot '${actor}' is not active/installed on ${owner}/${repo}`);
        return { isBot: true, isActive: false };
      }
      // For other errors, we'll treat as inactive to be safe
      const errorMessage = getErrorMessage(botError);
      core.warning(`Failed to check bot status: ${errorMessage}`);
      return { isBot: true, isActive: false, error: errorMessage };
    }
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    core.warning(`Error checking bot status: ${errorMessage}`);
    return { isBot: false, isActive: false, error: errorMessage };
  }
}

/**
 * Check if user has required repository permissions
 * @param {string} actor - GitHub username to check
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string[]} requiredPermissions - Array of required permission levels
 * @returns {Promise<{authorized: boolean, permission?: string, error?: string}>}
 */
async function checkRepositoryPermission(actor, owner, repo, requiredPermissions) {
  try {
    core.info(`Checking if user '${actor}' has required permissions for ${owner}/${repo}`);
    core.info(`Required permissions: ${requiredPermissions.join(", ")}`);

    const repoPermission = await github.rest.repos.getCollaboratorPermissionLevel({
      owner,
      repo,
      username: actor,
    });

    const permission = repoPermission.data.permission;
    core.info(`Repository permission level: ${permission}`);

    // Check if user has one of the required permission levels
    const hasPermission = requiredPermissions.some(requiredPerm => permission === requiredPerm || (requiredPerm === "maintainer" && permission === "maintain"));

    if (hasPermission) {
      core.info(`✅ User has ${permission} access to repository`);
      return { authorized: true, permission };
    }

    core.warning(`User permission '${permission}' does not meet requirements: ${requiredPermissions.join(", ")}`);
    return { authorized: false, permission };
  } catch (repoError) {
    const errorMessage = getErrorMessage(repoError);
    core.warning(`Repository permission check failed: ${errorMessage}`);
    return { authorized: false, error: errorMessage };
  }
}

module.exports = {
  parseRequiredPermissions,
  parseAllowedBots,
  canonicalizeBotIdentifier,
  isAllowedBot,
  checkRepositoryPermission,
  checkBotStatus,
};
