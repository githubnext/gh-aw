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

# Define the list of files to copy
FILES=(
  "safe_outputs_mcp_server.cjs"
  "safe_outputs_bootstrap.cjs"
  "safe_outputs_tools_loader.cjs"
  "safe_outputs_config.cjs"
  "safe_outputs_handlers.cjs"
  "safe_outputs_tools.json"
  "mcp_server_core.cjs"
  "mcp_logger.cjs"
  "messages.cjs"
)

# Source directory for the JavaScript files
# When running in GitHub Actions, these files are in the workflow/js directory
SOURCE_DIR="${GITHUB_ACTION_PATH}/js"

FILE_COUNT=0

# Copy each file
for file in "${FILES[@]}"; do
  if [ -f "${SOURCE_DIR}/${file}" ]; then
    cp "${SOURCE_DIR}/${file}" "${DESTINATION}/${file}"
    echo "Copied: ${file}"
    FILE_COUNT=$((FILE_COUNT + 1))
  else
    echo "Warning: File not found: ${SOURCE_DIR}/${file}"
  fi
done

# Set output
echo "files-copied=${FILE_COUNT}" >> "${GITHUB_OUTPUT}"
echo "✓ Successfully copied ${FILE_COUNT} files"
