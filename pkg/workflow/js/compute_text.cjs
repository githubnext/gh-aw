// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Sanitizes content for safe output in GitHub Actions
 * @param {string} content - The content to sanitize
 * @returns {string} The sanitized content
 */
const { sanitizeContent, writeRedactedDomainsLog } = require("./sanitize_content.cjs");

/**
 * Check if a user is a bot
 * @param {string} username - The username to check
 * @param {any} github - GitHub API instance
 * @returns {Promise<boolean>} True if the user is a bot
 */
async function isBot(username, github) {
  try {
    const { data: user } = await github.rest.users.getByUsername({
      username: username,
    });
    return user.type === "Bot";
  } catch (error) {
    core.warning(`Failed to check if user ${username} is a bot: ${error instanceof Error ? error.message : String(error)}`);
    return false;
  }
}

/**
 * Check if a user from the payload is a bot (checks user object type field if available)
 * @param {any} user - User object from GitHub payload
 * @returns {boolean} True if the user is a bot
 */
function isPayloadUserBot(user) {
  // Check if the user object has a type field set to "Bot"
  if (user && user.type === "Bot") {
    return true;
  }
  return false;
}

/**
 * Get repository team members (maintainers only)
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {any} github - GitHub API instance
 * @returns {Promise<string[]>} Array of team member usernames (excluding bots)
 */
async function getTeamMembers(owner, repo, github) {
  try {
    const collaborators = await github.rest.repos.listCollaborators({
      owner: owner,
      repo: repo,
      affiliation: "direct",
    });

    const teamMembers = [];
    for (const collaborator of collaborators.data) {
      // Only include maintainers (maintain or admin access)
      const permission = collaborator.permissions;
      if (permission && (permission.maintain || permission.admin)) {
        // Exclude bots
        if (collaborator.type !== "Bot") {
          teamMembers.push(collaborator.login);
        }
      }
    }

    return teamMembers;
  } catch (error) {
    core.warning(`Failed to fetch team members: ${error instanceof Error ? error.message : String(error)}`);
    return [];
  }
}

async function main() {
  let text = "";
  /** @type {string[]} */
  let allowedAliases = [];

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

  // Fetch team members in the repository (collaborators with push access or higher, excluding bots)
  const teamMembers = await getTeamMembers(owner, repo, github);
  core.info(`Found ${teamMembers.length} team members in repository`);
  allowedAliases.push(...teamMembers);

  // Determine current body text based on event context
  switch (context.eventName) {
    case "issues":
      // For issues: title + body
      if (context.payload.issue) {
        const title = context.payload.issue.title || "";
        const body = context.payload.issue.body || "";
        text = `${title}\n\n${body}`;
        // Add issue author to allowed aliases (if not a bot)
        if (context.payload.issue.user?.login && !isPayloadUserBot(context.payload.issue.user)) {
          allowedAliases.push(context.payload.issue.user.login);
        }
      }
      break;

    case "pull_request":
      // For pull requests: title + body
      if (context.payload.pull_request) {
        const title = context.payload.pull_request.title || "";
        const body = context.payload.pull_request.body || "";
        text = `${title}\n\n${body}`;
        // Add PR author to allowed aliases (if not a bot)
        if (context.payload.pull_request.user?.login && !isPayloadUserBot(context.payload.pull_request.user)) {
          allowedAliases.push(context.payload.pull_request.user.login);
        }
      }
      break;

    case "pull_request_target":
      // For pull request target events: title + body
      if (context.payload.pull_request) {
        const title = context.payload.pull_request.title || "";
        const body = context.payload.pull_request.body || "";
        text = `${title}\n\n${body}`;
        // Add PR author to allowed aliases (if not a bot)
        if (context.payload.pull_request.user?.login && !isPayloadUserBot(context.payload.pull_request.user)) {
          allowedAliases.push(context.payload.pull_request.user.login);
        }
      }
      break;

    case "issue_comment":
      // For issue comments: comment body
      if (context.payload.comment) {
        text = context.payload.comment.body || "";
        // Add comment author to allowed aliases (if not a bot)
        if (context.payload.comment.user?.login && !isPayloadUserBot(context.payload.comment.user)) {
          allowedAliases.push(context.payload.comment.user.login);
        }
        // Also add the parent issue author to allowed aliases (if not a bot)
        if (context.payload.issue?.user?.login && !isPayloadUserBot(context.payload.issue.user)) {
          allowedAliases.push(context.payload.issue.user.login);
        }
      }
      break;

    case "pull_request_review_comment":
      // For PR review comments: comment body
      if (context.payload.comment) {
        text = context.payload.comment.body || "";
        // Add comment author to allowed aliases (if not a bot)
        if (context.payload.comment.user?.login && !isPayloadUserBot(context.payload.comment.user)) {
          allowedAliases.push(context.payload.comment.user.login);
        }
        // Also add the parent PR author to allowed aliases (if not a bot)
        if (context.payload.pull_request?.user?.login && !isPayloadUserBot(context.payload.pull_request.user)) {
          allowedAliases.push(context.payload.pull_request.user.login);
        }
      }
      break;

    case "pull_request_review":
      // For PR reviews: review body
      if (context.payload.review) {
        text = context.payload.review.body || "";
        // Add review author to allowed aliases (if not a bot)
        if (context.payload.review.user?.login && !isPayloadUserBot(context.payload.review.user)) {
          allowedAliases.push(context.payload.review.user.login);
        }
        // Also add the parent PR author to allowed aliases (if not a bot)
        if (context.payload.pull_request?.user?.login && !isPayloadUserBot(context.payload.pull_request.user)) {
          allowedAliases.push(context.payload.pull_request.user.login);
        }
      }
      break;

    case "discussion":
      // For discussions: title + body
      if (context.payload.discussion) {
        const title = context.payload.discussion.title || "";
        const body = context.payload.discussion.body || "";
        text = `${title}\n\n${body}`;
        // Add discussion author to allowed aliases (if not a bot)
        if (context.payload.discussion.user?.login && !isPayloadUserBot(context.payload.discussion.user)) {
          allowedAliases.push(context.payload.discussion.user.login);
        }
      }
      break;

    case "discussion_comment":
      // For discussion comments: comment body
      if (context.payload.comment) {
        text = context.payload.comment.body || "";
        // Add comment author to allowed aliases (if not a bot)
        if (context.payload.comment.user?.login && !isPayloadUserBot(context.payload.comment.user)) {
          allowedAliases.push(context.payload.comment.user.login);
        }
        // Also add the parent discussion author to allowed aliases (if not a bot)
        if (context.payload.discussion?.user?.login && !isPayloadUserBot(context.payload.discussion.user)) {
          allowedAliases.push(context.payload.discussion.user.login);
        }
      }
      break;

    case "release":
      // For releases: name + body
      if (context.payload.release) {
        const name = context.payload.release.name || context.payload.release.tag_name || "";
        const body = context.payload.release.body || "";
        text = `${name}\n\n${body}`;
        // Add release author to allowed aliases (if not a bot)
        if (context.payload.release.author?.login && !isPayloadUserBot(context.payload.release.author)) {
          allowedAliases.push(context.payload.release.author.login);
        }
      }
      break;

    case "workflow_dispatch":
      // Add the actor who triggered the workflow_dispatch to allowed aliases
      // Note: actor may be a bot in workflow_dispatch, but we'll allow it since they explicitly triggered
      allowedAliases.push(actor);

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
              // Add release author to allowed aliases (if not a bot)
              if (release.author?.login && release.author.type !== "Bot") {
                allowedAliases.push(release.author.login);
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
            // Add release author to allowed aliases (if not a bot)
            if (release.author?.login && release.author.type !== "Bot") {
              allowedAliases.push(release.author.login);
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

  // Remove duplicate allowed aliases (case-insensitive deduplication)
  const uniqueAliases = [...new Set(allowedAliases.map(a => a.toLowerCase()))];
  const finalAllowedAliases = allowedAliases.filter(
    (alias, index) => uniqueAliases.indexOf(alias.toLowerCase()) === allowedAliases.map(a => a.toLowerCase()).indexOf(alias.toLowerCase())
  );

  // Log allowed mentions for documentation
  if (finalAllowedAliases.length > 0) {
    core.info(`Allowed mentions (will not be escaped): ${finalAllowedAliases.join(", ")}`);
  } else {
    core.info("No allowed mentions configured - all mentions will be escaped");
  }

  // Sanitize the text before output, passing the allowed aliases
  const sanitizedText = sanitizeContent(text, { allowedAliases: finalAllowedAliases });

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
