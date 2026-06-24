// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 * @typedef {{ status?: number, response?: { status?: number, data?: { errors?: Array<{ message?: string }>, message?: string } } }} IssueTypeAPIError
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
const AVAILABLE_TYPES_PATTERNS = [/one of:\s*(.+)$/i, /available(?: types?)?:\s*(.+)$/i];
const NO_ISSUE_TYPES_PATTERNS = [/no issue types? (?:are )?available/i, /issue types? (?:is|are) not (?:enabled|configured)/i];
const NO_ISSUE_TYPES_AVAILABLE_ERROR = "No issue types are available for this repository. Issue types must be configured in the repository or organization settings.";

/**
 * @param {{ rationale?: string, confidence?: "LOW"|"MEDIUM"|"HIGH", suggest?: boolean }} intentMetadata Intent metadata in GraphQL format.
 * @returns {{ rationale?: string, confidence?: "low"|"medium"|"high", suggest?: boolean }} Intent metadata formatted for REST.
 */
function toRestIssueIntentMetadata(intentMetadata) {
  /** @type {{ rationale?: string, confidence?: "low"|"medium"|"high", suggest?: boolean }} */
  const restMetadata = {};
  if (intentMetadata.rationale) {
    restMetadata.rationale = intentMetadata.rationale;
  }
  if (intentMetadata.suggest) {
    restMetadata.suggest = true;
  }
  if (intentMetadata.confidence) {
    switch (intentMetadata.confidence) {
      case "LOW":
        restMetadata.confidence = "low";
        break;
      case "MEDIUM":
        restMetadata.confidence = "medium";
        break;
      case "HIGH":
        restMetadata.confidence = "high";
        break;
      default:
        throw new Error(`Invalid confidence ${JSON.stringify(intentMetadata.confidence)}. Expected one of: LOW, MEDIUM, HIGH.`);
    }
  }
  return restMetadata;
}

/**
 * @param {unknown} error
 * @returns {unknown}
 */
function getErrorStatus(error) {
  if (typeof error !== "object" || error === null) {
    return undefined;
  }
  const status = "status" in error && typeof error.status === "number" ? error.status : undefined;
  if (status !== undefined) return status;
  if (!("response" in error) || typeof error.response !== "object" || error.response === null) {
    return undefined;
  }
  return "status" in error.response && typeof error.response.status === "number" ? error.response.status : undefined;
}

/**
 * @param {unknown} error
 * @returns {boolean}
 */
function isIssueTypeValidationError(error) {
  return getErrorStatus(error) === 422;
}

/**
 * @param {unknown} error
 * @returns {unknown}
 */
function getErrorResponseData(error) {
  if (typeof error !== "object" || error === null) {
    return undefined;
  }
  if (!("response" in error) || typeof error.response !== "object" || error.response === null) {
    return undefined;
  }
  if (!("data" in error.response)) {
    return undefined;
  }
  return error.response.data;
}

/**
 * @param {unknown} error
 * @param {string} issueTypeName
 * @returns {string}
 */
function mapInvalidIssueTypeError(error, issueTypeName) {
  const baseMessage = `Issue type ${JSON.stringify(issueTypeName)} not found.`;
  const responseData = getErrorResponseData(error);
  let errorDetails;
  if (typeof responseData === "object" && responseData !== null) {
    if ("message" in responseData && typeof responseData.message === "string") {
      errorDetails = responseData.message;
    }
    // Keep previous precedence: if detailed validation errors exist, prefer the first
    // entry over the top-level message.
    if ("errors" in responseData && Array.isArray(responseData.errors) && responseData.errors.length > 0) {
      const firstError = responseData.errors[0];
      if (typeof firstError === "object" && firstError !== null && "message" in firstError && typeof firstError.message === "string") {
        errorDetails = firstError.message;
      }
    }
  }
  if (!errorDetails) {
    return baseMessage;
  }
  // REST validation errors vary across endpoints and deployments; extract the list from
  // either "... one of: A, B" or "... available types: A, B" when present.
  const matchedPattern = AVAILABLE_TYPES_PATTERNS.find(pattern => pattern.test(errorDetails));
  const availableTypes = matchedPattern?.exec(errorDetails)?.[1]?.trim();
  if (availableTypes) {
    return `${baseMessage} Available types: ${availableTypes}`;
  }

  if (NO_ISSUE_TYPES_PATTERNS.some(pattern => pattern.test(errorDetails))) {
    return NO_ISSUE_TYPES_AVAILABLE_ERROR;
  }

  return baseMessage;
}

/**
 * @param {boolean} isClear
 * @param {string} issueTypeName
 * @param {{ rationale?: string, confidence?: "LOW"|"MEDIUM"|"HIGH", suggest?: boolean }} intentMetadata
 * @returns {string | { value: string, rationale?: string, confidence?: "low"|"medium"|"high", suggest?: boolean }}
 */
function buildIssueTypeValue(isClear, issueTypeName, intentMetadata) {
  if (isClear) {
    return "";
  }
  if (!hasIssueIntentsRuntimeFeature()) {
    return issueTypeName;
  }
  return {
    value: issueTypeName,
    ...toRestIssueIntentMetadata(intentMetadata),
  };
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
    let resolvedIssueTypeName = issueTypeName;

    core.info(`Setting issue type on issue #${issueNumber}: ${isClear ? "(clear)" : JSON.stringify(issueTypeName)}`);

    // Validate against allowed list if configured (empty string always allowed to clear)
    if (allowedTypes.length > 0 && !isClear) {
      const matchedAllowedType = allowedTypes.find(allowedType => allowedType.toLowerCase() === issueTypeName.toLowerCase());
      if (!matchedAllowedType) {
        const error = `Issue type ${JSON.stringify(issueTypeName)} is not in the allowed list: ${JSON.stringify(allowedTypes)}`;
        core.warning(error);
        return { success: false, error };
      }
      resolvedIssueTypeName = matchedAllowedType;
    }

    // If in staged mode, preview without executing
    if (isStaged) {
      const description = isClear ? `Would clear issue type on issue #${issueNumber} in ${itemRepo}` : `Would set issue type to ${JSON.stringify(resolvedIssueTypeName)} on issue #${issueNumber} in ${itemRepo}`;
      logStagedPreviewInfo(description);
      return {
        success: true,
        staged: true,
        previewInfo: {
          issue_number: issueNumber,
          issue_type: resolvedIssueTypeName,
          repo: itemRepo,
        },
      };
    }

    try {
      const { owner, repo } = repoParts;
      const intentMetadata = normalizeIssueIntentMetadata(item);
      const typeValue = buildIssueTypeValue(isClear, resolvedIssueTypeName, intentMetadata);
      await githubClient.rest.issues.update({
        owner,
        repo,
        issue_number: issueNumber,
        type: typeValue,
      });

      const successMsg = isClear ? `Successfully cleared issue type on issue #${issueNumber}` : `Successfully set issue type to ${JSON.stringify(resolvedIssueTypeName)} on issue #${issueNumber}`;
      core.info(successMsg);

      return {
        success: true,
        issue_number: issueNumber,
        issue_type: resolvedIssueTypeName,
        repo: itemRepo,
      };
    } catch (error) {
      if (!isClear && isIssueTypeValidationError(error)) {
        const mappedError = mapInvalidIssueTypeError(error, resolvedIssueTypeName);
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
