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
const { replaceTemporaryIdReferences, getOrGenerateTemporaryId } = require("./temporary_id.cjs");
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
const { renderTemplateFromFile, buildProtectedFileList, encodePathSegments } = require("./messages_core.cjs");
const { COPILOT_REVIEWER_BOT, FAQ_CREATE_PR_PERMISSIONS_URL, MAX_ASSIGNEES } = require("./constants.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { withRetry, isTransientError } = require("./error_recovery.cjs");
const { tryEnforceArrayLimit } = require("./limit_enforcement_helpers.cjs");
const { findAgent, getIssueDetails, assignAgentToIssue } = require("./assign_agent_helpers.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/**
 * Creates an authenticated GitHub client for copilot assignment on fallback issues.
 * Prefers the agent-specific token (GH_AW_ASSIGN_TO_AGENT_TOKEN) because the Copilot
 * assignment API requires a PAT rather than a GitHub App token.
 *
 * Token priority:
 *   1. config["github-token"] — explicit per-handler override
 *   2. GH_AW_ASSIGN_TO_AGENT_TOKEN — injected by the compiler when copilot is in assignees
 *   3. global github — step-level token (fallback when no agent token is available)
 *
 * @param {Object} config - Handler configuration
 * @returns {Promise<Object>} Authenticated GitHub client
 */
async function createCopilotAssignmentClient(config) {
  const token = config["github-token"] || process.env.GH_AW_ASSIGN_TO_AGENT_TOKEN;
  if (!token) {
    core.debug("No dedicated agent token configured — using step-level github client for copilot assignment");
    return github;
  }
  core.info("Using dedicated github client for copilot assignment");
  const { getOctokit } = await import("@actions/github");
  return getOctokit(token);
}

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "create_pull_request";

/** @type {string} Label always added to fallback issues so the triage system can find them */
const MANAGED_FALLBACK_ISSUE_LABEL = "agentic-workflows";

/**
 * Determines if a label API error is transient and worth retrying.
 * Returns true for:
 *  - The GitHub race condition where a newly-created PR's node ID is not immediately
 *    resolvable via the REST/GraphQL bridge (unprocessable validation error).
 *  - Any standard transient error matched by {@link isTransientError} (network issues,
 *    rate limits, 5xx gateway errors, etc.).
 * @param {any} error - The error to check
 * @returns {boolean} True if the error is transient and should be retried
 */
function isLabelTransientError(error) {
  const msg = getErrorMessage(error);
  if (msg.includes("Could not resolve to a node with the global id")) {
    return true;
  }
  return isTransientError(error);
}

/** @type {number} Number of retry attempts for label operations */
const LABEL_MAX_RETRIES = 3;
/** @type {number} Initial delay in ms before the first label retry (3 seconds) */
const LABEL_INITIAL_DELAY_MS = 3000;

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
 * Sanitizes configured assignees for fallback issue creation.
 * Filters invalid values, removes the special "copilot" username (not a valid GitHub user
 * for issue assignment), and enforces the MAX_ASSIGNEES limit.
 * Returns null (no assignees field) if the sanitized list is empty.
 * @param {string[]} assignees - Raw assignees from config
 * @returns {string[] | null} Sanitized assignees or null if none remain
 */
function sanitizeFallbackAssignees(assignees) {
  if (!assignees || assignees.length === 0) {
    return null;
  }
  const sanitized = assignees
    .filter(a => typeof a === "string")
    .map(a => a.trim())
    .filter(a => a.length > 0 && a.toLowerCase() !== "copilot");

  if (sanitized.length === 0) {
    return null;
  }

  const limitResult = tryEnforceArrayLimit(sanitized, MAX_ASSIGNEES, "assignees");
  if (!limitResult.success) {
    core.warning(`Assignees limit exceeded for fallback issue: ${limitResult.error}. Using first ${MAX_ASSIGNEES}.`);
    return sanitized.slice(0, MAX_ASSIGNEES);
  }

  return sanitized;
}

/**
 * Creates a fallback GitHub issue, retrying without assignees if the API rejects them.
 * This ensures fallback issue creation remains reliable even if an assignee username
 * is invalid or the repository does not have that collaborator.
 * @param {object} githubClient - Authenticated GitHub client
 * @param {{owner: string, repo: string}} repoParts - Repository owner and name
 * @param {string} title - Issue title
 * @param {string} body - Issue body
 * @param {string[]} labels - Issue labels
 * @param {string[] | null} assignees - Sanitized assignees (null = omit field)
 * @returns {Promise<any>}
 */
async function createFallbackIssue(githubClient, repoParts, title, body, labels, assignees) {
  const payload = {
    owner: repoParts.owner,
    repo: repoParts.repo,
    title,
    body,
    labels,
    ...(assignees && assignees.length > 0 && { assignees }),
  };

  try {
    return await githubClient.rest.issues.create(payload);
  } catch (error) {
    const status = typeof error === "object" && error !== null && "status" in error ? error.status : undefined;
    const message = getErrorMessage(error).toLowerCase();
    const isAssigneeError = status === 422 && (message.includes("assignee") || message.includes("assignees") || message.includes("unprocessable"));
    if (isAssigneeError && assignees && assignees.length > 0) {
      core.warning(`Fallback issue creation failed due to assignee error, retrying without assignees: ${getErrorMessage(error)}`);
      const { assignees: _removed, ...payloadWithoutAssignees } = payload;
      return await githubClient.rest.issues.create(payloadWithoutAssignees);
    }
    throw error;
  }
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
  const rawAssignees = config.assignees ? (Array.isArray(config.assignees) ? config.assignees : config.assignees.split(",")).map(a => String(a).trim()).filter(a => a) : [];
  const hasCopilotInAssignees = rawAssignees.some(a => a.toLowerCase() === "copilot");
  const configAssignees = sanitizeFallbackAssignees(rawAssignees);
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

  // Check if copilot assignment is enabled for fallback issues
  const assignCopilot = process.env.GH_AW_ASSIGN_COPILOT === "true";

  // Lazily-initialised client for copilot assignment (only allocated when needed).
  // Uses GH_AW_ASSIGN_TO_AGENT_TOKEN (agent token preference chain) when available,
  // otherwise falls back to the step-level github object.
  /** @type {Object|null} */
  let copilotClient = null;

  /**
   * Assigns copilot to a fallback issue using agent helpers, if copilot was requested
   * in the assignees config and the GH_AW_ASSIGN_COPILOT env var is set.
   * A no-op when either condition is false. The copilotClient is initialised lazily
   * on the first call and reused for subsequent issues.
   * @param {string} owner - Repository owner
   * @param {string} repo - Repository name
   * @param {number} issueNumber - Fallback issue number
   */
  async function assignCopilotToFallbackIssueIfEnabled(owner, repo, issueNumber) {
    if (!hasCopilotInAssignees || !assignCopilot) return;
    if (!copilotClient) {
      copilotClient = await createCopilotAssignmentClient(config);
    }
    core.info(`Assigning copilot coding agent to fallback issue #${issueNumber} in ${owner}/${repo}...`);
    try {
      const agentId = await findAgent(owner, repo, "copilot", copilotClient);
      if (!agentId) {
        core.warning(`copilot coding agent is not available for ${owner}/${repo}`);
        return;
      }
      const issueDetails = await getIssueDetails(owner, repo, issueNumber, copilotClient);
      if (!issueDetails) {
        core.warning(`Failed to get issue details for copilot assignment of fallback issue #${issueNumber}`);
        return;
      }
      if (issueDetails.currentAssignees.some(a => a.id === agentId)) {
        core.info(`copilot is already assigned to fallback issue #${issueNumber}`);
        return;
      }
      const assigned = await assignAgentToIssue(
        issueDetails.issueId,
        agentId,
        issueDetails.currentAssignees,
        "copilot",
        null, // allowedAgents — not restricted for fallback issues
        null, // pullRequestRepoId — not applicable (issue, not PR)
        null, // model — not applicable
        null, // customAgent — not applicable
        null, // customInstructions — not applicable
        null, // baseBranch — not applicable
        copilotClient
      );
      if (assigned) {
        core.info(`Successfully assigned copilot coding agent to fallback issue #${issueNumber}`);
      } else {
        core.warning(`Failed to assign copilot to fallback issue #${issueNumber}`);
      }
    } catch (error) {
      core.warning(`Failed to assign copilot to fallback issue #${issueNumber}: ${getErrorMessage(error)}`);
    }
  }

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
  const autoCloseIssue = parseBoolTemplatable(config.auto_close_issue, true); // Default to true (auto-close enabled)

  // Environment validation - fail early if required variables are missing
  const workflowId = process.env.GH_AW_WORKFLOW_ID;
  if (!workflowId) {
    throw new Error("GH_AW_WORKFLOW_ID environment variable is required");
  }

  // Extract triggering issue number from context (for auto-linking PRs to issues)
  const triggeringIssueNumber = typeof context !== "undefined" && context.payload?.issue?.number && !context.payload?.issue?.pull_request ? context.payload.issue.number : undefined;

  // Check if we're in staged mode
  const isStaged = isStagedMode(config);

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
  if (configAssignees && configAssignees.length > 0) {
    core.info(`Configured assignees (for fallback issues): ${configAssignees.join(", ")}`);
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

    const tempIdResult = getOrGenerateTemporaryId(pullRequestItem, "pull request");
    if (tempIdResult.error) {
      core.warning(`Skipping create_pull_request: ${tempIdResult.error}`);
      return { success: false, error: tempIdResult.error };
    }
    const temporaryId = tempIdResult.temporaryId;

    core.info(`Processing create_pull_request: title=${pullRequestItem.title || "No title"}, bodyLength=${pullRequestItem.body?.length || 0}`);

    // Determine the patch file path from the message (set by the MCP server handler)
    const patchFilePath = pullRequestItem.patch_path;
    core.info(`Patch file path: ${patchFilePath || "(not set)"}`);

    // Determine the bundle file path from the message (set when patch-format: bundle is configured)
    const bundleFilePath = pullRequestItem.bundle_path;
    if (bundleFilePath) {
      core.info(`Bundle file path: ${bundleFilePath}`);
    }
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
    // Skip this check when a bundle file is present (bundle transport does not use a patch file)
    const hasBundleFile = !!(bundleFilePath && fs.existsSync(bundleFilePath));
    if (!hasBundleFile && (!patchFilePath || !fs.existsSync(patchFilePath))) {
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
    let isEmpty = hasBundleFile ? false : true;

    if (!hasBundleFile && patchFilePath && fs.existsSync(patchFilePath)) {
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
    // This behavior can be disabled by setting auto-close-issue: false in the workflow config.
    if (triggeringIssueNumber && autoCloseIssue) {
      const hasClosingKeyword = /(?:fix|fixes|fixed|close|closes|closed|resolve|resolves|resolved)\s+#\d+/i.test(processedBody);
      if (!hasClosingKeyword) {
        processedBody = processedBody.trimEnd() + `\n\n- Fixes #${triggeringIssueNumber}`;
        core.info(`Auto-added "Fixes #${triggeringIssueNumber}" closing keyword to PR body as bullet point`);
      }
    } else if (triggeringIssueNumber && !autoCloseIssue) {
      core.info(`Skipping auto-close keyword for #${triggeringIssueNumber} (auto-close-issue: false)`);
    }

    let bodyLines = processedBody.split("\n");
    let branchName = pullRequestItem.branch ? pullRequestItem.branch.trim() : null;
    // Preserve the original agent branch name for bundle transport (the bundle was created
    // using this branch name as the refs/heads ref inside the bundle file).
    const originalAgentBranch = branchName;
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

    // Apply the patch/bundle using git CLI (skip if empty)
    // Track number of new commits pushed so we can restrict the extra empty commit
    // to branches with exactly one new commit (security: prevents use of CI trigger
    // token on multi-commit branches where workflow files may have been modified).
    let newCommitCount = 0;
    if (hasBundleFile) {
      // Bundle transport: fetch commits directly from the bundle file.
      // This preserves merge commit topology and per-commit metadata (messages, authorship)
      // unlike git format-patch which flattens history and drops merge resolution content.
      core.info(`Applying changes from bundle: ${bundleFilePath}`);
      const bundleBranchRef = originalAgentBranch || branchName;
      try {
        // Fetch from bundle: creates a local branch pointing to the bundle's tip commit.
        // The bundle contains refs/heads/<bundleBranchRef> which was the agent's working branch.
        await exec.exec("git", ["fetch", bundleFilePath, `refs/heads/${bundleBranchRef}:refs/heads/${branchName}`]);
        core.info(`Created local branch ${branchName} from bundle`);
        await exec.exec("git", ["checkout", branchName]);
        core.info(`Checked out branch ${branchName} from bundle`);
      } catch (bundleError) {
        core.error(`Failed to apply bundle: ${bundleError instanceof Error ? bundleError.message : String(bundleError)}`);
        return { success: false, error: "Failed to apply bundle" };
      }

      // Push the commits from the bundle to the remote branch
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
        core.info("Changes pushed to branch (from bundle)");

        // Count new commits on PR branch relative to base
        try {
          const { stdout: countStr } = await exec.getExecOutput("git", ["rev-list", "--count", `origin/${baseBranch}..HEAD`]);
          newCommitCount = parseInt(countStr.trim(), 10);
          core.info(`${newCommitCount} new commit(s) on branch relative to origin/${baseBranch}`);
        } catch {
          core.info("Could not count new commits - extra empty commit will be skipped");
        }
      } catch (pushError) {
        core.error(`Git push failed: ${pushError instanceof Error ? pushError.message : String(pushError)}`);

        if (!fallbackAsIssue) {
          const error = `Failed to push changes: ${pushError instanceof Error ? pushError.message : String(pushError)}`;
          return { success: false, error, error_type: "push_failed" };
        }

        core.warning("Git push operation failed - creating fallback issue instead of pull request");

        const runUrl = buildWorkflowRunUrl(context, context.repo);
        const runId = context.runId;

        const artifactFileName = bundleFilePath ? bundleFilePath.replace("/tmp/gh-aw/", "") : "aw-unknown.bundle";
        const fallbackBody = `${body}

---

> [!NOTE]
> This was originally intended as a pull request, but the git push operation failed.
>
> **Workflow Run:** [View run details and download bundle artifact](${runUrl})
>
> The bundle file is available in the \`agent\` artifact in the workflow run linked above.

To create a pull request with the changes:

\`\`\`sh
# Download the artifact from the workflow run
gh run download ${runId} -n agent -D /tmp/agent-${runId}

# Fetch the bundle into a local branch
git fetch /tmp/agent-${runId}/${artifactFileName} refs/heads/${bundleBranchRef}:refs/heads/${branchName}
git checkout ${branchName}

# Push the branch to origin
git push origin ${branchName}

# Create the pull request
gh pr create --title '${title}' --base ${baseBranch} --head ${branchName} --repo ${repoParts.owner}/${repoParts.repo}
\`\`\``;

        try {
          const { data: issue } = await createFallbackIssue(githubClient, repoParts, title, fallbackBody, mergeFallbackIssueLabels(labels), configAssignees);

          core.info(`Created fallback issue #${issue.number}: ${issue.html_url}`);
          await assignCopilotToFallbackIssueIfEnabled(repoParts.owner, repoParts.repo, issue.number);
          await updateActivationComment(github, context, core, issue.html_url, issue.number, "issue");

          return {
            success: true,
            fallback_used: true,
            issue_number: issue.number,
            issue_url: issue.html_url,
          };
        } catch (issueError) {
          const error = `Failed to push changes and failed to create fallback issue. Push error: ${pushError instanceof Error ? pushError.message : String(pushError)}. Issue error: ${issueError instanceof Error ? issueError.message : String(issueError)}`;
          return { success: false, error };
        }
      }
    } else {
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
        const patchLines = patchContent.split("\n");
        const previewLineCount = Math.min(500, patchLines.length);
        core.info(`Patch preview (first ${previewLineCount} of ${patchLines.length} lines):`);
        for (let i = 0; i < previewLineCount; i++) {
          core.info(patchLines[i]);
        }

        // Patches are created with git format-patch, so use git am to apply them
        // Use --3way to handle cross-repo patches where the patch base may differ from target repo
        // This allows git to resolve create-vs-modify mismatches when a file exists in target but not source
        let patchApplied = false;
        try {
          await exec.exec("git", ["am", "--3way", patchFilePath]);
          core.info("Patch applied successfully");
          patchApplied = true;
        } catch (patchError) {
          core.error(`Failed to apply patch with --3way: ${patchError instanceof Error ? patchError.message : String(patchError)}`);

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

          // Abort the failed git am before attempting any fallback
          try {
            await exec.exec("git am --abort");
            core.info("Aborted failed git am");
          } catch (abortError) {
            core.warning(`Failed to abort git am: ${abortError instanceof Error ? abortError.message : String(abortError)}`);
          }

          // Fallback (Option 1): create the PR branch at the original base commit so the PR
          // can still be created. GitHub will show the merge conflicts, allowing manual resolution.
          // This handles the case where the target branch received intervening commits after
          // the patch was generated, making --3way unable to resolve the conflicts automatically.
          core.info("Attempting fallback: create PR branch at original base commit...");
          try {
            // Use the base commit recorded at patch generation time.
            // The From <sha> header in format-patch output contains the agent's new commit SHA
            // which does not exist in this checkout, so we cannot derive the base from it.
            const originalBaseCommit = pullRequestItem.base_commit;
            if (!originalBaseCommit) {
              core.warning("No base_commit recorded in safe output entry - fallback not possible");
            } else {
              core.info(`Original base commit from patch generation: ${originalBaseCommit}`);

              // In shallow clones (fetch-depth: 1) the base commit may not be locally available.
              // Attempt to fetch it explicitly before checking whether it exists.
              try {
                await exec.exec("git", ["fetch", "origin", originalBaseCommit, "--depth=1"]);
              } catch (fetchError) {
                // Non-fatal: the commit may already be available, or the server may not support
                // fetching individual SHAs (e.g. some GHE configurations). Log for troubleshooting.
                core.info(`Note: could not fetch base commit ${originalBaseCommit} explicitly (${fetchError instanceof Error ? fetchError.message : String(fetchError)}); will verify local availability next`);
              }

              // Verify the base commit is available in this repo (may not exist cross-repo)
              await exec.exec("git", ["cat-file", "-e", originalBaseCommit]);
              core.info("Original base commit exists locally - proceeding with fallback");

              // Re-create the PR branch at the original base commit
              await exec.exec(`git checkout ${baseBranch}`);
              try {
                await exec.exec(`git branch -D ${branchName}`);
              } catch {
                // Branch may not exist yet, ignore
              }
              await exec.exec(`git checkout -b ${branchName} ${originalBaseCommit}`);
              core.info(`Created branch ${branchName} at original base commit ${originalBaseCommit}`);

              // Apply the patch without --3way; we are on the correct base so it should apply cleanly
              await exec.exec(`git am ${patchFilePath}`);
              core.info("Patch applied successfully at original base commit");
              core.warning(`PR branch ${branchName} is based on an earlier commit than the current ${baseBranch} HEAD. The pull request will show merge conflicts that require manual resolution.`);
              patchApplied = true;
            }
          } catch (fallbackError) {
            core.warning(`Fallback to original base commit failed: ${fallbackError instanceof Error ? fallbackError.message : String(fallbackError)}`);
          }

          if (!patchApplied) {
            return { success: false, error: "Failed to apply patch" };
          }
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
> The patch file is available in the \`agent\` artifact in the workflow run linked above.

To create a pull request with the changes:

\`\`\`sh
# Download the artifact from the workflow run
gh run download ${runId} -n agent -D /tmp/agent-${runId}

# Create a new branch
git checkout -b ${branchName}

# Apply the patch (--3way handles cross-repo patches where files may already exist)
git am --3way /tmp/agent-${runId}/${patchFileName}

# Push the branch to origin
git push origin ${branchName}

# Create the pull request
gh pr create --title '${title}' --base ${baseBranch} --head ${branchName} --repo ${repoParts.owner}/${repoParts.repo}
\`\`\`
${patchPreview}`;

            try {
              const { data: issue } = await createFallbackIssue(githubClient, repoParts, title, fallbackBody, mergeFallbackIssueLabels(labels), configAssignees);

              core.info(`Created fallback issue #${issue.number}: ${issue.html_url}`);
              await assignCopilotToFallbackIssueIfEnabled(repoParts.owner, repoParts.repo, issue.number);

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
      } // end if (!isEmpty) / else patch application block
    } // end else (!hasBundleFile - patch path)

    // Protected file protection – fallback-to-issue path:
    // The patch has been applied (and pushed, unless manifestProtectionPushFailedError is set).
    // Instead of creating a pull request, we create a review issue so a human can carefully
    // inspect the protected file changes before merging.
    // - Normal case (push succeeded): provides a GitHub compare URL to click and create the PR.
    // - Push-failed case: push was rejected (e.g. missing `workflows` permission); provides
    //   patch artifact download instructions instead of the compare URL.
    if (manifestProtectionFallback) {
      const allFound = manifestProtectionFallback;
      const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
      // Use head branch (branchName) for links when push succeeded; fall back to baseBranch
      // for the push-failed case where the head branch is not yet on the remote.
      const branchForLinks = manifestProtectionPushFailedError ? baseBranch : branchName;
      const fileList = buildProtectedFileList(allFound, githubServer, repoParts.owner, repoParts.repo, branchForLinks);

      let fallbackBody;
      if (manifestProtectionPushFailedError) {
        // Push failed — branch not on remote, so compare URL is unavailable.
        // Use the push-failed template with artifact download instructions.
        const runId = context.runId;
        const patchFileName = patchFilePath ? patchFilePath.replace("/tmp/gh-aw/", "") : "aw-unknown.patch";
        const pushFailedTemplatePath = `${process.env.RUNNER_TEMP}/gh-aw/prompts/manifest_protection_push_failed_fallback.md`;
        fallbackBody = renderTemplateFromFile(pushFailedTemplatePath, {
          main_body: mainBodyContent,
          footer: footerContent,
          files: fileList,
          run_id: String(runId),
          branch_name: branchName,
          base_branch: baseBranch,
          patch_file: patchFileName,
          title,
          repo: `${repoParts.owner}/${repoParts.repo}`,
        });
      } else {
        // Normal case — push succeeded, provide compare URL.
        const encodedBase = encodePathSegments(baseBranch);
        const encodedHead = encodePathSegments(branchName);
        const createPrUrl = `${githubServer}/${repoParts.owner}/${repoParts.repo}/compare/${encodedBase}...${encodedHead}?expand=1&title=${encodeURIComponent(title)}`;
        const templatePath = `${process.env.RUNNER_TEMP}/gh-aw/prompts/manifest_protection_create_pr_fallback.md`;
        fallbackBody = renderTemplateFromFile(templatePath, {
          main_body: mainBodyContent,
          footer: footerContent,
          files: fileList,
          create_pr_url: createPrUrl,
        });
      }

      try {
        const { data: issue } = await createFallbackIssue(githubClient, repoParts, title, fallbackBody, mergeFallbackIssueLabels(labels), configAssignees);

        core.info(`Created protected-file-protection review issue #${issue.number}: ${issue.html_url}`);
        await assignCopilotToFallbackIssueIfEnabled(repoParts.owner, repoParts.repo, issue.number);

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
        try {
          await withRetry(
            () =>
              githubClient.rest.issues.addLabels({
                owner: repoParts.owner,
                repo: repoParts.repo,
                issue_number: pullRequest.number,
                labels: labels,
              }),
            {
              maxRetries: LABEL_MAX_RETRIES,
              initialDelayMs: LABEL_INITIAL_DELAY_MS,
              backoffMultiplier: 2,
              shouldRetry: isLabelTransientError,
            },
            `add labels to PR #${pullRequest.number}`
          );
          core.info(`Added labels to pull request: ${JSON.stringify(labels)}`);
        } catch (labelError) {
          // Label addition is non-critical - warn but don't fail the PR creation.
          // GitHub's API may transiently fail to resolve the PR node ID immediately
          // after creation, which causes label operations to fail with an unprocessable error.
          core.warning(`Failed to add labels to PR #${pullRequest.number}: ${labelError instanceof Error ? labelError.message : String(labelError)}`);
        }
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
        const fallbackBody = renderTemplateFromFile(fallbackTemplatePath, {
          body,
          branch_name: branchName,
          create_pr_url: createPrUrl,
          faq_url: FAQ_CREATE_PR_PERMISSIONS_URL,
          patch_preview: patchPreview,
        });

        try {
          const { data: issue } = await createFallbackIssue(githubClient, repoParts, title, fallbackBody, mergeFallbackIssueLabels(labels), configAssignees);

          core.info(`Created fallback issue #${issue.number}: ${issue.html_url}`);
          await assignCopilotToFallbackIssueIfEnabled(repoParts.owner, repoParts.repo, issue.number);

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
        const { data: issue } = await createFallbackIssue(githubClient, repoParts, title, fallbackBody, mergeFallbackIssueLabels(labels), configAssignees);

        core.info(`Created fallback issue #${issue.number}: ${issue.html_url}`);
        await assignCopilotToFallbackIssueIfEnabled(repoParts.owner, repoParts.repo, issue.number);

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
