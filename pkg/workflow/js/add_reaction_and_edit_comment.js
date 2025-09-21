async function addReactionAndEditCommentMain() {
    const reaction = (process.env.GITHUB_AW_REACTION || "eyes");
    const command = process.env.GITHUB_AW_COMMAND;
    const runId = context.runId;
    const runUrl = context.payload.repository
        ? `${context.payload.repository.html_url}/actions/runs/${runId}`
        : `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;
    core.info(`Reaction type: ${reaction}`);
    core.info(`Command name: ${command || "none"}`);
    core.info(`Run ID: ${runId}`);
    core.info(`Run URL: ${runUrl}`);
    const validReactions = ["+1", "-1", "laugh", "confused", "heart", "hooray", "rocket", "eyes"];
    if (!validReactions.includes(reaction)) {
        core.setFailed(`Invalid reaction type: ${reaction}. Valid reactions are: ${validReactions.join(", ")}`);
        return;
    }
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
                shouldEditComment = true;
                break;
            case "pull_request":
                const prNumber = context.payload?.pull_request?.number;
                if (!prNumber) {
                    core.setFailed("Pull request number not found in event payload");
                    return;
                }
                reactionEndpoint = `/repos/${owner}/${repo}/issues/${prNumber}/reactions`;
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
                shouldEditComment = true;
                break;
            default:
                core.setFailed(`Unsupported event type: ${eventName}`);
                return;
        }
        core.info(`Reaction endpoint: ${reactionEndpoint}`);
        if (commentUpdateEndpoint) {
            core.info(`Comment update endpoint: ${commentUpdateEndpoint}`);
        }
        const reactionResponse = await github.request(`POST ${reactionEndpoint}`, {
            content: reaction,
        });
        if (reactionResponse.status === 200 || reactionResponse.status === 201) {
            core.info(`Successfully added ${reaction} reaction`);
            core.setOutput("reaction_added", "true");
            core.setOutput("reaction_type", reaction);
            core.setOutput("reaction_id", reactionResponse.data.id || "unknown");
        }
        else {
            core.warning(`Unexpected response status when adding reaction: ${reactionResponse.status}`);
        }
        if (shouldEditComment && commentUpdateEndpoint && command) {
            try {
                const currentCommentResponse = await github.request(`GET ${commentUpdateEndpoint}`);
                const currentBody = currentCommentResponse.data.body || "";
                const footerText = `\n\n> Processed by [${command}](${runUrl}) 🚀`;
                let newBody;
                if (currentBody.includes(`> Processed by [${command}]`)) {
                    core.info("Comment already has workflow footer, skipping edit");
                    return;
                }
                else {
                    newBody = currentBody + footerText;
                }
                const updateResponse = await github.request(`PATCH ${commentUpdateEndpoint}`, {
                    body: newBody,
                });
                if (updateResponse.status === 200) {
                    core.info("Successfully updated comment with workflow footer");
                    core.setOutput("comment_updated", "true");
                }
                else {
                    core.warning(`Unexpected response status when updating comment: ${updateResponse.status}`);
                }
            }
            catch (updateError) {
                core.warning(`Failed to update comment: ${updateError instanceof Error ? updateError.message : String(updateError)}`);
                core.setOutput("comment_updated", "false");
            }
        }
    }
    catch (error) {
        core.setFailed(`Failed to add reaction: ${error instanceof Error ? error.message : String(error)}`);
    }
}
(async () => {
    await addReactionAndEditCommentMain();
})();

