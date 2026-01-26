#!/usr/bin/env node
// @ts-check

// Safe-outputs MCP Server Entry Point
// This is the main entry point script for the safe-outputs MCP server
// It starts the HTTP server on the configured port

// Log immediately to verify Node.js execution starts
process.stderr.write("[safe-outputs-entry] Entry point script executing\n");
console.error("[safe-outputs-entry] Console.error test - Node.js is running");

const { startHttpServer } = require("./safe_outputs_mcp_server_http.cjs");

process.stderr.write("[safe-outputs-entry] Successfully required safe_outputs_mcp_server_http.cjs\n");

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
  process.stderr.write("[safe-outputs-entry] In require.main === module block\n");
  const port = parseInt(process.env.GH_AW_SAFE_OUTPUTS_PORT || "3001", 10);
  const logDir = process.env.GH_AW_MCP_LOG_DIR;
  process.stderr.write(`[safe-outputs-entry] Port: ${port}, LogDir: ${logDir}\n`);
  process.stderr.write("[safe-outputs-entry] Calling startHttpServer...\n");

  startHttpServer({ port, logDir, stateless: true }).catch(error => {
    console.error(`[safe-outputs-entry] Failed to start safe-outputs HTTP server: ${error.message}`);
    console.error(`[safe-outputs-entry] Error stack: ${error.stack}`);
    process.exit(1);
  });

  process.stderr.write("[safe-outputs-entry] startHttpServer call initiated (async)\n");
}

module.exports = { startHttpServer };
