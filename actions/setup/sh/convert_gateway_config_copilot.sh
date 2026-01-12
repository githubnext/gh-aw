#!/usr/bin/env bash
# Convert MCP Gateway Configuration to Copilot Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the format expected by GitHub Copilot CLI

set -e

# This script reads gateway configuration from stdin (in-memory)
# No files are created for the gateway output - security requirement

echo "Converting gateway configuration to Copilot format..."
echo "Reading configuration from stdin..."

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
# We also need to ensure headers use the actual API key value, not a placeholder.

# Read from stdin and convert
jq '
  .mcpServers |= with_entries(
    .value |= (
      if .tools then . else . + {"tools": ["*"]} end
    )
  )
' > /home/runner/.copilot/mcp-config.json

echo "Copilot configuration written to /home/runner/.copilot/mcp-config.json"
echo ""
echo "Converted configuration:"
cat /home/runner/.copilot/mcp-config.json
