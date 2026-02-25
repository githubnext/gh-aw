// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage, isLockedError } = require("./error_helpers.cjs");
const { ERR_API, ERR_NOT_FOUND, ERR_VALIDATION } = require("./error_codes.cjs");

/**
 * Add a reaction to the triggering item (issue, PR, comment, or discussion).
 * This provides immediate feedback to the user when a workflow is triggered.
 * This script only adds reactions - it does NOT create comments.
 * Use add_reaction_and_edit_comment.cjs in the activation job to create the comment with workflow link.
 */
async function main() {
  // Read inputs from environment variables
  const reaction = process.env.GH_AW_REACTION || "eyes";

  core.info(`Adding reaction: ${reaction}`);

  // Validate reaction type
  const validReactions = ["+1", "-1", "laugh", "confused", "heart", "hooray", "rocket", "eyes"];
  if (!validReactions.includes(reaction)) {
    core.setFailed(`${ERR_VALIDATION}: Invalid reaction type: ${reaction}. Valid reactions are: ${validReactions.join(", ")}`);
    return;
  }

  // Determine the API endpoint based on the event type
  let reactionEndpoint;
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
        reactionEndpoint = `/repos/${owner}/${repo}/issues/${issueNumber}/reactions`;
        break;
      }

      case "issue_comment": {
        const commentId = context.payload?.comment?.id;
        if (!commentId) {
          core.setFailed(`${ERR_VALIDATION}: Comment ID not found in event payload`);
          return;
        }
        reactionEndpoint = `/repos/${owner}/${repo}/issues/comments/${commentId}/reactions`;
        break;
      }

      case "pull_request": {
        const prNumber = context.payload?.pull_request?.number;
        if (!prNumber) {
          core.setFailed(`${ERR_NOT_FOUND}: Pull request number not found in event payload`);
          return;
        }
        // PRs are "issues" for the reactions endpoint
        reactionEndpoint = `/repos/${owner}/${repo}/issues/${prNumber}/reactions`;
        break;
      }

      case "pull_request_review_comment": {
        const reviewCommentId = context.payload?.comment?.id;
        if (!reviewCommentId) {
          core.setFailed(`${ERR_VALIDATION}: Review comment ID not found in event payload`);
          return;
        }
        reactionEndpoint = `/repos/${owner}/${repo}/pulls/comments/${reviewCommentId}/reactions`;
        break;
      }

      case "discussion": {
        const discussionNumber = context.payload?.discussion?.number;
        if (!discussionNumber) {
          core.setFailed(`${ERR_NOT_FOUND}: Discussion number not found in event payload`);
          return;
        }
        // Discussions use GraphQL API - get the node ID
        const discussionNodeId = await getDiscussionNodeId(owner, repo, discussionNumber);
        await addDiscussionReaction(discussionNodeId, reaction);
        return; // Early return for discussion events
      }

      case "discussion_comment": {
        const commentNodeId = context.payload?.comment?.node_id;
        if (!commentNodeId) {
          core.setFailed(`${ERR_NOT_FOUND}: Discussion comment node ID not found in event payload`);
          return;
        }
        await addDiscussionReaction(commentNodeId, reaction);
        return; // Early return for discussion comment events
      }

      default:
        core.setFailed(`${ERR_VALIDATION}: Unsupported event type: ${eventName}`);
        return;
    }

    // Add reaction using REST API (for non-discussion events)
    core.info(`Adding reaction to: ${reactionEndpoint}`);
    await addReaction(reactionEndpoint, reaction);
  } catch (error) {
    const errorMessage = getErrorMessage(error);

    // Check if the error is due to a locked issue/PR/discussion
    if (isLockedError(error)) {
      // Silently ignore locked resource errors - just log for debugging
      core.info(`Cannot add reaction: resource is locked (this is expected and not an error)`);
      return;
    }

    // For other errors, fail as before
    core.error(`Failed to add reaction: ${errorMessage}`);
    core.setFailed(`${ERR_API}: Failed to add reaction: ${errorMessage}`);
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
 * @returns {Promise<string>} Discussion node ID
 */
async function getDiscussionNodeId(owner, repo, discussionNumber) {
  const { repository } = await github.graphql(
    `
    query($owner: String!, $repo: String!, $num: Int!) {
      repository(owner: $owner, name: $repo) {
        discussion(number: $num) { 
          id 
        }
      }
    }`,
    { owner, repo, num: discussionNumber }
  );

  if (!repository || !repository.discussion) {
    throw new Error(`${ERR_NOT_FOUND}: Discussion #${discussionNumber} not found in ${owner}/${repo}`);
  }

  return repository.discussion.id;
}

module.exports = { main };
