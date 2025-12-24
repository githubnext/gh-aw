// @ts-check
// Auto-generated safe-outputs MCP server entry point
// This script uses individual module files (not bundled)

const { startSafeOutputsServer } = require("./safe_outputs_mcp_server.cjs");

// Start the server
// The server reads configuration from /tmp/gh-aw/safeoutputs/config.json
// Log directory is configured via GH_AW_MCP_LOG_DIR environment variable
if (require.main === module) {
  try {
    startSafeOutputsServer();
  } catch (error) {
    console.error(`Error starting safe-outputs server: ${error instanceof Error ? error.message : String(error)}`);
    process.exit(1);
  }
}

module.exports = { startSafeOutputsServer };
