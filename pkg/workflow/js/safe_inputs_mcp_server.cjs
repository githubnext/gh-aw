// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Safe Inputs MCP Server Module
 *
 * This module provides a reusable MCP server for safe-inputs configuration.
 * It uses the mcp_server_core module for JSON-RPC handling and tool registration.
 *
 * The server reads tool configuration from a JSON file and loads handlers from
 * JavaScript (.cjs) or shell script (.sh) files.
 *
 * Usage:
 *   node safe_inputs_mcp_server.cjs /path/to/tools.json
 *
 * Or as a module:
 *   const { startSafeInputsServer } = require("./safe_inputs_mcp_server.cjs");
 *   startSafeInputsServer("/path/to/tools.json");
 */

const fs = require("fs");
const path = require("path");
const { createServer, registerTool, loadToolHandlers, start } = require("./mcp_server_core.cjs");

/**
 * @typedef {Object} SafeInputsToolConfig
 * @property {string} name - Tool name
 * @property {string} description - Tool description
 * @property {Object} inputSchema - JSON Schema for tool inputs
 * @property {string} [handler] - Path to handler file (.cjs or .sh)
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

/**
 * Start the safe-inputs MCP server with the given configuration
 * @param {string} configPath - Path to the configuration JSON file
 * @param {Object} [options] - Additional options
 * @param {string} [options.logDir] - Override log directory from config
 */
function startSafeInputsServer(configPath, options = {}) {
  // Load configuration
  const config = loadConfig(configPath);

  // Determine base path for resolving relative handler paths
  const basePath = path.dirname(configPath);

  // Create server with configuration
  const serverName = config.serverName || "safeinputs";
  const version = config.version || "1.0.0";
  const logDir = options.logDir || config.logDir || undefined;

  const server = createServer({ name: serverName, version }, { logDir });

  server.debug(`Loading safe-inputs configuration from: ${configPath}`);
  server.debug(`Base path for handlers: ${basePath}`);
  server.debug(`Tools to load: ${config.tools.length}`);

  // Load tool handlers from file paths
  const tools = loadToolHandlers(server, config.tools, basePath);

  // Register all tools with the server
  for (const tool of tools) {
    registerTool(server, tool);
  }

  // Start the server
  start(server);
}

/**
 * Create tool configuration for a JavaScript handler
 * @param {string} name - Tool name
 * @param {string} description - Tool description
 * @param {Object} inputSchema - JSON Schema for tool inputs
 * @param {string} handlerPath - Relative path to the .cjs handler file
 * @returns {SafeInputsToolConfig} Tool configuration object
 */
function createJsToolConfig(name, description, inputSchema, handlerPath) {
  return {
    name,
    description,
    inputSchema,
    handler: handlerPath,
  };
}

/**
 * Create tool configuration for a shell script handler
 * @param {string} name - Tool name
 * @param {string} description - Tool description
 * @param {Object} inputSchema - JSON Schema for tool inputs
 * @param {string} handlerPath - Relative path to the .sh handler file
 * @returns {SafeInputsToolConfig} Tool configuration object
 */
function createShellToolConfig(name, description, inputSchema, handlerPath) {
  return {
    name,
    description,
    inputSchema,
    handler: handlerPath,
  };
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
    console.error(`Error starting safe-inputs server: ${error instanceof Error ? error.message : String(error)}`);
    process.exit(1);
  }
}

module.exports = {
  loadConfig,
  startSafeInputsServer,
  createJsToolConfig,
  createShellToolConfig,
};
