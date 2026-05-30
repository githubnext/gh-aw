// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "noop";

/**
 * Main handler factory for noop
 * Returns a message handler function that processes individual noop messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration with destructuring
  const { max: maxCount = 0 } = config; // 0 means unlimited

  core.info(`Max count: ${maxCount === 0 ? "unlimited" : maxCount}`);

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function that processes a single noop message
   * @param {Object} message - The noop message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number} (unused for noop)
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleNoop(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (maxCount > 0 && processedCount >= maxCount) {
      core.warning(`Skipping noop: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    // Validate required fields
    const { message: messageText } = message;
    if (!messageText || typeof messageText !== "string" || !messageText.trim()) {
      core.warning(`noop message missing or invalid 'message' field: ${JSON.stringify(message)}`);
      return {
        success: false,
        error: "Missing required field: message",
      };
    }

    processedCount++;

    const timestamp = new Date().toISOString();

    core.info(`✓ Recorded noop message: ${messageText}`);

    return {
      success: true,
      message: messageText,
      timestamp,
    };
  };
}

module.exports = { main };
