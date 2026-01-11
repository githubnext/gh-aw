#!/usr/bin/env bash
# Convert MCP Gateway Configuration to Copilot Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the format expected by GitHub Copilot CLI

set -e

# Required environment variable:
# - MCP_GATEWAY_OUTPUT: Path to gateway output configuration file

if [ -z "$MCP_GATEWAY_OUTPUT" ]; then
  echo "ERROR: MCP_GATEWAY_OUTPUT environment variable is required"
  exit 1
fi

if [ ! -f "$MCP_GATEWAY_OUTPUT" ]; then
  echo "ERROR: Gateway output file not found: $MCP_GATEWAY_OUTPUT"
  exit 1
fi

echo "Converting gateway configuration to Copilot format..."
echo "Input: $MCP_GATEWAY_OUTPUT"

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
# The main difference is that Copilot requires the "tools" field.
# All other fields (including headers) are preserved exactly as provided by the gateway.

jq '
  .mcpServers |= with_entries(
    .value |= (
      if .tools then . else . + {"tools": ["*"]} end
    )
  )
' "$MCP_GATEWAY_OUTPUT" > /home/runner/.copilot/mcp-config.json

echo "Copilot configuration written to /home/runner/.copilot/mcp-config.json"
echo ""
echo "Converted configuration:"
cat /home/runner/.copilot/mcp-config.json
