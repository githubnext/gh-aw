#!/usr/bin/env bash
# Convert MCP Gateway Configuration to Codex Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the TOML format expected by Codex

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

echo "Converting gateway configuration to Codex TOML format..."
echo "Input: $MCP_GATEWAY_OUTPUT"

# Convert gateway JSON output to Codex TOML format
# Gateway format (JSON):
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
# Codex format (TOML):
# [history]
# persistence = "none"
#
# [mcp_servers.server-name]
# url = "http://domain:port/mcp/server-name"
#
# [mcp_servers.server-name.headers]
# Authorization = "apiKey"
#
# Note: Codex doesn't use "type" or "tools" fields

# Create the TOML configuration
cat > /tmp/gh-aw/mcp-config/config.toml << 'TOML_EOF'
[history]
persistence = "none"

TOML_EOF

jq -r --arg apiKey "$MCP_GATEWAY_API_KEY" '
  .mcpServers | to_entries[] | 
  "[mcp_servers.\(.key)]\n" +
  "url = \"\(.value.url)\"\n" +
  "\n" +
  "[mcp_servers.\(.key).headers]\n" +
  if .value.headers.Authorization then
    "Authorization = \"\(.value.headers.Authorization)\"\n"
  else
    "Authorization = \"\($apiKey)\"\n"
  end
' "$MCP_GATEWAY_OUTPUT" >> /tmp/gh-aw/mcp-config/config.toml

echo "Codex configuration written to /tmp/gh-aw/mcp-config/config.toml"
echo ""
echo "Converted configuration:"
cat /tmp/gh-aw/mcp-config/config.toml
