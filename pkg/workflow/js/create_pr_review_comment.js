async function createPrReviewCommentMain() {
    const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
    if (!outputContent) {
        core.info("No GITHUB_AW_AGENT_OUTPUT environment variable found");
        return [];
    }
    if (outputContent.trim() === "") {
        core.info("Agent output content is empty");
        return [];
    }
    core.info(`Agent output content length: ${outputContent.length}`);
    let validatedOutput;
    try {
        validatedOutput = JSON.parse(outputContent);
    }
    catch (error) {
        core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
        return [];
    }
    if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
        core.info("No valid items found in agent output");
        return [];
    }
    const reviewCommentItems = validatedOutput.items.filter(item => item.type === "create-pull-request-review-comment");
    if (reviewCommentItems.length === 0) {
        core.info("No create-pull-request-review-comment items found in agent output");
        return [];
    }
    core.info(`Found ${reviewCommentItems.length} create-pull-request-review-comment item(s)`);
    if (process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true") {
        let summaryContent = "## 🎭 Staged Mode: Create PR Review Comments Preview\n\n";
        summaryContent += "The following review comments would be created if staged mode was disabled:\n\n";
        for (let i = 0; i < reviewCommentItems.length; i++) {
            const item = reviewCommentItems[i];
            summaryContent += `### Review Comment ${i + 1}\n`;
            summaryContent += `**File:** ${item.path || "No path provided"}\n\n`;
            summaryContent += `**Line:** ${item.line || "No line provided"}\n\n`;
            summaryContent += `**Side:** ${item.side || "RIGHT"}\n\n`;
            summaryContent += `**Body:**\n${item.body || "No content provided"}\n\n`;
            summaryContent += "---\n\n";
        }
        await core.summary.addRaw(summaryContent).write();
        core.info("📝 PR review comment creation preview written to step summary");
        return [];
    }
    let prNumber;
    const prTarget = process.env.GITHUB_AW_PR_TARGET || "triggering";
    if (prTarget === "triggering") {
        if (context.eventName === "pull_request" || context.eventName === "pull_request_review" || context.eventName === "pull_request_review_comment") {
            prNumber = context.payload.pull_request?.number;
        }
    }
    else {
        prNumber = parseInt(prTarget, 10);
    }
    if (!prNumber) {
        core.setFailed("Could not determine pull request number for review comments");
        return [];
    }
    const commitSha = context.payload.pull_request?.head?.sha || context.sha;
    if (!commitSha) {
        core.setFailed("Could not determine commit SHA for review comments");
        return [];
    }
    const createdComments = [];
    for (let i = 0; i < reviewCommentItems.length; i++) {
        const commentItem = reviewCommentItems[i];
        core.info(`Processing create-pull-request-review-comment item ${i + 1}/${reviewCommentItems.length}: path=${commentItem.path}`);
        const path = commentItem.path;
        const line = commentItem.line;
        const side = commentItem.side || "RIGHT";
        const body = commentItem.body;
        const startLine = commentItem.start_line;
        const startSide = commentItem.start_side;
        if (!path || !line || !body) {
            core.warning(`Skipping review comment ${i + 1}: missing required fields (path, line, or body)`);
            continue;
        }
        const commentPayload = {
            owner: context.repo.owner,
            repo: context.repo.repo,
            pull_number: prNumber,
            body: body.trim(),
            commit_id: commitSha,
            path: path,
            line: parseInt(line.toString(), 10),
            side: side,
        };
        if (startLine) {
            commentPayload.start_line = parseInt(startLine.toString(), 10);
        }
        if (startSide) {
            commentPayload.start_side = startSide;
        }
        core.info(`Creating review comment on PR #${prNumber}`);
        core.info(`File: ${path}, Line: ${line}, Side: ${side}`);
        core.info(`Comment length: ${body.length}`);
        try {
            const { data: comment } = await github.rest.pulls.createReviewComment(commentPayload);
            core.info("Created review comment #" + comment.id + ": " + comment.html_url);
            createdComments.push({
                id: comment.id,
                html_url: comment.html_url,
                path: comment.path,
                line: comment.line || 0,
            });
            if (i === reviewCommentItems.length - 1) {
                core.setOutput("comment_id", comment.id);
                core.setOutput("comment_url", comment.html_url);
                core.setOutput("pr_number", prNumber);
            }
        }
        catch (error) {
            core.error(`✗ Failed to create review comment: ${error instanceof Error ? error.message : String(error)}`);
            throw error;
        }
    }
    if (createdComments.length > 0) {
        let summaryContent = "\n\n## GitHub PR Review Comments\n";
        for (const comment of createdComments) {
            summaryContent += `- Comment #${comment.id}: [${comment.path}:${comment.line}](${comment.html_url})\n`;
        }
        await core.summary.addRaw(summaryContent).write();
    }
    core.info(`Successfully created ${createdComments.length} review comment(s)`);
    return createdComments;
}
(async () => {
    await createPrReviewCommentMain();
})();

