// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Sanitizes content for safe output in GitHub Actions
 * @param {string} content - The content to sanitize
 * @returns {string} The sanitized content
 */
const { sanitizeIncomingText, writeRedactedDomainsLog } = require("./sanitize_incoming_text.cjs");
const { isPayloadUserBot } = require("./resolve_mentions.cjs");

async function main() {
  let text = "";
  /** @type {string[]} */
  let knownAuthors = [];

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
        // Add issue author to known authors (if not a bot)
        if (context.payload.issue.user?.login && !isPayloadUserBot(context.payload.issue.user)) {
          knownAuthors.push(context.payload.issue.user.login);
        }
        // Add issue assignees to known authors (if not bots)
        if (context.payload.issue.assignees && Array.isArray(context.payload.issue.assignees)) {
          for (const assignee of context.payload.issue.assignees) {
            if (assignee?.login && !isPayloadUserBot(assignee)) {
              knownAuthors.push(assignee.login);
            }
          }
        }
      }
      break;

    case "pull_request":
      // For pull requests: title + body
      if (context.payload.pull_request) {
        const title = context.payload.pull_request.title || "";
        const body = context.payload.pull_request.body || "";
        text = `${title}\n\n${body}`;
        // Add PR author to known authors (if not a bot)
        if (context.payload.pull_request.user?.login && !isPayloadUserBot(context.payload.pull_request.user)) {
          knownAuthors.push(context.payload.pull_request.user.login);
        }
        // Add PR assignees to known authors (if not bots)
        if (context.payload.pull_request.assignees && Array.isArray(context.payload.pull_request.assignees)) {
          for (const assignee of context.payload.pull_request.assignees) {
            if (assignee?.login && !isPayloadUserBot(assignee)) {
              knownAuthors.push(assignee.login);
            }
          }
        }
      }
      break;

    case "pull_request_target":
      // For pull request target events: title + body
      if (context.payload.pull_request) {
        const title = context.payload.pull_request.title || "";
        const body = context.payload.pull_request.body || "";
        text = `${title}\n\n${body}`;
        // Add PR author to known authors (if not a bot)
        if (context.payload.pull_request.user?.login && !isPayloadUserBot(context.payload.pull_request.user)) {
          knownAuthors.push(context.payload.pull_request.user.login);
        }
        // Add PR assignees to known authors (if not bots)
        if (context.payload.pull_request.assignees && Array.isArray(context.payload.pull_request.assignees)) {
          for (const assignee of context.payload.pull_request.assignees) {
            if (assignee?.login && !isPayloadUserBot(assignee)) {
              knownAuthors.push(assignee.login);
            }
          }
        }
      }
      break;

    case "issue_comment":
      // For issue comments: comment body
      if (context.payload.comment) {
        text = context.payload.comment.body || "";
        // Add comment author to known authors (if not a bot)
        if (context.payload.comment.user?.login && !isPayloadUserBot(context.payload.comment.user)) {
          knownAuthors.push(context.payload.comment.user.login);
        }
        // Also add the parent issue author to known authors (if not a bot)
        if (context.payload.issue?.user?.login && !isPayloadUserBot(context.payload.issue.user)) {
          knownAuthors.push(context.payload.issue.user.login);
        }
        // Add issue assignees to known authors (if not bots)
        if (context.payload.issue?.assignees && Array.isArray(context.payload.issue.assignees)) {
          for (const assignee of context.payload.issue.assignees) {
            if (assignee?.login && !isPayloadUserBot(assignee)) {
              knownAuthors.push(assignee.login);
            }
          }
        }
      }
      break;

    case "pull_request_review_comment":
      // For PR review comments: comment body
      if (context.payload.comment) {
        text = context.payload.comment.body || "";
        // Add comment author to known authors (if not a bot)
        if (context.payload.comment.user?.login && !isPayloadUserBot(context.payload.comment.user)) {
          knownAuthors.push(context.payload.comment.user.login);
        }
        // Also add the parent PR author to known authors (if not a bot)
        if (context.payload.pull_request?.user?.login && !isPayloadUserBot(context.payload.pull_request.user)) {
          knownAuthors.push(context.payload.pull_request.user.login);
        }
        // Add PR assignees to known authors (if not bots)
        if (context.payload.pull_request?.assignees && Array.isArray(context.payload.pull_request.assignees)) {
          for (const assignee of context.payload.pull_request.assignees) {
            if (assignee?.login && !isPayloadUserBot(assignee)) {
              knownAuthors.push(assignee.login);
            }
          }
        }
      }
      break;

    case "pull_request_review":
      // For PR reviews: review body
      if (context.payload.review) {
        text = context.payload.review.body || "";
        // Add review author to known authors (if not a bot)
        if (context.payload.review.user?.login && !isPayloadUserBot(context.payload.review.user)) {
          knownAuthors.push(context.payload.review.user.login);
        }
        // Also add the parent PR author to known authors (if not a bot)
        if (context.payload.pull_request?.user?.login && !isPayloadUserBot(context.payload.pull_request.user)) {
          knownAuthors.push(context.payload.pull_request.user.login);
        }
        // Add PR assignees to known authors (if not bots)
        if (context.payload.pull_request?.assignees && Array.isArray(context.payload.pull_request.assignees)) {
          for (const assignee of context.payload.pull_request.assignees) {
            if (assignee?.login && !isPayloadUserBot(assignee)) {
              knownAuthors.push(assignee.login);
            }
          }
        }
      }
      break;

    case "discussion":
      // For discussions: title + body
      if (context.payload.discussion) {
        const title = context.payload.discussion.title || "";
        const body = context.payload.discussion.body || "";
        text = `${title}\n\n${body}`;
        // Add discussion author to known authors (if not a bot)
        if (context.payload.discussion.user?.login && !isPayloadUserBot(context.payload.discussion.user)) {
          knownAuthors.push(context.payload.discussion.user.login);
        }
      }
      break;

    case "discussion_comment":
      // For discussion comments: comment body
      if (context.payload.comment) {
        text = context.payload.comment.body || "";
        // Add comment author to known authors (if not a bot)
        if (context.payload.comment.user?.login && !isPayloadUserBot(context.payload.comment.user)) {
          knownAuthors.push(context.payload.comment.user.login);
        }
        // Also add the parent discussion author to known authors (if not a bot)
        if (context.payload.discussion?.user?.login && !isPayloadUserBot(context.payload.discussion.user)) {
          knownAuthors.push(context.payload.discussion.user.login);
        }
      }
      break;

    case "release":
      // For releases: name + body
      if (context.payload.release) {
        const name = context.payload.release.name || context.payload.release.tag_name || "";
        const body = context.payload.release.body || "";
        text = `${name}\n\n${body}`;
        // Add release author to known authors (if not a bot)
        if (context.payload.release.author?.login && !isPayloadUserBot(context.payload.release.author)) {
          knownAuthors.push(context.payload.release.author.login);
        }
      }
      break;

    case "workflow_dispatch":
      // Add the actor who triggered the workflow_dispatch to known authors
      // Note: actor may be a bot in workflow_dispatch, but we'll allow it since they explicitly triggered
      knownAuthors.push(actor);

      // For workflow dispatch: check for release_url or release_id in inputs
      if (context.payload.inputs) {
        const releaseUrl = context.payload.inputs.release_url;
        const releaseId = context.payload.inputs.release_id;

        // If release_url is provided, extract owner/repo/tag
        if (releaseUrl) {
          const urlMatch = releaseUrl.match(/github\.com\/([^\/]+)\/([^\/]+)\/releases\/tag\/([^\/]+)/);
          if (urlMatch) {
            const [, urlOwner, urlRepo, tag] = urlMatch;
            try {
              const { data: release } = await github.rest.repos.getReleaseByTag({
                owner: urlOwner,
                repo: urlRepo,
                tag: tag,
              });
              const name = release.name || release.tag_name || "";
              const body = release.body || "";
              text = `${name}\n\n${body}`;
              // Add release author to known authors (if not a bot)
              if (release.author?.login && release.author.type !== "Bot") {
                knownAuthors.push(release.author.login);
              }
            } catch (error) {
              core.warning(`Failed to fetch release from URL: ${error instanceof Error ? error.message : String(error)}`);
            }
          }
        } else if (releaseId) {
          // If release_id is provided, fetch the release
          try {
            const { data: release } = await github.rest.repos.getRelease({
              owner: owner,
              repo: repo,
              release_id: parseInt(releaseId, 10),
            });
            const name = release.name || release.tag_name || "";
            const body = release.body || "";
            text = `${name}\n\n${body}`;
            // Add release author to known authors (if not a bot)
            if (release.author?.login && release.author.type !== "Bot") {
              knownAuthors.push(release.author.login);
            }
          } catch (error) {
            core.warning(`Failed to fetch release by ID: ${error instanceof Error ? error.message : String(error)}`);
          }
        }
      }
      break;

    default:
      // Default: empty text
      text = "";
      break;
  }

  // Sanitize the text before output using slimmed-down sanitizer
  // Note: Mention filtering is NOT applied here - all mentions are escaped
  // Mention filtering will be applied by the agent output collector
  const sanitizedText = sanitizeIncomingText(text);

  // Display sanitized text in logs
  core.info(`text: ${sanitizedText}`);

  // Set the sanitized text as output
  core.setOutput("text", sanitizedText);

  // Write redacted URL domains to log file if any were collected
  const logPath = writeRedactedDomainsLog();
  if (logPath) {
    core.info(`Redacted URL domains written to: ${logPath}`);
  }
}

await main();
