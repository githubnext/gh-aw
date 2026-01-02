// @ts-check
/// <reference types="@actions/github-script" />

/** @type {typeof import("fs")} */
const fs = require("fs");
/** @type {typeof import("crypto")} */
const crypto = require("crypto");
const { updateActivationComment } = require("./update_activation_comment.cjs");
const { getTrackerID } = require("./get_tracker_id.cjs");
const { addExpirationComment } = require("./expiration_helpers.cjs");
const { removeDuplicateTitleFromDescription } = require("./remove_duplicate_title.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { replaceTemporaryIdReferences } = require("./temporary_id.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "create_pull_request";

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
  const summary = truncated ? `Show patch preview (${Math.min(maxLines, lines.length)} of ${lines.length} lines)` : `Show patch (${lines.length} lines)`;

  return `\n\n<details><summary>${summary}</summary>\n\n\`\`\`diff\n${preview}${truncated ? "\n... (truncated)" : ""}\n\`\`\`\n\n</details>`;
}

/**
 * Main handler factory for create_pull_request
 * Returns a message handler function that processes individual create_pull_request messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const titlePrefix = config.title_prefix || "";
  const envLabels = config.labels ? (Array.isArray(config.labels) ? config.labels : config.labels.split(",")).map(label => String(label).trim()).filter(label => label) : [];
  const draftDefault = config.draft !== undefined ? config.draft : true;
  const ifNoChanges = config.if_no_changes || "warn";
  const allowEmpty = config.allow_empty || false;
  const expiresHours = config.expires ? parseInt(String(config.expires), 10) : 0;
  const maxCount = config.max || 1; // PRs are typically limited to 1

  // Environment validation - fail early if required variables are missing
  const workflowId = process.env.GH_AW_WORKFLOW_ID;
  if (!workflowId) {
    throw new Error("GH_AW_WORKFLOW_ID environment variable is required");
  }

  const baseBranch = process.env.GH_AW_BASE_BRANCH;
  if (!baseBranch) {
    throw new Error("GH_AW_BASE_BRANCH environment variable is required");
  }

  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  core.info(`Base branch: ${baseBranch}`);
  if (envLabels.length > 0) {
    core.info(`Default labels: ${envLabels.join(", ")}`);
  }
  if (titlePrefix) {
    core.info(`Title prefix: ${titlePrefix}`);
  }
  core.info(`Draft default: ${draftDefault}`);
  core.info(`If no changes: ${ifNoChanges}`);
  core.info(`Allow empty: ${allowEmpty}`);
  if (expiresHours > 0) {
    core.info(`Pull requests expire after: ${expiresHours} hours`);
  }
  core.info(`Max count: ${maxCount}`);

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function that processes a single create_pull_request message
   * @param {Object} message - The create_pull_request message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status and PR details
   */
  return async function handleCreatePullRequest(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping create_pull_request: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const pullRequestItem = message;

    const pullRequestItem = message;

    core.info(`Processing create_pull_request: title=${pullRequestItem.title}, bodyLength=${pullRequestItem.body?.length || 0}`);

    // Read patch file if it exists
    let patchContent = "";
    let isEmpty = true;

    if (!fs.existsSync("/tmp/gh-aw/aw.patch")) {
      // If allow-empty is enabled, we can proceed without a patch file
      if (allowEmpty) {
        core.info("No patch file found, but allow-empty is enabled - will create empty PR");
      } else {
        const error = "No patch file found - cannot create pull request without changes";

        // If in staged mode, still show preview
        if (isStaged) {
          let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
          summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";
          summaryContent += `**Status:** ⚠️ No patch file found\n\n`;
          summaryContent += `**Message:** ${error}\n\n`;

          // Write to step summary
          await core.summary.addRaw(summaryContent).write();
          core.info("📝 Pull request creation preview written to step summary (no patch file)");
          return { success: false, error };
        }

        switch (ifNoChanges) {
          case "error":
            return { success: false, error };
          case "ignore":
            return { success: false, error, skipped: true };
          case "warn":
          default:
            core.warning(error);
            return { success: false, error };
        }
      }
    } else {
      patchContent = fs.readFileSync("/tmp/gh-aw/aw.patch", "utf8");
      isEmpty = !patchContent || !patchContent.trim();
    }

    // Check for actual error conditions (but allow empty patches as valid noop)
    if (patchContent.includes("Failed to generate patch")) {
      // If allow-empty is enabled, ignore patch errors and proceed
      if (allowEmpty) {
        core.info("Patch file contains error, but allow-empty is enabled - will create empty PR");
        patchContent = "";
        isEmpty = true;
      } else {
        const error = "Patch file contains error message - cannot create pull request without changes";

        // If in staged mode, still show preview
        if (isStaged) {
          let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
          summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";
          summaryContent += `**Status:** ⚠️ Patch file contains error\n\n`;
          summaryContent += `**Message:** ${error}\n\n`;

          // Write to step summary
          await core.summary.addRaw(summaryContent).write();
          core.info("📝 Pull request creation preview written to step summary (patch error)");
          return { success: false, error };
        }

        switch (ifNoChanges) {
          case "error":
            return { success: false, error };
          case "ignore":
            return { success: false, error, skipped: true };
          case "warn":
          default:
            core.warning(error);
            return { success: false, error };
        }
      }
    }

    // Validate patch size (unless empty)
    if (!isEmpty) {
      // Get maximum patch size from environment (default: 1MB = 1024 KB)
      const maxSizeKb = parseInt(process.env.GH_AW_MAX_PATCH_SIZE || "1024", 10);
      const patchSizeBytes = Buffer.byteLength(patchContent, "utf8");
      const patchSizeKb = Math.ceil(patchSizeBytes / 1024);

      core.info(`Patch size: ${patchSizeKb} KB (maximum allowed: ${maxSizeKb} KB)`);

      if (patchSizeKb > maxSizeKb) {
        const error = `Patch size (${patchSizeKb} KB) exceeds maximum allowed size (${maxSizeKb} KB)`;

        // If in staged mode, still show preview with error
        if (isStaged) {
          let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
          summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";
          summaryContent += `**Status:** ❌ Patch size exceeded\n\n`;
          summaryContent += `**Message:** ${error}\n\n`;

          // Write to step summary
          await core.summary.addRaw(summaryContent).write();
          core.info("📝 Pull request creation preview written to step summary (patch size error)");
          return { success: false, error };
        }

        return { success: false, error };
      }

      core.info("Patch size validation passed");
    }

    if (isEmpty && !isStaged && !allowEmpty) {
      const error = "Patch file is empty - no changes to apply (noop operation)";

      switch (ifNoChanges) {
        case "error":
          return { success: false, error };
        case "ignore":
          return { success: false, error, skipped: true };
        case "warn":
        default:
          core.warning(error);
          return { success: false, error };
      }
    }

    if (!isEmpty) {
      core.info("Patch content validation passed");
    } else if (allowEmpty) {
      core.info("Patch file is empty - processing empty PR creation (allow-empty is enabled)");
    } else {
      core.info("Patch file is empty - processing noop operation");
    }

    // Extract title, body, and branch from the message
    let title = pullRequestItem.title ? pullRequestItem.title.trim() : "";

    // Replace temporary ID references in the body
    let processedBody = pullRequestItem.body || "";
    if (resolvedTemporaryIds && Object.keys(resolvedTemporaryIds).length > 0) {
      // Convert object to Map for compatibility with replaceTemporaryIdReferences
      const tempIdMap = new Map(Object.entries(resolvedTemporaryIds));
      const currentRepo = `${context.repo.owner}/${context.repo.repo}`;
      processedBody = replaceTemporaryIdReferences(processedBody, tempIdMap, currentRepo);
      core.info("Resolved temporary ID references in PR body");
    }

    // Remove duplicate title from description if it starts with a header matching the title
    processedBody = removeDuplicateTitleFromDescription(title, processedBody);

    let bodyLines = processedBody.split("\n");
    let branchName = pullRequestItem.branch ? pullRequestItem.branch.trim() : null;

    // If no title was found, use a default
    if (!title) {
      title = "Agent Output";
    }

    // Apply title prefix if provided
    if (titlePrefix && !title.startsWith(titlePrefix)) {
      title = titlePrefix + title;
    }

    // Add AI disclaimer with workflow name and run url
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
    const runId = context.runId;
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    const runUrl = context.payload.repository ? `${context.payload.repository.html_url}/actions/runs/${runId}` : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

    // Add fingerprint comment if present
    const trackerIDComment = getTrackerID("markdown");
    if (trackerIDComment) {
      bodyLines.push(trackerIDComment);
    }

    // Add expiration comment if expires is set (only for same-repo PRs)
    if (expiresHours > 0) {
      const expiresDate = new Date();
      expiresDate.setHours(expiresDate.getHours() + expiresHours);
      const expiresString = expiresDate.toISOString();
      bodyLines.push(``, `<!-- gh-aw-expires: ${expiresString} -->`, `<!-- gh-aw-expires-type: Pull Request -->`);
    }

    bodyLines.push(``, ``, `> AI generated by [${workflowName}](${runUrl})`, "");

    // Prepare the body content
    const body = bodyLines.join("\n").trim();

    // Build labels array
    let labels = [...envLabels];
    if (pullRequestItem.labels && Array.isArray(pullRequestItem.labels)) {
      labels = [...labels, ...pullRequestItem.labels];
    }
    labels = labels
      .filter(label => !!label)
      .map(label => String(label).trim())
      .filter(label => label);

    // Determine draft setting - use message value if provided, otherwise use config default
    const draft = pullRequestItem.draft !== undefined ? pullRequestItem.draft : draftDefault;

    core.info(`Creating pull request with title: ${title}`);
    core.info(`Labels: ${JSON.stringify(labels)}`);
    core.info(`Draft: ${draft}`);
    core.info(`Body length: ${body.length}`);

    // If in staged mode, emit step summary instead of creating PR
    if (isStaged) {
      let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
      summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";

      summaryContent += `**Title:** ${title || "No title provided"}\n\n`;
      summaryContent += `**Branch:** ${branchName || "auto-generated"}\n\n`;
      summaryContent += `**Base:** ${baseBranch}\n\n`;

      if (body) {
        summaryContent += `**Body:**\n${body}\n\n`;
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
      return {
        success: true,
        staged: true,
        title,
        branch: branchName,
      };
    }

    const randomHex = crypto.randomBytes(8).toString("hex");
    // Use branch name from message if provided, otherwise generate unique branch name
    if (!branchName) {
      core.info("No branch name provided in message, generating unique branch name");
      // Generate unique branch name using cryptographic random hex
      branchName = `${workflowId}-${randomHex}`;
    } else {
      branchName = `${branchName}-${randomHex}`;
      core.info(`Using branch name from message with added salt: ${branchName}`);
    }

    core.info(`Generated branch name: ${branchName}`);
    core.info(`Base branch: ${baseBranch}`);

  // Create a new branch using git CLI, ensuring it's based on the correct base branch

  // First, fetch the base branch specifically (since we use shallow checkout)
  core.info(`Fetching base branch: ${baseBranch}`);

  // Fetch without creating/updating local branch to avoid conflicts with current branch
  // This works even when we're already on the base branch
  await exec.exec(`git fetch origin ${baseBranch}`);

  // Checkout the base branch (using origin/${baseBranch} if local doesn't exist)
  try {
    await exec.exec(`git checkout ${baseBranch}`);
  } catch (checkoutError) {
    // If local branch doesn't exist, create it from origin
    core.info(`Local branch ${baseBranch} doesn't exist, creating from origin/${baseBranch}`);
    await exec.exec(`git checkout -b ${baseBranch} origin/${baseBranch}`);
  }

  // Handle branch creation/checkout
  core.info(`Branch should not exist locally, creating new branch from base: ${branchName}`);
  await exec.exec(`git checkout -b ${branchName}`);
  core.info(`Created new branch from base: ${branchName}`);

  // Apply the patch using git CLI (skip if empty)
  if (!isEmpty) {
    core.info("Applying patch...");

    // Log first 500 lines of patch for debugging
    const patchLines = patchContent.split("\n");
    const previewLineCount = Math.min(500, patchLines.length);
    core.info(`Patch preview (first ${previewLineCount} of ${patchLines.length} lines):`);
    for (let i = 0; i < previewLineCount; i++) {
      core.info(patchLines[i]);
    }

    // Patches are created with git format-patch, so use git am to apply them
    try {
      await exec.exec("git am /tmp/gh-aw/aw.patch");
      core.info("Patch applied successfully");
    } catch (patchError) {
      core.error(`Failed to apply patch: ${patchError instanceof Error ? patchError.message : String(patchError)}`);

      // Investigate why the patch failed by logging git status and the failed patch
      try {
        core.info("Investigating patch failure...");

        // Log git status to see the current state
        const statusResult = await exec.getExecOutput("git", ["status"]);
        core.info("Git status output:");
        core.info(statusResult.stdout);

        // Log the failed patch diff
        const patchResult = await exec.getExecOutput("git", ["am", "--show-current-patch=diff"]);
        core.info("Failed patch content:");
        core.info(patchResult.stdout);
      } catch (investigateError) {
        core.warning(`Failed to investigate patch failure: ${investigateError instanceof Error ? investigateError.message : String(investigateError)}`);
      }

      core.setFailed("Failed to apply patch");
      return;
    }

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
        core.info(`Remote branch check failed (non-fatal): ${checkError instanceof Error ? checkError.message : String(checkError)}`);
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
      const runUrl = context.payload.repository ? `${context.payload.repository.html_url}/actions/runs/${runId}` : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

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

        // Update the activation comment with issue link (if a comment was created)
        await updateActivationComment(github, context, core, issue.html_url, issue.number, "issue");

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

    // For empty patches with allow-empty, we still need to push the branch
    if (allowEmpty) {
      core.info("allow-empty is enabled - will create branch and push with empty commit");
      // Push the branch with an empty commit to allow PR creation
      try {
        // Create an empty commit to ensure there's a commit difference
        await exec.exec(`git commit --allow-empty -m "Initialize"`);
        core.info("Created empty commit");

        // Check if remote branch already exists (optional precheck)
        let remoteBranchExists = false;
        try {
          const { stdout } = await exec.getExecOutput(`git ls-remote --heads origin ${branchName}`);
          if (stdout.trim()) {
            remoteBranchExists = true;
          }
        } catch (checkError) {
          core.info(`Remote branch check failed (non-fatal): ${checkError instanceof Error ? checkError.message : String(checkError)}`);
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
        core.info("Empty branch pushed successfully");
      } catch (pushError) {
        core.setFailed(`Failed to push empty branch: ${pushError instanceof Error ? pushError.message : String(pushError)}`);
        return;
      }
    } else {
      // For empty patches without allow-empty, handle if-no-changes configuration
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

    // Update the activation comment with PR link (if a comment was created)
    await updateActivationComment(github, context, core, pullRequest.html_url, pullRequest.number);

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
    const branchUrl = context.payload.repository ? `${context.payload.repository.html_url}/tree/${branchName}` : `${githubServer}/${context.repo.owner}/${context.repo.repo}/tree/${branchName}`;

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

      // Update the activation comment with issue link (if a comment was created)
      await updateActivationComment(github, context, core, issue.html_url, issue.number, "issue");

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
      core.setFailed(`Failed to create both pull request and fallback issue. PR error: ${prError instanceof Error ? prError.message : String(prError)}. Issue error: ${issueError instanceof Error ? issueError.message : String(issueError)}`);
      return;
    }
  }
}

module.exports = { main };
