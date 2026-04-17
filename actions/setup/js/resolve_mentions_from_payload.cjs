// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Helper module for resolving allowed mentions from GitHub event payloads
 */

const { resolveMentionsLazily, isPayloadUserBot } = require("./resolve_mentions.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Push a non-bot user's login to the array if present.
 * @param {string[]} users - Target array
 * @param {{ login?: string, type?: string } | null | undefined} user - User object from payload
 */
function pushNonBotUser(users, user) {
  if (user?.login && !isPayloadUserBot(user)) {
    users.push(user.login);
  }
}

/**
 * Push non-bot assignee logins to the array.
 * @param {string[]} users - Target array
 * @param {Array<{ login?: string, type?: string }> | null | undefined} assignees - Assignees from payload
 */
function pushNonBotAssignees(users, assignees) {
  if (Array.isArray(assignees)) {
    for (const assignee of assignees) {
      pushNonBotUser(users, assignee);
    }
  }
}

/**
 * Extract known authors from a GitHub event payload based on event type.
 * @param {any} context - GitHub Actions context
 * @returns {string[]} Array of known author logins from the payload
 */
function extractKnownAuthorsFromPayload(context) {
  if (!context || typeof context !== "object") {
    return [];
  }

  const users = /** @type {string[]} */ [];
  const { eventName, payload = {} } = context;

  switch (eventName) {
    case "issues":
      pushNonBotUser(users, payload.issue?.user);
      pushNonBotAssignees(users, payload.issue?.assignees);
      break;

    case "pull_request":
    case "pull_request_target":
      pushNonBotUser(users, payload.pull_request?.user);
      pushNonBotAssignees(users, payload.pull_request?.assignees);
      break;

    case "issue_comment":
      pushNonBotUser(users, payload.comment?.user);
      pushNonBotUser(users, payload.issue?.user);
      pushNonBotAssignees(users, payload.issue?.assignees);
      break;

    case "pull_request_review_comment":
      pushNonBotUser(users, payload.comment?.user);
      pushNonBotUser(users, payload.pull_request?.user);
      pushNonBotAssignees(users, payload.pull_request?.assignees);
      break;

    case "pull_request_review":
      pushNonBotUser(users, payload.review?.user);
      pushNonBotUser(users, payload.pull_request?.user);
      pushNonBotAssignees(users, payload.pull_request?.assignees);
      break;

    case "discussion":
      pushNonBotUser(users, payload.discussion?.user);
      break;

    case "discussion_comment":
      pushNonBotUser(users, payload.comment?.user);
      pushNonBotUser(users, payload.discussion?.user);
      break;

    case "release":
      pushNonBotUser(users, payload.release?.author);
      break;

    case "workflow_dispatch":
      if (typeof context.actor === "string" && context.actor.length > 0) {
        users.push(context.actor);
      }
      break;

    default:
      break;
  }

  return users;
}

/**
 * Resolve allowed mentions from the current GitHub event context
 * @param {any} context - GitHub Actions context
 * @param {any} github - GitHub API client
 * @param {any} core - GitHub Actions core
 * @param {any} [mentionsConfig] - Mentions configuration from safe-outputs
 * @param {string[]} [extraKnownAuthors] - Additional known authors to allow (e.g. pre-fetched target issue authors)
 * @returns {Promise<string[]>} Array of allowed mention usernames
 */
async function resolveAllowedMentionsFromPayload(context, github, core, mentionsConfig, extraKnownAuthors) {
  // Return empty array if context is not available (e.g., in tests)
  if (!context || !github || !core) {
    return [];
  }

  // If mentions is explicitly set to false, return empty array (all mentions escaped)
  if (mentionsConfig && mentionsConfig.enabled === false) {
    core.info("[MENTIONS] Mentions explicitly disabled - all mentions will be escaped");
    return [];
  }

  // Get configuration options (with defaults)
  const allowTeamMembers = mentionsConfig?.allowTeamMembers !== false; // default: true
  const allowContext = mentionsConfig?.allowContext !== false; // default: true
  const allowedList = mentionsConfig?.allowed || [];
  const maxMentions = mentionsConfig?.max || 50;

  try {
    const { owner, repo } = context.repo;
    const knownAuthors = allowContext ? extractKnownAuthorsFromPayload(context) : [];

    // Add allowed list (always included regardless of configuration)
    knownAuthors.push(...allowedList);

    // Add extra known authors (e.g. pre-fetched target issue authors for explicit item_number)
    if (extraKnownAuthors && extraKnownAuthors.length > 0) {
      core.info(`[MENTIONS] Adding ${extraKnownAuthors.length} extra known author(s): ${extraKnownAuthors.join(", ")}`);
      knownAuthors.push(...extraKnownAuthors);
    }

    // If allow-team-members is disabled, only use known authors (context + allowed list)
    if (!allowTeamMembers) {
      core.info(`[MENTIONS] Team members disabled - only allowing context (${knownAuthors.length} users)`);
      if (knownAuthors.length > maxMentions) {
        core.warning(`[MENTIONS] Mention limit exceeded: ${knownAuthors.length} mentions, limiting to ${maxMentions}`);
      }
      return knownAuthors.slice(0, maxMentions);
    }

    // Build allowed mentions list from known authors and collaborators
    // We pass the known authors as fake mentions in text so they get processed
    const fakeText = knownAuthors.map(author => `@${author}`).join(" ");
    const mentionResult = await resolveMentionsLazily(fakeText, knownAuthors, owner, repo, github, core);
    let allowedMentions = mentionResult.allowedMentions;

    // Apply max limit
    if (allowedMentions.length > maxMentions) {
      core.warning(`[MENTIONS] Mention limit exceeded: ${allowedMentions.length} mentions, limiting to ${maxMentions}`);
      allowedMentions = allowedMentions.slice(0, maxMentions);
    }

    if (allowedMentions.length > 0) {
      core.info(`[OUTPUT COLLECTOR] Allowed mentions: ${allowedMentions.join(", ")}`);
    } else {
      core.info("[OUTPUT COLLECTOR] No allowed mentions - all mentions will be escaped");
    }

    return allowedMentions;
  } catch (error) {
    core.warning(`Failed to resolve mentions for output collector: ${getErrorMessage(error)}`);
    return [];
  }
}

module.exports = {
  resolveAllowedMentionsFromPayload,
  extractKnownAuthorsFromPayload,
  pushNonBotUser,
  pushNonBotAssignees,
};
