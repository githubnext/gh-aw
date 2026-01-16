// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "missing_data";

/**
 * Main handler factory for missing_data
 * Returns a message handler function that processes individual missing_data messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const maxCount = config.max || 0; // 0 means unlimited

  core.info(`Max count: ${maxCount === 0 ? "unlimited" : maxCount}`);

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function that processes a single missing_data message
   * @param {Object} message - The missing_data message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number} (unused for missing_data)
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleMissingData(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (maxCount > 0 && processedCount >= maxCount) {
      core.warning(`Skipping missing_data: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    // No required fields - the model can just tell us it's missing something
    processedCount++;

    const missingData = {
      data_type: message.data_type || null,
      reason: message.reason || null,
      context: message.context || null,
      alternatives: message.alternatives || null,
      timestamp: new Date().toISOString(),
    };

    core.info(`✓ Recorded missing data${missingData.data_type ? `: ${missingData.data_type}` : ""}`);
    if (missingData.reason) {
      core.info(`   Reason: ${missingData.reason}`);
    }
    if (missingData.context) {
      core.info(`   Context: ${missingData.context}`);
    }
    if (missingData.alternatives) {
      core.info(`   Alternatives: ${missingData.alternatives}`);
    }

    return {
      success: true,
      data_type: missingData.data_type,
      reason: missingData.reason,
      context: missingData.context,
      alternatives: missingData.alternatives,
      timestamp: missingData.timestamp,
    };
  };
}

module.exports = { main };
