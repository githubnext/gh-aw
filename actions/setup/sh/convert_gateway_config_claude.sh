#!/usr/bin/env bash
# Convert MCP Gateway Configuration to Claude Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the JSON format expected by Claude (without Copilot-specific fields)

set -e

# Required environment variable:
# - MCP_GATEWAY_OUTPUT: Path to gateway output configuration file
# - MCP_GATEWAY_API_KEY: API key for gateway authentication

# Debug logging function - logs to stderr when DEBUG is set
debug_log() {
  if [ -n "$DEBUG" ] && { [ "$DEBUG" = "1" ] || [ "$DEBUG" = "*" ] || [[ "$DEBUG" == *"convert_gateway_config"* ]]; }; then
    echo "[DEBUG convert_gateway_config_claude] $*" >&2
  fi
}

debug_log "=== Starting Claude configuration conversion ==="
debug_log "Environment check:"
debug_log "  MCP_GATEWAY_OUTPUT=${MCP_GATEWAY_OUTPUT:-<not set>}"
debug_log "  MCP_GATEWAY_API_KEY=${MCP_GATEWAY_API_KEY:+<set>}${MCP_GATEWAY_API_KEY:-<not set>}"
debug_log "  DEBUG=${DEBUG:-<not set>}"

if [ -z "$MCP_GATEWAY_OUTPUT" ]; then
  echo "ERROR: MCP_GATEWAY_OUTPUT environment variable is required"
  debug_log "FAILED: MCP_GATEWAY_OUTPUT is required but not set"
  exit 1
fi
debug_log "✓ MCP_GATEWAY_OUTPUT is set"

if [ ! -f "$MCP_GATEWAY_OUTPUT" ]; then
  echo "ERROR: Gateway output file not found: $MCP_GATEWAY_OUTPUT"
  debug_log "FAILED: Gateway output file does not exist at: $MCP_GATEWAY_OUTPUT"
  exit 1
fi
debug_log "✓ Gateway output file exists: $MCP_GATEWAY_OUTPUT"
debug_log "  File size: $(wc -c < "$MCP_GATEWAY_OUTPUT") bytes"

if [ -z "$MCP_GATEWAY_API_KEY" ]; then
  echo "ERROR: MCP_GATEWAY_API_KEY environment variable is required"
  debug_log "FAILED: MCP_GATEWAY_API_KEY is required but not set"
  exit 1
fi
debug_log "✓ MCP_GATEWAY_API_KEY is set (length: ${#MCP_GATEWAY_API_KEY} characters)"

echo "Converting gateway configuration to Claude format..."
echo "Input: $MCP_GATEWAY_OUTPUT"

debug_log "Reading input configuration..."
if [ -n "$DEBUG" ] && { [ "$DEBUG" = "1" ] || [ "$DEBUG" = "*" ] || [[ "$DEBUG" == *"convert_gateway_config"* ]]; }; then
  debug_log "Input file contents:"
  cat "$MCP_GATEWAY_OUTPUT" >&2
fi

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
# Claude format (JSON without Copilot-specific fields):
# {
#   "mcpServers": {
#     "server-name": {
#       "url": "http://domain:port/mcp/server-name",
#       "headers": {
#         "Authorization": "apiKey"
#       }
#     }
#   }
# }
#
# Claude doesn't use "type" or "tools" fields like Copilot does.
# The format is cleaner JSON with just url and headers.

debug_log "Counting servers in input configuration..."
SERVER_COUNT=$(jq -r '.mcpServers | length' "$MCP_GATEWAY_OUTPUT")
debug_log "Found $SERVER_COUNT server(s) to convert"

if [ -n "$DEBUG" ] && { [ "$DEBUG" = "1" ] || [ "$DEBUG" = "*" ] || [[ "$DEBUG" == *"convert_gateway_config"* ]]; }; then
  debug_log "Listing server names:"
  jq -r '.mcpServers | keys[]' "$MCP_GATEWAY_OUTPUT" | while read -r server; do
    debug_log "  - $server"
  done
fi

debug_log "Starting jq transformation..."
debug_log "Transformation steps:"
debug_log "  1. Remove 'type' field (Claude doesn't use it)"
debug_log "  2. Remove 'tools' field (Claude doesn't use it)"
debug_log "  3. Replace Authorization header placeholder with actual API key"

jq --arg apiKey "$MCP_GATEWAY_API_KEY" '
  .mcpServers |= with_entries(
    .value |= (
      # Remove type field if present (Claude doesn'\''t use it)
      del(.type) |
      # Remove tools field if present (Claude doesn'\''t use it)
      del(.tools) |
      # Ensure headers Authorization uses actual API key
      if .headers and .headers.Authorization then
        .headers.Authorization = $apiKey
      else
        .
      end
    )
  )
' "$MCP_GATEWAY_OUTPUT" > /tmp/gh-aw/mcp-config/mcp-servers.json

debug_log "✓ jq transformation completed successfully"
debug_log "Output written to: /tmp/gh-aw/mcp-config/mcp-servers.json"
debug_log "Output file size: $(wc -c < /tmp/gh-aw/mcp-config/mcp-servers.json) bytes"

if [ -n "$DEBUG" ] && { [ "$DEBUG" = "1" ] || [ "$DEBUG" = "*" ] || [[ "$DEBUG" == *"convert_gateway_config"* ]]; }; then
  debug_log "Verifying each server was converted correctly..."
  jq -r '.mcpServers | keys[]' /tmp/gh-aw/mcp-config/mcp-servers.json | while read -r server; do
    debug_log "  Server: $server"
    HAS_TYPE=$(jq -r ".mcpServers[\"$server\"] | has(\"type\")" /tmp/gh-aw/mcp-config/mcp-servers.json)
    HAS_TOOLS=$(jq -r ".mcpServers[\"$server\"] | has(\"tools\")" /tmp/gh-aw/mcp-config/mcp-servers.json)
    HAS_URL=$(jq -r ".mcpServers[\"$server\"] | has(\"url\")" /tmp/gh-aw/mcp-config/mcp-servers.json)
    HAS_HEADERS=$(jq -r ".mcpServers[\"$server\"] | has(\"headers\")" /tmp/gh-aw/mcp-config/mcp-servers.json)
    debug_log "    - has 'type' field: $HAS_TYPE (should be false)"
    debug_log "    - has 'tools' field: $HAS_TOOLS (should be false)"
    debug_log "    - has 'url' field: $HAS_URL (should be true)"
    debug_log "    - has 'headers' field: $HAS_HEADERS (should be true)"
  done
fi

echo "Claude configuration written to /tmp/gh-aw/mcp-config/mcp-servers.json"
debug_log "=== Claude configuration conversion completed successfully ==="
echo ""
echo "Converted configuration:"
cat /tmp/gh-aw/mcp-config/mcp-servers.json
