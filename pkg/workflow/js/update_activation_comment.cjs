// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Update the activation comment with a link to the created pull request
 * @param {any} github - GitHub REST API instance
 * @param {any} context - GitHub Actions context
 * @param {any} core - GitHub Actions core
 * @param {string} pullRequestUrl - URL of the created pull request
 * @param {number} pullRequestNumber - Number of the pull request
 */
async function updateActivationComment(github, context, core, pullRequestUrl, pullRequestNumber) {
  const message = `\n\n✅ Pull request created: [#${pullRequestNumber}](${pullRequestUrl})`;
  await updateActivationCommentWithMessage(github, context, core, message);
}

/**
 * Update the activation comment with a commit link
 * @param {any} github - GitHub REST API instance
 * @param {any} context - GitHub Actions context
 * @param {any} core - GitHub Actions core
 * @param {string} commitSha - SHA of the commit
 * @param {string} commitUrl - URL of the commit
 */
async function updateActivationCommentWithCommit(github, context, core, commitSha, commitUrl) {
  const shortSha = commitSha.substring(0, 7);
  const message = `\n\n✅ Commit pushed: [\`${shortSha}\`](${commitUrl})`;
  await updateActivationCommentWithMessage(github, context, core, message);
}

/**
 * Update the activation comment with a custom message
 * @param {any} github - GitHub REST API instance
 * @param {any} context - GitHub Actions context
 * @param {any} core - GitHub Actions core
 * @param {string} message - Message to append to the comment
 */
async function updateActivationCommentWithMessage(github, context, core, message) {
  const commentId = process.env.GH_AW_COMMENT_ID;
  const commentRepo = process.env.GH_AW_COMMENT_REPO;

  // If no comment was created in activation, skip updating
  if (!commentId) {
    core.info("No activation comment to update (GH_AW_COMMENT_ID not set)");
    return;
  }

  core.info(`Updating activation comment ${commentId}`);

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
      core.info(`Successfully updated discussion comment`);
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
      const updatedBody = currentBody + message;

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

      core.info(`Successfully updated comment`);
      core.info(`Comment ID: ${response.data.id}`);
      core.info(`Comment URL: ${response.data.html_url}`);
    }
  } catch (error) {
    // Don't fail the workflow if we can't update the comment - just log a warning
    core.warning(`Failed to update activation comment: ${error instanceof Error ? error.message : String(error)}`);
  }
}

module.exports = {
  updateActivationComment,
  updateActivationCommentWithCommit,
};
