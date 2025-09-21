import type { SafeOutputItems } from "./types/safe-outputs";

type ValidReaction = "+1" | "-1" | "laugh" | "confused" | "heart" | "hooray" | "rocket" | "eyes";

async function addReactionAndEditCommentMain(): Promise<void> {
  // Read inputs from environment variables
  const reaction = (process.env.GITHUB_AW_REACTION || "eyes") as ValidReaction;
  const command = process.env.GITHUB_AW_COMMAND; // Only present for command workflows
  const runId = context.runId;
  const runUrl = context.payload.repository
    ? `${context.payload.repository.html_url}/actions/runs/${runId}`
    : `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

  core.info(`Reaction type: ${reaction}`);
  core.info(`Command name: ${command || "none"}`);
  core.info(`Run ID: ${runId}`);
  core.info(`Run URL: ${runUrl}`);

  // Validate reaction type
  const validReactions: ValidReaction[] = ["+1", "-1", "laugh", "confused", "heart", "hooray", "rocket", "eyes"];
  if (!validReactions.includes(reaction)) {
    core.setFailed(`Invalid reaction type: ${reaction}. Valid reactions are: ${validReactions.join(", ")}`);
    return;
  }

  // Determine the API endpoint based on the event type
  let reactionEndpoint: string;
  let commentUpdateEndpoint: string | undefined;
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
        // Only edit comments for command workflows  
        shouldEditComment = command ? true : false;
        break;

      case "pull_request":
        const prNumber = context.payload?.pull_request?.number;
        if (!prNumber) {
          core.setFailed("Pull request number not found in event payload");
          return;
        }
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
        // Only edit comments for command workflows
        shouldEditComment = command ? true : false;
        break;

      default:
        core.setFailed(`Unsupported event type: ${eventName}`);
        return;
    }

    core.info(`Reaction endpoint: ${reactionEndpoint}`);
    if (commentUpdateEndpoint) {
      core.info(`Comment update endpoint: ${commentUpdateEndpoint}`);
    }

    // Add the reaction
    const reactionResponse = await github.request(`POST ${reactionEndpoint}`, {
      content: reaction,
    });

    if (reactionResponse.status === 200 || reactionResponse.status === 201) {
      core.info(`Successfully added ${reaction} reaction`);
      core.setOutput("reaction_added", "true");
      core.setOutput("reaction_type", reaction);
      core.setOutput("reaction_id", reactionResponse.data.id || "unknown");
    } else {
      core.warning(`Unexpected response status when adding reaction: ${reactionResponse.status}`);
    }

    // Edit the comment if applicable and if it's a command workflow
    if (shouldEditComment && commentUpdateEndpoint && command) {
      core.info(`Comment update endpoint: ${commentUpdateEndpoint}`);
      await editCommentWithWorkflowLink(commentUpdateEndpoint, runUrl);
    }
  } catch (error) {
    core.setFailed(`Failed to add reaction: ${error instanceof Error ? error.message : String(error)}`);
  }
}

/**
 * Edit a comment to add a workflow run link
 * @param endpoint - The GitHub API endpoint to update the comment
 * @param runUrl - The URL of the workflow run
 */
async function editCommentWithWorkflowLink(endpoint: string, runUrl: string): Promise<void> {
  try {
    // Get the current comment content
    const currentCommentResponse = await github.request(`GET ${endpoint}`);
    const currentBody = currentCommentResponse.data.body || "";
    const command = process.env.GITHUB_AW_COMMAND;

    // Check if the footer already exists
    const footerText = `\n\n> Processed by [${command}](${runUrl}) 🚀`;
    
    let newBody: string;
    if (currentBody.includes(`> Processed by [${command}]`)) {
      // Footer already exists, don't duplicate it
      core.info("Comment already has workflow footer, skipping edit");
      return;
    } else {
      // Add the footer
      newBody = currentBody + footerText;
    }

    // Update the comment
    const updateResponse = await github.request(`PATCH ${endpoint}`, {
      body: newBody,
    });

    if (updateResponse.status === 200) {
      core.info("Successfully updated comment with workflow footer");
      core.setOutput("comment_updated", "true");
    } else {
      core.warning(`Unexpected response status when updating comment: ${updateResponse.status}`);
    }
  } catch (updateError) {
    // Non-fatal error - reaction was still added
    core.warning(`Failed to update comment: ${updateError instanceof Error ? updateError.message : String(updateError)}`);
    core.setOutput("comment_updated", "false");
  }
}

(async () => {
  await addReactionAndEditCommentMain();
})();