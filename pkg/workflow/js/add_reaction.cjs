async function main() {
  // Read inputs from environment variables
  const reaction = process.env.GITHUB_AW_REACTION || "eyes";

  console.log("Reaction type:", reaction);

  // Validate reaction type
  const validReactions = [
    "+1",
    "-1",
    "laugh",
    "confused",
    "heart",
    "hooray",
    "rocket",
    "eyes",
  ];
  if (!validReactions.includes(reaction)) {
    core.setFailed(
      `Invalid reaction type: ${reaction}. Valid reactions are: ${validReactions.join(", ")}`
    );
    return;
  }

  // Determine the API endpoint based on the event type
  let endpoint;
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
        endpoint = `/repos/${owner}/${repo}/issues/${issueNumber}/reactions`;
        break;

      case "issue_comment":
        const commentId = context.payload?.comment?.id;
        if (!commentId) {
          core.setFailed("Comment ID not found in event payload");
          return;
        }
        endpoint = `/repos/${owner}/${repo}/issues/comments/${commentId}/reactions`;
        break;

      case "pull_request":
      case "pull_request_target":
        const prNumber = context.payload?.pull_request?.number;
        if (!prNumber) {
          core.setFailed("Pull request number not found in event payload");
          return;
        }
        // PRs are "issues" for the reactions endpoint
        endpoint = `/repos/${owner}/${repo}/issues/${prNumber}/reactions`;
        break;

      case "pull_request_review_comment":
        const reviewCommentId = context.payload?.comment?.id;
        if (!reviewCommentId) {
          core.setFailed("Review comment ID not found in event payload");
          return;
        }
        endpoint = `/repos/${owner}/${repo}/pulls/comments/${reviewCommentId}/reactions`;
        break;

      default:
        core.setFailed(`Unsupported event type: ${eventName}`);
        return;
    }

    console.log("API endpoint:", endpoint);

    await addReaction(endpoint, reaction);
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    console.error("Failed to add reaction:", errorMessage);
    core.setFailed(`Failed to add reaction: ${errorMessage}`);
  }
}

/**
 * Add a reaction to a GitHub issue, PR, or comment
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
    console.log(`Successfully added reaction: ${reaction} (id: ${reactionId})`);
    core.setOutput("reaction-id", reactionId.toString());
  } else {
    console.log(`Successfully added reaction: ${reaction}`);
    core.setOutput("reaction-id", "");
  }
}

await main();
