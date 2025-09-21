async function updateIssueMain() {
    const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";
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
    const updateItems = validatedOutput.items.filter(item => item.type === "update-issue");
    if (updateItems.length === 0) {
        core.info("No update-issue items found in agent output");
        return [];
    }
    core.info(`Found ${updateItems.length} update-issue item(s)`);
    if (isStaged) {
        let summaryContent = "## 🎭 Staged Mode: Update Issues Preview\n\n";
        summaryContent += "The following issue updates would be applied if staged mode was disabled:\n\n";
        for (let i = 0; i < updateItems.length; i++) {
            const item = updateItems[i];
            summaryContent += `### Issue Update ${i + 1}\n`;
            if (item.issue_number) {
                summaryContent += `**Target Issue:** #${item.issue_number}\n\n`;
            }
            else {
                summaryContent += `**Target:** Current issue/PR\n\n`;
            }
            if (item.title) {
                summaryContent += `**New Title:** ${item.title}\n\n`;
            }
            if (item.body) {
                summaryContent += `**New Body:**\n${item.body}\n\n`;
            }
            if (item.state) {
                summaryContent += `**New State:** ${item.state}\n\n`;
            }
            summaryContent += "---\n\n";
        }
        await core.summary.addRaw(summaryContent).write();
        core.info("📝 Issue update preview written to step summary");
        return [];
    }
    const updateTarget = process.env.GITHUB_AW_UPDATE_TARGET || "triggering";
    core.info(`Update target configuration: ${updateTarget}`);
    const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";
    const isPRContext = context.eventName === "pull_request" ||
        context.eventName === "pull_request_review" ||
        context.eventName === "pull_request_review_comment";
    if (updateTarget === "triggering" && !isIssueContext && !isPRContext) {
        core.info('Target is "triggering" but not running in issue or pull request context, skipping issue updates');
        return [];
    }
    const updatedIssues = [];
    for (let i = 0; i < updateItems.length; i++) {
        const updateItem = updateItems[i];
        core.info(`Processing update-issue item ${i + 1}/${updateItems.length}`);
        let issueNumber;
        if (updateTarget === "*") {
            if (updateItem.issue_number) {
                issueNumber = parseInt(updateItem.issue_number, 10);
                if (isNaN(issueNumber) || issueNumber <= 0) {
                    core.info(`Invalid issue number specified: ${updateItem.issue_number}`);
                    continue;
                }
            }
            else {
                core.info('Target is "*" but no issue_number specified in update item');
                continue;
            }
        }
        else if (updateTarget && updateTarget !== "triggering") {
            issueNumber = parseInt(updateTarget, 10);
            if (isNaN(issueNumber) || issueNumber <= 0) {
                core.info(`Invalid issue number in target configuration: ${updateTarget}`);
                continue;
            }
        }
        else {
            if (isIssueContext) {
                if (context.payload.issue) {
                    issueNumber = context.payload.issue.number;
                }
                else {
                    core.info("Issue context detected but no issue found in payload");
                    continue;
                }
            }
            else if (isPRContext) {
                if (context.payload.pull_request) {
                    issueNumber = context.payload.pull_request.number;
                }
                else {
                    core.info("Pull request context detected but no pull request found in payload");
                    continue;
                }
            }
        }
        if (!issueNumber) {
            core.info("Could not determine issue or pull request number");
            continue;
        }
        const updatePayload = {
            owner: context.repo.owner,
            repo: context.repo.repo,
            issue_number: issueNumber,
        };
        if (updateItem.title) {
            updatePayload.title = updateItem.title.trim();
        }
        if (updateItem.body) {
            updatePayload.body = updateItem.body.trim();
        }
        if (updateItem.state) {
            updatePayload.state = updateItem.state;
        }
        core.info(`Updating issue #${issueNumber}`);
        if (updatePayload.title) {
            core.info(`New title: ${updatePayload.title}`);
        }
        if (updatePayload.body) {
            core.info(`New body length: ${updatePayload.body.length}`);
        }
        if (updatePayload.state) {
            core.info(`New state: ${updatePayload.state}`);
        }
        try {
            const { data: issue } = await github.rest.issues.update(updatePayload);
            core.info("Updated issue #" + issue.number + ": " + issue.html_url);
            updatedIssues.push(issue);
            if (i === updateItems.length - 1) {
                core.setOutput("issue_number", issue.number);
                core.setOutput("issue_url", issue.html_url);
            }
        }
        catch (error) {
            core.error(`✗ Failed to update issue #${issueNumber}: ${error instanceof Error ? error.message : String(error)}`);
            throw error;
        }
    }
    if (updatedIssues.length > 0) {
        let summaryContent = "\n\n## GitHub Issues Updated\n";
        for (const issue of updatedIssues) {
            summaryContent += `- Issue #${issue.number}: [${issue.title}](${issue.html_url})\n`;
        }
        await core.summary.addRaw(summaryContent).write();
    }
    core.info(`Successfully updated ${updatedIssues.length} issue(s)`);
    return updatedIssues;
}
(async () => {
    await updateIssueMain();
})();

