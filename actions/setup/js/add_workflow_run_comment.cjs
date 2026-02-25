// @ts-check
/// <reference types="@actions/github-script" />

const { getRunStartedMessage } = require("./messages_run_status.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { generateWorkflowIdMarker } = require("./generate_footer.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { ERR_NOT_FOUND, ERR_VALIDATION } = require("./error_codes.cjs");
const { getMessages } = require("./messages_core.cjs");
const { parseBoolTemplatable } = require("./templatable.cjs");

/**
 * Event type descriptions for comment messages
 */
const EVENT_TYPE_DESCRIPTIONS = {
  issues: "issue",
  pull_request: "pull request",
  issue_comment: "issue comment",
  pull_request_review_comment: "pull request review comment",
  discussion: "discussion",
  discussion_comment: "discussion comment",
};

/**
 * Helper function to get discussion node ID via GraphQL
 * @param {number} discussionNumber - The discussion number
 * @returns {Promise<string>} The discussion node ID
 */
async function getDiscussionNodeId(discussionNumber) {
  const { repository } = await github.graphql(
    `
    query($owner: String!, $repo: String!, $num: Int!) {
      repository(owner: $owner, name: $repo) {
        discussion(number: $num) { 
          id 
        }
      }
    }`,
    { owner: context.repo.owner, repo: context.repo.repo, num: discussionNumber }
  );
  return repository.discussion.id;
}

/**
 * Helper function to set comment outputs
 * @param {string|number} commentId - The comment ID
 * @param {string} commentUrl - The comment URL
 */
function setCommentOutputs(commentId, commentUrl) {
  core.info(`Successfully created comment with workflow link`);
  core.info(`Comment ID: ${commentId}`);
  core.info(`Comment URL: ${commentUrl}`);
  core.info(`Comment Repo: ${context.repo.owner}/${context.repo.repo}`);
  core.setOutput("comment-id", commentId.toString());
  core.setOutput("comment-url", commentUrl);
  core.setOutput("comment-repo", `${context.repo.owner}/${context.repo.repo}`);
}

/**
 * Add a comment with a workflow run link to the triggering item.
 * This script ONLY creates comments - it does NOT add reactions.
 * Use add_reaction.cjs in the pre-activation job to add reactions first for immediate feedback.
 */
async function main() {
  // Check if activation comments are disabled
  const messagesConfig = getMessages();
  if (!parseBoolTemplatable(messagesConfig?.activationComments, true)) {
    core.info("activation-comments is disabled: skipping activation comment creation");
    return;
  }

  const runId = context.runId;
  const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
  const runUrl = context.payload.repository ? `${context.payload.repository.html_url}/actions/runs/${runId}` : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

  core.info(`Run ID: ${runId}`);
  core.info(`Run URL: ${runUrl}`);

  // Determine the API endpoint based on the event type
  let commentEndpoint;
  const eventName = context.eventName;
  const owner = context.repo.owner;
  const repo = context.repo.repo;

  try {
    switch (eventName) {
      case "issues": {
        const issueNumber = context.payload?.issue?.number;
        if (!issueNumber) {
          core.setFailed(`${ERR_NOT_FOUND}: Issue number not found in event payload`);
          return;
        }
        commentEndpoint = `/repos/${owner}/${repo}/issues/${issueNumber}/comments`;
        break;
      }

      case "issue_comment": {
        const issueNumberForComment = context.payload?.issue?.number;
        if (!issueNumberForComment) {
          core.setFailed(`${ERR_NOT_FOUND}: Issue number not found in event payload`);
          return;
        }
        // Create new comment on the issue itself, not on the comment
        commentEndpoint = `/repos/${owner}/${repo}/issues/${issueNumberForComment}/comments`;
        break;
      }

      case "pull_request": {
        const prNumber = context.payload?.pull_request?.number;
        if (!prNumber) {
          core.setFailed(`${ERR_NOT_FOUND}: Pull request number not found in event payload`);
          return;
        }
        commentEndpoint = `/repos/${owner}/${repo}/issues/${prNumber}/comments`;
        break;
      }

      case "pull_request_review_comment": {
        const prNumberForReviewComment = context.payload?.pull_request?.number;
        if (!prNumberForReviewComment) {
          core.setFailed(`${ERR_NOT_FOUND}: Pull request number not found in event payload`);
          return;
        }
        // Create new comment on the PR itself (using issues endpoint since PRs are issues)
        commentEndpoint = `/repos/${owner}/${repo}/issues/${prNumberForReviewComment}/comments`;
        break;
      }

      case "discussion": {
        const discussionNumber = context.payload?.discussion?.number;
        if (!discussionNumber) {
          core.setFailed(`${ERR_NOT_FOUND}: Discussion number not found in event payload`);
          return;
        }
        commentEndpoint = `discussion:${discussionNumber}`; // Special format to indicate discussion
        break;
      }

      case "discussion_comment": {
        const discussionCommentNumber = context.payload?.discussion?.number;
        const discussionCommentId = context.payload?.comment?.id;
        if (!discussionCommentNumber || !discussionCommentId) {
          core.setFailed(`${ERR_NOT_FOUND}: Discussion or comment information not found in event payload`);
          return;
        }
        commentEndpoint = `discussion_comment:${discussionCommentNumber}:${discussionCommentId}`; // Special format
        break;
      }

      default:
        core.setFailed(`${ERR_VALIDATION}: Unsupported event type: ${eventName}`);
        return;
    }

    core.info(`Creating comment on: ${commentEndpoint}`);
    await addCommentWithWorkflowLink(commentEndpoint, runUrl, eventName);
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    core.error(`Failed to create comment: ${errorMessage}`);
    // Don't fail the job - just warn since this is not critical
    core.warning(`Failed to create comment with workflow link: ${errorMessage}`);
  }
}

/**
 * Add a comment with a workflow run link
 * @param {string} endpoint - The GitHub API endpoint to create the comment (or special format for discussions)
 * @param {string} runUrl - The URL of the workflow run
 * @param {string} eventName - The event type (to determine the comment text)
 */
async function addCommentWithWorkflowLink(endpoint, runUrl, eventName) {
  // Get workflow name from environment variable
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";

  // Determine the event type description using lookup object
  const eventTypeDescription = EVENT_TYPE_DESCRIPTIONS[eventName] ?? "event";

  // Use getRunStartedMessage for the workflow link text (supports custom messages)
  const workflowLinkText = getRunStartedMessage({
    workflowName: workflowName,
    runUrl: runUrl,
    eventType: eventTypeDescription,
  });

  // Sanitize the workflow link text to prevent injection attacks (defense in depth for custom message templates)
  // This must happen BEFORE adding workflow markers to preserve them
  let commentBody = sanitizeContent(workflowLinkText);

  // Add lock notice if lock-for-agent is enabled for issues or issue_comment
  const lockForAgent = process.env.GH_AW_LOCK_FOR_AGENT === "true";
  if (lockForAgent && (eventName === "issues" || eventName === "issue_comment")) {
    commentBody += "\n\n🔒 This issue has been locked while the workflow is running to prevent concurrent modifications.";
  }

  // Add workflow-id and tracker-id markers for hide-older-comments feature
  const workflowId = process.env.GITHUB_WORKFLOW || "";
  const trackerId = process.env.GH_AW_TRACKER_ID || "";

  // Add workflow-id marker if available
  if (workflowId) {
    commentBody += `\n\n${generateWorkflowIdMarker(workflowId)}`;
  }

  // Add tracker-id marker if available (for backwards compatibility)
  if (trackerId) {
    commentBody += `\n\n<!-- gh-aw-tracker-id: ${trackerId} -->`;
  }

  // Add comment type marker to identify this as a reaction comment
  // This prevents it from being hidden by hide-older-comments
  commentBody += `\n\n<!-- gh-aw-comment-type: reaction -->`;

  // Handle discussion events specially
  if (eventName === "discussion") {
    // Parse discussion number from special format: "discussion:NUMBER"
    const discussionNumber = parseInt(endpoint.split(":")[1], 10);

    // Get discussion node ID using helper function
    const discussionId = await getDiscussionNodeId(discussionNumber);

    const result = await github.graphql(
      `
      mutation($dId: ID!, $body: String!) {
        addDiscussionComment(input: { discussionId: $dId, body: $body }) {
          comment { 
            id 
            url
          }
        }
      }`,
      { dId: discussionId, body: commentBody }
    );

    const comment = result.addDiscussionComment.comment;
    setCommentOutputs(comment.id, comment.url);
    return;
  } else if (eventName === "discussion_comment") {
    // Parse discussion number from special format: "discussion_comment:NUMBER:COMMENT_ID"
    const discussionNumber = parseInt(endpoint.split(":")[1], 10);

    // Get discussion node ID using helper function
    const discussionId = await getDiscussionNodeId(discussionNumber);

    // Get the comment node ID to use as the parent for threading
    const commentNodeId = context.payload?.comment?.node_id;

    const result = await github.graphql(
      `
      mutation($dId: ID!, $body: String!, $replyToId: ID!) {
        addDiscussionComment(input: { discussionId: $dId, body: $body, replyToId: $replyToId }) {
          comment { 
            id 
            url
          }
        }
      }`,
      { dId: discussionId, body: commentBody, replyToId: commentNodeId }
    );

    const comment = result.addDiscussionComment.comment;
    setCommentOutputs(comment.id, comment.url);
    return;
  }

  // Create a new comment for non-discussion events
  const createResponse = await github.request("POST " + endpoint, {
    body: commentBody,
    headers: {
      Accept: "application/vnd.github+json",
    },
  });

  setCommentOutputs(createResponse.data.id, createResponse.data.html_url);
}

module.exports = { main, addCommentWithWorkflowLink };
