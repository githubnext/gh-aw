// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Helper module for resolving allowed mentions from GitHub event payloads
 */

const { resolveMentionsLazily, isPayloadUserBot } = require("./resolve_mentions.cjs");

/**
 * Resolve allowed mentions from the current GitHub event context
 * @param {any} context - GitHub Actions context
 * @param {any} github - GitHub API client
 * @param {any} core - GitHub Actions core
 * @returns {Promise<string[]>} Array of allowed mention usernames
 */
async function resolveAllowedMentionsFromPayload(context, github, core) {
  // Return empty array if context is not available (e.g., in tests)
  if (!context || !github || !core) {
    return [];
  }

  try {
    const { owner, repo } = context.repo;
    const knownAuthors = [];

    // Extract known authors from the event payload
    switch (context.eventName) {
      case "issues":
        if (context.payload.issue?.user?.login && !isPayloadUserBot(context.payload.issue.user)) {
          knownAuthors.push(context.payload.issue.user.login);
        }
        if (context.payload.issue?.assignees && Array.isArray(context.payload.issue.assignees)) {
          for (const assignee of context.payload.issue.assignees) {
            if (assignee?.login && !isPayloadUserBot(assignee)) {
              knownAuthors.push(assignee.login);
            }
          }
        }
        break;

      case "pull_request":
      case "pull_request_target":
        if (context.payload.pull_request?.user?.login && !isPayloadUserBot(context.payload.pull_request.user)) {
          knownAuthors.push(context.payload.pull_request.user.login);
        }
        if (context.payload.pull_request?.assignees && Array.isArray(context.payload.pull_request.assignees)) {
          for (const assignee of context.payload.pull_request.assignees) {
            if (assignee?.login && !isPayloadUserBot(assignee)) {
              knownAuthors.push(assignee.login);
            }
          }
        }
        break;

      case "issue_comment":
        if (context.payload.comment?.user?.login && !isPayloadUserBot(context.payload.comment.user)) {
          knownAuthors.push(context.payload.comment.user.login);
        }
        if (context.payload.issue?.user?.login && !isPayloadUserBot(context.payload.issue.user)) {
          knownAuthors.push(context.payload.issue.user.login);
        }
        if (context.payload.issue?.assignees && Array.isArray(context.payload.issue.assignees)) {
          for (const assignee of context.payload.issue.assignees) {
            if (assignee?.login && !isPayloadUserBot(assignee)) {
              knownAuthors.push(assignee.login);
            }
          }
        }
        break;

      case "pull_request_review_comment":
        if (context.payload.comment?.user?.login && !isPayloadUserBot(context.payload.comment.user)) {
          knownAuthors.push(context.payload.comment.user.login);
        }
        if (context.payload.pull_request?.user?.login && !isPayloadUserBot(context.payload.pull_request.user)) {
          knownAuthors.push(context.payload.pull_request.user.login);
        }
        if (context.payload.pull_request?.assignees && Array.isArray(context.payload.pull_request.assignees)) {
          for (const assignee of context.payload.pull_request.assignees) {
            if (assignee?.login && !isPayloadUserBot(assignee)) {
              knownAuthors.push(assignee.login);
            }
          }
        }
        break;

      case "pull_request_review":
        if (context.payload.review?.user?.login && !isPayloadUserBot(context.payload.review.user)) {
          knownAuthors.push(context.payload.review.user.login);
        }
        if (context.payload.pull_request?.user?.login && !isPayloadUserBot(context.payload.pull_request.user)) {
          knownAuthors.push(context.payload.pull_request.user.login);
        }
        if (context.payload.pull_request?.assignees && Array.isArray(context.payload.pull_request.assignees)) {
          for (const assignee of context.payload.pull_request.assignees) {
            if (assignee?.login && !isPayloadUserBot(assignee)) {
              knownAuthors.push(assignee.login);
            }
          }
        }
        break;

      case "discussion":
        if (context.payload.discussion?.user?.login && !isPayloadUserBot(context.payload.discussion.user)) {
          knownAuthors.push(context.payload.discussion.user.login);
        }
        break;

      case "discussion_comment":
        if (context.payload.comment?.user?.login && !isPayloadUserBot(context.payload.comment.user)) {
          knownAuthors.push(context.payload.comment.user.login);
        }
        if (context.payload.discussion?.user?.login && !isPayloadUserBot(context.payload.discussion.user)) {
          knownAuthors.push(context.payload.discussion.user.login);
        }
        break;

      case "release":
        if (context.payload.release?.author?.login && !isPayloadUserBot(context.payload.release.author)) {
          knownAuthors.push(context.payload.release.author.login);
        }
        break;

      case "workflow_dispatch":
        // Add the actor who triggered the workflow
        knownAuthors.push(context.actor);
        break;

      default:
        // No known authors for other event types
        break;
    }

    // Build allowed mentions list from known authors and collaborators
    // We pass the known authors as fake mentions in text so they get processed
    const fakeText = knownAuthors.map(author => `@${author}`).join(" ");
    const mentionResult = await resolveMentionsLazily(fakeText, knownAuthors, owner, repo, github, core);
    const allowedMentions = mentionResult.allowedMentions;

    // Log allowed mentions for debugging
    if (allowedMentions.length > 0) {
      core.info(`[OUTPUT COLLECTOR] Allowed mentions: ${allowedMentions.join(", ")}`);
    } else {
      core.info("[OUTPUT COLLECTOR] No allowed mentions - all mentions will be escaped");
    }

    return allowedMentions;
  } catch (error) {
    core.warning(`Failed to resolve mentions for output collector: ${error instanceof Error ? error.message : String(error)}`);
    // Return empty array on error
    return [];
  }
}

module.exports = {
  resolveAllowedMentionsFromPayload,
};
