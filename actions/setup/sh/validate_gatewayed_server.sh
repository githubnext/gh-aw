#!/bin/bash
set -e

# validate_gatewayed_server.sh - Validate that an MCP server is correctly gatewayed
#
# Usage: validate_gatewayed_server.sh SERVER_NAME MCP_CONFIG_PATH GATEWAY_URL
#
# Arguments:
#   SERVER_NAME      : Name of the MCP server to validate (e.g., "github", "playwright")
#   MCP_CONFIG_PATH  : Path to the MCP configuration file
#   GATEWAY_URL      : Expected gateway base URL (e.g., "http://localhost:8080")
#
# Validation checks:
#   1. Server exists in MCP config
#   2. Server has HTTP URL
#   3. Server type is "http"
#   4. URL points to gateway
#
# Exit codes:
#   0 - Server is correctly gatewayed
#   1 - Validation failed

# Parse arguments
if [ "$#" -ne 3 ]; then
  echo "Usage: $0 SERVER_NAME MCP_CONFIG_PATH GATEWAY_URL" >&2
  exit 1
fi

SERVER_NAME="$1"
MCP_CONFIG_PATH="$2"
GATEWAY_URL="$3"

# Check if server exists in config
if ! grep -q "\"${SERVER_NAME}\"" "$MCP_CONFIG_PATH"; then
  echo "ERROR: ${SERVER_NAME} server not found in MCP configuration" >&2
  exit 1
fi

# Extract server configuration
server_config=$(jq -r ".mcpServers.\"${SERVER_NAME}\"" "$MCP_CONFIG_PATH")
if [ "$server_config" = "null" ]; then
  echo "ERROR: ${SERVER_NAME} server configuration is null" >&2
  exit 1
fi

# Extract URL and type
server_url=$(echo "$server_config" | jq -r '.url // empty')
server_type=$(echo "$server_config" | jq -r '.type // empty')

# Verify URL exists
if [ -z "$server_url" ] || [ "$server_url" = "null" ]; then
  echo "ERROR: ${SERVER_NAME} server does not have HTTP URL (not gatewayed correctly)" >&2
  echo "Config: $server_config" >&2
  exit 1
fi

# Verify type is "http"
if [ "$server_type" != "http" ]; then
  echo "ERROR: ${SERVER_NAME} server type is not \"http\" (expected for gatewayed servers)" >&2
  echo "Type: $server_type" >&2
  exit 1
fi

# Verify URL points to gateway
if ! echo "$server_url" | grep -q "$GATEWAY_URL"; then
  echo "ERROR: ${SERVER_NAME} server URL does not point to gateway" >&2
  echo "Expected gateway URL: $GATEWAY_URL" >&2
  echo "Actual URL: $server_url" >&2
  exit 1
fi

# Success
echo "✓ ${SERVER_NAME} server is correctly gatewayed"
