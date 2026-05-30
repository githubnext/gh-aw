// @ts-check

/**
 * MCP Scripts Configuration Loader
 *
 * This module provides utilities for loading and validating mcp-scripts
 * configuration from JSON files.
 */

const fs = require("fs");
const { ERR_SYSTEM, ERR_VALIDATION } = require("./error_codes.cjs");

/**
 * @typedef {Object} MCPScriptsToolConfig
 * @property {string} name - Tool name
 * @property {string} description - Tool description
 * @property {Object} inputSchema - JSON Schema for tool inputs
 * @property {string} [handler] - Path to handler file (.cjs, .sh, or .py)
 * @property {number} [timeout] - Timeout in seconds for tool execution (default: 60)
 */

/**
 * @typedef {Object} MCPScriptsConfig
 * @property {string} [serverName] - Server name (defaults to "mcpscripts")
 * @property {string} [version] - Server version (defaults to "1.0.0")
 * @property {string} [logDir] - Log directory path
 * @property {MCPScriptsToolConfig[]} tools - Array of tool configurations
 */

/**
 * Load mcp-scripts configuration from a JSON file
 * @param {string} configPath - Path to the configuration JSON file
 * @returns {MCPScriptsConfig} The loaded configuration
 * @throws {Error} If the file doesn't exist or configuration is invalid
 */
function loadConfig(configPath) {
  if (!fs.existsSync(configPath)) {
    throw new Error(`${ERR_SYSTEM}: Configuration file not found: ${configPath}`);
  }

  const configContent = fs.readFileSync(configPath, "utf-8");
  const config = JSON.parse(configContent);

  // Validate required fields
  if (!config.tools || !Array.isArray(config.tools)) {
    throw new Error(`${ERR_VALIDATION}: Configuration must contain a 'tools' array`);
  }

  return config;
}

module.exports = {
  loadConfig,
};
