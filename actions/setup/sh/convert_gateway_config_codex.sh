#!/usr/bin/env bash
# Convert MCP Gateway Configuration to Codex Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the TOML format expected by Codex

set -e

# This script reads gateway configuration from stdin (in-memory)
# No files are created for the gateway output - security requirement

echo "Converting gateway configuration to Codex TOML format..."
echo "Reading configuration from stdin..."

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

# Read from stdin and append to TOML
jq -r '
  .mcpServers | to_entries[] | 
  "[mcp_servers.\(.key)]\n" +
  "url = \"\(.value.url)\"\n" +
  "\n" +
  "[mcp_servers.\(.key).headers]\n" +
  "Authorization = \"\(.value.headers.Authorization)\"\n"
' >> /tmp/gh-aw/mcp-config/config.toml

echo "Codex configuration written to /tmp/gh-aw/mcp-config/config.toml"
echo ""
echo "Converted configuration:"
cat /tmp/gh-aw/mcp-config/config.toml
