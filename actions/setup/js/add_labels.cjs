// @ts-check
/// <reference types="@actions/github-script" />

const { validateLabels } = require("./safe_output_validator.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Main handler factory for add_labels
 * Returns a message handler function that processes individual add_labels messages
 * @param {Object} config - Handler configuration from GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG
 * @returns {Promise<Function>} Message handler function (message, resolvedTemporaryIds) => result
 */
async function main(config = {}) {
  // Extract configuration
  const allowedLabels = config.allowed || [];
  const maxCount = config.max || 10;

  core.info(`Add labels configuration: max=${maxCount}`);
  if (allowedLabels.length > 0) {
    core.info(`Allowed labels: ${allowedLabels.join(", ")}`);
  }

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function that processes a single add_labels message
   * @param {Object} message - The add_labels message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleAddLabels(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping add_labels: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const item = message;

    // Determine target issue/PR number
    let itemNumber;
    if (item.item_number !== undefined) {
      itemNumber = parseInt(String(item.item_number), 10);
      if (isNaN(itemNumber)) {
        core.warning(`Invalid item number: ${item.item_number}`);
        return {
          success: false,
          error: `Invalid item number: ${item.item_number}`,
        };
      }
    } else {
      // Use context issue or PR if available
      const contextIssue = context.payload?.issue?.number;
      const contextPR = context.payload?.pull_request?.number;
      itemNumber = contextIssue || contextPR;

      if (!itemNumber) {
        core.warning("No item_number provided and not in issue/PR context");
        return {
          success: false,
          error: "No issue/PR number available",
        };
      }
    }

    // Determine context type
    const contextType = context.payload?.pull_request ? "pull request" : "issue";

    const requestedLabels = item.labels ?? [];
    core.info(`Requested labels: ${JSON.stringify(requestedLabels)}`);

    // Use validation helper to sanitize and validate labels
    const labelsResult = validateLabels(requestedLabels, allowedLabels, maxCount);
    if (!labelsResult.valid) {
      // If no valid labels, log info and return gracefully
      if (labelsResult.error?.includes("No valid labels")) {
        core.info("No labels to add");
        return {
          success: true,
          number: itemNumber,
          labelsAdded: [],
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
      core.info("No labels to add");
      return {
        success: true,
        number: itemNumber,
        labelsAdded: [],
        message: "No labels to add",
      };
    }

    core.info(`Adding ${uniqueLabels.length} labels to ${contextType} #${itemNumber}: ${JSON.stringify(uniqueLabels)}`);

    try {
      await github.rest.issues.addLabels({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: itemNumber,
        labels: uniqueLabels,
      });

      core.info(`Successfully added ${uniqueLabels.length} labels to ${contextType} #${itemNumber}`);

      return {
        success: true,
        number: itemNumber,
        labelsAdded: uniqueLabels,
        contextType: contextType,
      };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to add labels: ${errorMessage}`);
      return {
        success: false,
        error: errorMessage,
      };
    }
  };
}

module.exports = { main };
