async function main() {
  // Read inputs from environment variables
  const reaction = process.env.GITHUB_AW_REACTION || "eyes";
  const alias = process.env.GITHUB_AW_ALIAS; // Only present for alias workflows
  const runId = context.runId;
  const runUrl = context.payload.repository
    ? `${context.payload.repository.html_url}/actions/runs/${runId}`
    : `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

  console.log("Reaction type:", reaction);
  console.log("Alias name:", alias || "none");
  console.log("Run ID:", runId);
  console.log("Run URL:", runUrl);

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
  let reactionEndpoint;
  let commentUpdateEndpoint;
  let shouldEditComment = false;
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
        reactionEndpoint = `/repos/${owner}/${repo}/issues/${issueNumber}/reactions`;
        // Don't edit issue bodies for now - this might be more complex
        shouldEditComment = false;
        break;

      case "issue_comment":
        const commentId = context.payload?.comment?.id;
        if (!commentId) {
          core.setFailed("Comment ID not found in event payload");
          return;
        }
        reactionEndpoint = `/repos/${owner}/${repo}/issues/comments/${commentId}/reactions`;
        commentUpdateEndpoint = `/repos/${owner}/${repo}/issues/comments/${commentId}`;
        // Only edit comments for alias workflows
        shouldEditComment = alias ? true : false;
        break;

      case "pull_request":
        const prNumber = context.payload?.pull_request?.number;
        if (!prNumber) {
          core.setFailed("Pull request number not found in event payload");
          return;
        }
        // PRs are "issues" for the reactions endpoint
        reactionEndpoint = `/repos/${owner}/${repo}/issues/${prNumber}/reactions`;
        // Don't edit PR bodies for now - this might be more complex
        shouldEditComment = false;
        break;

      case "pull_request_review_comment":
        const reviewCommentId = context.payload?.comment?.id;
        if (!reviewCommentId) {
          core.setFailed("Review comment ID not found in event payload");
          return;
        }
        reactionEndpoint = `/repos/${owner}/${repo}/pulls/comments/${reviewCommentId}/reactions`;
        commentUpdateEndpoint = `/repos/${owner}/${repo}/pulls/comments/${reviewCommentId}`;
        // Only edit comments for alias workflows
        shouldEditComment = alias ? true : false;
        break;

      default:
        core.setFailed(`Unsupported event type: ${eventName}`);
        return;
    }

    console.log("Reaction API endpoint:", reactionEndpoint);

    // Add reaction first
    await addReaction(reactionEndpoint, reaction);

    // Then edit comment if applicable and if it's a comment event
    if (shouldEditComment && commentUpdateEndpoint) {
      console.log("Comment update endpoint:", commentUpdateEndpoint);
      await editCommentWithWorkflowLink(commentUpdateEndpoint, runUrl);
    } else {
      if (!alias && commentUpdateEndpoint) {
        console.log(
          "Skipping comment edit - only available for alias workflows"
        );
      } else {
        console.log("Skipping comment edit for event type:", eventName);
      }
    }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    console.error("Failed to process reaction and comment edit:", errorMessage);
    core.setFailed(
      `Failed to process reaction and comment edit: ${errorMessage}`
    );
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

/**
 * Edit a comment to add a workflow run link
 * @param {string} endpoint - The GitHub API endpoint to update the comment
 * @param {string} runUrl - The URL of the workflow run
 */
async function editCommentWithWorkflowLink(endpoint, runUrl) {
  try {
    // First, get the current comment content
    const getResponse = await github.request("GET " + endpoint, {
      headers: {
        Accept: "application/vnd.github+json",
      },
    });

    const originalBody = getResponse.data.body || "";
    const workflowLinkText = `\n\n---\n*🤖 [Workflow run](${runUrl}) triggered by this comment*`;

    // Check if we've already added a workflow link to avoid duplicates
    if (originalBody.includes("*🤖 [Workflow run](")) {
      console.log(
        "Comment already contains a workflow run link, skipping edit"
      );
      return;
    }

    const updatedBody = originalBody + workflowLinkText;

    // Update the comment
    const updateResponse = await github.request("PATCH " + endpoint, {
      body: updatedBody,
      headers: {
        Accept: "application/vnd.github+json",
      },
    });

    console.log(`Successfully updated comment with workflow link`);
    console.log(`Comment ID: ${updateResponse.data.id}`);
  } catch (error) {
    // Don't fail the entire job if comment editing fails - just log it
    const errorMessage = error instanceof Error ? error.message : String(error);
    console.warn("Failed to edit comment with workflow link:", errorMessage);
    console.warn(
      "This is not critical - the reaction was still added successfully"
    );
  }
}

await main();
