#!/usr/bin/env bash
# Convert MCP Gateway Configuration to Codex Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the TOML format expected by Codex

set -e

# Required environment variable:
# - MCP_GATEWAY_OUTPUT: Path to gateway output configuration file
# - MCP_GATEWAY_API_KEY: API key for gateway authentication

# Debug logging function - logs to stderr when DEBUG is set
debug_log() {
  if [ -n "$DEBUG" ] && { [ "$DEBUG" = "1" ] || [ "$DEBUG" = "*" ] || [[ "$DEBUG" == *"convert_gateway_config"* ]]; }; then
    echo "[DEBUG convert_gateway_config_codex] $*" >&2
  fi
}

debug_log "=== Starting Codex configuration conversion ==="
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

echo "Converting gateway configuration to Codex TOML format..."
echo "Input: $MCP_GATEWAY_OUTPUT"

debug_log "Reading input configuration..."
if [ -n "$DEBUG" ] && { [ "$DEBUG" = "1" ] || [ "$DEBUG" = "*" ] || [[ "$DEBUG" == *"convert_gateway_config"* ]]; }; then
  debug_log "Input file contents:"
  cat "$MCP_GATEWAY_OUTPUT" >&2
fi

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

debug_log "Counting servers in input configuration..."
SERVER_COUNT=$(jq -r '.mcpServers | length' "$MCP_GATEWAY_OUTPUT")
debug_log "Found $SERVER_COUNT server(s) to convert to TOML"

if [ -n "$DEBUG" ] && { [ "$DEBUG" = "1" ] || [ "$DEBUG" = "*" ] || [[ "$DEBUG" == *"convert_gateway_config"* ]]; }; then
  debug_log "Listing server names:"
  jq -r '.mcpServers | keys[]' "$MCP_GATEWAY_OUTPUT" | while read -r server; do
    debug_log "  - $server"
  done
fi

# Create the TOML configuration
debug_log "Creating TOML header with [history] section..."
cat > /tmp/gh-aw/mcp-config/config.toml << 'TOML_EOF'
[history]
persistence = "none"

TOML_EOF
debug_log "✓ TOML header written"

# Convert each server from JSON to TOML format
debug_log "Converting servers to TOML format..."
debug_log "Using jq to iterate over servers and generate TOML sections..."

if [ -n "$DEBUG" ] && { [ "$DEBUG" = "1" ] || [ "$DEBUG" = "*" ] || [[ "$DEBUG" == *"convert_gateway_config"* ]]; }; then
  # Process servers and log each one
  SERVER_NUM=0
  jq -r --arg apiKey "$MCP_GATEWAY_API_KEY" '
    .mcpServers | to_entries[] | 
    "[mcp_servers.\(.key)]\n" +
    "url = \"\(.value.url)\"\n" +
    "\n" +
    "[mcp_servers.\(.key).headers]\n" +
    "Authorization = \"\($apiKey)\"\n"
  ' "$MCP_GATEWAY_OUTPUT" | while IFS= read -r line; do
    echo "$line" >> /tmp/gh-aw/mcp-config/config.toml
    # Log section headers
    if [[ "$line" =~ ^\[mcp_servers\.([^\]]+)\]$ ]]; then
      SERVER_NUM=$((SERVER_NUM + 1))
      debug_log "  Server $SERVER_NUM: ${BASH_REMATCH[1]}"
    fi
  done
else
  # Non-debug mode: process all at once
  jq -r --arg apiKey "$MCP_GATEWAY_API_KEY" '
    .mcpServers | to_entries[] | 
    "[mcp_servers.\(.key)]\n" +
    "url = \"\(.value.url)\"\n" +
    "\n" +
    "[mcp_servers.\(.key).headers]\n" +
    "Authorization = \"\($apiKey)\"\n"
  ' "$MCP_GATEWAY_OUTPUT" >> /tmp/gh-aw/mcp-config/config.toml
fi

debug_log "✓ TOML conversion completed successfully"
debug_log "Output written to: /tmp/gh-aw/mcp-config/config.toml"
debug_log "Output file size: $(wc -c < /tmp/gh-aw/mcp-config/config.toml) bytes"
debug_log "Output line count: $(wc -l < /tmp/gh-aw/mcp-config/config.toml) lines"

if [ -n "$DEBUG" ] && { [ "$DEBUG" = "1" ] || [ "$DEBUG" = "*" ] || [[ "$DEBUG" == *"convert_gateway_config"* ]]; }; then
  debug_log "Verifying TOML structure..."
  debug_log "  History section present: $(grep -q '^\[history\]' /tmp/gh-aw/mcp-config/config.toml && echo 'yes' || echo 'no')"
  debug_log "  MCP servers sections: $(grep -c '^\[mcp_servers\.' /tmp/gh-aw/mcp-config/config.toml)"
  debug_log "  Authorization headers: $(grep -c '^Authorization = ' /tmp/gh-aw/mcp-config/config.toml)"
fi

echo "Codex configuration written to /tmp/gh-aw/mcp-config/config.toml"
debug_log "=== Codex configuration conversion completed successfully ==="
echo ""
echo "Converted configuration:"
cat /tmp/gh-aw/mcp-config/config.toml
