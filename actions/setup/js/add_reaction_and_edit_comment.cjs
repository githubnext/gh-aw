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
 * Maps reaction emoji names to GitHub's GraphQL ReactionContent enum values
 */
const DISCUSSION_REACTION_MAP = {
  "+1": "THUMBS_UP",
  "-1": "THUMBS_DOWN",
  laugh: "LAUGH",
  confused: "CONFUSED",
  heart: "HEART",
  hooray: "HOORAY",
  rocket: "ROCKET",
  eyes: "EYES",
};

async function main() {
  const reaction = process.env.GH_AW_REACTION || "eyes";
  const command = process.env.GH_AW_COMMAND; // Only present for command workflows
  const invocationContext = resolveInvocationContext(context);
  const runUrl = buildWorkflowRunUrl(context, invocationContext.workflowRepo);

  core.info(`Reaction type: ${reaction}`);
  core.info(`Command name: ${command || "none"}`);
  core.info(`Run ID: ${context.runId}`);
  core.info(`Run URL: ${runUrl}`);

  // Validate reaction type
  const validReactions = ["+1", "-1", "laugh", "confused", "heart", "hooray", "rocket", "eyes"];
  if (!validReactions.includes(reaction)) {
    core.setFailed(`${ERR_VALIDATION}: Invalid reaction type: ${reaction}. Valid reactions are: ${validReactions.join(", ")}`);
    return;
  }

  let reactionEndpoint;
  let commentUpdateEndpoint;
  const eventName = invocationContext.eventName;
  const owner = invocationContext.eventRepo.owner;
  const repo = invocationContext.eventRepo.repo;
  const payload = invocationContext.eventPayload;

  try {
    switch (eventName) {
      case "issues": {
        const issueNumber = payload?.issue?.number;
        if (!issueNumber) {
          core.setFailed(`${ERR_NOT_FOUND}: Issue number not found in event payload`);
          return;
        }
        reactionEndpoint = `/repos/${owner}/${repo}/issues/${issueNumber}/reactions`;
        commentUpdateEndpoint = `/repos/${owner}/${repo}/issues/${issueNumber}/comments`;
        break;
      }

      case "issue_comment": {
        const commentId = payload?.comment?.id;
        const issueNumberForComment = payload?.issue?.number;
        if (!commentId) {
          core.setFailed(`${ERR_VALIDATION}: Comment ID not found in event payload`);
          return;
        }
        if (!issueNumberForComment) {
          core.setFailed(`${ERR_NOT_FOUND}: Issue number not found in event payload`);
          return;
        }
        reactionEndpoint = `/repos/${owner}/${repo}/issues/comments/${commentId}/reactions`;
        // Create new comment on the issue itself, not on the comment
        commentUpdateEndpoint = `/repos/${owner}/${repo}/issues/${issueNumberForComment}/comments`;
        break;
      }

      case "pull_request": {
        const prNumber = payload?.pull_request?.number;
        if (!prNumber) {
          core.setFailed(`${ERR_NOT_FOUND}: Pull request number not found in event payload`);
          return;
        }
        // PRs are "issues" for the reactions endpoint
        reactionEndpoint = `/repos/${owner}/${repo}/issues/${prNumber}/reactions`;
        commentUpdateEndpoint = `/repos/${owner}/${repo}/issues/${prNumber}/comments`;
        break;
      }

      case "pull_request_review_comment": {
        const reviewCommentId = payload?.comment?.id;
        const prNumberForReviewComment = payload?.pull_request?.number;
        if (!reviewCommentId) {
          core.setFailed(`${ERR_VALIDATION}: Review comment ID not found in event payload`);
          return;
        }
        if (!prNumberForReviewComment) {
          core.setFailed(`${ERR_NOT_FOUND}: Pull request number not found in event payload`);
          return;
        }
        reactionEndpoint = `/repos/${owner}/${repo}/pulls/comments/${reviewCommentId}/reactions`;
        // Create new comment on the PR itself (using issues endpoint since PRs are issues)
        commentUpdateEndpoint = `/repos/${owner}/${repo}/issues/${prNumberForReviewComment}/comments`;
        break;
      }

      case "discussion": {
        const discussionNumber = payload?.discussion?.number;
        if (!discussionNumber) {
          core.setFailed(`${ERR_NOT_FOUND}: Discussion number not found in event payload`);
          return;
        }
        // Discussions use GraphQL API - get the node ID
        const discussion = await getDiscussionId(owner, repo, discussionNumber);
        reactionEndpoint = discussion.id; // Store node ID for GraphQL
        commentUpdateEndpoint = `discussion:${discussionNumber}`; // Special format to indicate discussion
        break;
      }

      case "discussion_comment": {
        const discussionCommentNumber = payload?.discussion?.number;
        const discussionCommentId = payload?.comment?.id;
        if (!discussionCommentNumber || !discussionCommentId) {
          core.setFailed(`${ERR_NOT_FOUND}: Discussion or comment information not found in event payload`);
          return;
        }
        const commentNodeId = payload?.comment?.node_id;
        if (!commentNodeId) {
          core.setFailed(`${ERR_NOT_FOUND}: Discussion comment node ID not found in event payload`);
          return;
        }
        reactionEndpoint = commentNodeId; // Store node ID for GraphQL
        commentUpdateEndpoint = `discussion_comment:${discussionCommentNumber}:${discussionCommentId}`; // Special format
        break;
      }

      default:
        core.setFailed(`${ERR_VALIDATION}: Unsupported event type: ${eventName}`);
        return;
    }

    core.info(`Reaction API endpoint: ${reactionEndpoint}`);

    // For discussions, reactionEndpoint is a node ID (GraphQL), otherwise it's a REST API path
    if (eventName === "discussion" || eventName === "discussion_comment") {
      await addDiscussionReaction(reactionEndpoint, reaction);
    } else {
      await addReaction(reactionEndpoint, reaction);
    }

    if (commentUpdateEndpoint) {
      core.info(`Comment endpoint: ${commentUpdateEndpoint}`);
      await addCommentWithWorkflowLink(commentUpdateEndpoint, runUrl, eventName, invocationContext);
    }
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
 * Add a reaction to a GitHub issue, PR, or comment using REST API
 * @param {string} endpoint - The GitHub API endpoint to add the reaction to
 * @param {string} reaction - The reaction type to add
 */
async function addReaction(endpoint, reaction) {
  const response = await github.request("POST " + endpoint, {
    content: reaction,
    headers: {
      Accept: "application/vnd.github+json",
    },
  });

  const reactionId = response.data?.id;
  if (reactionId) {
    core.info(`Successfully added reaction: ${reaction} (id: ${reactionId})`);
    core.setOutput("reaction-id", reactionId.toString());
  } else {
    core.info(`Successfully added reaction: ${reaction}`);
    core.setOutput("reaction-id", "");
  }
}

/**
 * Add a reaction to a GitHub discussion or discussion comment using GraphQL
 * @param {string} subjectId - The node ID of the discussion or comment
 * @param {string} reaction - The reaction type to add (mapped to GitHub's ReactionContent enum)
 */
async function addDiscussionReaction(subjectId, reaction) {
  const reactionContent = DISCUSSION_REACTION_MAP[reaction];
  if (!reactionContent) {
    throw new Error(`${ERR_VALIDATION}: Invalid reaction type for GraphQL: ${reaction}`);
  }

  const result = await github.graphql(
    `
    mutation($subjectId: ID!, $content: ReactionContent!) {
      addReaction(input: { subjectId: $subjectId, content: $content }) {
        reaction {
          id
          content
        }
      }
    }`,
    { subjectId, content: reactionContent }
  );

  const reactionId = result.addReaction.reaction.id;
  core.info(`Successfully added reaction: ${reaction} (id: ${reactionId})`);
  core.setOutput("reaction-id", reactionId);
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
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
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

    if (eventName === "discussion") {
      // Parse discussion number from special format: "discussion:NUMBER"
      const discussionNumber = parseInt(endpoint.split(":")[1], 10);
      const { id: discussionId } = await getDiscussionId(eventRepo.owner, eventRepo.repo, discussionNumber);

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
      setCommentOutputs(comment.id, comment.url, eventRepo);
      return;
    } else if (eventName === "discussion_comment") {
      // Parse discussion number from special format: "discussion_comment:NUMBER:COMMENT_ID"
      const discussionNumber = parseInt(endpoint.split(":")[1], 10);
      const { id: discussionId } = await getDiscussionId(eventRepo.owner, eventRepo.repo, discussionNumber);

      // Get the comment node ID to use as the parent for threading.
      // GitHub Discussions only supports two nesting levels, so if the triggering comment is
      // itself a reply, we resolve the top-level parent's node ID.
      const commentNodeId = await resolveTopLevelDiscussionCommentId(github, eventPayload?.comment?.node_id);

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
      setCommentOutputs(comment.id, comment.url, eventRepo);
      return;
    }

    // Create a new comment for non-discussion events
    const createResponse = await github.request("POST " + endpoint, {
      body: commentBody,
      headers: {
        Accept: "application/vnd.github+json",
      },
    });

    setCommentOutputs(createResponse.data.id.toString(), createResponse.data.html_url, eventRepo);
  } catch (error) {
    // Don't fail the entire job if comment creation fails - just log it
    const errorMessage = getErrorMessage(error);
    core.warning("Failed to create comment with workflow link (This is not critical - the reaction was still added successfully): " + errorMessage);
  }
}

module.exports = { main, addCommentWithWorkflowLink, addReaction, addDiscussionReaction };
