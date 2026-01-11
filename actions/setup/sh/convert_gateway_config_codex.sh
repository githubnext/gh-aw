#!/usr/bin/env bash
# Convert MCP Gateway Configuration to Codex Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the TOML format expected by Codex

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

echo "Converting gateway configuration to Codex TOML format..."
echo "Input: $MCP_GATEWAY_OUTPUT"
echo ""

# Validate gateway output structure
echo "Validating gateway output structure..."
if ! jq -e '.mcpServers' "$MCP_GATEWAY_OUTPUT" >/dev/null 2>&1; then
  echo "ERROR: Gateway output missing 'mcpServers' field"
  echo "Gateway output content:"
  cat "$MCP_GATEWAY_OUTPUT"
  exit 1
fi

# Check if any servers are configured
SERVER_COUNT=$(jq '.mcpServers | length' "$MCP_GATEWAY_OUTPUT")
echo "Found $SERVER_COUNT MCP server(s) in gateway output"

if [ "$SERVER_COUNT" -eq 0 ]; then
  echo "WARNING: No MCP servers found in gateway output"
fi

# List servers being converted
echo "Servers to convert:"
jq -r '.mcpServers | keys[]' "$MCP_GATEWAY_OUTPUT" | while read -r server; do
  echo "  - $server"
done
echo ""

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

# Convert each server with validation
jq -r '.mcpServers | to_entries[] | .key' "$MCP_GATEWAY_OUTPUT" | while read -r SERVER_NAME; do
  echo "Converting server: $SERVER_NAME"
  
  # Extract server configuration
  SERVER_URL=$(jq -r ".mcpServers[\"$SERVER_NAME\"].url // empty" "$MCP_GATEWAY_OUTPUT")
  SERVER_AUTH=$(jq -r ".mcpServers[\"$SERVER_NAME\"].headers.Authorization // empty" "$MCP_GATEWAY_OUTPUT")
  
  # Validate required fields
  if [ -z "$SERVER_URL" ]; then
    echo "  ERROR: Missing 'url' field for server '$SERVER_NAME'"
    exit 1
  fi
  
  if [ -z "$SERVER_AUTH" ]; then
    echo "  WARNING: Missing 'headers.Authorization' field for server '$SERVER_NAME'"
    # Don't fail - some servers may not require authentication
  fi
  
  echo "  URL: $SERVER_URL"
  if [ -n "$SERVER_AUTH" ]; then
    echo "  Auth: <present>"
  fi
  
  # Write server configuration to TOML
  {
    echo ""
    echo "[mcp_servers.$SERVER_NAME]"
    echo "url = \"$SERVER_URL\""
    
    # Only add headers section if Authorization is present
    if [ -n "$SERVER_AUTH" ]; then
      echo ""
      echo "[mcp_servers.$SERVER_NAME.headers]"
      echo "Authorization = \"$SERVER_AUTH\""
    fi
  } >> /tmp/gh-aw/mcp-config/config.toml
  
  echo "  ✓ Converted"
done

echo ""
echo "Codex configuration written to /tmp/gh-aw/mcp-config/config.toml"
echo ""
echo "Converted configuration:"
cat /tmp/gh-aw/mcp-config/config.toml
