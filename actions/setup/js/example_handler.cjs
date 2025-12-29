// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Example handler demonstrating the factory pattern for safe output handlers
 * 
 * Factory Pattern:
 * - main(config) is called once with configuration
 * - It returns a message processor function
 * - The message processor receives individual messages and processes them
 * - Returns { temporaryId?, repo, number } for tracking
 */

/**
 * Main factory function - called once with configuration
 * @param {Object} config - Handler configuration from GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG
 * @param {string[]} [config.allowed] - List of allowed labels
 * @param {number} [config.maxCount] - Maximum number of labels
 * @returns {Promise<Function>} Message processor function
 */
async function main(config = {}) {
  core.info(`Initializing example_handler with config: ${JSON.stringify(config)}`);
  
  const { allowed = [], maxCount = 10 } = config;
  
  // Return the message processor function
  return async function processMessage(outputItem, resolvedTemporaryIds) {
    core.info(`Processing example message: ${JSON.stringify(outputItem)}`);
    
    // Example: validate the message
    if (!outputItem.title) {
      core.warning("No title provided in example message");
      return null;
    }
    
    // Example: use configuration
    core.info(`Config allowed: ${allowed.join(",")}, maxCount: ${maxCount}`);
    
    // Example: access resolved temporary IDs
    if (resolvedTemporaryIds.size > 0) {
      core.info(`Resolved temporary IDs: ${Array.from(resolvedTemporaryIds.keys()).join(",")}`);
    }
    
    // Example: perform some action
    const result = {
      title: outputItem.title,
      processed: true,
    };
    
    // If this handler creates a temporary ID, return it
    if (outputItem.temporary_id) {
      return {
        temporaryId: outputItem.temporary_id,
        repo: `${context.repo.owner}/${context.repo.repo}`,
        number: 123, // Example issue number
      };
    }
    
    return result;
  };
}

module.exports = { main };
