// @ts-check
/// <reference types="@actions/github-script" />

/** @type {typeof import("fs")} */
const fs = require("fs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { pushSignedCommits } = require("./push_signed_commits.cjs");
const { updateActivationCommentWithCommit, updateActivationComment } = require("./update_activation_comment.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { normalizeBranchName } = require("./normalize_branch_name.cjs");
const { pushExtraEmptyCommit } = require("./extra_empty_commit.cjs");
const { detectForkPR } = require("./pr_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { checkFileProtection } = require("./manifest_file_helpers.cjs");
const { buildWorkflowRunUrl } = require("./workflow_metadata_helpers.cjs");
const { renderTemplateFromFile, buildProtectedFileList } = require("./messages_core.cjs");
const { getGitAuthEnv } = require("./git_helpers.cjs");

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
  const maxCount = config.max || 0; // 0 means no limit

  // Cross-repo support: resolve target repository from config
  // This allows pushing to PRs in a different repository than the workflow
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
  const githubClient = await createAuthenticatedGitHubClient(config);

  // Build git auth env once for all network operations in this handler.
  // clean_git_credentials.sh removes credentials from .git/config before the
  // agent runs, so git fetch/push must authenticate via GIT_CONFIG_* env vars.
  // Use the per-handler github-token (for cross-repo PAT) when available,
  // falling back to GITHUB_TOKEN for the default workflow token.
  const gitAuthEnv = getGitAuthEnv(config["github-token"]);

  // Base branch from config (if set) - used only for logging at factory level
  // Dynamic base branch resolution happens per-message after resolving the actual target repo
  const configBaseBranch = config.base_branch || null;

  // Check if we're in staged mode (either globally or per-handler config)
  const isStaged = isStagedMode(config);

  core.info(`Target: ${target}`);
  if (configBaseBranch) {
    core.info(`Base branch (from config): ${configBaseBranch}`);
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

    // Determine the bundle file path from the message (set when patch-format: bundle is configured)
    const bundleFilePath = message.bundle_path;
    if (bundleFilePath) {
      core.info(`Bundle file path: ${bundleFilePath}`);
    }

    // Check if bundle or patch file exists
    const hasBundleFile = !!(bundleFilePath && fs.existsSync(bundleFilePath));

    // Check if patch file exists and has valid content
    if (!hasBundleFile && (!patchFilePath || !fs.existsSync(patchFilePath))) {
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

    // For bundle transport, there is no patch content to read/validate.
    // The bundle file itself is the transport artifact.
    let patchContent = "";
    let isEmpty;

    if (hasBundleFile) {
      // Bundle transport: treat as non-empty (the bundle contains commits)
      isEmpty = false;
    } else {
      patchContent = fs.readFileSync(patchFilePath, "utf8");

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

      isEmpty = !patchContent || !patchContent.trim();
    }
    if (!hasBundleFile && !isEmpty) {
      const patchSizeBytes = Buffer.byteLength(patchContent, "utf8");
      const patchSizeKb = Math.ceil(patchSizeBytes / 1024);

      core.info(`Patch size: ${patchSizeKb} KB (maximum allowed: ${maxSizeKb} KB)`);

      if (patchSizeKb > maxSizeKb) {
        const msg = `Patch size (${patchSizeKb} KB) exceeds maximum allowed size (${maxSizeKb} KB)`;
        return { success: false, error: msg };
      }

      core.info("Patch size validation passed");
    }

    // Check file protection: allowlist (strict) or protected-files policy.
    // Fallback-to-issue detection is deferred until after PR metadata is resolved below.
    /** @type {string[] | null} Protected files found in the patch (manifest basenames + path-prefix matches) */
    let protectedFilesForFallback = null;
    if (!isEmpty) {
      const protection = checkFileProtection(patchContent, config);
      if (protection.action === "deny") {
        const filesStr = protection.files.join(", ");
        const msg =
          protection.source === "allowlist"
            ? `Cannot push to pull request branch: patch modifies files outside the allowed-files list (${filesStr}). Add the files to the allowed-files configuration field or remove them from the patch.`
            : `Cannot push to pull request branch: patch modifies protected files (${filesStr}). Add them to the allowed-files configuration field or set protected-files: fallback-to-issue to create a review issue instead.`;
        core.error(msg);
        return { success: false, error: msg };
      }
      if (protection.action === "fallback") {
        protectedFilesForFallback = protection.files;
        core.warning(`Protected file protection triggered (fallback-to-issue): ${protection.files.join(", ")}. Will create review issue instead of pushing.`);
      }
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
      pullNumber = typeof context !== "undefined" ? context.payload?.pull_request?.number || context.payload?.issue?.number : undefined;

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
      const response = await githubClient.rest.pulls.get({
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

    // SECURITY: Block pushing to the repository's default branch or any branch with
    // protection rules. PR head branches must never be default or protected branches.
    // This prevents agents from pushing directly to branches that should only receive
    // changes through reviewed pull requests.
    {
      // Check whether the branch is the repository default branch
      let defaultBranch = null;
      try {
        const { data: repoData } = await githubClient.rest.repos.get({
          owner: repoParts.owner,
          repo: repoParts.repo,
        });
        defaultBranch = repoData.default_branch;
      } catch (repoError) {
        core.warning(`Could not check repository default branch: ${getErrorMessage(repoError)}`);
      }

      if (defaultBranch && branchName === defaultBranch) {
        const msg = `Cannot push to branch "${branchName}": this is the repository's default branch. Agents must not push directly to the default branch.`;
        core.error(msg);
        return { success: false, error: msg };
      }

      // Check whether the branch has protection rules
      let isBranchProtected = false;
      try {
        await githubClient.rest.repos.getBranchProtection({
          owner: repoParts.owner,
          repo: repoParts.repo,
          branch: branchName,
        });
        // Successful response means branch protection rules exist
        isBranchProtected = true;
      } catch (protectionError) {
        const protectionStatus = protectionError && typeof protectionError === "object" && "status" in protectionError ? protectionError.status : undefined;
        if (protectionStatus === 404) {
          // 404 means no protection rules – safe to proceed
          core.info(`Branch "${branchName}" has no protection rules`);
        } else if (protectionStatus === 403) {
          // 403 means the token lacks permission to read branch protection rules.
          // The GitHub platform will still enforce branch protection at push time,
          // so warn and allow the push to proceed.
          core.warning(`Could not check branch protection rules for "${branchName}" (insufficient permissions): ${getErrorMessage(protectionError)}`);
        } else {
          // Unexpected errors (5xx, network failures, etc.) – fail closed to
          // avoid bypassing branch protection due to transient API issues.
          const msg = `Cannot verify branch protection rules for "${branchName}": ${getErrorMessage(protectionError)}. Push blocked to prevent accidental writes to protected branches.`;
          core.error(msg);
          return { success: false, error: msg };
        }
      }

      if (isBranchProtected) {
        const msg = `Cannot push to branch "${branchName}": this branch has protection rules. Agents must not push directly to protected branches.`;
        core.error(msg);
        return { success: false, error: msg };
      }
    }

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

    // Deferred protected file protection – fallback-to-issue path.
    // Create a review issue now that we have repoParts, pullNumber, and prTitle available.
    if (protectedFilesForFallback && protectedFilesForFallback.length > 0) {
      const runUrl = buildWorkflowRunUrl(context, context.repo);
      const runId = context.runId;
      const patchFileName = patchFilePath ? patchFilePath.replace("/tmp/gh-aw/", "") : "aw-unknown.patch";
      const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
      const prUrl = `${githubServer}/${repoParts.owner}/${repoParts.repo}/pull/${pullNumber}`;
      const issueTitle = `[gh-aw] Protected Files: ${prTitle || `PR #${pullNumber}`}`;
      const fileList = buildProtectedFileList(protectedFilesForFallback, githubServer, repoParts.owner, repoParts.repo, branchName);
      const templatePath = `${process.env.RUNNER_TEMP}/gh-aw/prompts/manifest_protection_push_to_pr_fallback.md`;
      const issueBody = renderTemplateFromFile(templatePath, {
        files: fileList,
        pull_number: pullNumber,
        pr_url: prUrl,
        run_url: runUrl,
        run_id: runId,
        branch_name: branchName,
        patch_file_name: patchFileName,
      });

      try {
        const { data: issue } = await githubClient.rest.issues.create({
          owner: repoParts.owner,
          repo: repoParts.repo,
          title: issueTitle,
          body: issueBody,
          labels: ["agentic-workflows"],
        });
        core.info(`Created manifest-protection review issue #${issue.number}: ${issue.html_url}`);
        await updateActivationComment(github, context, core, issue.html_url, issue.number, "issue");
        return {
          success: true,
          fallback_used: true,
          issue_number: issue.number,
          issue_url: issue.html_url,
        };
      } catch (issueError) {
        const error = `Manifest file protection: failed to create review issue. Error: ${issueError instanceof Error ? issueError.message : String(issueError)}`;
        core.error(error);
        return { success: false, error };
      }
    }

    const hasChanges = !isEmpty;

    // Switch to or create the target branch
    core.info(`Switching to branch: ${branchName}`);

    // Fetch the specific target branch from origin
    // Use GIT_CONFIG_* env vars for auth because .git/config credentials are
    // cleaned by clean_git_credentials.sh before the agent runs.
    try {
      core.info(`Fetching branch: ${branchName}`);
      await exec.exec("git", ["fetch", "origin", `${branchName}:refs/remotes/origin/${branchName}`], {
        env: { ...process.env, ...gitAuthEnv },
      });
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

    // Apply the patch/bundle using git CLI (skip if empty)
    // Track number of new commits added so we can restrict the extra empty commit
    // to branches with exactly one new commit (security: prevents use of CI trigger
    // token on multi-commit branches where workflow files may have been modified).
    let newCommitCount = 0;
    let remoteHeadBeforePatch = "";
    if (hasChanges) {
      // Capture HEAD before applying changes to compute new-commit count later
      try {
        const { stdout } = await exec.getExecOutput("git", ["rev-parse", "HEAD"]);
        remoteHeadBeforePatch = stdout.trim();
      } catch {
        // Non-fatal - extra empty commit will be skipped
      }

      if (hasBundleFile) {
        // Bundle transport: fetch commits directly from the bundle file.
        // This preserves merge commit topology and per-commit metadata.
        core.info(`Applying changes from bundle: ${bundleFilePath}`);
        const bundleRef = `refs/bundles/push-${branchName.replace(/[^a-zA-Z0-9-]/g, "-")}`;
        try {
          // Fetch from bundle into a temporary ref
          await exec.exec("git", ["fetch", bundleFilePath, `refs/heads/${message.branch}:${bundleRef}`]);
          core.info(`Fetched bundle to ${bundleRef}`);

          // Fast-forward the current branch to the bundle tip
          await exec.exec("git", ["merge", "--ff-only", bundleRef]);
          core.info("Fast-forwarded branch to bundle tip");

          // Clean up the temporary ref
          try {
            await exec.exec("git", ["update-ref", "-d", bundleRef]);
          } catch {
            // Non-fatal cleanup
          }
        } catch (bundleError) {
          core.error(`Failed to apply bundle: ${bundleError instanceof Error ? bundleError.message : String(bundleError)}`);
          // Clean up temp ref if it exists
          try {
            await exec.exec("git", ["update-ref", "-d", bundleRef]);
          } catch {
            // Ignore
          }
          return { success: false, error: "Failed to apply bundle" };
        }
      } else {
        // Patch transport (default): git am --3way
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

          // Use --3way to handle cross-repo patches where the patch base may differ from target repo
          // This allows git to resolve create-vs-modify mismatches when a file exists in target but not source
          await exec.exec(`git am --3way ${patchFilePath}`);
          core.info("Patch applied successfully");
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
      } // end else (patch path)

      // When threat detection produced a warning, create a review PR instead of pushing
      // directly to the existing PR branch. This allows manual review of the changes
      // before they are merged into the target PR.
      const detectionConclusionEnv = process.env.GH_AW_DETECTION_CONCLUSION;
      if (detectionConclusionEnv === "warning") {
        core.info("⚠️ Threat detection warning: creating review PR instead of direct push");

        // Create a review branch name based on the original branch, using
        // normalizeBranchName to enforce valid git ref characters + max length.
        const reviewBranchName = normalizeBranchName(`${branchName}-review`, String(Date.now()));
        try {
          // Rename current local branch to review branch
          await exec.exec("git", ["checkout", "-b", reviewBranchName]);
          core.info(`Created review branch: ${reviewBranchName}`);

          // Push the review branch
          await exec.exec("git", ["push", "origin", reviewBranchName], {
            env: { ...process.env, ...gitAuthEnv },
          });
          core.info(`Pushed review branch: ${reviewBranchName}`);

          // Create PR from review branch to original branch
          const detectionReasonEnv = process.env.GH_AW_DETECTION_REASON || "unknown";
          const prBody = [
            "> [!CAUTION]",
            "> **This PR requires manual review** because threat detection produced a warning.",
            ">",
            `> **Reason:** ${detectionReasonEnv}`,
            ">",
            `> Review the [workflow run logs](${buildWorkflowRunUrl(context, context.repo)}) for details.`,
            "",
            `This PR contains changes that were originally intended for PR #${pullNumber} (\`${branchName}\`).`,
            "Please review the changes carefully before merging.",
          ].join("\n");

          const { data: reviewPR } = await githubClient.rest.pulls.create({
            owner: repoParts.owner,
            repo: repoParts.repo,
            title: `[review] ${prTitle || `Changes for #${pullNumber}`}`,
            body: prBody,
            head: reviewBranchName,
            base: branchName,
          });

          core.info(`Created review PR #${reviewPR.number}: ${reviewPR.html_url}`);

          // Try to add needs-review label to the review PR
          try {
            await githubClient.rest.issues.addLabels({
              owner: repoParts.owner,
              repo: repoParts.repo,
              issue_number: reviewPR.number,
              labels: ["needs-review"],
            });
            core.info('Added "needs-review" label to review PR');
          } catch (labelError) {
            core.warning(`Failed to add "needs-review" label to review PR: ${getErrorMessage(labelError)}`);
          }

          // Update activation comment with review PR link
          await updateActivationComment(github, context, core, reviewPR.html_url, reviewPR.number, "pull_request");

          return {
            success: true,
            review_pr: true,
            branch_name: reviewBranchName,
            pr_number: reviewPR.number,
            pr_url: reviewPR.html_url,
          };
        } catch (reviewError) {
          core.error(`Failed to create review PR: ${getErrorMessage(reviewError)}`);
          return { success: false, error: `Failed to create review PR: ${getErrorMessage(reviewError)}` };
        }
      }

      // Push the applied commits to the branch using signed GraphQL commits (outside patch try/catch so push failures are not misattributed)
      try {
        await pushSignedCommits({
          githubClient,
          owner: repoParts.owner,
          repo: repoParts.repo,
          branch: branchName,
          baseRef: remoteHeadBeforePatch || `origin/${branchName}`,
          cwd: process.cwd(),
          gitAuthEnv,
        });
        core.info(`Changes committed and pushed to branch: ${branchName}`);
      } catch (pushError) {
        const pushErrorMessage = getErrorMessage(pushError);
        core.error(`Failed to push changes: ${pushErrorMessage}`);
        const nonFastForwardPatterns = ["non-fast-forward", "rejected", "fetch first", "Updates were rejected"];
        const isNonFastForward = nonFastForwardPatterns.some(pattern => pushErrorMessage.includes(pattern));
        const userMessage = isNonFastForward
          ? "Failed to push changes: remote PR branch changed while the workflow was running (non-fast-forward). Re-run the workflow on the latest PR branch state."
          : `Failed to push changes: ${pushErrorMessage}`;
        return { success: false, error_type: "push_failed", error: userMessage };
      }

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
    //
    // NOTE: we pass 'github' (global octokit) instead of githubClient (repo-scoped octokit) because the issue is created
    // in the same repo as the activation, so the global client has the correct context for updating the comment.
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
      commit_sha: commitSha,
      commit_url: commitUrl,
    };
  };
}

module.exports = { main, HANDLER_TYPE };
