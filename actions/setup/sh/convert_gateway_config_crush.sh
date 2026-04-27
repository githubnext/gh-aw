#!/usr/bin/env bash
set +o histexpand

# Convert MCP Gateway Configuration to Crush Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the JSON format expected by Crush (.crush.json)
#
# Crush reads MCP server configuration from .crush.json:
# - Project: ./.crush.json (used here)
# - Global: ~/.config/crush/crush.json
#
# See: https://github.com/charmbracelet/crush#mcps

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

echo "Converting gateway configuration to Crush format..."
echo "Input: $MCP_GATEWAY_OUTPUT"
echo "Target domain: $MCP_GATEWAY_DOMAIN:$MCP_GATEWAY_PORT"

# Convert gateway output to Crush .crush.json format
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
# Crush format:
# {
#   "mcp": {
#     "server-name": {
#       "type": "http",
#       "disabled": false,
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
# 2. Server type remains "http"
# 3. Uses "disabled": false
# 4. Remove "tools" field (Copilot-specific)
# 5. URLs must use the correct domain (host.docker.internal) for container access

# Build the correct URL prefix using the configured domain and port
URL_PREFIX="http://${MCP_GATEWAY_DOMAIN}:${MCP_GATEWAY_PORT}"

CRUSH_CONFIG_FILE="${GITHUB_WORKSPACE}/.crush.json"

# Build the MCP section from gateway output
MCP_SECTION=$(jq --arg urlPrefix "$URL_PREFIX" '
  .mcpServers | with_entries(
    .value |= {
      "type": "http",
      "disabled": false,
      "url": (.url | sub("^http://[^/]+/mcp/"; $urlPrefix + "/mcp/")),
      "headers": .headers
    }
  )
' "$MCP_GATEWAY_OUTPUT")

# Merge into existing .crush.json or create new one
if [ -f "$CRUSH_CONFIG_FILE" ]; then
  echo "Merging MCP config into existing .crush.json..."
  jq --argjson mcpSection "$MCP_SECTION" '.mcp = (.mcp // {}) * $mcpSection' "$CRUSH_CONFIG_FILE" > "${CRUSH_CONFIG_FILE}.tmp"
  mv "${CRUSH_CONFIG_FILE}.tmp" "$CRUSH_CONFIG_FILE"
else
  echo "Creating new .crush.json..."
  jq -n --argjson mcpSection "$MCP_SECTION" '{"mcp": $mcpSection}' > "$CRUSH_CONFIG_FILE"
fi

echo "Crush configuration written to $CRUSH_CONFIG_FILE"
chmod 600 "$CRUSH_CONFIG_FILE"
echo ""
echo "Converted configuration:"
cat "$CRUSH_CONFIG_FILE"
