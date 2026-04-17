#!/usr/bin/env bash
# Convert MCP Gateway Configuration to OpenCode Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the JSON format expected by OpenCode (opencode.jsonc)
#
# OpenCode reads MCP server configuration from opencode.jsonc:
# - Project: ./opencode.jsonc (used here)
# - Global: ~/.config/opencode/opencode.json
#
# See: https://opencode.ai/docs/mcp-servers/

set -euo pipefail

# Restrict permissions so credential-bearing files are not world-readable.
# umask 077 ensures new files are created with mode 0600 (owner-only read/write)
# even before a subsequent chmod, which would leave credential-bearing files
# world-readable (mode 0644) with a typical umask of 022.
umask 077

# Required environment variables:
# - MCP_GATEWAY_OUTPUT: Path to gateway output configuration file
# - MCP_GATEWAY_DOMAIN: Domain to use for MCP server URLs (e.g., host.docker.internal)
# - MCP_GATEWAY_PORT: Port for MCP gateway (e.g., 80)
# - GITHUB_WORKSPACE: Workspace directory for project-level config

if [ -z "$MCP_GATEWAY_OUTPUT" ]; then
  echo "ERROR: MCP_GATEWAY_OUTPUT environment variable is required"
  exit 1
fi

if [ ! -f "$MCP_GATEWAY_OUTPUT" ]; then
  echo "ERROR: Gateway output file not found: $MCP_GATEWAY_OUTPUT"
  exit 1
fi

if [ -z "$MCP_GATEWAY_DOMAIN" ]; then
  echo "ERROR: MCP_GATEWAY_DOMAIN environment variable is required"
  exit 1
fi

if [ -z "$MCP_GATEWAY_PORT" ]; then
  echo "ERROR: MCP_GATEWAY_PORT environment variable is required"
  exit 1
fi

if [ -z "$GITHUB_WORKSPACE" ]; then
  echo "ERROR: GITHUB_WORKSPACE environment variable is required"
  exit 1
fi

echo "Converting gateway configuration to OpenCode format..."
echo "Input: $MCP_GATEWAY_OUTPUT"
echo "Target domain: $MCP_GATEWAY_DOMAIN:$MCP_GATEWAY_PORT"

# Convert gateway output to OpenCode opencode.jsonc format
# Gateway format:
# {
#   "mcpServers": {
#     "server-name": {
#       "type": "http",
#       "url": "http://domain:port/mcp/server-name",
#       "headers": {
#         "Authorization": "apiKey"
#       }
#     }
#   }
# }
#
# OpenCode format:
# {
#   "mcp": {
#     "server-name": {
#       "type": "remote",
#       "enabled": true,
#       "url": "http://domain:port/mcp/server-name",
#       "headers": {
#         "Authorization": "apiKey"
#       }
#     }
#   }
# }
#
# The main differences:
# 1. Top-level key is "mcp" not "mcpServers"
# 2. Server type is "remote" not "http"
# 3. Has "enabled": true field
# 4. Remove "tools" field (Copilot-specific)
# 5. URLs must use the correct domain (host.docker.internal) for container access

# Build the correct URL prefix using the configured domain and port
URL_PREFIX="http://${MCP_GATEWAY_DOMAIN}:${MCP_GATEWAY_PORT}"

OPENCODE_CONFIG_FILE="${GITHUB_WORKSPACE}/opencode.jsonc"

# Build the MCP section from gateway output
MCP_SECTION=$(jq --arg urlPrefix "$URL_PREFIX" '
  .mcpServers | with_entries(
    .value |= {
      "type": "remote",
      "enabled": true,
      "url": (.url | sub("^http://[^/]+/mcp/"; $urlPrefix + "/mcp/")),
      "headers": .headers
    }
  )
' "$MCP_GATEWAY_OUTPUT")

# Merge into existing opencode.jsonc or create new one
if [ -f "$OPENCODE_CONFIG_FILE" ]; then
  echo "Merging MCP config into existing opencode.jsonc..."
  jq --argjson mcpSection "$MCP_SECTION" '.mcp = (.mcp // {}) * $mcpSection' "$OPENCODE_CONFIG_FILE" > "${OPENCODE_CONFIG_FILE}.tmp"
  mv "${OPENCODE_CONFIG_FILE}.tmp" "$OPENCODE_CONFIG_FILE"
else
  echo "Creating new opencode.jsonc..."
  jq -n --argjson mcpSection "$MCP_SECTION" '{"mcp": $mcpSection}' > "$OPENCODE_CONFIG_FILE"
fi

echo "OpenCode configuration written to $OPENCODE_CONFIG_FILE"
chmod 600 "$OPENCODE_CONFIG_FILE"
echo ""
echo "Converted configuration:"
cat "$OPENCODE_CONFIG_FILE"
