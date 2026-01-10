#!/usr/bin/env bash
# Convert MCP Gateway Configuration to Copilot Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the format expected by GitHub Copilot CLI

set -e

# Required environment variable:
# - MCP_GATEWAY_OUTPUT: Path to gateway output configuration file
# - MCP_GATEWAY_API_KEY: API key for gateway authentication

# Debug logging function - logs to stderr when DEBUG is set
debug_log() {
  if [ -n "$DEBUG" ] && { [ "$DEBUG" = "1" ] || [ "$DEBUG" = "*" ] || [[ "$DEBUG" == *"convert_gateway_config"* ]]; }; then
    echo "[DEBUG convert_gateway_config_copilot] $*" >&2
  fi
}

debug_log "=== Starting Copilot configuration conversion ==="
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

echo "Converting gateway configuration to Copilot format..."
echo "Input: $MCP_GATEWAY_OUTPUT"

debug_log "Reading input configuration..."
if [ -n "$DEBUG" ] && { [ "$DEBUG" = "1" ] || [ "$DEBUG" = "*" ] || [[ "$DEBUG" == *"convert_gateway_config"* ]]; }; then
  debug_log "Input file contents:"
  cat "$MCP_GATEWAY_OUTPUT" >&2
fi

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
debug_log "  1. Add 'tools': ['*'] field to each server (Copilot requirement)"
debug_log "  2. Replace Authorization header placeholder with actual API key"

jq --arg apiKey "$MCP_GATEWAY_API_KEY" '
  .mcpServers |= with_entries(
    .value |= (
      # Add tools field if not present
      if .tools then . else . + {"tools": ["*"]} end |
      # Ensure headers Authorization uses actual API key
      if .headers and .headers.Authorization then
        .headers.Authorization = $apiKey
      else
        .
      end
    )
  )
' "$MCP_GATEWAY_OUTPUT" > /home/runner/.copilot/mcp-config.json

debug_log "✓ jq transformation completed successfully"
debug_log "Output written to: /home/runner/.copilot/mcp-config.json"
debug_log "Output file size: $(wc -c < /home/runner/.copilot/mcp-config.json) bytes"

if [ -n "$DEBUG" ] && { [ "$DEBUG" = "1" ] || [ "$DEBUG" = "*" ] || [[ "$DEBUG" == *"convert_gateway_config"* ]]; }; then
  debug_log "Verifying each server was converted correctly..."
  jq -r '.mcpServers | keys[]' /home/runner/.copilot/mcp-config.json | while read -r server; do
    debug_log "  Server: $server"
    HAS_TOOLS=$(jq -r ".mcpServers[\"$server\"] | has(\"tools\")" /home/runner/.copilot/mcp-config.json)
    HAS_TYPE=$(jq -r ".mcpServers[\"$server\"] | has(\"type\")" /home/runner/.copilot/mcp-config.json)
    HAS_URL=$(jq -r ".mcpServers[\"$server\"] | has(\"url\")" /home/runner/.copilot/mcp-config.json)
    HAS_HEADERS=$(jq -r ".mcpServers[\"$server\"] | has(\"headers\")" /home/runner/.copilot/mcp-config.json)
    debug_log "    - has 'tools' field: $HAS_TOOLS"
    debug_log "    - has 'type' field: $HAS_TYPE"
    debug_log "    - has 'url' field: $HAS_URL"
    debug_log "    - has 'headers' field: $HAS_HEADERS"
  done
fi

echo "Copilot configuration written to /home/runner/.copilot/mcp-config.json"
debug_log "=== Copilot configuration conversion completed successfully ==="
echo ""
echo "Converted configuration:"
cat /home/runner/.copilot/mcp-config.json
