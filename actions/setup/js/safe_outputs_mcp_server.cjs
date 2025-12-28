// @ts-check

// Safe Outputs MCP Server Module
//
// This module provides a reusable MCP server for safe-outputs configuration.
// It uses the mcp_server_core module for JSON-RPC handling and tool registration.
//
// Usage:
//   node safe_outputs_mcp_server.cjs
//
// Or as a module:
//   const server = require("./safe_outputs_mcp_server.cjs");
//   server.startSafeOutputsServer();

const { createServer, registerTool, normalizeTool, start } = require("./mcp_server_core.cjs");
const { createAppendFunction } = require("./safe_outputs_append.cjs");
const { createHandlers } = require("./safe_outputs_handlers.cjs");
const { attachHandlers, registerPredefinedTools, registerDynamicTools } = require("./safe_outputs_tools_loader.cjs");
const { bootstrapSafeOutputsServer, cleanupConfigFile } = require("./safe_outputs_bootstrap.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Start the safe-outputs MCP server
 * @param {Object} [options] - Additional options
 * @param {string} [options.logDir] - Override log directory
 * @param {boolean} [options.skipCleanup] - Skip deletion of config file (useful for testing)
 */
function startSafeOutputsServer(options = {}) {
  // Server info for safe outputs MCP server
  const SERVER_INFO = { name: "safeoutputs", version: "1.0.0" };

  // Create the server instance with optional log directory
  const MCP_LOG_DIR = options.logDir || process.env.GH_AW_MCP_LOG_DIR;
  const server = createServer(SERVER_INFO, { logDir: MCP_LOG_DIR });

  // Bootstrap: load configuration and tools using shared logic
  const { config: safeOutputsConfig, outputFile, tools: ALL_TOOLS } = bootstrapSafeOutputsServer(server);

  // Create append function
  const appendSafeOutput = createAppendFunction(outputFile);

  // Create handlers with configuration
  const handlers = createHandlers(server, appendSafeOutput, safeOutputsConfig);
  const { defaultHandler } = handlers;

  // Attach handlers to tools
  const toolsWithHandlers = attachHandlers(ALL_TOOLS, handlers);

  server.debug(`  output file: ${outputFile}`);
  server.debug(`  config: ${JSON.stringify(safeOutputsConfig)}`);

  // Register predefined tools that are enabled in configuration
  registerPredefinedTools(server, toolsWithHandlers, safeOutputsConfig, registerTool, normalizeTool);

  // Add safe-jobs as dynamic tools
  registerDynamicTools(server, toolsWithHandlers, safeOutputsConfig, outputFile, registerTool, normalizeTool);

  server.debug(`  tools: ${Object.keys(server.tools).join(", ")}`);
  if (!Object.keys(server.tools).length) throw new Error("No tools enabled in configuration");

  // Note: We do NOT cleanup the config file here because it's needed by the ingestion
  // phase (collect_ndjson_output.cjs) that runs after the MCP server completes.
  // The config file only contains schema information (no secrets), so it's safe to leave.

  // Start the server with the default handler
  start(server, { defaultHandler });
}

// If run directly, start the server
if (require.main === module) {
  try {
    startSafeOutputsServer();
  } catch (error) {
    console.error(`Error starting safe-outputs server: ${getErrorMessage(error)}`);
    process.exit(1);
  }
}

module.exports = {
  startSafeOutputsServer,
};
