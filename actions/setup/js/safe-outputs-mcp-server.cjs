#!/usr/bin/env node
// @ts-check

// Safe-outputs MCP Server Entry Point
// This is the main entry point script for the safe-outputs MCP server
// It starts the HTTP server on the configured port

const { startHttpServer } = require("./safe_outputs_mcp_server_http.cjs");

// Start the HTTP server
// The server reads configuration from /opt/gh-aw/safeoutputs/config.json
// Port and API key are configured via environment variables:
// - GH_AW_SAFE_OUTPUTS_PORT
// - GH_AW_SAFE_OUTPUTS_API_KEY
// Log directory is configured via GH_AW_MCP_LOG_DIR environment variable
if (require.main === module) {
  const port = parseInt(process.env.GH_AW_SAFE_OUTPUTS_PORT || "3001", 10);
  const logDir = process.env.GH_AW_MCP_LOG_DIR;

  startHttpServer({ port, logDir }).catch(error => {
    console.error(`Failed to start safe-outputs HTTP server: ${error.message}`);
    process.exit(1);
  });
}

module.exports = { startHttpServer };
