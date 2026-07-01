// @ts-check
/// <reference types="@actions/github-script" />

const { getRunStartedMessage } = require("./messages_run_status.cjs");
const { getErrorMessage, isLockedError } = require("./error_helpers.cjs");
const { generateWorkflowIdMarker } = require("./generate_footer.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { ERR_API, ERR_NOT_FOUND, ERR_VALIDATION } = require("./error_codes.cjs");
const { buildWorkflowRunUrl } = require("./workflow_metadata_helpers.cjs");
const { resolveTopLevelDiscussionCommentId } = require("./github_api_helpers.cjs");
const { resolveInvocationContext } = require("./invocation_context_helpers.cjs");
const { addReaction, addDiscussionReaction } = require("./add_reaction.cjs");

/**
 * Event type descriptions for comment messages
 * @type {Record<string, string>}
 */
const EVENT_TYPE_DESCRIPTIONS = {
  issues: "issue",
  pull_request: "pull request",
  issue_comment: "issue comment",
  pull_request_review_comment: "pull request review comment",
  discussion: "discussion",
  discussion_comment: "discussion comment",
};

/** Valid GitHub reaction types */
const VALID_REACTIONS = Object.freeze(["+1", "-1", "laugh", "confused", "heart", "hooray", "rocket", "eyes"]);

/**
 * Resolve the reaction and comment API endpoints for a given event.
 * Returns null (after calling core.setFailed) when the event or payload is invalid.
 * @param {string} eventName - The GitHub event name
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {Record<string, any>} payload - The event payload
 * @returns {Promise<{reactionEndpoint: string, commentUpdateEndpoint: string} | null>}
 */
async function resolveEventEndpoints(eventName, owner, repo, payload) {
  switch (eventName) {
    case "issues": {
      const issueNumber = payload?.issue?.number;
      if (!issueNumber) {
        core.setFailed(`${ERR_NOT_FOUND}: Issue number not found in event payload`);
        return null;
      }
      return {
        reactionEndpoint: `/repos/${owner}/${repo}/issues/${issueNumber}/reactions`,
        commentUpdateEndpoint: `/repos/${owner}/${repo}/issues/${issueNumber}/comments`,
      };
    }

    case "issue_comment": {
      const commentId = payload?.comment?.id;
      const issueNumber = payload?.issue?.number;
      if (!commentId) {
        core.setFailed(`${ERR_VALIDATION}: Comment ID not found in event payload`);
        return null;
      }
      if (!issueNumber) {
        core.setFailed(`${ERR_NOT_FOUND}: Issue number not found in event payload`);
        return null;
      }
      return {
        reactionEndpoint: `/repos/${owner}/${repo}/issues/comments/${commentId}/reactions`,
        // Create new comment on the issue itself, not on the comment
        commentUpdateEndpoint: `/repos/${owner}/${repo}/issues/${issueNumber}/comments`,
      };
    }

    case "pull_request": {
      const prNumber = payload?.pull_request?.number;
      if (!prNumber) {
        core.setFailed(`${ERR_NOT_FOUND}: Pull request number not found in event payload`);
        return null;
      }
      // PRs are "issues" for the reactions endpoint
      return {
        reactionEndpoint: `/repos/${owner}/${repo}/issues/${prNumber}/reactions`,
        commentUpdateEndpoint: `/repos/${owner}/${repo}/issues/${prNumber}/comments`,
      };
    }

    case "pull_request_review_comment": {
      const reviewCommentId = payload?.comment?.id;
      const prNumber = payload?.pull_request?.number;
      if (!reviewCommentId) {
        core.setFailed(`${ERR_VALIDATION}: Review comment ID not found in event payload`);
        return null;
      }
      if (!prNumber) {
        core.setFailed(`${ERR_NOT_FOUND}: Pull request number not found in event payload`);
        return null;
      }
      return {
        reactionEndpoint: `/repos/${owner}/${repo}/pulls/comments/${reviewCommentId}/reactions`,
        // Create new comment on the PR itself (using issues endpoint since PRs are issues)
        commentUpdateEndpoint: `/repos/${owner}/${repo}/issues/${prNumber}/comments`,
      };
    }

    case "discussion": {
      const discussionNumber = payload?.discussion?.number;
      if (!discussionNumber) {
        core.setFailed(`${ERR_NOT_FOUND}: Discussion number not found in event payload`);
        return null;
      }
      // Discussions use GraphQL API - get the node ID
      const discussion = await getDiscussionId(owner, repo, discussionNumber);
      return {
        reactionEndpoint: discussion.id, // Store node ID for GraphQL
        commentUpdateEndpoint: `discussion:${discussionNumber}`, // Special format to indicate discussion
      };
    }

    case "discussion_comment": {
      const discussionNumber = payload?.discussion?.number;
      const commentId = payload?.comment?.id;
      if (!discussionNumber || !commentId) {
        core.setFailed(`${ERR_NOT_FOUND}: Discussion or comment information not found in event payload`);
        return null;
      }
      const commentNodeId = payload?.comment?.node_id;
      if (!commentNodeId) {
        core.setFailed(`${ERR_NOT_FOUND}: Discussion comment node ID not found in event payload`);
        return null;
      }
      return {
        reactionEndpoint: commentNodeId, // Store node ID for GraphQL
        commentUpdateEndpoint: `discussion_comment:${discussionNumber}:${commentId}`, // Special format
      };
    }

    default:
      core.setFailed(`${ERR_VALIDATION}: Unsupported event type: ${eventName}`);
      return null;
  }
}

async function main() {
  const reaction = process.env.GH_AW_REACTION || "eyes";
  const commandsJSON = process.env.GH_AW_COMMANDS;
  const command = commandsJSON ? (JSON.parse(commandsJSON)[0] ?? null) : null; // Only present for command workflows
  const invocationContext = resolveInvocationContext(context);
  const runUrl = buildWorkflowRunUrl(context, invocationContext.workflowRepo);

  core.info(`Reaction type: ${reaction}`);
  core.info(`Command name: ${command || "none"}`);
  core.info(`Run ID: ${context.runId}`);
  core.info(`Run URL: ${runUrl}`);

  if (!VALID_REACTIONS.includes(reaction)) {
    core.setFailed(`${ERR_VALIDATION}: Invalid reaction type: ${reaction}. Valid reactions are: ${VALID_REACTIONS.join(", ")}`);
    return;
  }

  const eventName = invocationContext.eventName;
  const { owner, repo } = invocationContext.eventRepo;
  const payload = invocationContext.eventPayload;

  try {
    const endpoints = await resolveEventEndpoints(eventName, owner, repo, payload);
    if (!endpoints) return;

    const { reactionEndpoint, commentUpdateEndpoint } = endpoints;

    core.info(`Reaction API endpoint: ${reactionEndpoint}`);

    // For discussions, reactionEndpoint is a node ID (GraphQL), otherwise it's a REST API path
    if (eventName === "discussion" || eventName === "discussion_comment") {
      await addDiscussionReaction(reactionEndpoint, reaction);
    } else {
      await addReaction(reactionEndpoint, reaction);
    }

    core.info(`Comment endpoint: ${commentUpdateEndpoint}`);
    await addCommentWithWorkflowLink(commentUpdateEndpoint, runUrl, eventName, invocationContext);
  } catch (error) {
    if (isLockedError(error)) {
      core.info(`Cannot add reaction: resource is locked (this is expected and not an error)`);
      return;
    }
    const errorMessage = getErrorMessage(error);
    core.error(`Failed to process reaction and comment creation: ${errorMessage}`);
    core.setFailed(`${ERR_API}: Failed to process reaction and comment creation: ${errorMessage}`);
  }
}

/**
 * Get the node ID for a discussion
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} discussionNumber - Discussion number
 * @returns {Promise<{id: string, url: string}>} Discussion details
 */
async function getDiscussionId(owner, repo, discussionNumber) {
  const { repository } = await github.graphql(
    `
    query($owner: String!, $repo: String!, $num: Int!) {
      repository(owner: $owner, name: $repo) {
        discussion(number: $num) { 
          id 
          url
        }
      }
    }`,
    { owner, repo, num: discussionNumber }
  );

  if (!repository || !repository.discussion) {
    throw new Error(`${ERR_NOT_FOUND}: Discussion #${discussionNumber} not found in ${owner}/${repo}`);
  }

  return {
    id: repository.discussion.id,
    url: repository.discussion.url,
  };
}

/**
 * Helper function to set comment outputs
 * @param {string} commentId - The comment ID
 * @param {string} commentUrl - The comment URL
 * @param {{ owner: string, repo: string }} [eventRepo=context.repo] - Repository where the comment was created
 */
function setCommentOutputs(commentId, commentUrl, eventRepo = context.repo) {
  core.info(`Successfully created comment with workflow link`);
  core.info(`Comment ID: ${commentId}`);
  core.info(`Comment URL: ${commentUrl}`);
  core.info(`Comment Repo: ${eventRepo.owner}/${eventRepo.repo}`);
  core.setOutput("comment-id", commentId);
  core.setOutput("comment-url", commentUrl);
  core.setOutput("comment-repo", `${eventRepo.owner}/${eventRepo.repo}`);
}

/**
 * Add a comment with a workflow run link
 * @param {string} endpoint - The GitHub API endpoint to create the comment (or special format for discussions)
 * @param {string} runUrl - The URL of the workflow run
 * @param {string} eventName - The event type (to determine the comment text)
 * @param {{
 *   source?: "native" | "workflow_dispatch" | "repository_dispatch",
 *   eventName?: string,
 *   eventPayload?: any,
 *   workflowRepo?: { owner: string, repo: string },
 *   eventRepo?: { owner: string, repo: string }
 * } | null} [invocationContext=null] - Resolved invocation event context. When omitted, falls back to global context payload/repo.
 */
async function addCommentWithWorkflowLink(endpoint, runUrl, eventName, invocationContext = null) {
  const eventPayload = invocationContext?.eventPayload || context.payload;
  const eventRepo = invocationContext?.eventRepo || context.repo;
  try {
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || process.env.GITHUB_WORKFLOW || "Workflow";
    const eventTypeDescription = EVENT_TYPE_DESCRIPTIONS[eventName] ?? "event";

    // Use getRunStartedMessage for the workflow link text (supports custom messages)
    const workflowLinkText = getRunStartedMessage({
      workflowName,
      runUrl,
      eventType: eventTypeDescription,
    });

    const lockForAgent = process.env.GH_AW_LOCK_FOR_AGENT === "true";
    const workflowId = process.env.GITHUB_WORKFLOW || "";
    const trackerId = process.env.GH_AW_TRACKER_ID || "";

    // Build comment body from parts, sanitizing first to preserve workflow markers
    const commentParts = [
      sanitizeContent(workflowLinkText),
      ...(lockForAgent && (eventName === "issues" || eventName === "issue_comment") ? ["🔒 This issue has been locked while the workflow is running to prevent concurrent modifications."] : []),
      ...(workflowId ? [generateWorkflowIdMarker(workflowId)] : []),
      ...(trackerId ? [`<!-- gh-aw-tracker-id: ${trackerId} -->`] : []),
      "<!-- gh-aw-comment-type: reaction -->",
    ];
    const commentBody = commentParts.join("\n\n");

    if (eventName === "discussion" || eventName === "discussion_comment") {
      // Parse discussion number from special format: "discussion:NUMBER" or "discussion_comment:NUMBER:COMMENT_ID"
      const discussionNumber = parseInt(endpoint.split(":")[1], 10);
      const { id: discussionId } = await getDiscussionId(eventRepo.owner, eventRepo.repo, discussionNumber);
      // For discussion_comment events, thread the reply under the triggering comment.
      // GitHub Discussions only supports two nesting levels, so resolve the top-level parent node ID.
      const replyToId = eventName === "discussion_comment"
        ? await resolveTopLevelDiscussionCommentId(github, eventPayload?.comment?.node_id)
        : null;
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
      const result = await github.graphql(mutation, { dId: discussionId, body: commentBody, ...(replyToId ? { replyToId } : {}) });
      const comment = result.addDiscussionComment.comment;
      setCommentOutputs(comment.id, comment.url, eventRepo);
      return;
    }

    // Create a new comment for non-discussion events
    const createResponse = await github.request(`POST ${endpoint}`, {
      body: commentBody,
      headers: {
        Accept: "application/vnd.github+json",
      },
    });

    setCommentOutputs(createResponse.data.id.toString(), createResponse.data.html_url, eventRepo);
  } catch (error) {
    // Don't fail the entire job if comment creation fails - just log it
    const errorMessage = getErrorMessage(error);
    core.warning(`Failed to create comment with workflow link (This is not critical - the reaction was still added successfully): ${errorMessage}`);
  }
}

module.exports = { main, addCommentWithWorkflowLink, resolveEventEndpoints, VALID_REACTIONS, addReaction, addDiscussionReaction };
