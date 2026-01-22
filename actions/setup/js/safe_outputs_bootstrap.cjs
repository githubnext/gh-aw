// @ts-check

/**
 * Safe Outputs Bootstrap Module
 *
 * This module provides shared bootstrap logic for safe-outputs MCP server.
 * It handles configuration loading, tools loading, and cleanup that is
 * common initialization logic.
 *
 * Usage:
 *   const { bootstrapSafeOutputsServer } = require("./safe_outputs_bootstrap.cjs");
 *   const { config, outputFile, tools } = bootstrapSafeOutputsServer(server);
 */

const fs = require("fs");
const { loadConfig } = require("./safe_outputs_config.cjs");
const { loadTools } = require("./safe_outputs_tools_loader.cjs");

/**
 * @typedef {Object} Logger
 * @property {Function} debug - Debug logging function
 * @property {Function} debugError - Error logging function
 */

/**
 * @typedef {Object} BootstrapResult
 * @property {Object} config - Loaded configuration
 * @property {string} outputFile - Path to the output file
 * @property {Array} tools - Loaded tool definitions
 */

/**
 * Bootstrap a safe-outputs server by loading configuration and tools.
 * This function performs the common initialization steps.
 *
 * @param {Logger} logger - Logger instance for debug messages
 * @returns {BootstrapResult} Configuration, output file path, and loaded tools
 */
function bootstrapSafeOutputsServer(logger) {
  // Load configuration
  logger.debug("Loading safe-outputs configuration");
  const { config, outputFile } = loadConfig(logger);

  // Load tools
  logger.debug("Loading safe-outputs tools");
  const tools = loadTools(logger);

  return { config, outputFile, tools };
}

/**
 * Delete the configuration file to ensure no secrets remain on disk.
 * This should be called after the server has been configured and started.
 *
 * @param {Logger} logger - Logger instance for debug messages
 */
function cleanupConfigFile(logger) {
  const configPath = process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH || "/opt/gh-aw/safeoutputs/config.json";

  try {
    if (fs.existsSync(configPath)) {
      fs.unlinkSync(configPath);
      logger.debug(`Deleted configuration file: ${configPath}`);
    }
  } catch (error) {
    logger.debugError("Warning: Could not delete configuration file: ", error);
    // Continue anyway - the server is already running
  }
}

module.exports = {
  bootstrapSafeOutputsServer,
  cleanupConfigFile,
};
