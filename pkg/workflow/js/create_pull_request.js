const fs = require("fs");
const crypto = require("crypto");
async function createPullRequestMain() {
    const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";
    const workflowId = process.env.GITHUB_AW_WORKFLOW_ID;
    if (!workflowId) {
        throw new Error("GITHUB_AW_WORKFLOW_ID environment variable is required");
    }
    const baseBranch = process.env.GITHUB_AW_BASE_BRANCH;
    if (!baseBranch) {
        throw new Error("GITHUB_AW_BASE_BRANCH environment variable is required");
    }
    const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT || "";
    if (outputContent.trim() === "") {
        core.info("Agent output content is empty");
    }
    const ifNoChanges = process.env.GITHUB_AW_PR_IF_NO_CHANGES || "warn";
    if (!fs.existsSync("/tmp/aw.patch")) {
        const message = "No patch file found - cannot create pull request without changes";
        if (isStaged) {
            let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
            summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";
            summaryContent += `**Status:** ⚠️ No patch file found\n\n`;
            summaryContent += `**Message:** ${message}\n\n`;
            await core.summary.addRaw(summaryContent).write();
            core.info("📝 Pull request creation preview written to step summary (no patch file)");
            return;
        }
        switch (ifNoChanges) {
            case "error":
                throw new Error(message);
            case "ignore":
                return;
            case "warn":
            default:
                core.warning(message);
                return;
        }
    }
    const patchContent = fs.readFileSync("/tmp/aw.patch", "utf8");
    if (patchContent.includes("Failed to generate patch")) {
        const message = "Patch file contains error message - cannot create pull request without changes";
        if (isStaged) {
            let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
            summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";
            summaryContent += `**Status:** ⚠️ Patch file contains error\n\n`;
            summaryContent += `**Message:** ${message}\n\n`;
            await core.summary.addRaw(summaryContent).write();
            core.info("📝 Pull request creation preview written to step summary (patch error)");
            return;
        }
        switch (ifNoChanges) {
            case "error":
                throw new Error(message);
            case "ignore":
                return;
            case "warn":
            default:
                core.warning(message);
                return;
        }
    }
    const isEmpty = !patchContent || !patchContent.trim();
    if (!isEmpty) {
        const maxSizeKb = parseInt(process.env.GITHUB_AW_MAX_PATCH_SIZE || "1024", 10);
        const { Buffer } = require("buffer");
        const patchSizeBytes = Buffer.byteLength(patchContent, "utf8");
        const patchSizeKb = Math.ceil(patchSizeBytes / 1024);
        core.info(`Patch size: ${patchSizeKb} KB (maximum allowed: ${maxSizeKb} KB)`);
        if (patchSizeKb > maxSizeKb) {
            const message = `Patch size (${patchSizeKb} KB) exceeds maximum allowed size (${maxSizeKb} KB)`;
            if (isStaged) {
                let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
                summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";
                summaryContent += `**Status:** ❌ Patch size exceeded\n\n`;
                summaryContent += `**Message:** ${message}\n\n`;
                await core.summary.addRaw(summaryContent).write();
                core.info("📝 Pull request creation preview written to step summary (patch size error)");
                return;
            }
            throw new Error(message);
        }
        core.info("Patch size validation passed");
    }
    if (isEmpty && !isStaged) {
        const message = "Patch file is empty - no changes to apply (noop operation)";
        switch (ifNoChanges) {
            case "error":
                throw new Error("No changes to push - failing as configured by if-no-changes: error");
            case "ignore":
                return;
            case "warn":
            default:
                core.warning(message);
                return;
        }
    }
    core.debug(`Agent output content length: ${outputContent.length}`);
    if (!isEmpty) {
        core.info("Patch content validation passed");
    }
    else {
        core.info("Patch file is empty - processing noop operation");
    }
    let validatedOutput;
    try {
        validatedOutput = JSON.parse(outputContent);
    }
    catch (error) {
        core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
        return;
    }
    if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
        core.warning("No valid items found in agent output");
        return;
    }
    const pullRequestItem = validatedOutput.items.find(item => item.type === "create-pull-request");
    if (!pullRequestItem) {
        core.warning("No create-pull-request item found in agent output");
        return;
    }
    core.debug(`Found create-pull-request item: title="${pullRequestItem.title}", bodyLength=${pullRequestItem.body.length}`);
    if (isStaged) {
        let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
        summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";
        summaryContent += `**Title:** ${pullRequestItem.title || "No title provided"}\n\n`;
        summaryContent += `**Branch:** ${pullRequestItem.branch || "auto-generated"}\n\n`;
        summaryContent += `**Base:** ${baseBranch}\n\n`;
        if (pullRequestItem.body) {
            summaryContent += `**Body:**\n${pullRequestItem.body}\n\n`;
        }
        if (fs.existsSync("/tmp/aw.patch")) {
            const patchStats = fs.readFileSync("/tmp/aw.patch", "utf8");
            if (patchStats.trim()) {
                summaryContent += `**Changes:** Patch file exists with ${patchStats.split("\n").length} lines\n\n`;
                summaryContent += `<details><summary>Show patch preview</summary>\n\n\`\`\`diff\n${patchStats.slice(0, 2000)}${patchStats.length > 2000 ? "\n... (truncated)" : ""}\n\`\`\`\n\n</details>\n\n`;
            }
            else {
                summaryContent += `**Changes:** No changes (empty patch)\n\n`;
            }
        }
        await core.summary.addRaw(summaryContent).write();
        core.info("📝 Pull request creation preview written to step summary");
        return;
    }
    let title = pullRequestItem.title.trim();
    let bodyLines = pullRequestItem.body.split("\n");
    let branchName = pullRequestItem.branch ? pullRequestItem.branch.trim() : null;
    if (!title) {
        title = "Agent Output";
    }
    const titlePrefix = process.env.GITHUB_AW_PR_TITLE_PREFIX;
    if (titlePrefix && !title.startsWith(titlePrefix)) {
        title = titlePrefix + title;
    }
    const runId = context.runId;
    const runUrl = context.payload.repository
        ? `${context.payload.repository.html_url}/actions/runs/${runId}`
        : `https://github.com/actions/runs/${runId}`;
    bodyLines.push(``, ``, `> Generated by Agentic Workflow [Run](${runUrl})`, "");
    const body = bodyLines.join("\n").trim();
    const labelsEnv = process.env.GITHUB_AW_PR_LABELS;
    const labels = labelsEnv
        ? labelsEnv
            .split(",")
            .map(label => label.trim())
            .filter(label => label)
        : [];
    const draftEnv = process.env.GITHUB_AW_PR_DRAFT;
    const draft = draftEnv ? draftEnv.toLowerCase() === "true" : true;
    core.info(`Creating pull request with title: ${title}`);
    core.debug(`Labels: ${JSON.stringify(labels)}`);
    core.debug(`Draft: ${draft}`);
    core.debug(`Body length: ${body.length}`);
    const randomHex = crypto.randomBytes(8).toString("hex");
    if (!branchName) {
        core.debug("No branch name provided in JSONL, generating unique branch name");
        branchName = `${workflowId}-${randomHex}`;
    }
    else {
        branchName = `${branchName}-${randomHex}`;
        core.debug(`Using branch name from JSONL with added salt: ${branchName}`);
    }
    core.info(`Generated branch name: ${branchName}`);
    core.debug(`Base branch: ${baseBranch}`);
    core.debug(`Fetching latest changes and checking out base branch: ${baseBranch}`);
    await exec.exec("git fetch origin");
    await exec.exec(`git checkout ${baseBranch}`);
    core.debug(`Branch should not exist locally, creating new branch from base: ${branchName}`);
    await exec.exec(`git checkout -b ${branchName}`);
    core.info(`Created new branch from base: ${branchName}`);
    if (!isEmpty) {
        core.info("Applying patch...");
        await exec.exec("git am /tmp/aw.patch");
        core.info("Patch applied successfully");
        await exec.exec(`git push origin ${branchName}`);
        core.info("Changes pushed to branch");
    }
    else {
        core.info("Skipping patch application (empty patch)");
        const message = "No changes to apply - noop operation completed successfully";
        switch (ifNoChanges) {
            case "error":
                throw new Error("No changes to apply - failing as configured by if-no-changes: error");
            case "ignore":
                return;
            case "warn":
            default:
                core.warning(message);
                return;
        }
    }
    const { data: pullRequest } = await github.rest.pulls.create({
        owner: context.repo.owner,
        repo: context.repo.repo,
        title: title,
        body: body,
        head: branchName,
        base: baseBranch,
        draft: draft,
    });
    core.info(`Created pull request #${pullRequest.number}: ${pullRequest.html_url}`);
    if (labels.length > 0) {
        await github.rest.issues.addLabels({
            owner: context.repo.owner,
            repo: context.repo.repo,
            issue_number: pullRequest.number,
            labels: labels,
        });
        core.info(`Added labels to pull request: ${JSON.stringify(labels)}`);
    }
    core.setOutput("pull_request_number", pullRequest.number);
    core.setOutput("pull_request_url", pullRequest.html_url);
    core.setOutput("branch_name", branchName);
    await core.summary
        .addRaw(`

## Pull Request
- **Pull Request**: [#${pullRequest.number}](${pullRequest.html_url})
- **Branch**: \`${branchName}\`
- **Base Branch**: \`${baseBranch}\`
`)
        .write();
}
(async () => {
    await createPullRequestMain();
})();
