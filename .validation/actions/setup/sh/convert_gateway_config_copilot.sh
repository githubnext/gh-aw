#!/usr/bin/env bash
set +o histexpand

# Convert MCP Gateway Configuration to Copilot Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the format expected by GitHub Copilot CLI

set -e

# Restrict default file creation mode to owner-only (rw-------) for all new files.
# This prevents the race window between file creation via output redirection and
# a subsequent chmod, which would leave credential-bearing files world-readable
# (mode 0644) with a typical umask of 022.
umask 077

# Required environment variables:
# - MCP_GATEWAY_OUTPUT: Path to gateway output configuration file
# - MCP_GATEWAY_DOMAIN: Domain to use for MCP server URLs (e.g., host.docker.internal)
# - MCP_GATEWAY_PORT: Port for MCP gateway (e.g., 80)

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

echo "Converting gateway configuration to Copilot format..."
echo "Input: $MCP_GATEWAY_OUTPUT"
echo "Target domain: $MCP_GATEWAY_DOMAIN:$MCP_GATEWAY_PORT"

# Convert gateway output to Copilot format
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
# Copilot format:
# {
#   "mcpServers": {
#     "server-name": {
#       "type": "http",
#       "url": "http://domain:port/mcp/server-name",
#       "headers": {
#         "Authorization": "apiKey"
#       },
#       "tools": ["*"]
#     }
#   }
# }
#
# The main differences:
# 1. Copilot requires the "tools" field
# 2. URLs must use the correct domain (host.docker.internal) for container access
#    The gateway may output 0.0.0.0 or localhost which won't work from within containers

# Build the correct URL prefix using the configured domain and port
URL_PREFIX="http://${MCP_GATEWAY_DOMAIN}:${MCP_GATEWAY_PORT}"

jq --arg urlPrefix "$URL_PREFIX" '
  .mcpServers |= with_entries(
    .value |= (
      # Add tools field if not present
      (if .tools then . else . + {"tools": ["*"]} end) |
      # Fix the URL to use the correct domain
      # Replace http://anything:port/mcp/ with http://domain:port/mcp/
      .url |= (. | sub("^http://[^/]+/mcp/"; $urlPrefix + "/mcp/"))
    )
  )
' "$MCP_GATEWAY_OUTPUT" > /home/runner/.copilot/mcp-config.json

# Restrict permissions so only the runner process owner can read this file.
# mcp-config.json contains the bearer token for the MCP gateway; an attacker
# who reads it could bypass the --allowed-tools constraint by issuing raw
# JSON-RPC calls directly to the gateway.
chmod 600 /home/runner/.copilot/mcp-config.json

echo "Copilot configuration written to /home/runner/.copilot/mcp-config.json"
echo ""
echo "Converted configuration:"
cat /home/runner/.copilot/mcp-config.json
