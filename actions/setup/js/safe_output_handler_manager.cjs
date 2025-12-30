// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Safe Output Handler Manager
 *
 * This module manages the dispatch of safe output messages to dedicated handlers.
 * It reads configuration, loads the appropriate handlers for enabled safe output types,
 * and processes messages from the agent output file while maintaining a shared temporary ID map.
 */

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Handler map configuration
 * Maps safe output types to their handler module file paths
 */
const HANDLER_MAP = {
  create_issue: "./create_issue.cjs",
  add_comment: "./add_comment.cjs",
  create_discussion: "./create_discussion.cjs",
  close_issue: "./close_issue.cjs",
  close_discussion: "./close_discussion.cjs",
  add_labels: "./add_labels.cjs",
  update_issue: "./update_issue.cjs",
  update_discussion: "./update_discussion.cjs",
};

/**
 * Load configuration for safe outputs
 * Reads configuration from GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG environment variable
 * @returns {Object} Safe outputs configuration
 */
function loadConfig() {
  if (!process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG) {
    throw new Error("GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG environment variable is required but not set");
  }

  try {
    const config = JSON.parse(process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG);
    core.info(`Loaded config from GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ${JSON.stringify(config)}`);
    // Normalize config keys: convert hyphens to underscores
    return Object.fromEntries(Object.entries(config).map(([k, v]) => [k.replace(/-/g, "_"), v]));
  } catch (error) {
    throw new Error(`Failed to parse GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ${getErrorMessage(error)}`);
  }
}

/**
 * Load and initialize handlers for enabled safe output types
 * Calls each handler's factory function (main) to get message processors
 * @param {Object} config - Safe outputs configuration
 * @returns {Promise<Map<string, Function>>} Map of type to message handler function
 */
async function loadHandlers(config) {
  const messageHandlers = new Map();

  core.info("Loading and initializing safe output handlers based on configuration...");

  for (const [type, handlerPath] of Object.entries(HANDLER_MAP)) {
    // Check if this safe output type is enabled in the config
    // The presence of the config key indicates the handler should be loaded
    if (config[type]) {
      try {
        const handlerModule = require(handlerPath);
        if (handlerModule && typeof handlerModule.main === "function") {
          // Call the factory function with config to get the message handler
          const handlerConfig = config[type] || {};
          const messageHandler = await handlerModule.main(handlerConfig);
          
          if (typeof messageHandler !== "function") {
            core.warning(`Handler ${type} main() did not return a function`);
            continue;
          }
          
          messageHandlers.set(type, messageHandler);
          core.info(`✓ Loaded and initialized handler for: ${type}`);
        } else {
          core.warning(`Handler module ${type} does not export a main function`);
        }
      } catch (error) {
        core.warning(`Failed to load handler for ${type}: ${getErrorMessage(error)}`);
      }
    } else {
      core.debug(`Handler not enabled: ${type}`);
    }
  }

  core.info(`Loaded ${messageHandlers.size} handler(s)`);
  return messageHandlers;
}

/**
 * Process all messages from agent output in the order they appear
 * Dispatches each message to the appropriate handler while maintaining shared state (temporary ID map)
 *
 * @param {Map<string, Function>} messageHandlers - Map of message handler functions
 * @param {Array<Object>} messages - Array of safe output messages
 * @returns {Promise<{success: boolean, results: Array<any>, temporaryIdMap: Map}>}
 */
async function processMessages(messageHandlers, messages) {
  const results = [];

  // Initialize shared temporary ID map
  // This will be populated by handlers as they create entities with temporary IDs
  /** @type {Map<string, {repo: string, number: number}>} */
  const temporaryIdMap = new Map();

  core.info(`Processing ${messages.length} message(s) in order of appearance...`);

  // Process messages in order of appearance
  for (let i = 0; i < messages.length; i++) {
    const message = messages[i];
    const messageType = message.type;

    if (!messageType) {
      core.warning(`Skipping message ${i + 1} without type`);
      continue;
    }

    const messageHandler = messageHandlers.get(messageType);

    if (!messageHandler) {
      core.debug(`No handler for type: ${messageType} (message ${i + 1})`);
      continue;
    }

    try {
      core.info(`Processing message ${i + 1}/${messages.length}: ${messageType}`);

      // Convert Map to plain object for handler
      const resolvedTemporaryIds = Object.fromEntries(temporaryIdMap);

      // Call the message handler with the individual message and resolved temp IDs
      const result = await messageHandler(message, resolvedTemporaryIds);

      // If handler returned a temp ID mapping, add it to our map
      if (result && result.temporaryId && result.repo && result.number) {
        temporaryIdMap.set(result.temporaryId, {
          repo: result.repo,
          number: result.number,
        });
        core.info(`Registered temporary ID: ${result.temporaryId} -> ${result.repo}#${result.number}`);
      }

      results.push({
        type: messageType,
        messageIndex: i,
        success: true,
        result,
      });

      core.info(`✓ Message ${i + 1} (${messageType}) completed successfully`);
    } catch (error) {
      core.error(`✗ Message ${i + 1} (${messageType}) failed: ${getErrorMessage(error)}`);
      results.push({
        type: messageType,
        messageIndex: i,
        success: false,
        error: getErrorMessage(error),
      });
    }
  }

  // Convert temporaryIdMap to plain object for serialization
  const temporaryIdMapObj = Object.fromEntries(temporaryIdMap);

  return {
    success: true,
    results,
    temporaryIdMap: temporaryIdMapObj,
  };
}

/**
 * Main entry point for the handler manager
 * This is called by the consolidated safe output step
 *
 * @returns {Promise<void>}
 */
async function main() {
  try {
    core.info("Safe Output Handler Manager starting...");

    // Load configuration
    const config = loadConfig();
    core.debug(`Configuration: ${JSON.stringify(Object.keys(config))}`);

    // Load agent output
    const agentOutput = loadAgentOutput();
    if (!agentOutput.success) {
      core.info("No agent output available - nothing to process");
      return;
    }

    core.info(`Found ${agentOutput.items.length} message(s) in agent output`);

    // Load and initialize handlers based on configuration (factory pattern)
    const messageHandlers = await loadHandlers(config);

    if (messageHandlers.size === 0) {
      core.info("No handlers loaded - nothing to process");
      return;
    }

    // Process all messages in order of appearance
    const processingResult = await processMessages(messageHandlers, agentOutput.items);

    // Log summary
    const successCount = processingResult.results.filter((r) => r.success).length;
    const failureCount = processingResult.results.filter((r) => !r.success).length;

    core.info(`\n=== Processing Summary ===`);
    core.info(`Total messages: ${processingResult.results.length}`);
    core.info(`Successful: ${successCount}`);
    core.info(`Failed: ${failureCount}`);
    core.info(`Temporary IDs registered: ${Object.keys(processingResult.temporaryIdMap).length}`);

    if (failureCount > 0) {
      core.warning(`${failureCount} message(s) failed to process`);
    }

    core.info("Safe Output Handler Manager completed");
  } catch (error) {
    core.setFailed(`Handler manager failed: ${getErrorMessage(error)}`);
  }
}

module.exports = { main };
