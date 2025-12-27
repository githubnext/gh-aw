// @ts-check

/**
 * Safe Inputs Bootstrap Module
 *
 * This module provides shared bootstrap logic for safe-inputs MCP servers.
 * It handles configuration loading, tool handler loading, and cleanup that is
 * common between stdio and HTTP transport implementations.
 *
 * Usage:
 *   const { bootstrapSafeInputsServer } = require("./safe_inputs_bootstrap.cjs");
 *   const { config, basePath, tools } = bootstrapSafeInputsServer(configPath, logger);
 */

const path = require("path");
const fs = require("fs");
const { loadConfig } = require("./safe_inputs_config_loader.cjs");
const { loadToolHandlers } = require("./mcp_server_core.cjs");

/**
 * @typedef {Object} Logger
 * @property {Function} debug - Debug logging function
 * @property {Function} debugError - Error logging function
 */

/**
 * @typedef {Object} BootstrapResult
 * @property {Object} config - Loaded configuration
 * @property {string} basePath - Base path for resolving handler files
 * @property {Array} tools - Loaded tool handlers
 */

/**
 * Bootstrap a safe-inputs server by loading configuration and tool handlers.
 * This function performs the common initialization steps shared by both stdio
 * and HTTP transport implementations.
 *
 * @param {string} configPath - Path to the configuration JSON file
 * @param {Logger} logger - Logger instance for debug messages
 * @returns {BootstrapResult} Configuration, base path, and loaded tools
 */
function bootstrapSafeInputsServer(configPath, logger) {
  // Load configuration
  logger.debug(`Loading safe-inputs configuration from: ${configPath}`);
  const config = loadConfig(configPath);

  // Determine base path for resolving relative handler paths
  const basePath = path.dirname(configPath);
  logger.debug(`Base path for handlers: ${basePath}`);
  logger.debug(`Tools to load: ${config.tools.length}`);

  // Load tool handlers from file paths
  // Logger implements the MCPServer interface needed for loadToolHandlers
  // prettier-ignore
  const tools = loadToolHandlers(/** @type {any} */ (logger), config.tools, basePath);

  return { config, basePath, tools };
}

/**
 * Delete the configuration file to ensure no secrets remain on disk.
 * This should be called after the server has been configured and started.
 *
 * @param {string} configPath - Path to the configuration file to delete
 * @param {Logger} logger - Logger instance for debug messages
 */
function cleanupConfigFile(configPath, logger) {
  try {
    if (fs.existsSync(configPath)) {
      fs.unlinkSync(configPath);
      logger.debug(`Deleted configuration file: ${configPath}`);
    }
  } catch (error) {
    logger.debugError(`Warning: Could not delete configuration file: `, error);
    // Continue anyway - the server is already running
  }
}

module.exports = {
  bootstrapSafeInputsServer,
  cleanupConfigFile,
};
