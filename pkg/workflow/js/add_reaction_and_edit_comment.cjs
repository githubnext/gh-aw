async function main() {
  // Read inputs from environment variables
  const reaction = process.env.GITHUB_AW_REACTION || "eyes";
  const command = process.env.GITHUB_AW_COMMAND; // Only present for command workflows
  const runId = context.runId;
  const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
  const runUrl = context.payload.repository
    ? `${context.payload.repository.html_url}/actions/runs/${runId}`
    : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

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
        commentUpdateEndpoint = `/repos/${owner}/${repo}/issues/${issueNumber}/comments`;
        // Create a comment for issue events
        shouldEditComment = true;
        break;

      case "issue_comment":
        const commentId = context.payload?.comment?.id;
        if (!commentId) {
          core.setFailed("Comment ID not found in event payload");
          return;
        }
        reactionEndpoint = `/repos/${owner}/${repo}/issues/comments/${commentId}/reactions`;
        commentUpdateEndpoint = `/repos/${owner}/${repo}/issues/comments/${commentId}`;
        // Only edit comments for command workflows
        shouldEditComment = command ? true : false;
        break;

      case "pull_request":
        const prNumber = context.payload?.pull_request?.number;
        if (!prNumber) {
          core.setFailed("Pull request number not found in event payload");
          return;
        }
        // PRs are "issues" for the reactions endpoint
        reactionEndpoint = `/repos/${owner}/${repo}/issues/${prNumber}/reactions`;
        commentUpdateEndpoint = `/repos/${owner}/${repo}/issues/${prNumber}/comments`;
        // Create a comment for pull request events
        shouldEditComment = true;
        break;

      case "pull_request_review_comment":
        const reviewCommentId = context.payload?.comment?.id;
        if (!reviewCommentId) {
          core.setFailed("Review comment ID not found in event payload");
          return;
        }
        reactionEndpoint = `/repos/${owner}/${repo}/pulls/comments/${reviewCommentId}/reactions`;
        commentUpdateEndpoint = `/repos/${owner}/${repo}/pulls/comments/${reviewCommentId}`;
        // Only edit comments for command workflows
        shouldEditComment = command ? true : false;
        break;

      default:
        core.setFailed(`Unsupported event type: ${eventName}`);
        return;
    }

    core.info(`Reaction API endpoint: ${reactionEndpoint}`);

    // Add reaction first
    await addReaction(reactionEndpoint, reaction);

    // Then add or edit comment if applicable
    if (shouldEditComment && commentUpdateEndpoint) {
      core.info(`Comment endpoint: ${commentUpdateEndpoint}`);
      await addOrEditCommentWithWorkflowLink(commentUpdateEndpoint, runUrl, eventName);
    } else {
      if (!command && commentUpdateEndpoint) {
        core.info("Skipping comment edit - only available for command workflows");
      } else {
        core.info(`Skipping comment for event type: ${eventName}`);
      }
    }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to process reaction and comment edit: ${errorMessage}`);
    core.setFailed(`Failed to process reaction and comment edit: ${errorMessage}`);
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
    core.info(`Successfully added reaction: ${reaction} (id: ${reactionId})`);
    core.setOutput("reaction-id", reactionId.toString());
  } else {
    core.info(`Successfully added reaction: ${reaction}`);
    core.setOutput("reaction-id", "");
  }
}

/**
 * Add or edit a comment to add a workflow run link
 * @param {string} endpoint - The GitHub API endpoint to update or create the comment
 * @param {string} runUrl - The URL of the workflow run
 * @param {string} eventName - The event type (to determine if we create or edit)
 */
async function addOrEditCommentWithWorkflowLink(endpoint, runUrl, eventName) {
  try {
    // Get workflow name from environment variable
    const workflowName = process.env.GITHUB_AW_WORKFLOW_NAME || "Workflow";

    // For issues and pull_request events, create a new comment
    // For comment events (issue_comment, pull_request_review_comment), edit the existing comment
    const isCreateComment = eventName === "issues" || eventName === "pull_request";

    if (isCreateComment) {
      // Create a new comment
      const workflowLinkText = `Agentic [${workflowName}](${runUrl}) triggered by this ${eventName === "issues" ? "issue" : "pull request"}`;

      const createResponse = await github.request("POST " + endpoint, {
        body: workflowLinkText,
        headers: {
          Accept: "application/vnd.github+json",
        },
      });

      core.info(`Successfully created comment with workflow link`);
      core.info(`Comment ID: ${createResponse.data.id}`);
      core.info(`Comment URL: ${createResponse.data.html_url}`);
      core.setOutput("comment-id", createResponse.data.id.toString());
      core.setOutput("comment-url", createResponse.data.html_url);
    } else {
      // Edit existing comment
      // First, get the current comment content
      const getResponse = await github.request("GET " + endpoint, {
        headers: {
          Accept: "application/vnd.github+json",
        },
      });

      const originalBody = getResponse.data.body || "";
      const workflowLinkText = `\n\nAgentic [${workflowName}](${runUrl}) triggered by this comment`;

      // Check if we've already added a workflow link to avoid duplicates
      // Look for the specific pattern "Agentic [<workflow-name>](<url>) triggered by this comment"
      const duplicatePattern = /Agentic \[.+?\]\(.+?\) triggered by this comment/;
      if (duplicatePattern.test(originalBody)) {
        core.info("Comment already contains a workflow run link, skipping edit");
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

      core.info(`Successfully updated comment with workflow link`);
      core.info(`Comment ID: ${updateResponse.data.id}`);
    }
  } catch (error) {
    // Don't fail the entire job if comment editing/creation fails - just log it
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.warning(
      "Failed to add/edit comment with workflow link (This is not critical - the reaction was still added successfully): " + errorMessage
    );
  }
}

await main();
