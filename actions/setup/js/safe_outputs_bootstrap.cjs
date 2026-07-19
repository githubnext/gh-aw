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
const { ERR_CONFIG } = require("./error_codes.cjs");

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

  enforceCreatePullRequestRuntimePolicy(config, logger);

  // Load tools
  logger.debug("Loading safe-outputs tools");
  const tools = loadTools(logger);

  return { config, outputFile, tools };
}

/**
 * Refuse startup when runtime policy disables create-pull-request.
 * @param {Record<string, any>} config
 * @param {Logger} logger
 */
function enforceCreatePullRequestRuntimePolicy(config, logger) {
  const policyVarName = "GH_AW_POLICY_ALLOW_CREATE_PULL_REQUEST";
  const rawValue = process.env[policyVarName];
  const normalizedValue = typeof rawValue === "string" ? rawValue.trim().toLowerCase() : "";
  // config is always snake_case after loadConfig normalises keys (k.replace(/-/g, '_'))
  const createPullRequestConfigured = !!config && Object.prototype.hasOwnProperty.call(config, "create_pull_request");

  if (!createPullRequestConfigured || normalizedValue !== "false") {
    return;
  }

  const message = `create-pull-request is disabled by runtime policy: ${policyVarName}=false. ` + `Remove safe-outputs.create-pull-request or set ${policyVarName}=true.`;
  logger.debugError(message);
  throw new Error(`${ERR_CONFIG}: ${message}`);
}

/**
 * Delete the configuration file to ensure no secrets remain on disk.
 * This should be called after the server has been configured and started.
 *
 * @param {Logger} logger - Logger instance for debug messages
 */
function cleanupConfigFile(logger) {
  const configPath = process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH || `${process.env.RUNNER_TEMP}/gh-aw/safeoutputs/config.json`;

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
  enforceCreatePullRequestRuntimePolicy,
};
