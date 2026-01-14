// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "add_code_scanning_autofix";

/**
 * Main handler factory for add_code_scanning_autofix
 * Returns a message handler function that processes individual add_code_scanning_autofix messages
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
   * Message handler function that processes a single add_code_scanning_autofix message
   * @param {Object} message - The add_code_scanning_autofix message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleAddCodeScanningAutofix(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping add_code_scanning_autofix: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const autofixItem = message;

    // Validate required fields
    if (autofixItem.alert_number === undefined || autofixItem.alert_number === null) {
      core.warning("Skipping add_code_scanning_autofix: alert_number is missing");
      return {
        success: false,
        error: "alert_number is required",
      };
    }

    if (!autofixItem.fix_description) {
      core.warning("Skipping add_code_scanning_autofix: fix_description is missing");
      return {
        success: false,
        error: "fix_description is required",
      };
    }

    if (!autofixItem.fix_code) {
      core.warning("Skipping add_code_scanning_autofix: fix_code is missing");
      return {
        success: false,
        error: "fix_code is required",
      };
    }

    // Parse alert number
    const alertNumber = parseInt(String(autofixItem.alert_number), 10);
    if (isNaN(alertNumber) || alertNumber <= 0) {
      core.warning(`Invalid alert_number: ${autofixItem.alert_number}`);
      return {
        success: false,
        error: `Invalid alert_number: ${autofixItem.alert_number}`,
      };
    }

    core.info(`Processing add_code_scanning_autofix: alert_number=${alertNumber}, fix_description="${autofixItem.fix_description.substring(0, 50)}..."`);

    // Staged mode: collect for preview
    if (isStaged) {
      processedAutofixes.push({
        alert_number: alertNumber,
        fix_description: autofixItem.fix_description,
        fix_code_length: autofixItem.fix_code.length,
      });

      return {
        success: true,
        staged: true,
        alertNumber,
      };
    }

    // Create autofix via GitHub REST API
    try {
      core.info(`Creating autofix for code scanning alert ${alertNumber}`);
      core.info(`Fix description: ${autofixItem.fix_description}`);
      core.info(`Fix code length: ${autofixItem.fix_code.length} characters`);

      // Call the GitHub REST API to create the autofix
      // Reference: https://docs.github.com/en/rest/code-scanning/code-scanning?apiVersion=2022-11-28#create-an-autofix-for-a-code-scanning-alert
      // Note: As of the time of writing, the createAutofix method may not be available in @actions/github
      // We'll use the generic request method to call the API endpoint directly
      const result = await github.request("POST /repos/{owner}/{repo}/code-scanning/alerts/{alert_number}/fixes", {
        owner: context.repo.owner,
        repo: context.repo.repo,
        alert_number: alertNumber,
        fix: {
          description: autofixItem.fix_description,
          code: autofixItem.fix_code,
        },
        headers: {
          "X-GitHub-Api-Version": "2022-11-28",
        },
      });

      const autofixUrl = `https://github.com/${context.repo.owner}/${context.repo.repo}/security/code-scanning/${alertNumber}`;
      core.info(`✓ Successfully created autofix for code scanning alert ${alertNumber}: ${autofixUrl}`);

      processedAutofixes.push({
        alert_number: alertNumber,
        fix_description: autofixItem.fix_description,
        url: autofixUrl,
      });

      return {
        success: true,
        alertNumber,
        autofixUrl,
      };
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

      return {
        success: false,
        error: errorMessage,
      };
    }
  };
}

module.exports = { main };
