// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { getErrorMessage } = require("./error_helpers.cjs");

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "missing_tool";

/**
 * Main handler factory for missing_tool
 * Returns a message handler function that processes individual missing_tool messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Get max reports from config
  const maxReports = config.max || null;

  core.info("Initializing missing-tool handler...");
  if (maxReports) {
    core.info(`Maximum reports allowed: ${maxReports}`);
  }

  // Track processed messages
  let processedCount = 0;
  const missingTools = [];

  /**
   * Message handler function that processes a single missing_tool message
   * @param {Object} message - The missing_tool message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number} (unused for missing_tool)
   * @returns {Promise<Object>} Result with success status
   */
  return async function handleMissingTool(message, resolvedTemporaryIds) {
    // Validate required fields
    if (!message.tool) {
      core.warning(`missing-tool entry missing 'tool' field: ${JSON.stringify(message)}`);
      return {
        success: false,
        error: "Missing 'tool' field",
      };
    }
    if (!message.reason) {
      core.warning(`missing-tool entry missing 'reason' field: ${JSON.stringify(message)}`);
      return {
        success: false,
        error: "Missing 'reason' field",
      };
    }

    // Check max limit
    if (maxReports && processedCount >= maxReports) {
      core.info(`Reached maximum number of missing tool reports (${maxReports})`);
      return {
        success: false,
        error: `Max count of ${maxReports} reached`,
      };
    }

    processedCount++;

    const missingTool = {
      tool: message.tool,
      reason: message.reason,
      alternatives: message.alternatives || null,
      timestamp: new Date().toISOString(),
    };

    missingTools.push(missingTool);
    core.info(`Recorded missing tool: ${missingTool.tool}`);

    // Log details and create step summary
    core.info(`Missing tool: ${missingTool.tool}`);
    core.info(`   Reason: ${missingTool.reason}`);
    if (missingTool.alternatives) {
      core.info(`   Alternatives: ${missingTool.alternatives}`);
    }
    core.info(`   Reported at: ${missingTool.timestamp}`);

    // Create structured summary for GitHub Actions step summary
    let summaryContent = `### Missing Tool: \`${missingTool.tool}\`\n\n`;
    summaryContent += `**Reason:** ${missingTool.reason}\n\n`;

    if (missingTool.alternatives) {
      summaryContent += `**Alternatives:** ${missingTool.alternatives}\n\n`;
    }

    summaryContent += `**Reported at:** ${missingTool.timestamp}\n\n`;

    await core.summary.addRaw(summaryContent).write();

    // Set outputs for first missing tool
    if (processedCount === 1) {
      core.setOutput("tools_reported", JSON.stringify(missingTools));
      core.setOutput("total_count", missingTools.length.toString());
    }

    core.info(`Successfully processed missing tool report ${processedCount}`);

    return {
      success: true,
      tool: missingTool,
    };
  };
}

module.exports = { main };
