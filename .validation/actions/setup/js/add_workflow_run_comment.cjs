// @ts-check
/// <reference types="@actions/github-script" />

const { getRunStartedMessage } = require("./messages_run_status.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { generateWorkflowIdMarker } = require("./generate_footer.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { ERR_NOT_FOUND, ERR_VALIDATION } = require("./error_codes.cjs");
const { getMessages } = require("./messages_core.cjs");
const { parseBoolTemplatable } = require("./templatable.cjs");
const { buildWorkflowRunUrl } = require("./workflow_metadata_helpers.cjs");
const { resolveTopLevelDiscussionCommentId } = require("./github_api_helpers.cjs");
const { resolveInvocationContext } = require("./invocation_context_helpers.cjs");

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
 * @param {{ owner: string, repo: string }} [eventRepo] - Repository where the discussion event occurred (defaults to context.repo at runtime)
 * @returns {Promise<string>} The discussion node ID
 */
async function getDiscussionNodeId(discussionNumber, eventRepo = context.repo) {
  const { repository } = await github.graphql(
    `
    query($owner: String!, $repo: String!, $num: Int!) {
      repository(owner: $owner, name: $repo) {
        discussion(number: $num) { 
          id 
        }
      }
    }`,
    { owner: eventRepo.owner, repo: eventRepo.repo, num: discussionNumber }
  );
  return repository.discussion.id;
}

/**
 * Helper function to set comment outputs
 * @param {string|number} commentId - The comment ID
 * @param {string} commentUrl - The comment URL
 * @param {{ owner: string, repo: string }} [eventRepo] - Repository where the comment was created (defaults to context.repo at runtime)
 */
function setCommentOutputs(commentId, commentUrl, eventRepo = context.repo) {
  core.info(`Successfully created comment with workflow link`);
  core.info(`Comment ID: ${commentId}`);
  core.info(`Comment URL: ${commentUrl}`);
  core.info(`Comment Repo: ${eventRepo.owner}/${eventRepo.repo}`);
  core.setOutput("comment-id", commentId.toString());
  core.setOutput("comment-url", commentUrl);
  core.setOutput("comment-repo", `${eventRepo.owner}/${eventRepo.repo}`);
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

  const invocationContext = resolveInvocationContext(context);
  const runUrl = buildWorkflowRunUrl(context, invocationContext.workflowRepo);

  core.info(`Run ID: ${context.runId}`);
  core.info(`Run URL: ${runUrl}`);
  core.info(`Event source: ${invocationContext.source}`);

  // Determine the API endpoint based on the event type
  let commentEndpoint;
  const eventName = invocationContext.eventName;
  const owner = invocationContext.eventRepo.owner;
  const repo = invocationContext.eventRepo.repo;
  const payload = invocationContext.eventPayload;

  try {
    switch (eventName) {
      case "issues":
      case "issue_comment": {
        const number = payload?.issue?.number;
        if (!number) {
          core.setFailed(`${ERR_NOT_FOUND}: Issue number not found in event payload`);
          return;
        }
        commentEndpoint = `/repos/${owner}/${repo}/issues/${number}/comments`;
        break;
      }

      case "pull_request":
      case "pull_request_review_comment": {
        const number = payload?.pull_request?.number;
        if (!number) {
          core.setFailed(`${ERR_NOT_FOUND}: Pull request number not found in event payload`);
          return;
        }
        // PRs use the issues comment endpoint
        commentEndpoint = `/repos/${owner}/${repo}/issues/${number}/comments`;
        break;
      }

      case "discussion": {
        const discussionNumber = payload?.discussion?.number;
        if (!discussionNumber) {
          core.setFailed(`${ERR_NOT_FOUND}: Discussion number not found in event payload`);
          return;
        }
        commentEndpoint = `discussion:${discussionNumber}`; // Special format to indicate discussion
        break;
      }

      case "discussion_comment": {
        const discussionCommentNumber = payload?.discussion?.number;
        const discussionCommentId = payload?.comment?.id;
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
    await addCommentWithWorkflowLink(commentEndpoint, runUrl, eventName, invocationContext);
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    // Don't fail the job - just warn since this is not critical
    core.warning(`Failed to create comment with workflow link: ${errorMessage}`);
  }
}

/**
 * Build the comment body text for a workflow run link.
 * Sanitizes the content and appends all required markers.
 * @param {string} eventName - The event type
 * @param {string} runUrl - The URL of the workflow run
 * @returns {string} The assembled comment body
 */
function buildCommentBody(eventName, runUrl) {
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
  const eventTypeDescription = EVENT_TYPE_DESCRIPTIONS[eventName] ?? "event";

  // Sanitize before adding markers (defense in depth for custom message templates)
  let body = sanitizeContent(getRunStartedMessage({ workflowName, runUrl, eventType: eventTypeDescription }));

  // Add lock notice if lock-for-agent is enabled for issues or issue_comment
  if (process.env.GH_AW_LOCK_FOR_AGENT === "true" && (eventName === "issues" || eventName === "issue_comment")) {
    body += "\n\n🔒 This issue has been locked while the workflow is running to prevent concurrent modifications.";
  }

  // Add workflow-id marker for hide-older-comments feature
  const workflowId = process.env.GITHUB_WORKFLOW || "";
  if (workflowId) {
    body += `\n\n${generateWorkflowIdMarker(workflowId)}`;
  }

  // Add tracker-id marker for backwards compatibility
  const trackerId = process.env.GH_AW_TRACKER_ID || "";
  if (trackerId) {
    body += `\n\n<!-- gh-aw-tracker-id: ${trackerId} -->`;
  }

  // Identify this as a reaction comment (prevents it from being hidden by hide-older-comments)
  body += `\n\n<!-- gh-aw-comment-type: reaction -->`;

  return body;
}

/**
 * Post a GraphQL comment to a discussion, optionally as a threaded reply.
 * @param {number} discussionNumber - The discussion number
 * @param {string} commentBody - The comment body
 * @param {string|null} replyToNodeId - Parent comment node ID for threading (null for top-level)
 * @param {{ owner: string, repo: string }} [eventRepo] - Repository where the discussion exists (defaults to context.repo at runtime)
 */
async function postDiscussionComment(discussionNumber, commentBody, replyToNodeId = null, eventRepo = context.repo) {
  const discussionId = await getDiscussionNodeId(discussionNumber, eventRepo);

  /** @type {any} */
  let result;
  if (replyToNodeId) {
    result = await github.graphql(
      `
      mutation($dId: ID!, $body: String!, $replyToId: ID!) {
        addDiscussionComment(input: { discussionId: $dId, body: $body, replyToId: $replyToId }) {
          comment { id url }
        }
      }`,
      { dId: discussionId, body: commentBody, replyToId: replyToNodeId }
    );
  } else {
    result = await github.graphql(
      `
      mutation($dId: ID!, $body: String!) {
        addDiscussionComment(input: { discussionId: $dId, body: $body }) {
          comment { id url }
        }
      }`,
      { dId: discussionId, body: commentBody }
    );
  }

  const comment = result.addDiscussionComment.comment;
  setCommentOutputs(comment.id, comment.url, eventRepo);
}

/**
 * Add a comment with a workflow run link
 * @param {string} endpoint - The GitHub API endpoint to create the comment (or special format for discussions)
 * @param {string} runUrl - The URL of the workflow run
 * @param {string} eventName - The event type (to determine the comment text)
 * @param {{
 *   source: "native" | "workflow_dispatch" | "repository_dispatch";
 *   eventName: string;
 *   eventPayload: any;
 *   workflowRepo: { owner: string, repo: string };
 *   eventRepo: { owner: string, repo: string };
 * }|null} [invocationContext=null] - Invocation context overrides for event payload and repo
 */
async function addCommentWithWorkflowLink(endpoint, runUrl, eventName, invocationContext = null) {
  const eventPayload = invocationContext?.eventPayload || context.payload;
  const eventRepo = invocationContext?.eventRepo || context.repo;
  const commentBody = buildCommentBody(eventName, runUrl);

  if (eventName === "discussion") {
    // Parse discussion number from special format: "discussion:NUMBER"
    const discussionNumber = parseInt(endpoint.split(":")[1], 10);
    await postDiscussionComment(discussionNumber, commentBody, null, eventRepo);
    return;
  }

  if (eventName === "discussion_comment") {
    // Parse discussion number from special format: "discussion_comment:NUMBER:COMMENT_ID"
    const discussionNumber = parseInt(endpoint.split(":")[1], 10);

    // GitHub Discussions only supports two nesting levels, so resolve the top-level parent's node ID
    const commentNodeId = await resolveTopLevelDiscussionCommentId(github, eventPayload?.comment?.node_id);
    await postDiscussionComment(discussionNumber, commentBody, commentNodeId, eventRepo);
    return;
  }

  // Create a new comment for non-discussion events
  const createResponse = await github.request("POST " + endpoint, {
    body: commentBody,
    headers: { Accept: "application/vnd.github+json" },
  });

  setCommentOutputs(createResponse.data.id, createResponse.data.html_url, eventRepo);
}

module.exports = { main, addCommentWithWorkflowLink, buildCommentBody, postDiscussionComment };
