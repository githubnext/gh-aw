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
const { hasUnresolvedTemporaryIds, replaceTemporaryIdReferences, normalizeTemporaryId } = require("./temporary_id.cjs");

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
 * Tracks outputs created with unresolved temporary IDs and generates synthetic updates after resolution
 *
 * @param {Map<string, Function>} messageHandlers - Map of message handler functions
 * @param {Array<Object>} messages - Array of safe output messages
 * @returns {Promise<{success: boolean, results: Array<any>, temporaryIdMap: Map, pendingUpdates: Array<any>}>}
 */
async function processMessages(messageHandlers, messages) {
  const results = [];

  // Initialize shared temporary ID map
  // This will be populated by handlers as they create entities with temporary IDs
  /** @type {Map<string, {repo: string, number: number}>} */
  const temporaryIdMap = new Map();

  // Track outputs that were created with unresolved temporary IDs
  // Format: {type, message, result, originalTempIdMapSize}
  /** @type {Array<{type: string, message: any, result: any, originalTempIdMapSize: number}>} */
  const outputsWithUnresolvedIds = [];

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
      
      // Record the temp ID map size before processing to detect new IDs
      const tempIdMapSizeBefore = temporaryIdMap.size;

      // Call the message handler with the individual message and resolved temp IDs
      const result = await messageHandler(message, resolvedTemporaryIds);

      // If handler returned a temp ID mapping, add it to our map
      if (result && result.temporaryId && result.repo && result.number) {
        const normalizedTempId = normalizeTemporaryId(result.temporaryId);
        temporaryIdMap.set(normalizedTempId, {
          repo: result.repo,
          number: result.number,
        });
        core.info(`Registered temporary ID: ${result.temporaryId} -> ${result.repo}#${result.number}`);
      }

      // Check if this output was created with unresolved temporary IDs
      // For create_issue, create_discussion, add_comment - check if body has unresolved IDs
      if (result && result.number && result.repo) {
        const contentToCheck = getContentToCheck(messageType, message);
        if (contentToCheck && hasUnresolvedTemporaryIds(contentToCheck, temporaryIdMap)) {
          core.info(`Output ${result.repo}#${result.number} was created with unresolved temporary IDs - tracking for update`);
          outputsWithUnresolvedIds.push({
            type: messageType,
            message: message,
            result: result,
            originalTempIdMapSize: tempIdMapSizeBefore,
          });
        }
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

  // After processing all original messages, check if any new temporary IDs were resolved
  // If so, generate synthetic update messages for outputs that had unresolved IDs
  const pendingUpdates = [];
  if (outputsWithUnresolvedIds.length > 0) {
    core.info(`\n=== Checking for Synthetic Updates ===`);
    core.info(`Found ${outputsWithUnresolvedIds.length} output(s) with unresolved temporary IDs`);
    
    for (const tracked of outputsWithUnresolvedIds) {
      // Check if any new temporary IDs were resolved since this output was created
      if (temporaryIdMap.size > tracked.originalTempIdMapSize) {
        const contentToCheck = getContentToCheck(tracked.type, tracked.message);
        
        // Check if the content still has unresolved IDs (some may now be resolved)
        const stillHasUnresolved = hasUnresolvedTemporaryIds(contentToCheck, temporaryIdMap);
        const resolvedCount = temporaryIdMap.size - tracked.originalTempIdMapSize;
        
        if (!stillHasUnresolved) {
          // All temporary IDs are now resolved - generate synthetic update
          core.info(`Generating synthetic update for ${tracked.result.repo}#${tracked.result.number} (${resolvedCount} temp ID(s) resolved)`);
          
          const syntheticUpdate = createSyntheticUpdate(tracked, temporaryIdMap);
          if (syntheticUpdate) {
            pendingUpdates.push(syntheticUpdate);
          }
        } else {
          core.debug(`Output ${tracked.result.repo}#${tracked.result.number} still has unresolved temporary IDs`);
        }
      }
    }
    
    if (pendingUpdates.length > 0) {
      core.info(`Generated ${pendingUpdates.length} synthetic update(s)`);
    } else {
      core.info(`No synthetic updates needed`);
    }
  }

  // Convert temporaryIdMap to plain object for serialization
  const temporaryIdMapObj = Object.fromEntries(temporaryIdMap);

  return {
    success: true,
    results,
    temporaryIdMap: temporaryIdMapObj,
    pendingUpdates,
  };
}

/**
 * Get the content field to check for unresolved temporary IDs based on message type
 * @param {string} messageType - Type of the message
 * @param {any} message - The message object
 * @returns {string|null} Content to check for temporary IDs
 */
function getContentToCheck(messageType, message) {
  switch (messageType) {
    case "create_issue":
      return message.body || "";
    case "create_discussion":
      return message.body || "";
    case "add_comment":
      return message.body || "";
    default:
      return null;
  }
}

/**
 * Create a synthetic update message for an output with now-resolved temporary IDs
 * @param {{type: string, message: any, result: any}} tracked - Tracked output info
 * @param {Map<string, {repo: string, number: number}>} temporaryIdMap - Current temporary ID map
 * @returns {Object|null} Synthetic update message or null if not applicable
 */
function createSyntheticUpdate(tracked, temporaryIdMap) {
  const { type, message, result } = tracked;
  
  // Get the original content with unresolved temporary IDs
  const originalContent = getContentToCheck(type, message);
  if (!originalContent) {
    return null;
  }
  
  // Replace temporary ID references with resolved values
  const updatedContent = replaceTemporaryIdReferences(originalContent, temporaryIdMap, result.repo);
  
  // Generate appropriate update message based on the original type
  switch (type) {
    case "create_issue":
      return {
        type: "update_issue",
        issue_number: result.number,
        body: updatedContent,
        _synthetic: true,
        _original_type: type,
      };
    case "create_discussion":
      return {
        type: "update_discussion",
        discussion_number: result.number,
        body: updatedContent,
        _synthetic: true,
        _original_type: type,
      };
    case "add_comment":
      // For comments, we would need to update the comment, but GitHub API doesn't support
      // updating comments easily without the comment ID. Skip for now.
      core.debug(`Skipping synthetic update for comment - comment updates not yet supported`);
      return null;
    default:
      return null;
  }
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

    // Process synthetic updates if any were generated
    let syntheticUpdateCount = 0;
    if (processingResult.pendingUpdates && processingResult.pendingUpdates.length > 0) {
      core.info(`\n=== Processing Synthetic Updates ===`);
      
      for (const syntheticUpdate of processingResult.pendingUpdates) {
        const updateType = syntheticUpdate.type;
        const messageHandler = messageHandlers.get(updateType);
        
        if (!messageHandler) {
          core.warning(`No handler for synthetic update type: ${updateType}`);
          continue;
        }
        
        try {
          core.info(`Processing synthetic ${updateType} for ${syntheticUpdate._original_type}`);
          
          // Convert temp ID map to plain object for handler
          const resolvedTemporaryIds = processingResult.temporaryIdMap;
          
          // Call the message handler with the synthetic update
          await messageHandler(syntheticUpdate, resolvedTemporaryIds);
          
          syntheticUpdateCount++;
          core.info(`✓ Synthetic update completed`);
        } catch (error) {
          core.warning(`✗ Synthetic update failed: ${getErrorMessage(error)}`);
        }
      }
      
      core.info(`Processed ${syntheticUpdateCount}/${processingResult.pendingUpdates.length} synthetic update(s)`);
    }

    // Log summary
    const successCount = processingResult.results.filter((r) => r.success).length;
    const failureCount = processingResult.results.filter((r) => !r.success).length;

    core.info(`\n=== Processing Summary ===`);
    core.info(`Total messages: ${processingResult.results.length}`);
    core.info(`Successful: ${successCount}`);
    core.info(`Failed: ${failureCount}`);
    core.info(`Temporary IDs registered: ${Object.keys(processingResult.temporaryIdMap).length}`);
    core.info(`Synthetic updates: ${syntheticUpdateCount}`);

    if (failureCount > 0) {
      core.warning(`${failureCount} message(s) failed to process`);
    }

    core.info("Safe Output Handler Manager completed");
  } catch (error) {
    core.setFailed(`Handler manager failed: ${getErrorMessage(error)}`);
  }
}

module.exports = { main, loadConfig, loadHandlers, processMessages };
