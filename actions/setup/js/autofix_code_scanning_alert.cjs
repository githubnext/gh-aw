// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "autofix_code_scanning_alert";

/**
 * Main handler factory for autofix_code_scanning_alert
 * Returns a message handler function that processes individual autofix_code_scanning_alert messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const maxCount = config.max || 10;
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  core.info(`Add code scanning autofix configuration: max=${maxCount}`);
  core.info(`Staged mode: ${isStaged}`);

  // Track how many items we've processed for max limit
  let processedCount = 0;

  // Track processed autofixes for outputs
  const processedAutofixes = [];

  /**
   * Message handler function that processes a single autofix_code_scanning_alert message
   * @param {Object} message - The autofix_code_scanning_alert message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleAutofixCodeScanningAlert(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping autofix_code_scanning_alert: max count of ${maxCount} reached`);
      return { success: false, error: `Max count of ${maxCount} reached` };
    }

    processedCount++;

    // Validate required fields
    if (message.alert_number === undefined || message.alert_number === null) {
      core.warning("Skipping autofix_code_scanning_alert: alert_number is missing");
      return { success: false, error: "alert_number is required" };
    }

    if (!message.fix_description) {
      core.warning("Skipping autofix_code_scanning_alert: fix_description is missing");
      return { success: false, error: "fix_description is required" };
    }

    if (!message.fix_code) {
      core.warning("Skipping autofix_code_scanning_alert: fix_code is missing");
      return { success: false, error: "fix_code is required" };
    }

    // Parse alert number
    const alertNumber = parseInt(String(message.alert_number), 10);
    if (isNaN(alertNumber) || alertNumber <= 0) {
      core.warning(`Invalid alert_number: ${message.alert_number}`);
      return { success: false, error: `Invalid alert_number: ${message.alert_number}` };
    }

    core.info(`Processing autofix_code_scanning_alert: alert_number=${alertNumber}, fix_description="${message.fix_description.substring(0, 50)}..."`);

    // Staged mode: collect for preview
    if (isStaged) {
      processedAutofixes.push({
        alert_number: alertNumber,
        fix_description: message.fix_description,
        fix_code_length: message.fix_code.length,
      });

      return { success: true, staged: true, alertNumber };
    }

    // Create autofix via GitHub REST API
    try {
      core.info(`Creating autofix for code scanning alert ${alertNumber}`);
      core.info(`Fix description: ${message.fix_description}`);
      core.info(`Fix code length: ${message.fix_code.length} characters`);

      // Call the GitHub REST API to create the autofix
      // Reference: https://docs.github.com/en/rest/code-scanning/code-scanning?apiVersion=2022-11-28#create-an-autofix-for-a-code-scanning-alert
      // Note: As of the time of writing, the createAutofix method may not be available in @actions/github
      await github.request("POST /repos/{owner}/{repo}/code-scanning/alerts/{alert_number}/fixes", {
        ...context.repo,
        alert_number: alertNumber,
        fix: {
          description: message.fix_description,
          code: message.fix_code,
        },
        headers: { "X-GitHub-Api-Version": "2022-11-28" },
      });

      const autofixUrl = `https://github.com/${context.repo.owner}/${context.repo.repo}/security/code-scanning/${alertNumber}`;
      core.info(`✓ Successfully created autofix for code scanning alert ${alertNumber}: ${autofixUrl}`);

      processedAutofixes.push({
        alert_number: alertNumber,
        fix_description: message.fix_description,
        url: autofixUrl,
      });

      return { success: true, alertNumber, autofixUrl };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`✗ Failed to create autofix for alert ${alertNumber}: ${errorMessage}`);

      // Provide helpful error messages
      if (errorMessage.includes("404")) {
        core.error(`Alert ${alertNumber} not found. Ensure the alert exists and you have access to it.`);
      } else if (errorMessage.includes("403")) {
        core.error("Permission denied. Ensure the workflow has 'security-events: write' permission.");
      } else if (errorMessage.includes("422")) {
        core.error("Invalid request. Check that the fix_description and fix_code are valid.");
      }

      return { success: false, error: errorMessage };
    }
  };
}

module.exports = { main };
