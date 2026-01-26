#!/usr/bin/env node
// @ts-check

// Safe-outputs MCP Server Entry Point
// This is the main entry point script for the safe-outputs MCP server
// It starts the HTTP server on the configured port

const { createLogger } = require("./mcp_logger.cjs");
const logger = createLogger("safe-outputs-entry");

// Log immediately to verify Node.js execution starts
logger.debug("Entry point script executing");

const { startHttpServer } = require("./safe_outputs_mcp_server_http.cjs");

logger.debug("Successfully required safe_outputs_mcp_server_http.cjs");

// Start the HTTP server
// The server reads configuration from /opt/gh-aw/safeoutputs/config.json
// Port and API key are configured via environment variables:
// - GH_AW_SAFE_OUTPUTS_PORT
// - GH_AW_SAFE_OUTPUTS_API_KEY
// Log directory is configured via GH_AW_MCP_LOG_DIR environment variable
//
// NOTE: The server runs in stateless mode (no session management) because
// the MCP gateway doesn't perform the MCP protocol initialization handshake.
// It directly calls methods like tools/list without the Mcp-Session-Id header.
if (require.main === module) {
  logger.debug("In require.main === module block");
  const port = parseInt(process.env.GH_AW_SAFE_OUTPUTS_PORT || "3001", 10);
  const logDir = process.env.GH_AW_MCP_LOG_DIR;
  logger.debug(`Port: ${port}, LogDir: ${logDir}`);
  logger.debug("Calling startHttpServer...");

  startHttpServer({ port, logDir, stateless: true }).catch(error => {
    logger.debugError("Failed to start safe-outputs HTTP server: ", error);
    process.exit(1);
  });

  logger.debug("startHttpServer call initiated (async)");
}

module.exports = { startHttpServer };
