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
const fs = require("fs");
const path = require("path");

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
 * Tries to load from config file first, then falls back to inferring from environment variables
 * @returns {Object} Safe outputs configuration
 */
function loadConfig() {
  const configPath = process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH || "/tmp/gh-aw/safeoutputs/config.json";

  // Try to load from config file first
  try {
    if (fs.existsSync(configPath)) {
      const configContent = fs.readFileSync(configPath, "utf8");
      const config = JSON.parse(configContent);

      // Normalize config keys: convert hyphens to underscores
      return Object.fromEntries(Object.entries(config).map(([k, v]) => [k.replace(/-/g, "_"), v]));
    }
  } catch (error) {
    core.debug(`Failed to load config from file: ${getErrorMessage(error)}`);
  }

  // Fallback: infer config from environment variables
  // When running in the safe_outputs job, the config file doesn't exist,
  // but individual handler env vars are present (e.g., GH_AW_ISSUE_EXPIRES, GH_AW_HIDE_OLDER_COMMENTS)
  core.debug("Config file not found, inferring configuration from environment variables");
  const config = {};

  // Check for create_issue indicators
  if (process.env.GH_AW_ISSUE_EXPIRES || process.env.GH_AW_ISSUE_TITLE_PREFIX || process.env.GH_AW_ISSUE_LABELS || process.env.GH_AW_ISSUE_ALLOWED_LABELS) {
    config.create_issue = { enabled: true };
  }

  // Check for add_comment indicators
  if (process.env.GH_AW_COMMENT_TARGET || process.env.GH_AW_HIDE_OLDER_COMMENTS || process.env.GITHUB_AW_COMMENT_DISCUSSION) {
    config.add_comment = { enabled: true };
  }

  // Check for create_discussion indicators (always enable if other safe outputs are present, as it's common)
  if (Object.keys(config).length > 0) {
    config.create_discussion = { enabled: true };
  }

  // Check for close_issue indicators (always enable if create_issue is present)
  if (config.create_issue) {
    config.close_issue = { enabled: true };
  }

  // Check for close_discussion indicators (always enable if create_discussion is present)
  if (config.create_discussion) {
    config.close_discussion = { enabled: true };
  }

  // Check for add_labels indicators
  if (process.env.GH_AW_LABELS_ALLOWED || process.env.GH_AW_LABELS_MAX_COUNT) {
    config.add_labels = { enabled: true };
  }

  // Check for update_issue indicators (always enable if create_issue is present)
  if (config.create_issue) {
    config.update_issue = { enabled: true };
  }

  // Check for update_discussion indicators (always enable if create_discussion is present)
  if (config.create_discussion) {
    config.update_discussion = { enabled: true };
  }

  core.debug(`Inferred config: ${JSON.stringify(config)}`);
  return config;
}

/**
 * Load handlers for enabled safe output types
 * @param {Object} config - Safe outputs configuration
 * @returns {Map<string, {main: Function}>} Map of type to handler module
 */
function loadHandlers(config) {
  const handlers = new Map();

  core.info("Loading safe output handlers based on configuration...");

  for (const [type, handlerPath] of Object.entries(HANDLER_MAP)) {
    // Check if this safe output type is enabled in the config
    // Config keys use underscores (e.g., create_issue)
    const configKey = type;

    // Check if handler is enabled (config entry exists)
    // The presence of the config key indicates the handler should be loaded
    if (config[configKey]) {
      try {
        const handler = require(handlerPath);
        if (handler && typeof handler.main === "function") {
          handlers.set(type, handler);
          core.info(`✓ Loaded handler for: ${type}`);
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

  core.info(`Loaded ${handlers.size} handler(s)`);
  return handlers;
}

/**
 * Process all messages from agent output in the order they appear
 * Dispatches each message to the appropriate handler while maintaining shared state (temporary ID map)
 *
 * @param {Map<string, {main: Function}>} handlers - Map of loaded handlers
 * @param {Array<Object>} messages - Array of safe output messages
 * @returns {Promise<{success: boolean, results: Array<any>, temporaryIdMap: Object}>}
 */
async function processMessages(handlers, messages) {
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

    const handler = handlers.get(messageType);

    if (!handler) {
      core.debug(`No handler for type: ${messageType} (message ${i + 1})`);
      continue;
    }

    try {
      core.info(`Processing message ${i + 1}/${messages.length}: ${messageType}`);

      // Call the handler's main function
      // The handler will access agent output internally via loadAgentOutput()
      // and will populate/use the temporaryIdMap as needed
      const result = await handler.main();

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

    // Load handlers based on configuration
    const handlers = loadHandlers(config);

    if (handlers.size === 0) {
      core.info("No handlers enabled in configuration");
      return;
    }

    // Process all messages with loaded handlers
    const result = await processMessages(handlers, agentOutput.items);

    // Log summary
    core.info("=== Processing Summary ===");
    core.info(`Total handlers invoked: ${result.results.length}`);
    core.info(`Successful: ${result.results.filter(r => r.success).length}`);
    core.info(`Failed: ${result.results.filter(r => !r.success).length}`);

    // Set outputs for downstream steps
    core.setOutput("temporary_id_map", JSON.stringify(result.temporaryIdMap));
    core.setOutput("processed_count", result.results.length);

    core.info("Safe Output Handler Manager completed successfully");
  } catch (error) {
    const errorMsg = getErrorMessage(error);
    core.error(`Handler manager failed: ${errorMsg}`);
    core.setFailed(errorMsg);
  }
}

module.exports = {
  main,
  loadConfig,
  loadHandlers,
  processMessages,
};
