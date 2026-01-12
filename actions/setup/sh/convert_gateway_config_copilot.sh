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
# Gateway format (per MCP Gateway Specification v1.3.0):
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
# Copilot format (required by GitHub Copilot CLI):
# {
#   "mcpServers": {
#     "server-name": {
#       "type": "http",
#       "url": "http://domain:port/mcp/server-name",
#       "headers": {
#         "Authorization": "Bearer apiKey"
#       },
#       "tools": ["*"]
#     }
#   }
# }
#
# Key differences:
# 1. Copilot requires the "tools" field
# 2. Copilot expects "Bearer " prefix in Authorization header (standard HTTP authentication scheme)
#
# Note: The MCP Gateway Specification v1.3.0 Section 7.1 states that the gateway outputs
# Authorization headers without the "Bearer" prefix. However, GitHub Copilot CLI follows
# standard HTTP authentication schemes and requires the "Bearer" prefix for token-based
# authentication. This converter adds the prefix to ensure compatibility with Copilot CLI.

jq '
  .mcpServers |= with_entries(
    .value |= (
      # Add tools field if not present
      (if .tools then . else . + {"tools": ["*"]} end) |
      # Add "Bearer " prefix to Authorization header if headers exist and not already prefixed
      (if .headers and .headers.Authorization then
        if (.headers.Authorization | startswith("Bearer ")) then
          .
        else
          .headers.Authorization = "Bearer " + .headers.Authorization
        end
      else . end)
    )
  )
' "$MCP_GATEWAY_OUTPUT" > /home/runner/.copilot/mcp-config.json

echo "Copilot configuration written to /home/runner/.copilot/mcp-config.json"
echo ""
echo "Converted configuration:"
cat /home/runner/.copilot/mcp-config.json
