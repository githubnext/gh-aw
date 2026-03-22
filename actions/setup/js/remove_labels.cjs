// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "remove_labels";

const { validateLabels } = require("./safe_output_validator.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { resolveRepoIssueTarget, loadTemporaryIdMapFromResolved } = require("./temporary_id.cjs");

/**
 * Main handler factory for remove_labels
 * Returns a message handler function that processes individual remove_labels messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const allowedLabels = config.allowed || [];
  const blockedPatterns = config.blocked || [];
  const maxCount = config.max || 10;
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
  const githubClient = await createAuthenticatedGitHubClient(config);

  // Check if we're in staged mode
  const isStaged = isStagedMode(config);

  core.info(`Remove labels configuration: max=${maxCount}`);
  if (allowedLabels.length > 0) {
    core.info(`Allowed labels to remove: ${allowedLabels.join(", ")}`);
  }
  if (blockedPatterns.length > 0) {
    core.info(`Blocked patterns: ${blockedPatterns.join(", ")}`);
  }
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${Array.from(allowedRepos).join(", ")}`);
  }

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function that processes a single remove_labels message
   * @param {Object} message - The remove_labels message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleRemoveLabels(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping remove_labels: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    // Resolve and validate target repository
    const repoResult = resolveAndValidateRepo(message, defaultTargetRepo, allowedRepos, "label");
    if (!repoResult.success) {
      core.warning(`Skipping remove_labels: ${repoResult.error}`);
      return {
        success: false,
        error: repoResult.error,
      };
    }
    const { repo: itemRepo, repoParts } = repoResult;
    core.info(`Target repository: ${itemRepo}`);

    // Determine target issue/PR number
    let itemNumber;
    if (message.item_number !== undefined) {
      // Resolve temporary IDs if present
      const tempIdMap = loadTemporaryIdMapFromResolved(resolvedTemporaryIds);
      const resolvedTarget = resolveRepoIssueTarget(message.item_number, tempIdMap, repoParts.owner, repoParts.repo);

      // Check if this is an unresolved temporary ID
      if (resolvedTarget.wasTemporaryId && !resolvedTarget.resolved) {
        core.info(`Deferring remove_labels: unresolved temporary ID (${message.item_number})`);
        return {
          success: false,
          deferred: true,
          error: resolvedTarget.errorMessage || `Unresolved temporary ID: ${message.item_number}`,
        };
      }

      // Check for other resolution errors
      if (resolvedTarget.errorMessage || !resolvedTarget.resolved) {
        const error = `Invalid item number: ${message.item_number}`;
        core.warning(error);
        return { success: false, error };
      }

      itemNumber = resolvedTarget.resolved.number;
    } else {
      itemNumber = context.payload?.issue?.number || context.payload?.pull_request?.number;
    }

    if (!itemNumber || isNaN(itemNumber)) {
      const error = "No issue/PR number available";
      core.warning(error);
      return { success: false, error };
    }

    const contextType = context.payload?.pull_request ? "pull request" : "issue";
    const requestedLabels = message.labels ?? [];
    core.info(`Requested labels to remove: ${JSON.stringify(requestedLabels)}`);

    // If no labels provided, return a helpful message with allowed labels if configured
    if (!requestedLabels || requestedLabels.length === 0) {
      let errorMessage = "No labels provided. Please provide at least one label from";
      if (allowedLabels.length > 0) {
        errorMessage += ` the allowed list: ${JSON.stringify(allowedLabels)}`;
      } else {
        errorMessage += " the issue/PR's current labels";
      }
      core.info(errorMessage);
      return {
        success: false,
        error: errorMessage,
      };
    }

    // Use validation helper to sanitize and validate labels
    const labelsResult = validateLabels(requestedLabels, allowedLabels, maxCount, blockedPatterns);
    if (!labelsResult.valid) {
      // If no valid labels, log info and return gracefully
      if (labelsResult.error?.includes("No valid labels")) {
        core.info("No labels to remove");
        return {
          success: true,
          number: itemNumber,
          labelsRemoved: [],
          message: "No valid labels found",
        };
      }
      // For other validation errors, return error
      core.warning(`Label validation failed: ${labelsResult.error}`);
      return {
        success: false,
        error: labelsResult.error ?? "Invalid labels",
      };
    }

    const uniqueLabels = labelsResult.value ?? [];

    if (uniqueLabels.length === 0) {
      core.info("No labels to remove");
      return {
        success: true,
        number: itemNumber,
        labelsRemoved: [],
        message: "No labels to remove",
      };
    }

    core.info(`Removing ${uniqueLabels.length} labels from ${contextType} #${itemNumber} in ${itemRepo}: ${JSON.stringify(uniqueLabels)}`);

    // If in staged mode, preview the label removal without actually removing
    if (isStaged) {
      logStagedPreviewInfo(`Would remove ${uniqueLabels.length} labels from ${contextType} #${itemNumber} in ${itemRepo}`);
      return {
        success: true,
        staged: true,
        previewInfo: {
          number: itemNumber,
          repo: itemRepo,
          labels: uniqueLabels,
          contextType,
        },
      };
    }

    // Track successfully removed labels
    const removedLabels = [];
    const failedLabels = [];

    // Remove labels one at a time (GitHub API doesn't have a bulk remove endpoint)
    for (const label of uniqueLabels) {
      try {
        await githubClient.rest.issues.removeLabel({
          owner: repoParts.owner,
          repo: repoParts.repo,
          issue_number: itemNumber,
          name: label,
        });
        removedLabels.push(label);
        core.info(`Removed label "${label}" from ${contextType} #${itemNumber} in ${itemRepo}`);
      } catch (error) {
        // Label might not exist on the issue/PR - this is not a failure
        const errorMessage = getErrorMessage(error);
        if (errorMessage.includes("Label does not exist") || errorMessage.includes("404")) {
          core.info(`Label "${label}" was not present on ${contextType} #${itemNumber} in ${itemRepo}, skipping`);
        } else {
          core.warning(`Failed to remove label "${label}": ${errorMessage}`);
          failedLabels.push({ label, error: errorMessage });
        }
      }
    }

    if (removedLabels.length > 0) {
      core.info(`Successfully removed ${removedLabels.length} labels from ${contextType} #${itemNumber} in ${itemRepo}`);
    }

    return {
      success: true,
      number: itemNumber,
      labelsRemoved: removedLabels,
      failedLabels: failedLabels.length > 0 ? failedLabels : undefined,
      contextType,
    };
  };
}

module.exports = { main };
