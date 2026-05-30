// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "report_incomplete";

/**
 * Main handler factory for report_incomplete
 * Returns a message handler function that processes individual report_incomplete messages.
 * report_incomplete is a first-class signal that the agent could not complete its task
 * (e.g. due to tool failures, missing auth, or inaccessible resources).
 * The handler records and logs the reason; handle_agent_failure.cjs detects these
 * items in the raw output and activates failure handling even when the agent exited 0.
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration with destructuring
  const { max: maxCount = 0 } = config; // 0 means unlimited

  core.info(`Max count: ${maxCount === 0 ? "unlimited" : maxCount}`);

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function that processes a single report_incomplete message
   * @param {Object} message - The report_incomplete message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number} (unused for report_incomplete)
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleReportIncomplete(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (maxCount > 0 && processedCount >= maxCount) {
      core.warning(`Skipping report_incomplete: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    // Validate required fields
    const { reason } = message;
    if (!reason || typeof reason !== "string" || !reason.trim()) {
      core.warning(`report_incomplete message missing or invalid 'reason' field: ${JSON.stringify(message)}`);
      return {
        success: false,
        error: "Missing required field: reason",
      };
    }

    processedCount++;

    const timestamp = new Date().toISOString();

    core.warning(`⚠️ report_incomplete: ${reason}`);
    if (message.details) {
      core.info(`   Details: ${message.details}`);
    }

    return {
      success: true,
      reason,
      details: message.details || null,
      timestamp,
    };
  };
}

module.exports = { main };
