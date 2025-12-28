// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Safe Inputs MCP Server Module
 *
 * This module provides a reusable MCP server for safe-inputs configuration.
 * It uses the mcp_server_core module for JSON-RPC handling and tool registration.
 *
 * The server reads tool configuration from a JSON file and loads handlers from
 * JavaScript (.cjs), shell script (.sh), or Python script (.py) files.
 *
 * Usage:
 *   node safe_inputs_mcp_server.cjs /path/to/tools.json
 *
 * Or as a module:
 *   const { startSafeInputsServer } = require("./safe_inputs_mcp_server.cjs");
 *   startSafeInputsServer("/path/to/tools.json");
 */

const { createServer, registerTool, start } = require("./mcp_server_core.cjs");
const { loadConfig } = require("./safe_inputs_config_loader.cjs");
const { createToolConfig } = require("./safe_inputs_tool_factory.cjs");
const { bootstrapSafeInputsServer, cleanupConfigFile } = require("./safe_inputs_bootstrap.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * @typedef {Object} SafeInputsToolConfig
 * @property {string} name - Tool name
 * @property {string} description - Tool description
 * @property {Object} inputSchema - JSON Schema for tool inputs
 * @property {string} [handler] - Path to handler file (.cjs, .sh, or .py)
 */

/**
 * @typedef {Object} SafeInputsConfig
 * @property {string} [serverName] - Server name (defaults to "safeinputs")
 * @property {string} [version] - Server version (defaults to "1.0.0")
 * @property {string} [logDir] - Log directory path
 * @property {SafeInputsToolConfig[]} tools - Array of tool configurations
 */

/**
 * Start the safe-inputs MCP server with the given configuration
 * @param {string} configPath - Path to the configuration JSON file
 * @param {Object} [options] - Additional options
 * @param {string} [options.logDir] - Override log directory from config
 * @param {boolean} [options.skipCleanup] - Skip deletion of config file (useful for stdio mode with agent restarts)
 */
function startSafeInputsServer(configPath, options = {}) {
  // Create server first to have logger available
  const logDir = options.logDir || undefined;
  const server = createServer({ name: "safeinputs", version: "1.0.0" }, { logDir });

  // Bootstrap: load configuration and tools using shared logic
  const { config, tools } = bootstrapSafeInputsServer(configPath, server);

  // Update server info with actual config values
  server.serverInfo.name = config.serverName || "safeinputs";
  server.serverInfo.version = config.version || "1.0.0";

  // Use logDir from config if not overridden by options
  if (!options.logDir && config.logDir) {
    server.logDir = config.logDir;
  }

  // Register all tools with the server
  for (const tool of tools) {
    registerTool(server, tool);
  }

  // Cleanup: delete the configuration file after loading (unless skipCleanup is true)
  if (!options.skipCleanup) {
    cleanupConfigFile(configPath, server);
  }

  // Start the server
  start(server);
}

// If run directly, start the server with command-line arguments
if (require.main === module) {
  const args = process.argv.slice(2);

  if (args.length < 1) {
    console.error("Usage: node safe_inputs_mcp_server.cjs <config.json> [--log-dir <path>]");
    process.exit(1);
  }

  const configPath = args[0];
  const options = {};

  // Parse optional arguments
  for (let i = 1; i < args.length; i++) {
    if (args[i] === "--log-dir" && args[i + 1]) {
      options.logDir = args[i + 1];
      i++;
    }
  }

  try {
    startSafeInputsServer(configPath, options);
  } catch (error) {
    console.error(`Error starting safe-inputs server: ${getErrorMessage(error)}`);
    process.exit(1);
  }
}

module.exports = {
  startSafeInputsServer,
  // Re-export helpers for convenience
  loadConfig,
  createToolConfig,
};
