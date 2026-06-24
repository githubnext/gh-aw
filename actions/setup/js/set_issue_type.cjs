// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { isStagedMode, checkRequiredFilter } = require("./safe_output_helpers.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { resolveSafeOutputIssueTarget } = require("./temporary_id.cjs");
const { hasIssueIntentsRuntimeFeature, normalizeIssueIntentMetadata } = require("./issue_intents.cjs");

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "set_issue_type";

/**
 * @param {{ rationale?: string, confidence?: "LOW"|"MEDIUM"|"HIGH", suggest?: boolean }} intentMetadata
 * @returns {{ rationale?: string, confidence?: "low"|"medium"|"high", suggest?: boolean }}
 */
function toRestIssueIntentMetadata(intentMetadata) {
  if (!intentMetadata.confidence) {
    return intentMetadata;
  }
  return {
    ...intentMetadata,
    confidence: intentMetadata.confidence.toLowerCase(),
  };
}

/**
 * @param {unknown} error
 * @returns {boolean}
 */
function isIssueTypeValidationError(error) {
  const status =
    typeof error === "object" && error !== null
      ? /** @type {{ status?: unknown, response?: { status?: unknown } }} */ (error.status ?? /** @type {{ status?: unknown, response?: { status?: unknown } }} */ error.response?.status)
      : undefined;
  return status === 422;
}

/**
 * @param {unknown} error
 * @param {string} issueTypeName
 * @returns {string}
 */
function mapInvalidIssueTypeError(error, issueTypeName) {
  if (typeof error !== "object" || error === null) {
    return `Issue type ${JSON.stringify(issueTypeName)} not found.`;
  }
  const responseData = /** @type {{ response?: { data?: { errors?: Array<{ message?: string }>, message?: string } } }} */ error.response?.data;
  const errorDetails = Array.isArray(responseData?.errors)
    ? responseData.errors
        .map(err => err?.message)
        .filter(Boolean)
        .join("; ")
    : responseData?.message;
  if (!errorDetails) {
    return `Issue type ${JSON.stringify(issueTypeName)} not found.`;
  }
  return `Issue type ${JSON.stringify(issueTypeName)} not found. Available types: ${errorDetails}`;
}

/**
 * Main handler factory for set_issue_type
 * Returns a message handler function that processes individual set_issue_type messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const allowedTypes = config.allowed || [];
  const maxCount = config.max || 5;
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
  const githubClient = await createAuthenticatedGitHubClient(config);

  // Check if we're in staged mode
  const isStaged = isStagedMode(config);

  core.info(`Set issue type configuration: max=${maxCount}`);
  const requiredLabels = Array.isArray(config.required_labels) ? config.required_labels : [];
  const requiredTitlePrefix = config.required_title_prefix || "";
  if (requiredLabels.length > 0) core.info(`Required labels (all): ${requiredLabels.join(", ")}`);
  if (requiredTitlePrefix) core.info(`Required title prefix: ${requiredTitlePrefix}`);
  if (allowedTypes.length > 0) {
    core.info(`Allowed issue types: ${allowedTypes.join(", ")}`);
  }
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${Array.from(allowedRepos).join(", ")}`);
  }

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function that processes a single set_issue_type message
   * @param {Object} message - The set_issue_type message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleSetIssueType(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping set_issue_type: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const item = message;

    // Resolve and validate target repository
    const repoResult = resolveAndValidateRepo(item, defaultTargetRepo, allowedRepos, "issue");
    if (!repoResult.success) {
      core.warning(`Skipping set_issue_type: ${repoResult.error}`);
      return {
        success: false,
        error: repoResult.error,
      };
    }
    const { repo: itemRepo, repoParts } = repoResult;
    core.info(`Target repository: ${itemRepo}`);

    // Determine target issue number, with temporary ID support
    const targetResult = resolveSafeOutputIssueTarget({ message: item, resolvedTemporaryIds, repoParts, handlerType: HANDLER_TYPE, aliases: ["issue_number"] });
    if (!targetResult.success) return targetResult;
    let issueNumber;
    if (targetResult.number !== null) {
      issueNumber = targetResult.number;
      core.info(`Resolved issue number: #${issueNumber}`);
    } else {
      const contextIssueNumber = context.payload?.issue?.number;
      if (!contextIssueNumber) {
        core.warning("No issue_number provided and not in issue context");
        return {
          success: false,
          error: "No issue number available",
        };
      }
      issueNumber = contextIssueNumber;
    }

    const filterResult = await checkRequiredFilter(githubClient, repoParts, issueNumber, requiredLabels, requiredTitlePrefix, HANDLER_TYPE);
    if (filterResult) return filterResult;

    const issueTypeName = item.issue_type ?? "";
    const isClear = issueTypeName === "";

    core.info(`Setting issue type on issue #${issueNumber}: ${isClear ? "(clear)" : JSON.stringify(issueTypeName)}`);

    // Validate against allowed list if configured (empty string always allowed to clear)
    if (allowedTypes.length > 0 && !isClear) {
      const normalizedAllowed = allowedTypes.map(t => t.toLowerCase());
      if (!normalizedAllowed.includes(issueTypeName.toLowerCase())) {
        const error = `Issue type ${JSON.stringify(issueTypeName)} is not in the allowed list: ${JSON.stringify(allowedTypes)}`;
        core.warning(error);
        return { success: false, error };
      }
    }

    // If in staged mode, preview without executing
    if (isStaged) {
      const description = isClear ? `Would clear issue type on issue #${issueNumber} in ${itemRepo}` : `Would set issue type to ${JSON.stringify(issueTypeName)} on issue #${issueNumber} in ${itemRepo}`;
      logStagedPreviewInfo(description);
      return {
        success: true,
        staged: true,
        previewInfo: {
          issue_number: issueNumber,
          issue_type: issueTypeName,
          repo: itemRepo,
        },
      };
    }

    try {
      const { owner, repo } = repoParts;
      const intentMetadata = normalizeIssueIntentMetadata(item);
      const typeValue = isClear
        ? ""
        : hasIssueIntentsRuntimeFeature()
          ? {
              value: issueTypeName,
              ...toRestIssueIntentMetadata(intentMetadata),
            }
          : issueTypeName;
      await githubClient.rest.issues.update({
        owner,
        repo,
        issue_number: issueNumber,
        type: typeValue,
      });

      const successMsg = isClear ? `Successfully cleared issue type on issue #${issueNumber}` : `Successfully set issue type to ${JSON.stringify(issueTypeName)} on issue #${issueNumber}`;
      core.info(successMsg);

      return {
        success: true,
        issue_number: issueNumber,
        issue_type: issueTypeName,
        repo: itemRepo,
      };
    } catch (error) {
      if (!isClear && isIssueTypeValidationError(error)) {
        const mappedError = mapInvalidIssueTypeError(error, issueTypeName);
        core.error(`Failed to set issue type on issue #${issueNumber}: ${mappedError}`);
        return { success: false, error: mappedError };
      }
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to set issue type on issue #${issueNumber}: ${errorMessage}`);
      return { success: false, error: errorMessage };
    }
  };
}

module.exports = { main };
