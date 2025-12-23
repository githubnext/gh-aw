#!/bin/bash
# Safe Outputs Copy Action
# Copies safe-outputs MCP server files to the agent environment

set -e

# Get destination from input or use default
DESTINATION="${INPUT_DESTINATION:-/tmp/gh-aw/safeoutputs}"

echo "Copying safe-outputs files to ${DESTINATION}"

# Create destination directory if it doesn't exist
mkdir -p "${DESTINATION}"
echo "Created directory: ${DESTINATION}"

# Source directory for the JavaScript files
# When running in GitHub Actions, these files are in the action's js directory
SOURCE_DIR="${GITHUB_ACTION_PATH}/js"

FILE_COUNT=0

# Copy all .cjs files from js/ to destination
for file in "${SOURCE_DIR}"/*.cjs; do
  if [ -f "$file" ]; then
    filename=$(basename "$file")
    cp "$file" "${DESTINATION}/${filename}"
    echo "Copied: ${filename}"
    FILE_COUNT=$((FILE_COUNT + 1))
  fi
done

# Copy any .json files as well
for file in "${SOURCE_DIR}"/*.json; do
  if [ -f "$file" ]; then
    filename=$(basename "$file")
    cp "$file" "${DESTINATION}/${filename}"
    echo "Copied: ${filename}"
    FILE_COUNT=$((FILE_COUNT + 1))
  fi
done

# Generate and copy the main MCP server entry point
cat > "${DESTINATION}/mcp-server.cjs" << 'EOF_ENTRYPOINT'
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
EOF_ENTRYPOINT
chmod +x "${DESTINATION}/mcp-server.cjs"
echo "Copied: mcp-server.cjs (generated)"
FILE_COUNT=$((FILE_COUNT + 1))

# Set output
echo "files-copied=${FILE_COUNT}" >> "${GITHUB_OUTPUT}"
echo "✓ Successfully copied ${FILE_COUNT} files"
