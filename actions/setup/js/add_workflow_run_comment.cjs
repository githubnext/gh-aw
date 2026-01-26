// @ts-check
/// <reference types="@actions/github-script" />

const { getRunStartedMessage } = require("./messages_run_status.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { generateWorkflowIdMarker } = require("./generate_footer.cjs");

/**
 * Add a comment with a workflow run link to the triggering item.
 * This script ONLY creates comments - it does NOT add reactions.
 * Use add_reaction.cjs in the pre-activation job to add reactions first for immediate feedback.
 */
async function main() {
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
      case "issues":
        const issueNumber = context.payload?.issue?.number;
        if (!issueNumber) {
          core.setFailed("Issue number not found in event payload");
          return;
        }
        commentEndpoint = `/repos/${owner}/${repo}/issues/${issueNumber}/comments`;
        break;

      case "issue_comment":
        const issueNumberForComment = context.payload?.issue?.number;
        if (!issueNumberForComment) {
          core.setFailed("Issue number not found in event payload");
          return;
        }
        // Create new comment on the issue itself, not on the comment
        commentEndpoint = `/repos/${owner}/${repo}/issues/${issueNumberForComment}/comments`;
        break;

      case "pull_request":
        const prNumber = context.payload?.pull_request?.number;
        if (!prNumber) {
          core.setFailed("Pull request number not found in event payload");
          return;
        }
        commentEndpoint = `/repos/${owner}/${repo}/issues/${prNumber}/comments`;
        break;

      case "pull_request_review_comment":
        const prNumberForReviewComment = context.payload?.pull_request?.number;
        if (!prNumberForReviewComment) {
          core.setFailed("Pull request number not found in event payload");
          return;
        }
        // Create new comment on the PR itself (using issues endpoint since PRs are issues)
        commentEndpoint = `/repos/${owner}/${repo}/issues/${prNumberForReviewComment}/comments`;
        break;

      case "discussion":
        const discussionNumber = context.payload?.discussion?.number;
        if (!discussionNumber) {
          core.setFailed("Discussion number not found in event payload");
          return;
        }
        commentEndpoint = `discussion:${discussionNumber}`; // Special format to indicate discussion
        break;

      case "discussion_comment":
        const discussionCommentNumber = context.payload?.discussion?.number;
        const discussionCommentId = context.payload?.comment?.id;
        if (!discussionCommentNumber || !discussionCommentId) {
          core.setFailed("Discussion or comment information not found in event payload");
          return;
        }
        commentEndpoint = `discussion_comment:${discussionCommentNumber}:${discussionCommentId}`; // Special format
        break;

      default:
        core.setFailed(`Unsupported event type: ${eventName}`);
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

  // Determine the event type description
  let eventTypeDescription;
  switch (eventName) {
    case "issues":
      eventTypeDescription = "issue";
      break;
    case "pull_request":
      eventTypeDescription = "pull request";
      break;
    case "issue_comment":
      eventTypeDescription = "issue comment";
      break;
    case "pull_request_review_comment":
      eventTypeDescription = "pull request review comment";
      break;
    case "discussion":
      eventTypeDescription = "discussion";
      break;
    case "discussion_comment":
      eventTypeDescription = "discussion comment";
      break;
    default:
      eventTypeDescription = "event";
  }

  // Use getRunStartedMessage for the workflow link text (supports custom messages)
  const workflowLinkText = getRunStartedMessage({
    workflowName: workflowName,
    runUrl: runUrl,
    eventType: eventTypeDescription,
  });

  // Add workflow-id and tracker-id markers for hide-older-comments feature
  const workflowId = process.env.GITHUB_WORKFLOW || "";
  const trackerId = process.env.GH_AW_TRACKER_ID || "";

  let commentBody = workflowLinkText;

  // Add lock notice if lock-for-agent is enabled for issues or issue_comment
  const lockForAgent = process.env.GH_AW_LOCK_FOR_AGENT === "true";
  if (lockForAgent && (eventName === "issues" || eventName === "issue_comment")) {
    commentBody += "\n\n🔒 This issue has been locked while the workflow is running to prevent concurrent modifications.";
  }

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

    // Create a new comment on the discussion using GraphQL
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

    const discussionId = repository.discussion.id;

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
    core.info(`Successfully created discussion comment with workflow link`);
    core.info(`Comment ID: ${comment.id}`);
    core.info(`Comment URL: ${comment.url}`);
    core.info(`Comment Repo: ${context.repo.owner}/${context.repo.repo}`);
    core.setOutput("comment-id", comment.id);
    core.setOutput("comment-url", comment.url);
    core.setOutput("comment-repo", `${context.repo.owner}/${context.repo.repo}`);
    return;
  } else if (eventName === "discussion_comment") {
    // Parse discussion number from special format: "discussion_comment:NUMBER:COMMENT_ID"
    const discussionNumber = parseInt(endpoint.split(":")[1], 10);

    // Create a new comment on the discussion using GraphQL
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

    const discussionId = repository.discussion.id;

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
    core.info(`Successfully created discussion comment with workflow link`);
    core.info(`Comment ID: ${comment.id}`);
    core.info(`Comment URL: ${comment.url}`);
    core.info(`Comment Repo: ${context.repo.owner}/${context.repo.repo}`);
    core.setOutput("comment-id", comment.id);
    core.setOutput("comment-url", comment.url);
    core.setOutput("comment-repo", `${context.repo.owner}/${context.repo.repo}`);
    return;
  }

  // Create a new comment for non-discussion events
  const createResponse = await github.request("POST " + endpoint, {
    body: commentBody,
    headers: {
      Accept: "application/vnd.github+json",
    },
  });

  core.info(`Successfully created comment with workflow link`);
  core.info(`Comment ID: ${createResponse.data.id}`);
  core.info(`Comment URL: ${createResponse.data.html_url}`);
  core.info(`Comment Repo: ${context.repo.owner}/${context.repo.repo}`);
  core.setOutput("comment-id", createResponse.data.id.toString());
  core.setOutput("comment-url", createResponse.data.html_url);
  core.setOutput("comment-repo", `${context.repo.owner}/${context.repo.repo}`);
}

module.exports = { main };
