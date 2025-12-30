// @ts-check
/// <reference types="@actions/github-script" />

const { getRunStartedMessage } = require("./messages_run_status.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

async function main() {
  // Read inputs from environment variables
  const reaction = process.env.GH_AW_REACTION || "eyes";
  const command = process.env.GH_AW_COMMAND; // Only present for command workflows
  const runId = context.runId;
  const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
  const runUrl = context.payload.repository ? `${context.payload.repository.html_url}/actions/runs/${runId}` : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

  core.info(`Reaction type: ${reaction}`);
  core.info(`Command name: ${command || "none"}`);
  core.info(`Run ID: ${runId}`);
  core.info(`Run URL: ${runUrl}`);

  // Validate reaction type
  const validReactions = ["+1", "-1", "laugh", "confused", "heart", "hooray", "rocket", "eyes"];
  if (!validReactions.includes(reaction)) {
    core.setFailed(`Invalid reaction type: ${reaction}. Valid reactions are: ${validReactions.join(", ")}`);
    return;
  }

  const {
    eventName,
    repo: { owner, repo },
  } = context;
  let reactionEndpoint;
  let commentUpdateEndpoint;
  const shouldCreateComment = true; // Always create comments for all events

  try {
    // Event-specific endpoint configuration
    const eventConfig = {
      issues: () => {
        const issueNumber = context.payload?.issue?.number;
        if (!issueNumber) {
          core.setFailed("Issue number not found in event payload");
          return null;
        }
        return {
          reactionEndpoint: `/repos/${owner}/${repo}/issues/${issueNumber}/reactions`,
          commentUpdateEndpoint: `/repos/${owner}/${repo}/issues/${issueNumber}/comments`,
        };
      },
      issue_comment: () => {
        const commentId = context.payload?.comment?.id;
        const issueNumber = context.payload?.issue?.number;
        if (!commentId || !issueNumber) {
          core.setFailed("Comment ID or issue number not found in event payload");
          return null;
        }
        return {
          reactionEndpoint: `/repos/${owner}/${repo}/issues/comments/${commentId}/reactions`,
          commentUpdateEndpoint: `/repos/${owner}/${repo}/issues/${issueNumber}/comments`,
        };
      },
      pull_request: () => {
        const prNumber = context.payload?.pull_request?.number;
        if (!prNumber) {
          core.setFailed("Pull request number not found in event payload");
          return null;
        }
        return {
          reactionEndpoint: `/repos/${owner}/${repo}/issues/${prNumber}/reactions`,
          commentUpdateEndpoint: `/repos/${owner}/${repo}/issues/${prNumber}/comments`,
        };
      },
      pull_request_review_comment: () => {
        const reviewCommentId = context.payload?.comment?.id;
        const prNumber = context.payload?.pull_request?.number;
        if (!reviewCommentId || !prNumber) {
          core.setFailed("Review comment ID or pull request number not found in event payload");
          return null;
        }
        return {
          reactionEndpoint: `/repos/${owner}/${repo}/pulls/comments/${reviewCommentId}/reactions`,
          commentUpdateEndpoint: `/repos/${owner}/${repo}/issues/${prNumber}/comments`,
        };
      },
      discussion: async () => {
        const discussionNumber = context.payload?.discussion?.number;
        if (!discussionNumber) {
          core.setFailed("Discussion number not found in event payload");
          return null;
        }
        const discussion = await getDiscussionId(owner, repo, discussionNumber);
        return {
          reactionEndpoint: discussion.id,
          commentUpdateEndpoint: `discussion:${discussionNumber}`,
        };
      },
      discussion_comment: () => {
        const discussionNumber = context.payload?.discussion?.number;
        const commentId = context.payload?.comment?.id;
        const commentNodeId = context.payload?.comment?.node_id;
        if (!discussionNumber || !commentId || !commentNodeId) {
          core.setFailed("Discussion or comment information not found in event payload");
          return null;
        }
        return {
          reactionEndpoint: commentNodeId,
          commentUpdateEndpoint: `discussion_comment:${discussionNumber}:${commentId}`,
        };
      },
    };

    const configFn = eventConfig[eventName];
    if (!configFn) {
      core.setFailed(`Unsupported event type: ${eventName}`);
      return;
    }

    const config = await configFn();
    if (!config) return;

    ({ reactionEndpoint, commentUpdateEndpoint } = config);

    core.info(`Reaction API endpoint: ${reactionEndpoint}`);

    // Add reaction first (use GraphQL for discussions, REST for others)
    const isDiscussionEvent = eventName === "discussion" || eventName === "discussion_comment";
    await (isDiscussionEvent ? addDiscussionReaction(reactionEndpoint, reaction) : addReaction(reactionEndpoint, reaction));

    // Then add comment
    core.info(`Comment endpoint: ${commentUpdateEndpoint}`);
    await addCommentWithWorkflowLink(commentUpdateEndpoint, runUrl, eventName);
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    core.error(`Failed to process reaction and comment creation: ${errorMessage}`);
    core.setFailed(`Failed to process reaction and comment creation: ${errorMessage}`);
  }
}

/**
 * Add a reaction to a GitHub issue, PR, or comment using REST API
 * @param {string} endpoint - The GitHub API endpoint to add the reaction to
 * @param {string} reaction - The reaction type to add
 */
async function addReaction(endpoint, reaction) {
  const response = await github.request(`POST ${endpoint}`, {
    content: reaction,
    headers: { Accept: "application/vnd.github+json" },
  });

  const reactionId = response.data?.id;
  core.info(`Successfully added reaction: ${reaction}${reactionId ? ` (id: ${reactionId})` : ""}`);
  core.setOutput("reaction-id", reactionId?.toString() || "");
}

/**
 * Add a reaction to a GitHub discussion or discussion comment using GraphQL
 * @param {string} subjectId - The node ID of the discussion or comment
 * @param {string} reaction - The reaction type to add (mapped to GitHub's ReactionContent enum)
 */
async function addDiscussionReaction(subjectId, reaction) {
  // Map reaction names to GitHub's GraphQL ReactionContent enum
  const reactionMap = {
    "+1": "THUMBS_UP",
    "-1": "THUMBS_DOWN",
    laugh: "LAUGH",
    confused: "CONFUSED",
    heart: "HEART",
    hooray: "HOORAY",
    rocket: "ROCKET",
    eyes: "EYES",
  };

  const reactionContent = reactionMap[reaction];
  if (!reactionContent) {
    throw new Error(`Invalid reaction type for GraphQL: ${reaction}`);
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
    `query($owner: String!, $repo: String!, $num: Int!) {
      repository(owner: $owner, name: $repo) {
        discussion(number: $num) { id url }
      }
    }`,
    { owner, repo, num: discussionNumber }
  );

  if (!repository?.discussion) {
    throw new Error(`Discussion #${discussionNumber} not found in ${owner}/${repo}`);
  }

  return repository.discussion;
}

/**
 * Add a comment with a workflow run link
 * @param {string} endpoint - The GitHub API endpoint to create the comment (or special format for discussions)
 * @param {string} runUrl - The URL of the workflow run
 * @param {string} eventName - The event type (to determine the comment text)
 */
async function addCommentWithWorkflowLink(endpoint, runUrl, eventName) {
  try {
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";

    // Map event names to descriptions
    const eventDescriptions = {
      issues: "issue",
      pull_request: "pull request",
      issue_comment: "issue comment",
      pull_request_review_comment: "pull request review comment",
      discussion: "discussion",
      discussion_comment: "discussion comment",
    };
    const eventTypeDescription = eventDescriptions[eventName] || "event";

    const workflowLinkText = getRunStartedMessage({
      workflowName,
      runUrl,
      eventType: eventTypeDescription,
    });

    const workflowId = process.env.GITHUB_WORKFLOW || "";
    const trackerId = process.env.GH_AW_TRACKER_ID || "";
    const lockForAgent = process.env.GH_AW_LOCK_FOR_AGENT === "true";

    // Build comment body with optional sections
    const commentParts = [workflowLinkText];

    if (lockForAgent && (eventName === "issues" || eventName === "issue_comment")) {
      commentParts.push("🔒 This issue has been locked while the workflow is running to prevent concurrent modifications.");
    }

    if (workflowId) {
      commentParts.push(`<!-- workflow-id: ${workflowId} -->`);
    }

    if (trackerId) {
      commentParts.push(`<!-- tracker-id: ${trackerId} -->`);
    }

    commentParts.push("<!-- comment-type: reaction -->");

    const commentBody = commentParts.join("\n\n");

    // Create comment (discussion vs REST API)
    const { owner, repo } = context.repo;
    let comment;

    if (eventName === "discussion" || eventName === "discussion_comment") {
      const discussionNumber = parseInt(endpoint.split(":")[1], 10);
      const { repository } = await github.graphql(
        `query($owner: String!, $repo: String!, $num: Int!) {
          repository(owner: $owner, name: $repo) {
            discussion(number: $num) { id }
          }
        }`,
        { owner, repo, num: discussionNumber }
      );

      const discussionId = repository.discussion.id;
      const mutationParams = { dId: discussionId, body: commentBody };

      // Add replyToId for comment threads
      if (eventName === "discussion_comment") {
        mutationParams.replyToId = context.payload?.comment?.node_id;
      }

      const result = await github.graphql(
        `mutation($dId: ID!, $body: String!${eventName === "discussion_comment" ? ", $replyToId: ID!" : ""}) {
          addDiscussionComment(input: { 
            discussionId: $dId, 
            body: $body
            ${eventName === "discussion_comment" ? ", replyToId: $replyToId" : ""}
          }) {
            comment { id url }
          }
        }`,
        mutationParams
      );

      comment = result.addDiscussionComment.comment;
    } else {
      const response = await github.request(`POST ${endpoint}`, {
        body: commentBody,
        headers: { Accept: "application/vnd.github+json" },
      });
      comment = { id: response.data.id.toString(), url: response.data.html_url };
    }

    core.info(`Successfully created comment with workflow link`);
    core.info(`Comment ID: ${comment.id}`);
    core.info(`Comment URL: ${comment.url}`);
    core.info(`Comment Repo: ${owner}/${repo}`);
    core.setOutput("comment-id", comment.id);
    core.setOutput("comment-url", comment.url);
    core.setOutput("comment-repo", `${owner}/${repo}`);
  } catch (error) {
    // Don't fail the entire job if comment creation fails - just log it
    const errorMessage = getErrorMessage(error);
    core.warning("Failed to create comment with workflow link (This is not critical - the reaction was still added successfully): " + errorMessage);
  }
}

module.exports = { main };
