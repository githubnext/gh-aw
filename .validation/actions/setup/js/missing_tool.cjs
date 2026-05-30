// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "missing_tool";

/**
 * Main handler factory for missing_tool
 * Returns a message handler function that processes individual missing_tool messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const maxCount = config.max || 0; // 0 means unlimited

  core.info(`Max count: ${maxCount === 0 ? "unlimited" : maxCount}`);

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function that processes a single missing_tool message
   * @param {Object} message - The missing_tool message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number} (unused for missing_tool)
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleMissingTool(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (maxCount > 0 && processedCount >= maxCount) {
      core.warning(`Skipping missing_tool: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    // Validate required fields (only reason is required now)
    if (!message.reason) {
      core.warning(`missing_tool message missing 'reason' field: ${JSON.stringify(message)}`);
      return {
        success: false,
        error: "Missing required field: reason",
      };
    }

    processedCount++;

    const missingTool = {
      tool: message.tool || null,
      reason: message.reason,
      alternatives: message.alternatives || null,
      timestamp: new Date().toISOString(),
    };

    if (missingTool.tool) {
      core.info(`✓ Recorded missing tool: ${missingTool.tool}`);
      core.info(`   Reason: ${missingTool.reason}`);
    } else {
      core.info(`✓ Recorded missing functionality/limitation`);
      core.info(`   Reason: ${missingTool.reason}`);
    }
    if (missingTool.alternatives) {
      core.info(`   Alternatives: ${missingTool.alternatives}`);
    }

    return {
      success: true,
      tool: missingTool.tool,
      reason: missingTool.reason,
      alternatives: missingTool.alternatives,
      timestamp: missingTool.timestamp,
    };
  };
}

module.exports = { main };
