// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Safe Output Handler Manager
 * 
 * This manager orchestrates the processing of safe output messages by:
 * 1. Loading handler configuration from GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG
 * 2. Loading agent output items from GH_AW_AGENT_OUTPUT
 * 3. Creating handler factory functions for each enabled handler type
 * 4. Processing messages sequentially, maintaining a shared temporary ID map
 * 
 * Each handler is converted to a factory pattern:
 *   async function main(config = {}) {
 *     return async function(outputItem, resolvedTemporaryIds) {
 *       // Process single message
 *       // Return { temporaryId?, repo, number }
 *     };
 *   }
 */

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Load handler configuration from environment variable
 * @returns {{success: true, config: Object} | {success: false, error?: string}}
 */
function loadHandlerConfig() {
  const configEnv = process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG;
  
  if (!configEnv) {
    core.info("No GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG environment variable found");
    return { success: false };
  }
  
  try {
    const config = JSON.parse(configEnv);
    core.info(`Loaded handler configuration with ${Object.keys(config).length} handler types`);
    return { success: true, config };
  } catch (error) {
    const errorMessage = `Error parsing handler configuration JSON: ${getErrorMessage(error)}`;
    core.error(errorMessage);
    return { success: false, error: errorMessage };
  }
}

/**
 * Main entry point for the handler manager
 */
async function main() {
  core.info("Starting safe output handler manager");
  
  // Load agent output
  const outputResult = loadAgentOutput();
  if (!outputResult.success) {
    core.info("No agent output to process");
    return;
  }
  
  const items = outputResult.items;
  if (items.length === 0) {
    core.info("No items found in agent output");
    return;
  }
  
  core.info(`Found ${items.length} item(s) in agent output`);
  
  // Load handler configuration
  const configResult = loadHandlerConfig();
  if (!configResult.success) {
    core.info("No handler configuration found");
    return;
  }
  
  const handlerConfig = configResult.config;
  
  // Check if staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";
  if (isStaged) {
    core.info("Running in staged mode");
  }
  
  // Initialize shared temporary ID map
  // Map<string, {repo: string, number: number}>
  const resolvedTemporaryIds = new Map();
  
  // Create handler factories for each handler type
  const handlerFactories = new Map();
  
  // Get unique handler types from items
  const handlerTypes = new Set(items.map(item => item.type));
  core.info(`Handler types needed: ${Array.from(handlerTypes).join(", ")}`);
  
  // Load and initialize handlers
  for (const handlerType of handlerTypes) {
    const config = handlerConfig[handlerType] || {};
    
    try {
      // Convert handler type to module name (e.g., create_issue -> ./create_issue.cjs)
      const handlerModule = require(`./${handlerType}.cjs`);
      
      if (typeof handlerModule.main !== "function") {
        core.warning(`Handler ${handlerType} does not export a main function, skipping`);
        continue;
      }
      
      // Create handler factory by calling main(config)
      // This returns a function that processes individual messages
      const messageHandler = await handlerModule.main(config);
      
      if (typeof messageHandler !== "function") {
        core.warning(`Handler ${handlerType} main() did not return a function, skipping`);
        continue;
      }
      
      handlerFactories.set(handlerType, messageHandler);
      core.info(`Initialized handler: ${handlerType}`);
    } catch (error) {
      core.error(`Failed to load handler ${handlerType}: ${getErrorMessage(error)}`);
      // Continue with other handlers
    }
  }
  
  if (handlerFactories.size === 0) {
    core.warning("No handlers were successfully initialized");
    return;
  }
  
  // Process each item sequentially
  let processedCount = 0;
  let errorCount = 0;
  
  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    const handlerType = item.type;
    
    core.info(`Processing item ${i + 1}/${items.length}: type=${handlerType}`);
    
    const messageHandler = handlerFactories.get(handlerType);
    if (!messageHandler) {
      core.warning(`No handler found for type: ${handlerType}, skipping`);
      continue;
    }
    
    try {
      // Call the message handler with the item and temporary ID map
      const result = await messageHandler(item, resolvedTemporaryIds);
      
      // If handler returns a temporary ID mapping, add it to the map
      if (result && result.temporaryId) {
        const repo = result.repo || `${context.repo.owner}/${context.repo.repo}`;
        const number = result.number;
        
        if (number) {
          resolvedTemporaryIds.set(result.temporaryId.toLowerCase(), { repo, number });
          core.info(`Mapped temporary ID ${result.temporaryId} -> ${repo}#${number}`);
        }
      }
      
      processedCount++;
    } catch (error) {
      core.error(`Failed to process item ${i + 1} (type=${handlerType}): ${getErrorMessage(error)}`);
      errorCount++;
      // Continue processing other items
    }
  }
  
  core.info(`Processed ${processedCount} items successfully, ${errorCount} errors`);
  
  // Set output with temporary ID map for downstream steps
  if (resolvedTemporaryIds.size > 0) {
    const tempIdMapObj = {};
    for (const [tempId, value] of resolvedTemporaryIds) {
      tempIdMapObj[tempId] = value;
    }
    core.setOutput("temporary_id_map", JSON.stringify(tempIdMapObj));
  }
}

module.exports = { main, loadHandlerConfig };
