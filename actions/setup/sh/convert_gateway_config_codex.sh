#!/usr/bin/env bash
# Convert MCP Gateway Configuration to Codex Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the TOML format expected by Codex

set -e

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

echo "Converting gateway configuration to Codex TOML format..."
echo "Input: $MCP_GATEWAY_OUTPUT"
echo "Target domain: $MCP_GATEWAY_DOMAIN:$MCP_GATEWAY_PORT"

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
# http_headers = { Authorization = "apiKey" }
#
# Note: Codex doesn't use "type" or "tools" fields
# Note: Codex uses http_headers as an inline table, not a separate section
# Note: URLs must use the correct domain (host.docker.internal) for container access

# Build the correct URL prefix using the configured domain and port
URL_PREFIX="http://${MCP_GATEWAY_DOMAIN}:${MCP_GATEWAY_PORT}"

# Validate MCP_GATEWAY_API_KEY is set (required for authentication)
if [ -z "$MCP_GATEWAY_API_KEY" ]; then
  echo "ERROR: MCP_GATEWAY_API_KEY environment variable is required"
  exit 1
fi

# Create the TOML configuration
cat > /tmp/gh-aw/mcp-config/config.toml << 'TOML_EOF'
[history]
persistence = "none"

TOML_EOF

jq -r --arg urlPrefix "$URL_PREFIX" --arg apiKey "$MCP_GATEWAY_API_KEY" '
  .mcpServers | to_entries[] |
  "[mcp_servers.\(.key)]\n" +
  "url = \"" + ($urlPrefix + "/mcp/" + .key) + "\"\n" +
  "http_headers = { Authorization = \"" + $apiKey + "\" }\n"
' "$MCP_GATEWAY_OUTPUT" >> /tmp/gh-aw/mcp-config/config.toml

echo "Codex configuration written to /tmp/gh-aw/mcp-config/config.toml"
echo ""
echo "Converted configuration:"
cat /tmp/gh-aw/mcp-config/config.toml
