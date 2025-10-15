/** @type {typeof import("fs")} */
const fs = require("fs");
/** @type {typeof import("crypto")} */
const crypto = require("crypto");

/**
 * Generate a patch preview with max 500 lines and 2000 chars for issue body
 * @param {string} patchContent - The full patch content
 * @returns {string} Formatted patch preview
 */
function generatePatchPreview(patchContent) {
  if (!patchContent || !patchContent.trim()) {
    return "";
  }

  const lines = patchContent.split("\n");
  const maxLines = 500;
  const maxChars = 2000;

  // Apply line limit first
  let preview = lines.length <= maxLines ? patchContent : lines.slice(0, maxLines).join("\n");
  const lineTruncated = lines.length > maxLines;

  // Apply character limit
  const charTruncated = preview.length > maxChars;
  if (charTruncated) {
    preview = preview.slice(0, maxChars);
  }

  const truncated = lineTruncated || charTruncated;
  const summary = truncated
    ? `Show patch preview (${Math.min(maxLines, lines.length)} of ${lines.length} lines)`
    : `Show patch (${lines.length} lines)`;

  return `\n\n<details><summary>${summary}</summary>\n\n\`\`\`diff\n${preview}${truncated ? "\n... (truncated)" : ""}\n\`\`\`\n\n</details>`;
}

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";

  // Environment validation - fail early if required variables are missing
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

  // Check if patch file exists and has valid content
  if (!fs.existsSync("/tmp/gh-aw/aw.patch")) {
    const message = "No patch file found - cannot create pull request without changes";

    // If in staged mode, still show preview
    if (isStaged) {
      let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
      summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";
      summaryContent += `**Status:** ⚠️ No patch file found\n\n`;
      summaryContent += `**Message:** ${message}\n\n`;

      // Write to step summary
      await core.summary.addRaw(summaryContent).write();
      core.info("📝 Pull request creation preview written to step summary (no patch file)");
      return;
    }

    switch (ifNoChanges) {
      case "error":
        throw new Error(message);
      case "ignore":
        // Silent success - no console output
        return;
      case "warn":
      default:
        core.warning(message);
        return;
    }
  }

  const patchContent = fs.readFileSync("/tmp/gh-aw/aw.patch", "utf8");

  // Check for actual error conditions (but allow empty patches as valid noop)
  if (patchContent.includes("Failed to generate patch")) {
    const message = "Patch file contains error message - cannot create pull request without changes";

    // If in staged mode, still show preview
    if (isStaged) {
      let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
      summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";
      summaryContent += `**Status:** ⚠️ Patch file contains error\n\n`;
      summaryContent += `**Message:** ${message}\n\n`;

      // Write to step summary
      await core.summary.addRaw(summaryContent).write();
      core.info("📝 Pull request creation preview written to step summary (patch error)");
      return;
    }

    switch (ifNoChanges) {
      case "error":
        throw new Error(message);
      case "ignore":
        // Silent success - no console output
        return;
      case "warn":
      default:
        core.warning(message);
        return;
    }
  }

  // Validate patch size (unless empty)
  const isEmpty = !patchContent || !patchContent.trim();
  if (!isEmpty) {
    // Get maximum patch size from environment (default: 1MB = 1024 KB)
    const maxSizeKb = parseInt(process.env.GITHUB_AW_MAX_PATCH_SIZE || "1024", 10);
    const patchSizeBytes = Buffer.byteLength(patchContent, "utf8");
    const patchSizeKb = Math.ceil(patchSizeBytes / 1024);

    core.info(`Patch size: ${patchSizeKb} KB (maximum allowed: ${maxSizeKb} KB)`);

    if (patchSizeKb > maxSizeKb) {
      const message = `Patch size (${patchSizeKb} KB) exceeds maximum allowed size (${maxSizeKb} KB)`;

      // If in staged mode, still show preview with error
      if (isStaged) {
        let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
        summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";
        summaryContent += `**Status:** ❌ Patch size exceeded\n\n`;
        summaryContent += `**Message:** ${message}\n\n`;

        // Write to step summary
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
        // Silent success - no console output
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
  } else {
    core.info("Patch file is empty - processing noop operation");
  }

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.warning("No valid items found in agent output");
    return;
  }

  // Find the create-pull-request item
  const pullRequestItem = validatedOutput.items.find(/** @param {any} item */ item => item.type === "create_pull_request");
  if (!pullRequestItem) {
    core.warning("No create-pull-request item found in agent output");
    return;
  }

  core.debug(`Found create-pull-request item: title="${pullRequestItem.title}", bodyLength=${pullRequestItem.body.length}`);

  // If in staged mode, emit step summary instead of creating PR
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
    summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";

    summaryContent += `**Title:** ${pullRequestItem.title || "No title provided"}\n\n`;
    summaryContent += `**Branch:** ${pullRequestItem.branch || "auto-generated"}\n\n`;
    summaryContent += `**Base:** ${baseBranch}\n\n`;

    if (pullRequestItem.body) {
      summaryContent += `**Body:**\n${pullRequestItem.body}\n\n`;
    }

    if (fs.existsSync("/tmp/gh-aw/aw.patch")) {
      const patchStats = fs.readFileSync("/tmp/gh-aw/aw.patch", "utf8");
      if (patchStats.trim()) {
        summaryContent += `**Changes:** Patch file exists with ${patchStats.split("\n").length} lines\n\n`;
        summaryContent += `<details><summary>Show patch preview</summary>\n\n\`\`\`diff\n${patchStats.slice(0, 2000)}${patchStats.length > 2000 ? "\n... (truncated)" : ""}\n\`\`\`\n\n</details>\n\n`;
      } else {
        summaryContent += `**Changes:** No changes (empty patch)\n\n`;
      }
    }

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Pull request creation preview written to step summary");
    return;
  }

  // Extract title, body, and branch from the JSON item
  let title = pullRequestItem.title.trim();
  let bodyLines = pullRequestItem.body.split("\n");
  let branchName = pullRequestItem.branch ? pullRequestItem.branch.trim() : null;

  // If no title was found, use a default
  if (!title) {
    title = "Agent Output";
  }

  // Apply title prefix if provided via environment variable
  const titlePrefix = process.env.GITHUB_AW_PR_TITLE_PREFIX;
  if (titlePrefix && !title.startsWith(titlePrefix)) {
    title = titlePrefix + title;
  }

  // Determine triggering context for footer
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";
  const isPRContext =
    context.eventName === "pull_request" ||
    context.eventName === "pull_request_review" ||
    context.eventName === "pull_request_review_comment";
  const isDiscussionContext = context.eventName === "discussion" || context.eventName === "discussion_comment";

  let triggeringReference = "";
  if (isIssueContext && context.payload.issue) {
    triggeringReference = ` for #${context.payload.issue.number}`;
  } else if (isPRContext && context.payload.pull_request) {
    triggeringReference = ` for #${context.payload.pull_request.number}`;
  } else if (isDiscussionContext && context.payload.discussion) {
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    const repoUrl = context.payload.repository
      ? context.payload.repository.html_url
      : `${githubServer}/${context.repo.owner}/${context.repo.repo}`;
    const discussionUrl = `${repoUrl}/discussions/${context.payload.discussion.number}`;
    triggeringReference = ` for [discussion #${context.payload.discussion.number}](${discussionUrl})`;
  }

  // Add AI disclaimer with workflow name and run url
  const workflowName = process.env.GITHUB_AW_WORKFLOW_NAME || "Workflow";
  const runId = context.runId;
  const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
  const runUrl = context.payload.repository
    ? `${context.payload.repository.html_url}/actions/runs/${runId}`
    : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;
  bodyLines.push(``, ``, `> AI generated by [${workflowName}](${runUrl})${triggeringReference}`, "");

  // Prepare the body content
  const body = bodyLines.join("\n").trim();

  // Parse labels from environment variable (comma-separated string)
  const labelsEnv = process.env.GITHUB_AW_PR_LABELS;
  const labels = labelsEnv
    ? labelsEnv
        .split(",")
        .map(/** @param {string} label */ label => label.trim())
        .filter(/** @param {string} label */ label => label)
    : [];

  // Parse draft setting from environment variable (defaults to true)
  const draftEnv = process.env.GITHUB_AW_PR_DRAFT;
  const draft = draftEnv ? draftEnv.toLowerCase() === "true" : true;

  core.info(`Creating pull request with title: ${title}`);
  core.debug(`Labels: ${JSON.stringify(labels)}`);
  core.debug(`Draft: ${draft}`);
  core.debug(`Body length: ${body.length}`);

  const randomHex = crypto.randomBytes(8).toString("hex");
  // Use branch name from JSONL if provided, otherwise generate unique branch name
  if (!branchName) {
    core.debug("No branch name provided in JSONL, generating unique branch name");
    // Generate unique branch name using cryptographic random hex
    branchName = `${workflowId}-${randomHex}`;
  } else {
    branchName = `${branchName}-${randomHex}`;
    core.debug(`Using branch name from JSONL with added salt: ${branchName}`);
  }

  core.info(`Generated branch name: ${branchName}`);
  core.debug(`Base branch: ${baseBranch}`);

  // Create a new branch using git CLI, ensuring it's based on the correct base branch

  // First, fetch latest changes and checkout the base branch
  core.debug(`Fetching latest changes and checking out base branch: ${baseBranch}`);
  await exec.exec("git fetch origin");
  await exec.exec(`git checkout ${baseBranch}`);

  // Handle branch creation/checkout
  core.debug(`Branch should not exist locally, creating new branch from base: ${branchName}`);
  await exec.exec(`git checkout -b ${branchName}`);
  core.info(`Created new branch from base: ${branchName}`);

  // Apply the patch using git CLI (skip if empty)
  if (!isEmpty) {
    core.info("Applying patch...");
    // Patches are created with git format-patch, so use git am to apply them
    await exec.exec("git am /tmp/gh-aw/aw.patch");
    core.info("Patch applied successfully");

    // Push the applied commits to the branch (with fallback to issue creation on failure)
    try {
      // Check if remote branch already exists (optional precheck)
      let remoteBranchExists = false;
      try {
        const { stdout } = await exec.getExecOutput(`git ls-remote --heads origin ${branchName}`);
        if (stdout.trim()) {
          remoteBranchExists = true;
        }
      } catch (checkError) {
        core.debug(`Remote branch check failed (non-fatal): ${checkError instanceof Error ? checkError.message : String(checkError)}`);
      }

      if (remoteBranchExists) {
        core.warning(`Remote branch ${branchName} already exists - appending random suffix`);
        const extraHex = crypto.randomBytes(4).toString("hex");
        const oldBranch = branchName;
        branchName = `${branchName}-${extraHex}`;
        // Rename local branch
        await exec.exec(`git branch -m ${oldBranch} ${branchName}`);
        core.info(`Renamed branch to ${branchName}`);
      }

      await exec.exec(`git push origin ${branchName}`);
      core.info("Changes pushed to branch");
    } catch (pushError) {
      // Push failed - create fallback issue instead of PR
      core.error(`Git push failed: ${pushError instanceof Error ? pushError.message : String(pushError)}`);
      core.warning("Git push operation failed - creating fallback issue instead of pull request");

      const runId = context.runId;
      const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
      const runUrl = context.payload.repository
        ? `${context.payload.repository.html_url}/actions/runs/${runId}`
        : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

      // Read patch content for preview
      let patchPreview = "";
      if (fs.existsSync("/tmp/gh-aw/aw.patch")) {
        const patchContent = fs.readFileSync("/tmp/gh-aw/aw.patch", "utf8");
        patchPreview = generatePatchPreview(patchContent);
      }

      const fallbackBody = `${body}

---

> [!NOTE]
> This was originally intended as a pull request, but the git push operation failed.
>
> **Workflow Run:** [View run details and download patch artifact](${runUrl})
>
> The patch file is available as an artifact (\`aw.patch\`) in the workflow run linked above.

To apply the patch locally:

\`\`\`sh
# Download the artifact from the workflow run ${runUrl}
# (Use GitHub MCP tools if gh CLI is not available)
gh run download ${runId} -n aw.patch

# Apply the patch
git am aw.patch
\`\`\`
${patchPreview}`;

      try {
        const { data: issue } = await github.rest.issues.create({
          owner: context.repo.owner,
          repo: context.repo.repo,
          title: title,
          body: fallbackBody,
          labels: labels,
        });

        core.info(`Created fallback issue #${issue.number}: ${issue.html_url}`);

        // Set outputs for push failure fallback
        core.setOutput("issue_number", issue.number);
        core.setOutput("issue_url", issue.html_url);
        core.setOutput("branch_name", branchName);
        core.setOutput("fallback_used", "true");
        core.setOutput("push_failed", "true");

        // Write summary to GitHub Actions summary
        await core.summary
          .addRaw(
            `

## Push Failure Fallback
- **Push Error:** ${pushError instanceof Error ? pushError.message : String(pushError)}
- **Fallback Issue:** [#${issue.number}](${issue.html_url})
- **Patch Artifact:** Available in workflow run artifacts
- **Note:** Push failed, created issue as fallback
`
          )
          .write();

        return;
      } catch (issueError) {
        core.setFailed(
          `Failed to push and failed to create fallback issue. Push error: ${pushError instanceof Error ? pushError.message : String(pushError)}. Issue error: ${issueError instanceof Error ? issueError.message : String(issueError)}`
        );
        return;
      }
    }
  } else {
    core.info("Skipping patch application (empty patch)");

    // For empty patches, handle if-no-changes configuration
    const message = "No changes to apply - noop operation completed successfully";

    switch (ifNoChanges) {
      case "error":
        throw new Error("No changes to apply - failing as configured by if-no-changes: error");
      case "ignore":
        // Silent success - no console output
        return;
      case "warn":
      default:
        core.warning(message);
        return;
    }
  }

  // Try to create the pull request, with fallback to issue creation
  try {
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

    // Add labels if specified
    if (labels.length > 0) {
      await github.rest.issues.addLabels({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: pullRequest.number,
        labels: labels,
      });
      core.info(`Added labels to pull request: ${JSON.stringify(labels)}`);
    }

    // Set output for other jobs to use
    core.setOutput("pull_request_number", pullRequest.number);
    core.setOutput("pull_request_url", pullRequest.html_url);
    core.setOutput("branch_name", branchName);

    // Write summary to GitHub Actions summary
    await core.summary
      .addRaw(
        `

## Pull Request
- **Pull Request**: [#${pullRequest.number}](${pullRequest.html_url})
- **Branch**: \`${branchName}\`
- **Base Branch**: \`${baseBranch}\`
`
      )
      .write();
  } catch (prError) {
    core.warning(`Failed to create pull request: ${prError instanceof Error ? prError.message : String(prError)}`);
    core.info("Falling back to creating an issue instead");

    // Create issue as fallback with enhanced body content
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    const branchUrl = context.payload.repository
      ? `${context.payload.repository.html_url}/tree/${branchName}`
      : `${githubServer}/${context.repo.owner}/${context.repo.repo}/tree/${branchName}`;

    // Read patch content for preview
    let patchPreview = "";
    if (fs.existsSync("/tmp/gh-aw/aw.patch")) {
      const patchContent = fs.readFileSync("/tmp/gh-aw/aw.patch", "utf8");
      patchPreview = generatePatchPreview(patchContent);
    }

    const fallbackBody = `${body}

---

**Note:** This was originally intended as a pull request, but PR creation failed. The changes have been pushed to the branch [\`${branchName}\`](${branchUrl}).

**Original error:** ${prError instanceof Error ? prError.message : String(prError)}

You can manually create a pull request from the branch if needed.${patchPreview}`;

    try {
      const { data: issue } = await github.rest.issues.create({
        owner: context.repo.owner,
        repo: context.repo.repo,
        title: title,
        body: fallbackBody,
        labels: labels,
      });

      core.info(`Created fallback issue #${issue.number}: ${issue.html_url}`);

      // Set output for other jobs to use (issue instead of PR)
      core.setOutput("issue_number", issue.number);
      core.setOutput("issue_url", issue.html_url);
      core.setOutput("branch_name", branchName);
      core.setOutput("fallback_used", "true");

      // Write summary to GitHub Actions summary
      await core.summary
        .addRaw(
          `

## Fallback Issue Created
- **Issue**: [#${issue.number}](${issue.html_url})
- **Branch**: [\`${branchName}\`](${branchUrl})
- **Base Branch**: \`${baseBranch}\`
- **Note**: Pull request creation failed, created issue as fallback
`
        )
        .write();
    } catch (issueError) {
      core.setFailed(
        `Failed to create both pull request and fallback issue. PR error: ${prError instanceof Error ? prError.message : String(prError)}. Issue error: ${issueError instanceof Error ? issueError.message : String(issueError)}`
      );
      return;
    }
  }
}
await main();
