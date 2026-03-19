// @ts-check
/// <reference types="@actions/github-script" />

/** @type {typeof import("fs")} */
const fs = require("fs");
/** @type {typeof import("crypto")} */
const crypto = require("crypto");
const { updateActivationComment } = require("./update_activation_comment.cjs");
const { pushSignedCommits } = require("./push_signed_commits.cjs");
const { getTrackerID } = require("./get_tracker_id.cjs");
const { removeDuplicateTitleFromDescription } = require("./remove_duplicate_title.cjs");
const { sanitizeTitle, applyTitlePrefix } = require("./sanitize_title.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { replaceTemporaryIdReferences, isTemporaryId } = require("./temporary_id.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { addExpirationToFooter } = require("./ephemerals.cjs");
const { generateWorkflowIdMarker } = require("./generate_footer.cjs");
const { parseBoolTemplatable } = require("./templatable.cjs");
const { generateFooterWithMessages } = require("./messages_footer.cjs");
const { generateHistoryUrl } = require("./generate_history_link.cjs");
const { normalizeBranchName } = require("./normalize_branch_name.cjs");
const { pushExtraEmptyCommit } = require("./extra_empty_commit.cjs");
const { createCheckoutManager } = require("./dynamic_checkout.cjs");
const { getBaseBranch } = require("./get_base_branch.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { buildWorkflowRunUrl } = require("./workflow_metadata_helpers.cjs");
const { checkFileProtection } = require("./manifest_file_helpers.cjs");
const { renderTemplate } = require("./messages_core.cjs");
const { COPILOT_REVIEWER_BOT, FAQ_CREATE_PR_PERMISSIONS_URL } = require("./constants.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "create_pull_request";

/** @type {string} Label always added to fallback issues so the triage system can find them */
const MANAGED_FALLBACK_ISSUE_LABEL = "agentic-workflows";

/**
 * Merges the required fallback label with any workflow-configured labels,
 * deduplicating and filtering empty values.
 * @param {string[]} [labels]
 * @returns {string[]}
 */
function mergeFallbackIssueLabels(labels = []) {
  const normalizedLabels = labels
    .filter(label => !!label)
    .map(label => String(label).trim())
    .filter(label => label);
  return [...new Set([MANAGED_FALLBACK_ISSUE_LABEL, ...normalizedLabels])];
}

/**
 * Maximum limits for pull request parameters to prevent resource exhaustion.
 * These limits align with GitHub's API constraints and security best practices.
 */
/** @type {number} Maximum number of files allowed per pull request */
const MAX_FILES = 100;

/**
 * Enforces maximum limits on pull request parameters to prevent resource exhaustion attacks.
 * Per Safe Outputs specification requirement SEC-003, limits must be enforced before API calls.
 *
 * @param {string} patchContent - Patch content to validate
 * @throws {Error} When any limit is exceeded, with error code E003 and details
 */
function enforcePullRequestLimits(patchContent) {
  if (!patchContent || !patchContent.trim()) {
    return;
  }

  // Count files in patch by looking for "diff --git" lines
  const fileMatches = patchContent.match(/^diff --git /gm);
  const fileCount = fileMatches ? fileMatches.length : 0;

  // Check file count - max limit exceeded check
  if (fileCount > MAX_FILES) {
    throw new Error(`E003: Cannot create pull request with more than ${MAX_FILES} files (received ${fileCount})`);
  }
}

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
  const configReviewers = config.reviewers ? (Array.isArray(config.reviewers) ? config.reviewers : config.reviewers.split(",")).map(r => String(r).trim()).filter(r => r) : [];
  const draftDefault = parseBoolTemplatable(config.draft, true);
  const ifNoChanges = config.if_no_changes || "warn";
  const allowEmpty = parseBoolTemplatable(config.allow_empty, false);
  const autoMerge = parseBoolTemplatable(config.auto_merge, false);
  const preserveBranchName = config.preserve_branch_name === true;
  const expiresHours = config.expires ? parseInt(String(config.expires), 10) : 0;
  const maxCount = config.max || 1; // PRs are typically limited to 1
  const maxSizeKb = config.max_patch_size ? parseInt(String(config.max_patch_size), 10) : 1024;
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
  const githubClient = await createAuthenticatedGitHubClient(config);

  // Base branch from config (if set) - validated at factory level if explicit
  // Dynamic base branch resolution happens per-message after resolving the actual target repo
  const configBaseBranch = config.base_branch || null;

  // SECURITY: If base branch is explicitly configured, validate it at factory level
  if (configBaseBranch) {
    const normalizedConfigBase = normalizeBranchName(configBaseBranch);
    if (!normalizedConfigBase) {
      throw new Error(`Invalid baseBranch: sanitization resulted in empty string (original: "${configBaseBranch}")`);
    }
    if (configBaseBranch !== normalizedConfigBase) {
      throw new Error(`Invalid baseBranch: contains invalid characters (original: "${configBaseBranch}", normalized: "${normalizedConfigBase}")`);
    }
  }

  const includeFooter = parseBoolTemplatable(config.footer, true);
  const fallbackAsIssue = config.fallback_as_issue !== false; // Default to true (fallback enabled)

  // Environment validation - fail early if required variables are missing
  const workflowId = process.env.GH_AW_WORKFLOW_ID;
  if (!workflowId) {
    throw new Error("GH_AW_WORKFLOW_ID environment variable is required");
  }

  // Extract triggering issue number from context (for auto-linking PRs to issues)
  const triggeringIssueNumber = typeof context !== "undefined" && context.payload?.issue?.number && !context.payload?.issue?.pull_request ? context.payload.issue.number : undefined;

  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  core.info(`Base branch: ${configBaseBranch || "(dynamic - resolved per target repo)"}`);
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${Array.from(allowedRepos).join(", ")}`);
  }
  if (envLabels.length > 0) {
    core.info(`Default labels: ${envLabels.join(", ")}`);
  }
  if (configReviewers.length > 0) {
    core.info(`Configured reviewers: ${configReviewers.join(", ")}`);
  }
  if (titlePrefix) {
    core.info(`Title prefix: ${titlePrefix}`);
  }
  core.info(`Draft default: ${draftDefault}`);
  core.info(`If no changes: ${ifNoChanges}`);
  core.info(`Allow empty: ${allowEmpty}`);
  core.info(`Auto-merge: ${autoMerge}`);
  if (expiresHours > 0) {
    core.info(`Pull requests expire after: ${expiresHours} hours`);
  }
  core.info(`Max count: ${maxCount}`);
  core.info(`Max patch size: ${maxSizeKb} KB`);

  // Track how many items we've processed for max limit
  let processedCount = 0;

  // Create checkout manager for multi-repo support
  // Token is available via GITHUB_TOKEN environment variable (set by the workflow job)
  const checkoutToken = process.env.GITHUB_TOKEN;
  const checkoutManager = checkoutToken ? createCheckoutManager(checkoutToken, { defaultBaseBranch: configBaseBranch }) : null;

  // Log multi-repo support status
  if (allowedRepos.size > 0 && checkoutManager) {
    core.info(`Multi-repo support enabled: can switch between repos in allowed-repos list`);
  } else if (allowedRepos.size > 0 && !checkoutManager) {
    core.warning(`Multi-repo support disabled: GITHUB_TOKEN not available for dynamic checkout`);
  }

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

    let temporaryId;
    if (pullRequestItem.temporary_id !== undefined && pullRequestItem.temporary_id !== null) {
      if (typeof pullRequestItem.temporary_id !== "string") {
        core.warning(`Skipping create_pull_request: temporary_id must be a string (got ${typeof pullRequestItem.temporary_id})`);
        return {
          success: false,
          error: `temporary_id must be a string (got ${typeof pullRequestItem.temporary_id})`,
        };
      }

      const rawTemporaryId = pullRequestItem.temporary_id.trim();
      const normalized = rawTemporaryId.startsWith("#") ? rawTemporaryId.substring(1).trim() : rawTemporaryId;

      if (!isTemporaryId(normalized)) {
        core.warning(
          `Skipping create_pull_request: Invalid temporary_id format: '${pullRequestItem.temporary_id}'. Temporary IDs must be in format 'aw_' followed by 3 to 12 alphanumeric characters (A-Za-z0-9). Example: 'aw_abc' or 'aw_Test123'`
        );
        return {
          success: false,
          error: `Invalid temporary_id format: '${pullRequestItem.temporary_id}'. Temporary IDs must be in format 'aw_' followed by 3 to 12 alphanumeric characters (A-Za-z0-9). Example: 'aw_abc' or 'aw_Test123'`,
        };
      }

      temporaryId = normalized.toLowerCase();
    }

    core.info(`Processing create_pull_request: title=${pullRequestItem.title || "No title"}, bodyLength=${pullRequestItem.body?.length || 0}`);

    // Determine the patch file path from the message (set by the MCP server handler)
    const patchFilePath = pullRequestItem.patch_path;
    core.info(`Patch file path: ${patchFilePath || "(not set)"}`);

    // Resolve and validate target repository
    const repoResult = resolveAndValidateRepo(pullRequestItem, defaultTargetRepo, allowedRepos, "pull request");
    if (!repoResult.success) {
      core.warning(`Skipping pull request: ${repoResult.error}`);
      return {
        success: false,
        error: repoResult.error,
      };
    }
    const { repo: itemRepo, repoParts } = repoResult;
    core.info(`Target repository: ${itemRepo}`);

    // Resolve base branch for this target repository
    // Use config value if set, otherwise resolve dynamically for the specific target repo
    // Dynamic resolution is needed for issue_comment events on PRs where the base branch
    // is not available in GitHub Actions expressions and requires an API call
    // NOTE: Must be resolved before checkout so cross-repo checkout uses the correct branch
    let baseBranch = configBaseBranch || (await getBaseBranch(repoParts));

    // Multi-repo support: Switch checkout to target repo if different from current
    // This enables creating PRs in multiple repos from a single workflow run
    if (checkoutManager && itemRepo) {
      const switchResult = await checkoutManager.switchTo(itemRepo, { baseBranch });
      if (!switchResult.success) {
        core.warning(`Failed to switch to repository ${itemRepo}: ${switchResult.error}`);
        return {
          success: false,
          error: `Failed to checkout repository ${itemRepo}: ${switchResult.error}`,
        };
      }
      if (switchResult.switched) {
        core.info(`Switched checkout to repository: ${itemRepo}`);
      }
    }

    // SECURITY: Sanitize dynamically resolved base branch to prevent shell injection
    const originalBaseBranch = baseBranch;
    baseBranch = normalizeBranchName(baseBranch);
    if (!baseBranch) {
      return {
        success: false,
        error: `Invalid base branch: sanitization resulted in empty string (original: "${originalBaseBranch}")`,
      };
    }
    if (originalBaseBranch !== baseBranch) {
      return {
        success: false,
        error: `Invalid base branch: contains invalid characters (original: "${originalBaseBranch}", normalized: "${baseBranch}")`,
      };
    }
    core.info(`Base branch for ${itemRepo}: ${baseBranch}`);

    // Check if patch file exists and has valid content
    if (!patchFilePath || !fs.existsSync(patchFilePath)) {
      // If allow-empty is enabled, we can proceed without a patch file
      if (allowEmpty) {
        core.info("No patch file found, but allow-empty is enabled - will create empty PR");
      } else {
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
          return { success: true, staged: true };
        }

        switch (ifNoChanges) {
          case "error":
            return { success: false, error: message };

          case "ignore":
            // Silent success - no console output
            return { success: false, skipped: true };

          case "warn":
          default:
            core.warning(message);
            return { success: false, error: message, skipped: true };
        }
      }
    }

    let patchContent = "";
    let isEmpty = true;

    if (patchFilePath && fs.existsSync(patchFilePath)) {
      patchContent = fs.readFileSync(patchFilePath, "utf8");
      isEmpty = !patchContent || !patchContent.trim();
    }

    // Enforce max limits on patch before processing
    try {
      enforcePullRequestLimits(patchContent);
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.warning(`Pull request limit exceeded: ${errorMessage}`);
      return { success: false, error: errorMessage };
    }

    // Check for actual error conditions (but allow empty patches as valid noop)
    if (patchContent.includes("Failed to generate patch")) {
      // If allow-empty is enabled, ignore patch errors and proceed
      if (allowEmpty) {
        core.info("Patch file contains error, but allow-empty is enabled - will create empty PR");
        patchContent = "";
        isEmpty = true;
      } else {
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
          return { success: true, staged: true };
        }

        switch (ifNoChanges) {
          case "error":
            return { success: false, error: message };

          case "ignore":
            // Silent success - no console output
            return { success: false, skipped: true };

          case "warn":
          default:
            core.warning(message);
            return { success: false, error: message, skipped: true };
        }
      }
    }

    // Validate patch size (unless empty)
    if (!isEmpty) {
      // maxSizeKb is already extracted from config at the top
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
          return { success: true, staged: true };
        }

        return { success: false, error: message };
      }

      core.info("Patch size validation passed");
    }

    // Check file protection: allowlist (strict) or protected-files policy.
    /** @type {string[] | null} Protected files that trigger fallback-to-issue handling */
    let manifestProtectionFallback = null;
    /** @type {unknown} */
    let manifestProtectionPushFailedError = null;
    if (!isEmpty) {
      const protection = checkFileProtection(patchContent, config);
      if (protection.action === "deny") {
        const filesStr = protection.files.join(", ");
        const message =
          protection.source === "allowlist"
            ? `Cannot create pull request: patch modifies files outside the allowed-files list (${filesStr}). Add the files to the allowed-files configuration field or remove them from the patch.`
            : `Cannot create pull request: patch modifies protected files (${filesStr}). Add them to the allowed-files configuration field or set protected-files: fallback-to-issue to create a review issue instead.`;
        core.error(message);
        return { success: false, error: message };
      }
      if (protection.action === "fallback") {
        manifestProtectionFallback = protection.files;
        core.warning(`Protected file protection triggered (fallback-to-issue): ${protection.files.join(", ")}. Will create review issue instead of pull request.`);
      }
    }

    if (isEmpty && !isStaged && !allowEmpty) {
      const message = "Patch file is empty - no changes to apply (noop operation)";

      switch (ifNoChanges) {
        case "error":
          return { success: false, error: "No changes to push - failing as configured by if-no-changes: error" };

        case "ignore":
          // Silent success - no console output
          return { success: false, skipped: true };

        case "warn":
        default:
          core.warning(message);
          return { success: false, error: message, skipped: true };
      }
    }

    if (!isEmpty) {
      core.info("Patch content validation passed");
    } else if (allowEmpty) {
      core.info("Patch file is empty - processing empty PR creation (allow-empty is enabled)");
    } else {
      core.info("Patch file is empty - processing noop operation");
    }

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

      if (patchFilePath && fs.existsSync(patchFilePath)) {
        const patchStats = fs.readFileSync(patchFilePath, "utf8");
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
      return { success: true, staged: true };
    }

    // Extract title, body, and branch from the JSON item
    let title = pullRequestItem.title.trim();
    let processedBody = pullRequestItem.body;

    // Replace temporary ID references in the body with resolved issue/PR numbers
    // This allows PRs to reference issues created earlier in the same workflow
    // by using temporary IDs like #aw_123abc456def
    if (resolvedTemporaryIds && Object.keys(resolvedTemporaryIds).length > 0) {
      // Convert object to Map for compatibility with replaceTemporaryIdReferences
      const tempIdMap = new Map(Object.entries(resolvedTemporaryIds));
      processedBody = replaceTemporaryIdReferences(processedBody, tempIdMap, itemRepo);
      core.info(`Resolved ${tempIdMap.size} temporary ID references in PR body`);
    }

    // Remove duplicate title from description if it starts with a header matching the title
    processedBody = removeDuplicateTitleFromDescription(title, processedBody);

    // Auto-add "Fixes #N" closing keyword if triggered from an issue and not already present.
    // This ensures the triggering issue is auto-closed when the PR is merged.
    // Agents are instructed to include this but don't reliably do so.
    if (triggeringIssueNumber) {
      const hasClosingKeyword = /(?:fix|fixes|fixed|close|closes|closed|resolve|resolves|resolved)\s+#\d+/i.test(processedBody);
      if (!hasClosingKeyword) {
        processedBody = processedBody.trimEnd() + `\n\n- Fixes #${triggeringIssueNumber}`;
        core.info(`Auto-added "Fixes #${triggeringIssueNumber}" closing keyword to PR body as bullet point`);
      }
    }

    let bodyLines = processedBody.split("\n");
    let branchName = pullRequestItem.branch ? pullRequestItem.branch.trim() : null;
    const randomHex = crypto.randomBytes(8).toString("hex");

    // SECURITY: Sanitize branch name to prevent shell injection (CWE-78)
    // Branch names from user input must be normalized before use in git commands.
    // When preserve-branch-name is disabled (default), a random salt suffix is
    // appended to avoid collisions.
    if (branchName) {
      const originalBranchName = branchName;
      branchName = normalizeBranchName(branchName, preserveBranchName ? null : randomHex);

      // Validate it's not empty after normalization
      if (!branchName) {
        throw new Error(`Invalid branch name: sanitization resulted in empty string (original: "${originalBranchName}")`);
      }

      if (preserveBranchName) {
        core.info(`Using branch name from JSONL without salt suffix (preserve-branch-name enabled): ${branchName}`);
      } else {
        core.info(`Using branch name from JSONL with added salt: ${branchName}`);
      }
      if (originalBranchName !== branchName) {
        core.info(`Branch name sanitized: "${originalBranchName}" -> "${branchName}"`);
      }
    }

    // If no title was found, use a default
    if (!title) {
      title = "Agent Output";
    }

    // Sanitize title for Unicode security and remove any duplicate prefixes
    title = sanitizeTitle(title, titlePrefix);

    // Apply title prefix (only if it doesn't already exist)
    title = applyTitlePrefix(title, titlePrefix);

    // Add AI disclaimer with workflow name and run url
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
    const workflowId = process.env.GH_AW_WORKFLOW_ID || "";
    const runUrl = buildWorkflowRunUrl(context, context.repo);
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE ?? "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL ?? "";
    const triggeringPRNumber = context.payload.pull_request?.number;
    const triggeringDiscussionNumber = context.payload.discussion?.number;

    // Add fingerprint comment if present
    const trackerIDComment = getTrackerID("markdown");
    if (trackerIDComment) {
      bodyLines.push(trackerIDComment);
    }

    // Snapshot the body content (without footer) for use in protected-files fallback ordering.
    // The protected-files section must appear before the footer (including guard notices such as
    // the integrity-filtering note) so that the footer always comes last in the issue body.
    const mainBodyContent = bodyLines.join("\n").trim();

    // Generate footer using messages template system (respects custom messages.footer config)
    // When footer is disabled, only add XML markers (no visible footer content)
    const footerParts = [];
    if (includeFooter) {
      const historyUrl = generateHistoryUrl({
        owner: repoParts.owner,
        repo: repoParts.repo,
        itemType: "pull_request",
        workflowId,
        serverUrl: context.serverUrl,
      });
      let footer = generateFooterWithMessages(workflowName, runUrl, workflowSource, workflowSourceURL, triggeringIssueNumber, triggeringPRNumber, triggeringDiscussionNumber, historyUrl).trimEnd();
      footer = addExpirationToFooter(footer, expiresHours, "Pull Request");
      if (expiresHours > 0) {
        footer += "\n\n<!-- gh-aw-expires-type: pull-request -->";
      }
      bodyLines.push(``, ``, footer);
      footerParts.push(footer);
    }

    // Add standalone workflow-id marker for searchability (consistent with comments)
    // Always add XML markers even when footer is disabled
    if (workflowId) {
      const workflowIdMarker = generateWorkflowIdMarker(workflowId);
      // Add to bodyLines for the normal PR body path.
      // Add to footerParts so the fallback issue body places it after the protected-files section.
      bodyLines.push(``, workflowIdMarker);
      footerParts.push(workflowIdMarker);
    }

    bodyLines.push("");

    // Prepare the body content
    const body = bodyLines.join("\n").trim();
    // Footer section (footer + workflow-id marker) used when ordering protected-files notices
    const footerContent = footerParts.join("\n\n");

    // Build labels array - merge config labels with message labels
    let labels = [...envLabels];
    if (pullRequestItem.labels && Array.isArray(pullRequestItem.labels)) {
      labels = [...labels, ...pullRequestItem.labels];
    }
    labels = labels
      .filter(label => !!label)
      .map(label => String(label).trim())
      .filter(label => label);

    // Configuration enforces draft as a policy, not a fallback (consistent with autoMerge/allowEmpty)
    const draft = draftDefault;
    if (pullRequestItem.draft !== undefined && pullRequestItem.draft !== draftDefault) {
      core.warning(
        `Agent requested draft: ${pullRequestItem.draft}, but configuration enforces draft: ${draftDefault}. ` +
          `Configuration takes precedence for security. To change this, update safe-outputs.create-pull-request.draft in the workflow file.`
      );
    }

    core.info(`Creating pull request with title: ${title}`);
    core.info(`Labels: ${JSON.stringify(labels)}`);
    core.info(`Draft: ${draft}`);
    core.info(`Body length: ${body.length}`);

    // When no branch name was provided by the agent, generate a unique one.
    if (!branchName) {
      core.info("No branch name provided in JSONL, generating unique branch name");
      branchName = `${workflowId}-${randomHex}`;
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
    // Track number of new commits pushed so we can restrict the extra empty commit
    // to branches with exactly one new commit (security: prevents use of CI trigger
    // token on multi-commit branches where workflow files may have been modified).
    let newCommitCount = 0;
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
      // Use --3way to handle cross-repo patches where the patch base may differ from target repo
      // This allows git to resolve create-vs-modify mismatches when a file exists in target but not source
      try {
        await exec.exec(`git am --3way ${patchFilePath}`);
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

        return { success: false, error: "Failed to apply patch" };
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

        await pushSignedCommits({
          githubClient,
          owner: repoParts.owner,
          repo: repoParts.repo,
          branch: branchName,
          baseRef: `origin/${baseBranch}`,
          cwd: process.cwd(),
        });
        core.info("Changes pushed to branch");

        // Count new commits on PR branch relative to base, used to restrict
        // the extra empty CI-trigger commit to exactly 1 new commit.
        try {
          const { stdout: countStr } = await exec.getExecOutput("git", ["rev-list", "--count", `origin/${baseBranch}..HEAD`]);
          newCommitCount = parseInt(countStr.trim(), 10);
          core.info(`${newCommitCount} new commit(s) on branch relative to origin/${baseBranch}`);
        } catch {
          // Non-fatal - newCommitCount stays 0, extra empty commit will be skipped
          core.info("Could not count new commits - extra empty commit will be skipped");
        }
      } catch (pushError) {
        // Push failed - create fallback issue instead of PR (if fallback is enabled)
        core.error(`Git push failed: ${pushError instanceof Error ? pushError.message : String(pushError)}`);

        if (manifestProtectionFallback) {
          // Push failed specifically for a protected-file modification. Don't create
          // a generic push-failed issue — fall through to the manifestProtectionFallback
          // block below, which will create the proper protected-file review issue with
          // patch artifact download instructions (since the branch was not pushed).
          core.warning("Git push failed for protected-file modification - deferring to protected-file review issue");
          manifestProtectionPushFailedError = pushError;
        } else if (!fallbackAsIssue) {
          // Fallback is disabled - return error without creating issue
          core.error("fallback-as-issue is disabled - not creating fallback issue");
          const error = `Failed to push changes: ${pushError instanceof Error ? pushError.message : String(pushError)}`;
          return {
            success: false,
            error,
            error_type: "push_failed",
          };
        } else {
          core.warning("Git push operation failed - creating fallback issue instead of pull request");

          const runUrl = buildWorkflowRunUrl(context, context.repo);
          const runId = context.runId;

          // Read patch content for preview
          let patchPreview = "";
          if (patchFilePath && fs.existsSync(patchFilePath)) {
            const patchContent = fs.readFileSync(patchFilePath, "utf8");
            patchPreview = generatePatchPreview(patchContent);
          }

          const patchFileName = patchFilePath ? patchFilePath.replace("/tmp/gh-aw/", "") : "aw-unknown.patch";
          const fallbackBody = `${body}

---

> [!NOTE]
> This was originally intended as a pull request, but the git push operation failed.
>
> **Workflow Run:** [View run details and download patch artifact](${runUrl})
>
> The patch file is available in the \`agent-artifacts\` artifact in the workflow run linked above.

To create a pull request with the changes:

\`\`\`sh
# Download the artifact from the workflow run
gh run download ${runId} -n agent-artifacts -D /tmp/agent-artifacts-${runId}

# Create a new branch
git checkout -b ${branchName}

# Apply the patch (--3way handles cross-repo patches where files may already exist)
git am --3way /tmp/agent-artifacts-${runId}/${patchFileName}

# Push the branch to origin
git push origin ${branchName}

# Create the pull request
gh pr create --title '${title}' --base ${baseBranch} --head ${branchName} --repo ${repoParts.owner}/${repoParts.repo}
\`\`\`
${patchPreview}`;

          try {
            const { data: issue } = await githubClient.rest.issues.create({
              owner: repoParts.owner,
              repo: repoParts.repo,
              title: title,
              body: fallbackBody,
              labels: mergeFallbackIssueLabels(labels),
            });

            core.info(`Created fallback issue #${issue.number}: ${issue.html_url}`);

            // Update the activation comment with issue link (if a comment was created)
            //
            // NOTE: we pass 'github' (global octokit) instead of githubClient (repo-scoped octokit) because the issue is created
            // in the same repo as the activation, so the global client has the correct context for updating the comment.
            await updateActivationComment(github, context, core, issue.html_url, issue.number, "issue");

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

            return {
              success: true,
              fallback_used: true,
              push_failed: true,
              issue_number: issue.number,
              issue_url: issue.html_url,
              branch_name: branchName,
              repo: itemRepo,
            };
          } catch (issueError) {
            const error = `Failed to push and failed to create fallback issue. Push error: ${pushError instanceof Error ? pushError.message : String(pushError)}. Issue error: ${issueError instanceof Error ? issueError.message : String(issueError)}`;
            core.error(error);
            return {
              success: false,
              error,
            };
          }
        } // end else (generic push-failed fallback)
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

          await pushSignedCommits({
            githubClient,
            owner: repoParts.owner,
            repo: repoParts.repo,
            branch: branchName,
            baseRef: `origin/${baseBranch}`,
            cwd: process.cwd(),
          });
          core.info("Empty branch pushed successfully");

          // Count new commits (will be 1 from the Initialize commit)
          try {
            const { stdout: countStr } = await exec.getExecOutput("git", ["rev-list", "--count", `origin/${baseBranch}..HEAD`]);
            newCommitCount = parseInt(countStr.trim(), 10);
            core.info(`${newCommitCount} new commit(s) on branch relative to origin/${baseBranch}`);
          } catch {
            // Non-fatal - newCommitCount stays 0, extra empty commit will be skipped
            core.info("Could not count new commits - extra empty commit will be skipped");
          }
        } catch (pushError) {
          const error = `Failed to push empty branch: ${pushError instanceof Error ? pushError.message : String(pushError)}`;
          core.error(error);
          return {
            success: false,
            error,
          };
        }
      } else {
        // For empty patches without allow-empty, handle if-no-changes configuration
        const message = "No changes to apply - noop operation completed successfully";

        switch (ifNoChanges) {
          case "error":
            return { success: false, error: "No changes to apply - failing as configured by if-no-changes: error" };

          case "ignore":
            // Silent success - no console output
            return { success: false, skipped: true };

          case "warn":
          default:
            core.warning(message);
            return { success: false, error: message, skipped: true };
        }
      }
    }

    // Protected file protection – fallback-to-issue path:
    // The patch has been applied (and pushed, unless manifestProtectionPushFailedError is set).
    // Instead of creating a pull request, we create a review issue so a human can carefully
    // inspect the protected file changes before merging.
    // - Normal case (push succeeded): provides a GitHub compare URL to click and create the PR.
    // - Push-failed case: push was rejected (e.g. missing `workflows` permission); provides
    //   patch artifact download instructions instead of the compare URL.
    if (manifestProtectionFallback) {
      const allFound = manifestProtectionFallback;
      const filesFormatted = allFound.map(f => `\`${f}\``).join(", ");

      let fallbackBody;
      if (manifestProtectionPushFailedError) {
        // Push failed — branch not on remote, so compare URL is unavailable.
        // Use the push-failed template with artifact download instructions.
        const runId = context.runId;
        const patchFileName = patchFilePath ? patchFilePath.replace("/tmp/gh-aw/", "") : "aw-unknown.patch";
        const pushFailedTemplatePath = `${process.env.RUNNER_TEMP}/gh-aw/prompts/manifest_protection_push_failed_fallback.md`;
        const pushFailedTemplate = fs.readFileSync(pushFailedTemplatePath, "utf8");
        fallbackBody = renderTemplate(pushFailedTemplate, {
          main_body: mainBodyContent,
          footer: footerContent,
          files: filesFormatted,
          run_id: String(runId),
          branch_name: branchName,
          base_branch: baseBranch,
          patch_file: patchFileName,
          title,
          repo: `${repoParts.owner}/${repoParts.repo}`,
        });
      } else {
        // Normal case — push succeeded, provide compare URL.
        const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
        const encodedBase = baseBranch.split("/").map(encodeURIComponent).join("/");
        const encodedHead = branchName.split("/").map(encodeURIComponent).join("/");
        const createPrUrl = `${githubServer}/${repoParts.owner}/${repoParts.repo}/compare/${encodedBase}...${encodedHead}?expand=1&title=${encodeURIComponent(title)}`;
        const templatePath = `${process.env.RUNNER_TEMP}/gh-aw/prompts/manifest_protection_create_pr_fallback.md`;
        const template = fs.readFileSync(templatePath, "utf8");
        fallbackBody = renderTemplate(template, {
          main_body: mainBodyContent,
          footer: footerContent,
          files: filesFormatted,
          create_pr_url: createPrUrl,
        });
      }

      try {
        const { data: issue } = await githubClient.rest.issues.create({
          owner: repoParts.owner,
          repo: repoParts.repo,
          title: title,
          body: fallbackBody,
          labels: mergeFallbackIssueLabels(labels),
        });

        core.info(`Created protected-file-protection review issue #${issue.number}: ${issue.html_url}`);

        await updateActivationComment(github, context, core, issue.html_url, issue.number, "issue");

        return {
          success: true,
          fallback_used: true,
          issue_number: issue.number,
          issue_url: issue.html_url,
          branch_name: branchName,
          repo: itemRepo,
        };
      } catch (issueError) {
        const error = `Protected file protection: failed to create review issue. Error: ${issueError instanceof Error ? issueError.message : String(issueError)}`;
        core.error(error);
        return { success: false, error };
      }
    }

    // Try to create the pull request, with fallback to issue creation
    try {
      const { data: pullRequest } = await githubClient.rest.pulls.create({
        owner: repoParts.owner,
        repo: repoParts.repo,
        title: title,
        body: body,
        head: branchName,
        base: baseBranch,
        draft: draft,
      });

      core.info(`Created pull request #${pullRequest.number}: ${pullRequest.html_url}`);

      // Add labels if specified
      if (labels.length > 0) {
        await githubClient.rest.issues.addLabels({
          owner: repoParts.owner,
          repo: repoParts.repo,
          issue_number: pullRequest.number,
          labels: labels,
        });
        core.info(`Added labels to pull request: ${JSON.stringify(labels)}`);
      }

      // Add configured reviewers if specified
      if (configReviewers.length > 0) {
        const hasCopilot = configReviewers.includes("copilot");
        const otherReviewers = configReviewers.filter(r => r !== "copilot");

        if (otherReviewers.length > 0) {
          core.info(`Requesting ${otherReviewers.length} reviewer(s) for pull request #${pullRequest.number}: ${JSON.stringify(otherReviewers)}`);
          try {
            await githubClient.rest.pulls.requestReviewers({
              owner: repoParts.owner,
              repo: repoParts.repo,
              pull_number: pullRequest.number,
              reviewers: otherReviewers,
            });
            core.info(`Requested reviewers for pull request #${pullRequest.number}: ${JSON.stringify(otherReviewers)}`);
          } catch (reviewerError) {
            core.warning(`Failed to request reviewers for PR #${pullRequest.number}: ${reviewerError instanceof Error ? reviewerError.message : String(reviewerError)}`);
          }
        }

        if (hasCopilot) {
          core.info(`Requesting copilot as reviewer for pull request #${pullRequest.number}`);
          try {
            await githubClient.rest.pulls.requestReviewers({
              owner: repoParts.owner,
              repo: repoParts.repo,
              pull_number: pullRequest.number,
              reviewers: [COPILOT_REVIEWER_BOT],
            });
            core.info(`Requested copilot as reviewer for pull request #${pullRequest.number}`);
          } catch (copilotError) {
            core.warning(`Failed to request copilot as reviewer for PR #${pullRequest.number}: ${copilotError instanceof Error ? copilotError.message : String(copilotError)}`);
          }
        }
      }

      // Enable auto-merge if configured
      if (autoMerge) {
        try {
          await githubClient.graphql(
            `mutation($prId: ID!) {
              enablePullRequestAutoMerge(input: {pullRequestId: $prId}) {
                pullRequest {
                  id
                }
              }
            }`,
            {
              prId: pullRequest.node_id,
            }
          );
          core.info(`Enabled auto-merge for pull request #${pullRequest.number}`);
        } catch (autoMergeError) {
          core.warning(`Failed to enable auto-merge for PR #${pullRequest.number}: ${autoMergeError instanceof Error ? autoMergeError.message : String(autoMergeError)}`);
        }
      }

      // Update the activation comment with PR link (if a comment was created)
      //
      // NOTE: we pass 'github' (global octokit) instead of githubClient (repo-scoped octokit) because the issue is created
      // in the same repo as the activation, so the global client has the correct context for updating the comment.
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

      // Push an extra empty commit if a token is configured and exactly 1 new commit was pushed.
      // This works around the GITHUB_TOKEN limitation where pushes don't trigger CI events.
      // Restricting to exactly 1 new commit prevents the CI trigger token being used on
      // multi-commit branches where workflow files may have been iteratively modified.
      const ciTriggerResult = await pushExtraEmptyCommit({
        branchName,
        repoOwner: repoParts.owner,
        repoName: repoParts.repo,
        newCommitCount,
      });
      if (ciTriggerResult.success && !ciTriggerResult.skipped) {
        core.info("Extra empty commit pushed - CI checks should start shortly");
      }

      // Return success with PR details
      return {
        success: true,
        pull_request_number: pullRequest.number,
        pull_request_url: pullRequest.html_url,
        branch_name: branchName,
        temporary_id: temporaryId,
        repo: itemRepo,
      };
    } catch (prError) {
      const errorMessage = prError instanceof Error ? prError.message : String(prError);
      core.warning(`Failed to create pull request: ${errorMessage}`);

      // Check if the error is the specific "GitHub actions is not permitted to create or approve pull requests" error
      if (errorMessage.includes("GitHub Actions is not permitted to create or approve pull requests")) {
        core.error(`Permission error: GitHub Actions is not permitted to create or approve pull requests. See FAQ: ${FAQ_CREATE_PR_PERMISSIONS_URL}`);

        // Branch has already been pushed - create a fallback issue with a link to create the PR via GitHub UI
        const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
        // Encode branch name path segments individually to preserve '/' while encoding other special characters
        const encodedBase = baseBranch.split("/").map(encodeURIComponent).join("/");
        const encodedHead = branchName.split("/").map(encodeURIComponent).join("/");
        const createPrUrl = `${githubServer}/${repoParts.owner}/${repoParts.repo}/compare/${encodedBase}...${encodedHead}?expand=1&title=${encodeURIComponent(title)}`;

        // Read patch content for preview
        let patchPreview = "";
        if (patchFilePath && fs.existsSync(patchFilePath)) {
          const patchContent = fs.readFileSync(patchFilePath, "utf8");
          patchPreview = generatePatchPreview(patchContent);
        }

        const fallbackTemplatePath = `${process.env.RUNNER_TEMP}/gh-aw/prompts/pr_permission_denied_fallback.md`;
        const fallbackTemplate = fs.readFileSync(fallbackTemplatePath, "utf8");
        const fallbackBody = renderTemplate(fallbackTemplate, {
          body,
          branch_name: branchName,
          create_pr_url: createPrUrl,
          faq_url: FAQ_CREATE_PR_PERMISSIONS_URL,
          patch_preview: patchPreview,
        });

        try {
          const { data: issue } = await githubClient.rest.issues.create({
            owner: repoParts.owner,
            repo: repoParts.repo,
            title: title,
            body: fallbackBody,
            labels: mergeFallbackIssueLabels(labels),
          });

          core.info(`Created fallback issue #${issue.number}: ${issue.html_url}`);

          await updateActivationComment(github, context, core, issue.html_url, issue.number, "issue");

          return {
            success: true,
            fallback_used: true,
            issue_number: issue.number,
            issue_url: issue.html_url,
            branch_name: branchName,
            repo: itemRepo,
          };
        } catch (issueError) {
          const error = `Failed to create pull request (permission denied) and failed to create fallback issue. PR error: ${errorMessage}. Issue error: ${issueError instanceof Error ? issueError.message : String(issueError)}`;
          core.error(error);
          return {
            success: false,
            error,
            error_type: "permission_denied",
          };
        }
      }

      if (!fallbackAsIssue) {
        // Fallback is disabled - return error without creating issue
        core.error("fallback-as-issue is disabled - not creating fallback issue");
        return {
          success: false,
          error: errorMessage,
          error_type: "pr_creation_failed",
        };
      }

      core.info("Falling back to creating an issue instead");

      // Create issue as fallback with enhanced body content
      const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
      const branchUrl = context.payload.repository ? `${context.payload.repository.html_url}/tree/${branchName}` : `${githubServer}/${repoParts.owner}/${repoParts.repo}/tree/${branchName}`;

      // Read patch content for preview
      let patchPreview = "";
      if (patchFilePath && fs.existsSync(patchFilePath)) {
        const patchContent = fs.readFileSync(patchFilePath, "utf8");
        patchPreview = generatePatchPreview(patchContent);
      }

      const fallbackBody = `${body}

---

> [!NOTE]
> This was originally intended as a pull request, but PR creation failed. The changes have been pushed to the branch [\`${branchName}\`](${branchUrl}).
>
> **Original error:** ${errorMessage}

To create the pull request manually:

\`\`\`sh
gh pr create --title "${title}" --base ${baseBranch} --head ${branchName} --repo ${repoParts.owner}/${repoParts.repo}
\`\`\`
${patchPreview}`;

      try {
        const { data: issue } = await githubClient.rest.issues.create({
          owner: repoParts.owner,
          repo: repoParts.repo,
          title: title,
          body: fallbackBody,
          labels: mergeFallbackIssueLabels(labels),
        });

        core.info(`Created fallback issue #${issue.number}: ${issue.html_url}`);

        // Update the activation comment with issue link (if a comment was created)
        // NOTE: we pass 'github' (global octokit) instead of githubClient (repo-scoped octokit) because the issue is created
        // in the same repo as the activation, so the global client has the correct context for updating the comment.
        await updateActivationComment(github, context, core, issue.html_url, issue.number, "issue");

        // Return success with fallback flag
        return {
          success: true,
          fallback_used: true,
          issue_number: issue.number,
          issue_url: issue.html_url,
          branch_name: branchName,
          repo: itemRepo,
        };
      } catch (issueError) {
        const error = `Failed to create both pull request and fallback issue. PR error: ${errorMessage}. Issue error: ${issueError instanceof Error ? issueError.message : String(issueError)}`;
        core.error(error);
        return {
          success: false,
          error,
        };
      }
    }
  }; // End of handleCreatePullRequest
} // End of main

module.exports = { main, enforcePullRequestLimits };
