#!/usr/bin/env node
// @ts-check

// Safe-outputs MCP Server Entry Point
// This is the main entry point script for the safe-outputs MCP server
// It starts the HTTP server on the configured port

// Load core shim before any other modules so that global.core is available
// for modules that rely on it (e.g. generate_git_patch.cjs).
require("./shim.cjs");

const { createLogger } = require("./mcp_logger.cjs");
const logger = createLogger("safe-outputs-entry");

// Log immediately to verify Node.js execution starts
logger.debug("Entry point script executing");

const { startHttpServer } = require("./safe_outputs_mcp_server_http.cjs");

logger.debug("Successfully required safe_outputs_mcp_server_http.cjs");

// Start the HTTP server
// The server reads configuration from ${RUNNER_TEMP}/gh-aw/safeoutputs/config.json
// Port and API key are configured via environment variables:
// - GH_AW_SAFE_OUTPUTS_PORT
// - GH_AW_SAFE_OUTPUTS_API_KEY
// Log directory is configured via GH_AW_MCP_LOG_DIR environment variable
//
// NOTE: The server runs in stateless mode (no session management).
// The MCP gateway manages sessions on behalf of clients (e.g. the Copilot CLI).
// The safe-outputs HTTP server itself does not track sessions — each POST request
// is handled independently. Session lifecycle (including SSE keepalive pings to
// prevent the 5-minute idle timeout) is handled by the MCP gateway configuration
// via the sseKeepAliveInterval gateway option.
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
