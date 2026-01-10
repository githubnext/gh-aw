#!/usr/bin/env bash
# Convert MCP Gateway Configuration to Claude Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the JSON format expected by Claude (without Copilot-specific fields)

set -e

# Required environment variable:
# - MCP_GATEWAY_OUTPUT: Path to gateway output configuration file
# - MCP_GATEWAY_API_KEY: API key for gateway authentication

if [ -z "$MCP_GATEWAY_OUTPUT" ]; then
  echo "ERROR: MCP_GATEWAY_OUTPUT environment variable is required"
  exit 1
fi

if [ ! -f "$MCP_GATEWAY_OUTPUT" ]; then
  echo "ERROR: Gateway output file not found: $MCP_GATEWAY_OUTPUT"
  exit 1
fi

if [ -z "$MCP_GATEWAY_API_KEY" ]; then
  echo "ERROR: MCP_GATEWAY_API_KEY environment variable is required"
  exit 1
fi

echo "Converting gateway configuration to Claude format..."
echo "Input: $MCP_GATEWAY_OUTPUT"

# Convert gateway output to Claude format
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
# Claude format (JSON with HTTP type and headers):
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
# Claude uses "type": "http" for HTTP-based MCP servers.
# The "tools" field is removed as it's Copilot-specific.

jq --arg apiKey "$MCP_GATEWAY_API_KEY" '
  .mcpServers |= with_entries(
    .value |= (
      # Ensure type is set to http for HTTP-based servers
      .type = "http" |
      # Remove tools field if present (Claude doesn'\''t use it)
      del(.tools) |
      # Always ensure headers object exists with Authorization
      .headers = {
        "Authorization": $apiKey
      }
    )
  )
' "$MCP_GATEWAY_OUTPUT" > /tmp/gh-aw/mcp-config/mcp-servers.json

echo "Claude configuration written to /tmp/gh-aw/mcp-config/mcp-servers.json"
echo ""
echo "Converted configuration:"
cat /tmp/gh-aw/mcp-config/mcp-servers.json
