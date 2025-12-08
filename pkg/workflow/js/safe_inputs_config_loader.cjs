// @ts-check

/**
 * Safe Inputs Configuration Loader
 *
 * This module provides utilities for loading and validating safe-inputs
 * configuration from JSON files.
 */

const fs = require("fs");

/**
 * @typedef {Object} SafeInputsToolConfig
 * @property {string} name - Tool name
 * @property {string} description - Tool description
 * @property {Object} inputSchema - JSON Schema for tool inputs
 * @property {string} [handler] - Path to handler file (.cjs, .sh, or .py)
 * @property {number} [timeout] - Timeout in seconds for tool execution (default: 60)
 */

/**
 * @typedef {Object} SafeInputsConfig
 * @property {string} [serverName] - Server name (defaults to "safeinputs")
 * @property {string} [version] - Server version (defaults to "1.0.0")
 * @property {string} [logDir] - Log directory path
 * @property {SafeInputsToolConfig[]} tools - Array of tool configurations
 */

/**
 * Load safe-inputs configuration from a JSON file
 * @param {string} configPath - Path to the configuration JSON file
 * @returns {SafeInputsConfig} The loaded configuration
 * @throws {Error} If the file doesn't exist or configuration is invalid
 */
function loadConfig(configPath) {
  if (!fs.existsSync(configPath)) {
    throw new Error(`Configuration file not found: ${configPath}`);
  }

  const configContent = fs.readFileSync(configPath, "utf-8");
  const config = JSON.parse(configContent);

  // Validate required fields
  if (!config.tools || !Array.isArray(config.tools)) {
    throw new Error("Configuration must contain a 'tools' array");
  }

  return config;
}

module.exports = {
  loadConfig,
};
