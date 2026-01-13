// @ts-check
/// <reference types="@actions/github-script" />

/** @type {typeof import("fs")} */
const fs = require("fs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { updateActivationCommentWithCommit } = require("./update_activation_comment.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { replaceTemporaryIdReferences } = require("./temporary_id.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "push_to_pull_request_branch";

/**
 * Main handler factory for push_to_pull_request_branch
 * Returns a message handler function that processes individual push_to_pull_request_branch messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration from config parameter
  const target = config.target || "triggering";
  const titlePrefix = config.title_prefix || "";
  const envLabels = config.labels ? (Array.isArray(config.labels) ? config.labels : config.labels.split(",")).map(label => String(label).trim()).filter(label => label) : [];
  const ifNoChanges = config.if_no_changes || "warn";
  const commitTitleSuffix = config.commit_title_suffix || "";
  const maxSizeKb = config.max_patch_size ? parseInt(String(config.max_patch_size), 10) : 1024;
  const baseBranch = config.base_branch || "";
  const maxCount = config.max || 0; // 0 means no limit

  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  core.info(`Target: ${target}`);
  if (baseBranch) {
    core.info(`Base branch: ${baseBranch}`);
  }
  if (titlePrefix) {
    core.info(`Title prefix: ${titlePrefix}`);
  }
  if (envLabels.length > 0) {
    core.info(`Required labels: ${envLabels.join(", ")}`);
  }
  core.info(`If no changes: ${ifNoChanges}`);
  if (commitTitleSuffix) {
    core.info(`Commit title suffix: ${commitTitleSuffix}`);
  }
  core.info(`Max patch size: ${maxSizeKb} KB`);
  core.info(`Max count: ${maxCount || "unlimited"}`);

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function - processes individual push_to_pull_request_branch messages
   * @param {any} message - The push_to_pull_request_branch message to process
   * @param {import('./types/handler-factory').ResolvedTemporaryIds} resolvedTemporaryIds - Map of temporary IDs to resolved IDs
   * @returns {Promise<import('./types/handler-factory').HandlerResult>}
   */
  return async function handlePushToPullRequestBranch(message, resolvedTemporaryIds) {
    // Check max count
    if (maxCount > 0 && processedCount >= maxCount) {
      core.info(`Skipping message - max count (${maxCount}) reached`);
      return { success: false, error: `Max count (${maxCount}) reached`, skipped: true };
    }

    processedCount++;

    // Check if patch file exists and has valid content
    if (!fs.existsSync("/tmp/gh-aw/aw.patch")) {
      const msg = "No patch file found - cannot push without changes";

      switch (ifNoChanges) {
        case "error":
          return { success: false, error: msg };
        case "ignore":
          return { success: false, error: msg, skipped: true };
        case "warn":
        default:
          core.info(msg);
          return { success: false, error: msg, skipped: true };
      }
    }

    const patchContent = fs.readFileSync("/tmp/gh-aw/aw.patch", "utf8");

    // Check for actual error conditions
    if (patchContent.includes("Failed to generate patch")) {
      const msg = "Patch file contains error message - cannot push without changes";
      core.error("Patch file generation failed");
      core.error(`Patch file location: /tmp/gh-aw/aw.patch`);
      core.error(`Patch file size: ${Buffer.byteLength(patchContent, "utf8")} bytes`);
      const previewLength = Math.min(500, patchContent.length);
      core.error(`Patch file preview (first ${previewLength} characters):`);
      core.error(patchContent.substring(0, previewLength));
      return { success: false, error: msg };
    }

    // Validate patch size (unless empty)
    const isEmpty = !patchContent || !patchContent.trim();
    if (!isEmpty) {
      const patchSizeBytes = Buffer.byteLength(patchContent, "utf8");
      const patchSizeKb = Math.ceil(patchSizeBytes / 1024);

      core.info(`Patch size: ${patchSizeKb} KB (maximum allowed: ${maxSizeKb} KB)`);

      if (patchSizeKb > maxSizeKb) {
        const msg = `Patch size (${patchSizeKb} KB) exceeds maximum allowed size (${maxSizeKb} KB)`;
        return { success: false, error: msg };
      }

      core.info("Patch size validation passed");
    }

    if (isEmpty) {
      const msg = "Patch file is empty - no changes to apply (noop operation)";

      switch (ifNoChanges) {
        case "error":
          return { success: false, error: "No changes to push - failing as configured by if-no-changes: error" };
        case "ignore":
          return { success: false, error: msg, skipped: true };
        case "warn":
        default:
          core.info(msg);
          return { success: false, error: msg, skipped: true };
      }
    }

    core.info("Patch content validation passed");
    core.info(`Target configuration: ${target}`);

    // If in staged mode, emit preview
    if (isStaged) {
      await generateStagedPreview({
        title: "Push to PR Branch",
        description: "The following changes would be pushed if staged mode was disabled:",
        items: [{ target, commit_message: message.commit_message }],
        renderItem: item => {
          let content = `**Target:** ${item.target}\n\n`;

          if (item.commit_message) {
            content += `**Commit Message:** ${item.commit_message}\n\n`;
          }

          if (fs.existsSync("/tmp/gh-aw/aw.patch")) {
            const patchStats = fs.readFileSync("/tmp/gh-aw/aw.patch", "utf8");
            if (patchStats.trim()) {
              content += `**Changes:** Patch file exists with ${patchStats.split("\n").length} lines\n\n`;
              content += `<details><summary>Show patch preview</summary>\n\n\`\`\`diff\n${patchStats.slice(0, 2000)}${patchStats.length > 2000 ? "\n... (truncated)" : ""}\n\`\`\`\n\n</details>\n\n`;
            } else {
              content += `**Changes:** No changes (empty patch)\n\n`;
            }
          }
          return content;
        },
      });
      return { success: true, staged: true };
    }

    // Validate target configuration
    if (target !== "*" && target !== "triggering") {
      const pullNumber = parseInt(target, 10);
      if (isNaN(pullNumber)) {
        return { success: false, error: 'Invalid target configuration: must be "triggering", "*", or a valid pull request number' };
      }
    }

    // Compute the target branch name based on target configuration
    let pullNumber;
    if (target === "triggering") {
      pullNumber = context.payload?.pull_request?.number || context.payload?.issue?.number;

      if (!pullNumber) {
        return { success: false, error: 'push-to-pull-request-branch with target "triggering" requires pull request context' };
      }
    } else if (target === "*") {
      if (message.pull_request_number) {
        pullNumber = parseInt(message.pull_request_number, 10);
      }
    } else {
      pullNumber = parseInt(target, 10);
    }

    let branchName;
    let prTitle = "";
    let prLabels = [];

    if (!pullNumber) {
      return { success: false, error: "Pull request number is required but not found" };
    }

    // Fetch the specific PR to get its head branch, title, and labels
    try {
      const { data: pullRequest } = await github.rest.pulls.get({
        owner: context.repo.owner,
        repo: context.repo.repo,
        pull_number: pullNumber,
      });
      branchName = pullRequest.head.ref;
      prTitle = pullRequest.title || "";
      prLabels = pullRequest.labels.map(label => label.name);
    } catch (error) {
      core.info(`Warning: Could not fetch PR ${pullNumber} details: ${getErrorMessage(error)}`);
      return { success: false, error: `Failed to determine branch name for PR ${pullNumber}` };
    }

    core.info(`Target branch: ${branchName}`);
    core.info(`PR title: ${prTitle}`);
    core.info(`PR labels: ${prLabels.join(", ")}`);

    // Validate title prefix if specified
    if (titlePrefix && !prTitle.startsWith(titlePrefix)) {
      return { success: false, error: `Pull request title "${prTitle}" does not start with required prefix "${titlePrefix}"` };
    }

    // Validate labels if specified
    if (envLabels.length > 0) {
      const missingLabels = envLabels.filter(label => !prLabels.includes(label));
      if (missingLabels.length > 0) {
        return { success: false, error: `Pull request is missing required labels: ${missingLabels.join(", ")}. Current labels: ${prLabels.join(", ")}` };
      }
    }

    if (titlePrefix) {
      core.info(`✓ Title prefix validation passed: "${titlePrefix}"`);
    }
    if (envLabels.length > 0) {
      core.info(`✓ Labels validation passed: ${envLabels.join(", ")}`);
    }

    const hasChanges = !isEmpty;

    // Switch to or create the target branch
    core.info(`Switching to branch: ${branchName}`);

    // Fetch the specific target branch from origin
    try {
      core.info(`Fetching branch: ${branchName}`);
      await exec.exec(`git fetch origin ${branchName}:refs/remotes/origin/${branchName}`);
    } catch (fetchError) {
      return { success: false, error: `Failed to fetch branch ${branchName}: ${fetchError instanceof Error ? fetchError.message : String(fetchError)}` };
    }

    // Check if branch exists on origin
    try {
      await exec.exec(`git rev-parse --verify origin/${branchName}`);
    } catch (verifyError) {
      return { success: false, error: `Branch ${branchName} does not exist on origin, can't push to it: ${verifyError instanceof Error ? verifyError.message : String(verifyError)}` };
    }

    // Checkout the branch from origin
    try {
      await exec.exec(`git checkout -B ${branchName} origin/${branchName}`);
      core.info(`Checked out existing branch from origin: ${branchName}`);
    } catch (checkoutError) {
      return { success: false, error: `Failed to checkout branch ${branchName}: ${checkoutError instanceof Error ? checkoutError.message : String(checkoutError)}` };
    }

    // Apply the patch using git CLI (skip if empty)
    if (hasChanges) {
      core.info("Applying patch...");
      try {
        if (commitTitleSuffix) {
          core.info(`Appending commit title suffix: "${commitTitleSuffix}"`);

          // Read the patch file
          let patchContent = fs.readFileSync("/tmp/gh-aw/aw.patch", "utf8");

          // Modify Subject lines in the patch to append the suffix
          patchContent = patchContent.replace(/^Subject: (?:\[PATCH\] )?(.*)$/gm, (match, title) => `Subject: [PATCH] ${title}${commitTitleSuffix}`);

          // Write the modified patch back
          fs.writeFileSync("/tmp/gh-aw/aw.patch", patchContent, "utf8");
          core.info(`Patch modified with commit title suffix: "${commitTitleSuffix}"`);
        }

        // Log first 100 lines of patch for debugging
        const finalPatchContent = fs.readFileSync("/tmp/gh-aw/aw.patch", "utf8");
        const patchLines = finalPatchContent.split("\n");
        const previewLineCount = Math.min(100, patchLines.length);
        core.info(`Patch preview (first ${previewLineCount} of ${patchLines.length} lines):`);
        for (let i = 0; i < previewLineCount; i++) {
          core.info(patchLines[i]);
        }

        // Apply patch
        await exec.exec("git am /tmp/gh-aw/aw.patch");
        core.info("Patch applied successfully");

        // Push the applied commits to the branch
        await exec.exec(`git push origin ${branchName}`);
        core.info(`Changes committed and pushed to branch: ${branchName}`);
      } catch (error) {
        core.error(`Failed to apply patch: ${getErrorMessage(error)}`);

        // Investigate patch failure
        try {
          core.info("Investigating patch failure...");

          const statusResult = await exec.getExecOutput("git", ["status"]);
          core.info("Git status output:");
          core.info(statusResult.stdout);

          const logResult = await exec.getExecOutput("git", ["log", "--oneline", "-5"]);
          core.info("Recent commits (last 5):");
          core.info(logResult.stdout);

          const diffResult = await exec.getExecOutput("git", ["diff", "HEAD"]);
          core.info("Uncommitted changes:");
          core.info(diffResult.stdout && diffResult.stdout.trim() ? diffResult.stdout : "(no uncommitted changes)");

          const patchDiffResult = await exec.getExecOutput("git", ["am", "--show-current-patch=diff"]);
          core.info("Failed patch diff:");
          core.info(patchDiffResult.stdout);

          const patchFullResult = await exec.getExecOutput("git", ["am", "--show-current-patch"]);
          core.info("Failed patch (full):");
          core.info(patchFullResult.stdout);
        } catch (investigateError) {
          core.warning(`Failed to investigate patch failure: ${investigateError instanceof Error ? investigateError.message : String(investigateError)}`);
        }

        return { success: false, error: "Failed to apply patch" };
      }
    } else {
      core.info("Skipping patch application (empty patch)");

      const msg = "No changes to apply - noop operation completed successfully";

      switch (ifNoChanges) {
        case "error":
          return { success: false, error: "No changes to apply - failing as configured by if-no-changes: error" };
        case "ignore":
          // Silent success
          break;
        case "warn":
        default:
          core.info(msg);
          break;
      }
    }

    // Get commit SHA and push URL
    const commitShaRes = await exec.getExecOutput("git", ["rev-parse", "HEAD"]);
    if (commitShaRes.exitCode !== 0) {
      return { success: false, error: "Failed to get commit SHA" };
    }
    const commitSha = commitShaRes.stdout.trim();

    // Get repository base URL and construct URLs
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    const repoUrl = context.payload.repository ? context.payload.repository.html_url : `${githubServer}/${context.repo.owner}/${context.repo.repo}`;
    const pushUrl = `${repoUrl}/tree/${branchName}`;
    const commitUrl = `${repoUrl}/commit/${commitSha}`;

    // Update the activation comment with commit link (if a comment was created and changes were pushed)
    if (hasChanges) {
      await updateActivationCommentWithCommit(github, context, core, commitSha, commitUrl);
    }

    // Write summary to GitHub Actions summary
    const summaryTitle = hasChanges ? "Push to Branch" : "Push to Branch (No Changes)";
    const summaryContent = hasChanges
      ? `
## ${summaryTitle}
- **Branch**: \`${branchName}\`
- **Commit**: [${commitSha.substring(0, 7)}](${commitUrl})
- **URL**: [${pushUrl}](${pushUrl})
`
      : `
## ${summaryTitle}
- **Branch**: \`${branchName}\`
- **Status**: No changes to apply (noop operation)
- **URL**: [${pushUrl}](${pushUrl})
`;

    await core.summary.addRaw(summaryContent).write();

    return {
      success: true,
      branch_name: branchName,
      commit_url: commitUrl,
    };
  };
}

module.exports = { main, HANDLER_TYPE };
