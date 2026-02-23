// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { getMessages } = require("./messages_core.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { getPullRequestCreatedMessage, getIssueCreatedMessage, getCommitPushedMessage } = require("./messages_run_status.cjs");
const { parseBoolTemplatable } = require("./templatable.cjs");
const { generateXMLMarker } = require("./generate_footer.cjs");

/**
 * Build the workflow run URL from context and environment.
 * @param {any} context - GitHub Actions context
 * @returns {string} The workflow run URL
 */
function getRunUrl(context) {
  const runId = context.runId || process.env.GITHUB_RUN_ID || "";
  const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
  return context.payload?.repository?.html_url ? `${context.payload.repository.html_url}/actions/runs/${runId}` : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;
}

/**
 * Update the activation comment with a link to the created pull request or issue
 * @param {any} github - GitHub REST API instance
 * @param {any} context - GitHub Actions context
 * @param {any} core - GitHub Actions core
 * @param {string} itemUrl - URL of the created item (pull request or issue)
 * @param {number} itemNumber - Number of the item (pull request or issue)
 * @param {string} itemType - Type of item: "pull_request" or "issue" (defaults to "pull_request")
 */
async function updateActivationComment(github, context, core, itemUrl, itemNumber, itemType = "pull_request") {
  const itemLabel = itemType === "issue" ? "issue" : "pull request";
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
  const runUrl = getRunUrl(context);
  const body = itemType === "issue" ? getIssueCreatedMessage({ itemNumber, itemUrl }) : getPullRequestCreatedMessage({ itemNumber, itemUrl });
  const linkMessage = `\n\n${body}\n\n${generateXMLMarker(workflowName, runUrl)}`;
  await updateActivationCommentWithMessage(github, context, core, linkMessage, itemLabel, { targetIssueNumber: itemNumber });
}

/**
 * Update the activation comment with a commit link
 * @param {any} github - GitHub REST API instance
 * @param {any} context - GitHub Actions context
 * @param {any} core - GitHub Actions core
 * @param {string} commitSha - SHA of the commit
 * @param {string} commitUrl - URL of the commit
 * @param {object} [options] - Optional configuration
 * @param {number} [options.targetIssueNumber] - PR/issue number to post a new comment on if no activation comment exists
 */
async function updateActivationCommentWithCommit(github, context, core, commitSha, commitUrl, options = {}) {
  const shortSha = commitSha.substring(0, 7);
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
  const runUrl = getRunUrl(context);
  const message = `\n\n${getCommitPushedMessage({ commitSha, shortSha, commitUrl })}\n\n${generateXMLMarker(workflowName, runUrl)}`;
  await updateActivationCommentWithMessage(github, context, core, message, "commit", options);
}

/**
 * Update the activation comment with a custom message
 * @param {any} github - GitHub REST API instance
 * @param {any} context - GitHub Actions context
 * @param {any} core - GitHub Actions core
 * @param {string} message - Message to append to the comment
 * @param {string} label - Optional label for log messages (e.g., "pull request", "issue", "commit")
 * @param {object} [options] - Optional configuration
 * @param {number} [options.targetIssueNumber] - PR/issue number to post a new comment on if no activation comment exists
 */
async function updateActivationCommentWithMessage(github, context, core, message, label = "", options = {}) {
  const commentId = process.env.GH_AW_COMMENT_ID;
  const commentRepo = process.env.GH_AW_COMMENT_REPO;

  // Check if append-only-comments is enabled
  const messagesConfig = getMessages();
  const appendOnlyComments = messagesConfig?.appendOnlyComments === true;

  // Check if activation comments are disabled entirely
  if (!parseBoolTemplatable(messagesConfig?.activationComments, true)) {
    core.info("activation-comments is disabled: skipping activation comment update");
    return;
  }

  // Parse comment repo (format: "owner/repo") with validation
  let repoOwner = context.repo.owner;
  let repoName = context.repo.repo;
  if (commentRepo) {
    const parts = commentRepo.split("/");
    if (parts.length === 2) {
      repoOwner = parts[0];
      repoName = parts[1];
    } else {
      core.warning(`Invalid comment repo format: ${commentRepo}, expected "owner/repo". Falling back to context.repo.`);
    }
  }

  // Append-only mode: create a new comment instead of updating the activation comment
  if (appendOnlyComments) {
    core.info("Append-only-comments enabled: creating a new comment");
    try {
      const eventName = context.eventName;

      // Discussions: create a new discussion comment (threaded reply for discussion_comment)
      if (eventName === "discussion" || eventName === "discussion_comment") {
        const discussionNumber = context.payload?.discussion?.number;
        if (!discussionNumber) {
          core.warning("Unable to determine discussion number for append-only comment; skipping");
          return;
        }

        const { repository } = await github.graphql(
          `
          query($owner: String!, $repo: String!, $num: Int!) {
            repository(owner: $owner, name: $repo) {
              discussion(number: $num) {
                id
              }
            }
          }`,
          { owner: repoOwner, repo: repoName, num: discussionNumber }
        );

        const discussionId = repository?.discussion?.id;
        if (!discussionId) {
          core.warning("Unable to resolve discussion id for append-only comment; skipping");
          return;
        }

        const replyToId = eventName === "discussion_comment" ? context.payload?.comment?.node_id : null;
        const mutation = replyToId
          ? `mutation($dId: ID!, $body: String!, $replyToId: ID!) {
              addDiscussionComment(input: { discussionId: $dId, body: $body, replyToId: $replyToId }) {
                comment { id url }
              }
            }`
          : `mutation($dId: ID!, $body: String!) {
              addDiscussionComment(input: { discussionId: $dId, body: $body }) {
                comment { id url }
              }
            }`;

        const sanitizedMessage = sanitizeContent(message);
        const variables = replyToId ? { dId: discussionId, body: sanitizedMessage, replyToId } : { dId: discussionId, body: sanitizedMessage };
        const result = await github.graphql(mutation, variables);
        const created = result?.addDiscussionComment?.comment;
        const successMessage = label ? `Successfully created append-only discussion comment with ${label} link` : "Successfully created append-only discussion comment";
        core.info(successMessage);
        if (created?.id) core.info(`Comment ID: ${created.id}`);
        if (created?.url) core.info(`Comment URL: ${created.url}`);
        return;
      }

      // Issues/PRs: determine issue number from event payload and create a new issue comment
      const issueNumber = context.payload?.issue?.number || context.payload?.pull_request?.number;
      if (!issueNumber) {
        core.warning("Unable to determine issue/PR number for append-only comment; skipping");
        return;
      }

      const sanitizedMessage = sanitizeContent(message);
      const response = await github.request("POST /repos/{owner}/{repo}/issues/{issue_number}/comments", {
        owner: repoOwner,
        repo: repoName,
        issue_number: issueNumber,
        body: sanitizedMessage,
        headers: {
          Accept: "application/vnd.github+json",
        },
      });

      const successMessage = label ? `Successfully created append-only comment with ${label} link` : "Successfully created append-only comment";
      core.info(successMessage);
      if (response?.data?.id) core.info(`Comment ID: ${response.data.id}`);
      if (response?.data?.html_url) core.info(`Comment URL: ${response.data.html_url}`);
      return;
    } catch (error) {
      // Don't fail the workflow if we can't create the comment - just log a warning
      core.warning(`Failed to create append-only comment: ${getErrorMessage(error)}`);
      return;
    }
  }

  // Standard mode: update the existing activation comment
  // If no comment was created in activation, try to create a new comment on the target PR/issue
  if (!commentId) {
    // Determine target issue/PR number: explicit option, or from event payload
    const targetNumber = options.targetIssueNumber || context.payload?.issue?.number || context.payload?.pull_request?.number;
    if (targetNumber) {
      core.info(`No activation comment to update (GH_AW_COMMENT_ID not set), creating new comment on #${targetNumber}`);
      try {
        const sanitizedMessage = sanitizeContent(message);
        const response = await github.request("POST /repos/{owner}/{repo}/issues/{issue_number}/comments", {
          owner: repoOwner,
          repo: repoName,
          issue_number: targetNumber,
          body: sanitizedMessage,
          headers: {
            Accept: "application/vnd.github+json",
          },
        });

        const successMessage = label ? `Successfully created comment with ${label} link on #${targetNumber}` : `Successfully created comment on #${targetNumber}`;
        core.info(successMessage);
        if (response?.data?.id) core.info(`Comment ID: ${response.data.id}`);
        if (response?.data?.html_url) core.info(`Comment URL: ${response.data.html_url}`);
      } catch (error) {
        core.warning(`Failed to create comment on #${targetNumber}: ${getErrorMessage(error)}`);
      }
    } else {
      core.info("No activation comment to update (GH_AW_COMMENT_ID not set)");
    }
    return;
  }

  core.info(`Updating activation comment ${commentId}`);
  core.info(`Updating comment in ${repoOwner}/${repoName}`);

  // Check if this is a discussion comment (GraphQL node ID format)
  const isDiscussionComment = commentId.startsWith("DC_");

  try {
    if (isDiscussionComment) {
      // Get current comment body using GraphQL
      const currentComment = await github.graphql(
        `
        query($commentId: ID!) {
          node(id: $commentId) {
            ... on DiscussionComment {
              body
            }
          }
        }`,
        { commentId: commentId }
      );

      if (!currentComment?.node?.body) {
        core.warning("Unable to fetch current comment body, comment may have been deleted or is inaccessible");
        return;
      }
      const currentBody = currentComment.node.body;
      const updatedBody = currentBody + message;

      // Update discussion comment using GraphQL
      const result = await github.graphql(
        `
        mutation($commentId: ID!, $body: String!) {
          updateDiscussionComment(input: { commentId: $commentId, body: $body }) {
            comment {
              id
              url
            }
          }
        }`,
        { commentId: commentId, body: updatedBody }
      );

      const comment = result.updateDiscussionComment.comment;
      const successMessage = label ? `Successfully updated discussion comment with ${label} link` : "Successfully updated discussion comment";
      core.info(successMessage);
      core.info(`Comment ID: ${comment.id}`);
      core.info(`Comment URL: ${comment.url}`);
    } else {
      // Get current comment body using REST API
      const currentComment = await github.request("GET /repos/{owner}/{repo}/issues/comments/{comment_id}", {
        owner: repoOwner,
        repo: repoName,
        comment_id: parseInt(commentId, 10),
        headers: {
          Accept: "application/vnd.github+json",
        },
      });

      if (!currentComment?.data?.body) {
        core.warning("Unable to fetch current comment body, comment may have been deleted");
        return;
      }
      const currentBody = currentComment.data.body;
      const updatedBody = sanitizeContent(currentBody + message);

      // Update issue/PR comment using REST API
      const response = await github.request("PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}", {
        owner: repoOwner,
        repo: repoName,
        comment_id: parseInt(commentId, 10),
        body: updatedBody,
        headers: {
          Accept: "application/vnd.github+json",
        },
      });

      const successMessage = label ? `Successfully updated comment with ${label} link` : "Successfully updated comment";
      core.info(successMessage);
      core.info(`Comment ID: ${response.data.id}`);
      core.info(`Comment URL: ${response.data.html_url}`);
    }
  } catch (error) {
    // Don't fail the workflow if we can't update the comment - just log a warning
    core.warning(`Failed to update activation comment: ${getErrorMessage(error)}`);
  }
}

module.exports = {
  updateActivationComment,
  updateActivationCommentWithCommit,
};
