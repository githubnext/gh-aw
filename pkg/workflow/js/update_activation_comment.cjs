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
  const commentId = process.env.GH_AW_COMMENT_ID;
  const commentRepo = process.env.GH_AW_COMMENT_REPO;

  // If no comment was created in activation, skip updating
  if (!commentId) {
    core.info("No activation comment to update (GH_AW_COMMENT_ID not set)");
    return;
  }

  core.info(`Updating activation comment ${commentId} with PR link`);

  // Parse comment repo (format: "owner/repo")
  const repoOwner = commentRepo ? commentRepo.split("/")[0] : context.repo.owner;
  const repoName = commentRepo ? commentRepo.split("/")[1] : context.repo.repo;

  core.info(`Updating comment in ${repoOwner}/${repoName}`);

  // Prepare the message to append
  const prLinkMessage = `\n\n✅ Pull request created: [#${pullRequestNumber}](${pullRequestUrl})`;

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

      const currentBody = currentComment.node.body;
      const updatedBody = currentBody + prLinkMessage;

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
      core.info(`Successfully updated discussion comment with PR link`);
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

      const currentBody = currentComment.data.body;
      const updatedBody = currentBody + prLinkMessage;

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

      core.info(`Successfully updated comment with PR link`);
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
};
