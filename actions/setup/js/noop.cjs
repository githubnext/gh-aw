// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "noop";

/**
 * Main handler factory for noop
 * Returns a message handler function that processes individual noop messages
 * No-op is a fallback output type that logs messages for transparency
 * without taking any GitHub API actions
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Track processed messages for outputs
  let messageCount = 0;

  /**
   * Message handler function that processes a single noop message
   * @param {Object} message - The noop message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number} (unused for noop)
   * @returns {Promise<Object>} Result with success status
   */
  return async function handleNoop(message, resolvedTemporaryIds) {
    messageCount++;

    const noopMessage = message.message || "(no message)";

    // If in staged mode, emit step summary instead of logging
    if (isStaged) {
      let summaryContent = `## 🎭 Staged Mode: No-Op Message ${messageCount} Preview\n\n`;
      summaryContent += "The following message would be logged if staged mode was disabled:\n\n";
      summaryContent += `${noopMessage}\n`;

      await core.summary.addRaw(summaryContent).write();
      core.info(`📝 No-op message ${messageCount} preview written to step summary`);

      return {
        success: true,
        message: noopMessage,
      };
    }

    // Process noop item - just log the message for transparency
    core.info(`No-op message ${messageCount}: ${noopMessage}`);

    let summaryContent = `## No-Op Message ${messageCount}\n\n`;
    summaryContent += `${noopMessage}\n`;

    // Write summary for this noop message
    await core.summary.addRaw(summaryContent).write();

    // Export the first noop message for use in add-comment default reporting
    if (messageCount === 1) {
      core.setOutput("noop_message", noopMessage);
      core.exportVariable("GH_AW_NOOP_MESSAGE", noopMessage);
    }

    core.info(`Successfully processed noop message ${messageCount}`);

    return {
      success: true,
      message: noopMessage,
    };
  };
}

module.exports = { main };
