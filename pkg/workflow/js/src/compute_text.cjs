// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Sanitizes content for safe output in GitHub Actions
 * @param {string} content - The content to sanitize
 * @returns {string} The sanitized content
 */
const { sanitizeContent } = require("./lib/sanitize.cjs");

async function main() {
  let text = "";

  const actor = context.actor;
  const { owner, repo } = context.repo;

  // Check if the actor has repository access (admin, maintain permissions)
  const repoPermission = await github.rest.repos.getCollaboratorPermissionLevel({
    owner: owner,
    repo: repo,
    username: actor,
  });

  const permission = repoPermission.data.permission;
  core.info(`Repository permission level: ${permission}`);

  if (permission !== "admin" && permission !== "maintain") {
    core.setOutput("text", "");
    return;
  }

  // Determine current body text based on event context
  switch (context.eventName) {
    case "issues":
      // For issues: title + body
      if (context.payload.issue) {
        const title = context.payload.issue.title || "";
        const body = context.payload.issue.body || "";
        text = `${title}\n\n${body}`;
      }
      break;

    case "pull_request":
      // For pull requests: title + body
      if (context.payload.pull_request) {
        const title = context.payload.pull_request.title || "";
        const body = context.payload.pull_request.body || "";
        text = `${title}\n\n${body}`;
      }
      break;

    case "pull_request_target":
      // For pull request target events: title + body
      if (context.payload.pull_request) {
        const title = context.payload.pull_request.title || "";
        const body = context.payload.pull_request.body || "";
        text = `${title}\n\n${body}`;
      }
      break;

    case "issue_comment":
      // For issue comments: comment body
      if (context.payload.comment) {
        text = context.payload.comment.body || "";
      }
      break;

    case "pull_request_review_comment":
      // For PR review comments: comment body
      if (context.payload.comment) {
        text = context.payload.comment.body || "";
      }
      break;

    case "pull_request_review":
      // For PR reviews: review body
      if (context.payload.review) {
        text = context.payload.review.body || "";
      }
      break;

    case "discussion":
      // For discussions: title + body
      if (context.payload.discussion) {
        const title = context.payload.discussion.title || "";
        const body = context.payload.discussion.body || "";
        text = `${title}\n\n${body}`;
      }
      break;

    case "discussion_comment":
      // For discussion comments: comment body
      if (context.payload.comment) {
        text = context.payload.comment.body || "";
      }
      break;

    default:
      // Default: empty text
      text = "";
      break;
  }

  // Sanitize the text before output
  const sanitizedText = sanitizeContent(text);

  // Display sanitized text in logs
  core.info(`text: ${sanitizedText}`);

  // Set the sanitized text as output
  core.setOutput("text", sanitizedText);
}

await main();
