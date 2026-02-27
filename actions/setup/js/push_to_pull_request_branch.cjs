// @ts-check
/// <reference types="@actions/github-script" />

/** @type {typeof import("fs")} */
const fs = require("fs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { updateActivationCommentWithCommit } = require("./update_activation_comment.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { normalizeBranchName } = require("./normalize_branch_name.cjs");
const { pushExtraEmptyCommit } = require("./extra_empty_commit.cjs");
const { detectForkPR } = require("./pr_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { getBaseBranch } = require("./get_base_branch.cjs");

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
  // Resolve base branch: use config value if set, otherwise resolve dynamically
  // Dynamic resolution is needed for issue_comment events on PRs where the base branch
  // is not available in GitHub Actions expressions and requires an API call
  const baseBranch = config.base_branch || (await getBaseBranch());
  const maxCount = config.max || 0; // 0 means no limit

  // Cross-repo support: resolve target repository from config
  // This allows pushing to PRs in a different repository than the workflow
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);

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
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${[...allowedRepos].join(", ")}`);
  }

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

    // Determine the patch file path from the message (set by the MCP server handler)
    const patchFilePath = message.patch_path;
    core.info(`Patch file path: ${patchFilePath || "(not set)"}`);

    // Check if patch file exists and has valid content
    if (!patchFilePath || !fs.existsSync(patchFilePath)) {
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

    const patchContent = fs.readFileSync(patchFilePath, "utf8");

    // Check for actual error conditions
    if (patchContent.includes("Failed to generate patch")) {
      const msg = "Patch file contains error message - cannot push without changes";
      core.error("Patch file generation failed");
      core.error(`Patch file location: ${patchFilePath}`);
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

    // If in staged mode, emit 🎭 Staged Mode Preview via generateStagedPreview
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

          if (patchFilePath && fs.existsSync(patchFilePath)) {
            const patchStats = fs.readFileSync(patchFilePath, "utf8");
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

    // Resolve and validate target repository
    // For cross-repo scenarios, the PR may be in a different repository than the workflow
    const repoResult = resolveAndValidateRepo(message, defaultTargetRepo, allowedRepos, "push to PR branch");
    if (!repoResult.success) {
      return { success: false, error: repoResult.error };
    }
    const itemRepo = repoResult.repo;
    const repoParts = repoResult.repoParts;

    core.info(`Target repository: ${itemRepo}`);

    // Fetch the specific PR to get its head branch, title, and labels
    let pullRequest;
    try {
      const response = await github.rest.pulls.get({
        owner: repoParts.owner,
        repo: repoParts.repo,
        pull_number: pullNumber,
      });
      pullRequest = response.data;
      branchName = pullRequest.head.ref;
      prTitle = pullRequest.title || "";
      prLabels = pullRequest.labels.map(label => label.name);
    } catch (error) {
      core.info(`Warning: Could not fetch PR ${pullNumber} from ${itemRepo}: ${getErrorMessage(error)}`);
      return { success: false, error: `Failed to determine branch name for PR ${pullNumber} in ${itemRepo}` };
    }

    // SECURITY: Check if this is a fork PR - we cannot push to fork branches
    // The workflow token only has access to the base repository, not the fork
    const { isFork, reason: forkReason } = detectForkPR(pullRequest);
    if (isFork) {
      core.error(`Cannot push to fork PR branch: ${forkReason}`);
      core.error("The workflow token does not have permission to push to fork repositories.");
      core.error("Fork PRs must be updated by the fork owner or through other mechanisms.");
      return {
        success: false,
        error: `Cannot push to fork PR: ${forkReason}. The workflow token does not have permission to push to fork repositories.`,
      };
    }
    core.info(`Fork PR check: not a fork (${forkReason})`);

    // SECURITY: Sanitize branch name to prevent shell injection (CWE-78)
    // Branch names from GitHub API must be normalized before use in git commands
    if (branchName) {
      const originalBranchName = branchName;
      branchName = normalizeBranchName(branchName);

      // Validate it's not empty after normalization
      if (!branchName) {
        return { success: false, error: `Invalid branch name: sanitization resulted in empty string (original: "${originalBranchName}")` };
      }

      if (originalBranchName !== branchName) {
        core.info(`Branch name sanitized: "${originalBranchName}" -> "${branchName}"`);
      }
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
    // Track number of new commits added so we can restrict the extra empty commit
    // to branches with exactly one new commit (security: prevents use of CI trigger
    // token on multi-commit branches where workflow files may have been modified).
    let newCommitCount = 0;
    if (hasChanges) {
      core.info("Applying patch...");
      try {
        if (commitTitleSuffix) {
          core.info(`Appending commit title suffix: "${commitTitleSuffix}"`);

          // Read the patch file
          let patchContent = fs.readFileSync(patchFilePath, "utf8");

          // Modify Subject lines in the patch to append the suffix
          patchContent = patchContent.replace(/^Subject: (?:\[PATCH\] )?(.*)$/gm, (match, title) => `Subject: [PATCH] ${title}${commitTitleSuffix}`);

          // Write the modified patch back
          fs.writeFileSync(patchFilePath, patchContent, "utf8");
          core.info(`Patch modified with commit title suffix: "${commitTitleSuffix}"`);
        }

        // Log first 100 lines of patch for debugging
        const finalPatchContent = fs.readFileSync(patchFilePath, "utf8");
        const patchLines = finalPatchContent.split("\n");
        const previewLineCount = Math.min(100, patchLines.length);
        core.info(`Patch preview (first ${previewLineCount} of ${patchLines.length} lines):`);
        for (let i = 0; i < previewLineCount; i++) {
          core.info(patchLines[i]);
        }

        // Apply patch
        // Capture HEAD before applying patch to compute new-commit count later
        let remoteHeadBeforePatch = "";
        try {
          const { stdout } = await exec.getExecOutput("git", ["rev-parse", "HEAD"]);
          remoteHeadBeforePatch = stdout.trim();
        } catch {
          // Non-fatal - extra empty commit will be skipped
        }

        // Use --3way to handle cross-repo patches where the patch base may differ from target repo
        // This allows git to resolve create-vs-modify mismatches when a file exists in target but not source
        await exec.exec(`git am --3way ${patchFilePath}`);
        core.info("Patch applied successfully");

        // Push the applied commits to the branch
        await exec.exec(`git push origin ${branchName}`);
        core.info(`Changes committed and pushed to branch: ${branchName}`);

        // Count new commits pushed for the CI trigger decision
        if (remoteHeadBeforePatch) {
          try {
            const { stdout: countStr } = await exec.getExecOutput("git", ["rev-list", "--count", `${remoteHeadBeforePatch}..HEAD`]);
            newCommitCount = parseInt(countStr.trim(), 10);
            core.info(`${newCommitCount} new commit(s) pushed to branch`);
          } catch {
            // Non-fatal - newCommitCount stays 0, extra empty commit will be skipped
            core.info("Could not count new commits - extra empty commit will be skipped");
          }
        }
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
    // For cross-repo scenarios, use repoParts (the target repo) not context.repo (the workflow repo)
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    const repoUrl = `${githubServer}/${repoParts.owner}/${repoParts.repo}`;
    const pushUrl = `${repoUrl}/tree/${branchName}`;
    const commitUrl = `${repoUrl}/commit/${commitSha}`;

    // Update the activation comment with commit link (if a comment was created and changes were pushed)
    // Pass pullNumber so a new comment is created on the PR when no activation comment exists (e.g., schedule triggers)
    if (hasChanges) {
      await updateActivationCommentWithCommit(github, context, core, commitSha, commitUrl, { targetIssueNumber: pullNumber });
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

    // Push an extra empty commit if a token is configured and exactly 1 new commit was pushed.
    // This works around the GITHUB_TOKEN limitation where pushes don't trigger CI events.
    // Restricting to exactly 1 new commit prevents the CI trigger token being used on
    // multi-commit branches where workflow files may have been iteratively modified.
    if (hasChanges) {
      const ciTriggerResult = await pushExtraEmptyCommit({
        branchName,
        repoOwner: repoParts.owner,
        repoName: repoParts.repo,
        newCommitCount,
      });
      if (ciTriggerResult.success && !ciTriggerResult.skipped) {
        core.info("Extra empty commit pushed - CI checks should start shortly");
      }
    }

    return {
      success: true,
      branch_name: branchName,
      commit_url: commitUrl,
    };
  };
}

module.exports = { main, HANDLER_TYPE };
