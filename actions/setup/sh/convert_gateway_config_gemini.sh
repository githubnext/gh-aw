#!/usr/bin/env bash
# Convert MCP Gateway Configuration to Gemini Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the JSON format expected by Gemini CLI (.gemini/settings.json)
#
# Gemini CLI reads MCP server configuration from settings.json files:
# - Global: ~/.gemini/settings.json
# - Project: .gemini/settings.json (used here)
#
# See: https://geminicli.com/docs/tools/mcp-server/

set -e

# Required environment variables:
# - MCP_GATEWAY_OUTPUT: Path to gateway output configuration file
# - MCP_GATEWAY_DOMAIN: Domain to use for MCP server URLs (e.g., host.docker.internal)
# - MCP_GATEWAY_PORT: Port for MCP gateway (e.g., 80)
# - GITHUB_WORKSPACE: Workspace directory for project-level settings

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

if [ -z "$GITHUB_WORKSPACE" ]; then
  echo "ERROR: GITHUB_WORKSPACE environment variable is required"
  exit 1
fi

echo "Converting gateway configuration to Gemini format..."
echo "Input: $MCP_GATEWAY_OUTPUT"
echo "Target domain: $MCP_GATEWAY_DOMAIN:$MCP_GATEWAY_PORT"

# Convert gateway output to Gemini settings.json format
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
# Gemini settings.json format:
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
# The main differences:
# 1. Remove "type" field (Gemini uses transport auto-detection from url/httpUrl)
# 2. Remove "tools" field (Copilot-specific)
# 3. URLs must use the correct domain (host.docker.internal) for container access

# Build the correct URL prefix using the configured domain and port
URL_PREFIX="http://${MCP_GATEWAY_DOMAIN}:${MCP_GATEWAY_PORT}"

# Create .gemini directory in the workspace (project-level settings)
GEMINI_SETTINGS_DIR="${GITHUB_WORKSPACE}/.gemini"
GEMINI_SETTINGS_FILE="${GEMINI_SETTINGS_DIR}/settings.json"

mkdir -p "$GEMINI_SETTINGS_DIR"

jq --arg urlPrefix "$URL_PREFIX" '
  .mcpServers |= with_entries(
    .value |= (
      (del(.type)) |
      (del(.tools)) |
      # Fix the URL to use the correct domain
      .url |= (. | sub("^http://[^/]+/mcp/"; $urlPrefix + "/mcp/"))
    )
  )
' "$MCP_GATEWAY_OUTPUT" > "$GEMINI_SETTINGS_FILE"

echo "Gemini configuration written to $GEMINI_SETTINGS_FILE"
echo ""
echo "Converted configuration:"
cat "$GEMINI_SETTINGS_FILE"
