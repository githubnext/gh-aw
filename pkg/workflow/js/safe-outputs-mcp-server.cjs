#!/usr/bin/env node
// @ts-check

// Safe-outputs MCP Server Entry Point
// This is the main entry point script for the safe-outputs MCP server
// It requires the bootstrap module and starts the server

const { startSafeOutputsServer } = require("./safe_outputs_mcp_server.cjs");

// Start the server
// The server reads configuration from /tmp/gh-aw/safeoutputs/config.json
// Log directory is configured via GH_AW_MCP_LOG_DIR environment variable
if (require.main === module) {
  startSafeOutputsServer();
}

module.exports = { startSafeOutputsServer };
