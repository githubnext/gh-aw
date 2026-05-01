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
 * Detect a potential Dependabot Confused Deputy attack.
 *
 * Attack vectors defended against:
 *   - @dependabot recreate on a PR not authored by dependabot causes
 *     github.actor = 'dependabot[bot]' while pull_request.user.login remains
 *     the original (human) PR author.
 *   - @dependabot show on an issue causes github.actor = 'dependabot[bot]'
 *     while comment.user.login differs from the actor.
 *
 * Reference: https://labs.boostsecurity.io/articles/weaponizing-dependabot-pwn-request-at-its-finest/
 *
 * @param {string} actor - The current github.actor
 * @param {string} eventName - The GitHub event name (e.g. "pull_request", "issue_comment")
 * @param {object|undefined} payload - The GitHub event payload (context.payload)
 * @returns {boolean} true if the event looks like a confused deputy attack
 */
function isConfusedDeputyAttack(actor, eventName, payload) {
  if (!payload) return false;

  // For pull_request events, only check on the `synchronize` action.
  // The confused deputy attack (@dependabot recreate) triggers a synchronize event
  // with actor=dependabot[bot] but pull_request.user = original human author.
  // Other pull_request actions (labeled, unlabeled, assigned, review_requested, etc.)
  // legitimately have actor != pr_author — the actor is whoever performed the action,
  // not the PR author — so checking those would cause false positives.
  if (eventName === "pull_request" && payload.action === "synchronize") {
    const prAuthor = payload.pull_request?.user?.login;
    if (prAuthor !== undefined && prAuthor !== actor) {
      return true;
    }
  }

  // For pull_request_review events, the reviewer must match the actor.
  // The PR author (pull_request.user.login) is irrelevant here — the actor
  // is the person submitting the review, not the PR author.
  if (eventName === "pull_request_review") {
    const reviewAuthor = payload.review?.user?.login;
    if (reviewAuthor !== undefined && reviewAuthor !== actor) {
      return true;
    }
  }

  // For pull_request_review_comment events, the comment author must match the actor.
  // The PR author (pull_request.user.login) is irrelevant here — the actor
  // is the person writing the review comment, not the PR author.
  if (eventName === "pull_request_review_comment") {
    const commentAuthor = payload.comment?.user?.login;
    if (commentAuthor !== undefined && commentAuthor !== actor) {
      return true;
    }
  }

  // For issue_comment events, @dependabot show can trigger a comment from dependabot
  // with actor=dependabot[bot]. Verify the comment itself was authored by the actor.
  if (eventName === "issue_comment") {
    const commentAuthor = payload.comment?.user?.login;
    if (commentAuthor !== undefined && commentAuthor !== actor) {
      return true;
    }
  }

  return false;
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
    const actorSlug = canonicalizeBotIdentifier(actor);
    const actorForApi = actor.endsWith("[bot]") ? actor : `${actorSlug}[bot]`;

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
      // If we get a 404, the [bot]-suffixed form may not be listed as a collaborator.
      // Fall back to checking the non-[bot] (slug) form, as some GitHub Apps appear
      // under their plain slug name rather than the [bot]-suffixed form.
      if (botError?.status === 404) {
        try {
          const slugPermission = await github.rest.repos.getCollaboratorPermissionLevel({
            owner,
            repo,
            username: actorSlug,
          });
          core.info(`Bot '${actor}' is active (via slug form) with permission level: ${slugPermission.data.permission}`);
          return { isBot: true, isActive: true };
        } catch (slugError) {
          if (slugError?.status === 404) {
            core.warning(`Bot '${actor}' is not active/installed on ${owner}/${repo}`);
            return { isBot: true, isActive: false };
          }
          const errorMessage = getErrorMessage(slugError);
          core.warning(`Failed to check bot status: ${errorMessage}`);
          return { isBot: true, isActive: false, error: errorMessage };
        }
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
  isConfusedDeputyAttack,
  checkRepositoryPermission,
  checkBotStatus,
};
