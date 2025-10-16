// This script updates an existing comment created by the activation job
// to notify that the workflow failed or didn't produce a result.

async function main() {
  const commentId = process.env.GITHUB_AW_COMMENT_ID;
  const commentRepo = process.env.GITHUB_AW_COMMENT_REPO;
  const runUrl = process.env.GITHUB_AW_RUN_URL;
  const workflowName = process.env.GITHUB_AW_WORKFLOW_NAME || "Workflow";
  const agentConclusion = process.env.GITHUB_AW_AGENT_CONCLUSION || "failure";

  core.info(`Comment ID: ${commentId}`);
  core.info(`Comment Repo: ${commentRepo}`);
  core.info(`Run URL: ${runUrl}`);
  core.info(`Workflow Name: ${workflowName}`);
  core.info(`Agent Conclusion: ${agentConclusion}`);

  if (!commentId) {
    core.info("No comment ID found, skipping comment update");
    return;
  }

  if (!runUrl) {
    core.setFailed("Run URL is required");
    return;
  }

  // Parse comment repo (format: "owner/repo")
  const repoOwner = commentRepo ? commentRepo.split("/")[0] : context.repo.owner;
  const repoName = commentRepo ? commentRepo.split("/")[1] : context.repo.repo;

  core.info(`Updating comment in ${repoOwner}/${repoName}`);

  // Determine the error message based on agent conclusion
  let statusEmoji = "❌";
  let statusText = "failed";
  
  if (agentConclusion === "cancelled") {
    statusEmoji = "🚫";
    statusText = "was cancelled";
  } else if (agentConclusion === "skipped") {
    statusEmoji = "⏭️";
    statusText = "was skipped";
  } else if (agentConclusion === "timed_out") {
    statusEmoji = "⏱️";
    statusText = "timed out";
  }

  const errorMessage = `${statusEmoji} Agentic [${workflowName}](${runUrl}) ${statusText} and wasn't able to produce a result.`;

  // Check if this is a discussion comment (GraphQL node ID format)
  const isDiscussionComment = commentId.startsWith("DC_");

  try {
    if (isDiscussionComment) {
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
        { commentId: commentId, body: errorMessage }
      );

      const comment = result.updateDiscussionComment.comment;
      core.info(`Successfully updated discussion comment`);
      core.info(`Comment ID: ${comment.id}`);
      core.info(`Comment URL: ${comment.url}`);
    } else {
      // Update issue/PR comment using REST API
      const response = await github.request(
        "PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}",
        {
          owner: repoOwner,
          repo: repoName,
          comment_id: parseInt(commentId, 10),
          body: errorMessage,
          headers: {
            Accept: "application/vnd.github+json",
          },
        }
      );

      core.info(`Successfully updated comment`);
      core.info(`Comment ID: ${response.data.id}`);
      core.info(`Comment URL: ${response.data.html_url}`);
    }
  } catch (error) {
    // Don't fail the workflow if we can't update the comment
    core.warning(`Failed to update comment: ${error instanceof Error ? error.message : String(error)}`);
  }
}

main().catch((error) => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
