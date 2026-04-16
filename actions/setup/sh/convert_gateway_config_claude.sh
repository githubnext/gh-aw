#!/usr/bin/env bash
# Convert MCP Gateway Configuration to Claude Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the JSON format expected by Claude (without Copilot-specific fields)

set -e

# Restrict default file creation mode to owner-only (rw-------) for all new files.
# This prevents the race window between file creation via output redirection and
# a subsequent chmod, which would leave credential-bearing files world-readable
# (mode 0644) with a typical umask of 022.
umask 077

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

echo "Converting gateway configuration to Claude format..."
echo "Input: $MCP_GATEWAY_OUTPUT"
echo "Target domain: $MCP_GATEWAY_DOMAIN:$MCP_GATEWAY_PORT"

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
# Claude format (JSON with HTTP type and headers):
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
# The main differences:
# 1. Claude uses "type": "http" for HTTP-based MCP servers
# 2. The "tools" field is preserved from the gateway config to enforce the tool allowlist
#    at the gateway layer (not removed, unlike older versions that treated it as Copilot-specific)
# 3. URLs must use the correct domain (host.docker.internal) for container access

# Build the correct URL prefix using the configured domain and port
URL_PREFIX="http://${MCP_GATEWAY_DOMAIN}:${MCP_GATEWAY_PORT}"

# The safeoutputs write-sink gets a direct connection (bypassing the gateway)
# with an env var reference for the Authorization header. This prevents bash
# subprocesses from reading the shared MCP gateway bearer token and using it
# to call the safeoutputs write-sink, bypassing the read-only permission ceiling.
# Claude Code expands ${GH_AW_SAFE_OUTPUTS_API_KEY} at connection time; the
# LD_PRELOAD one-shot library protects the env var from bash subprocess access.
SAFE_OUTPUTS_PORT="${GH_AW_SAFE_OUTPUTS_PORT:-3001}"
SAFE_OUTPUTS_DIRECT_URL="http://${MCP_GATEWAY_DOMAIN}:${SAFE_OUTPUTS_PORT}"

jq --arg urlPrefix "$URL_PREFIX" \
   --arg safeOutputsUrl "$SAFE_OUTPUTS_DIRECT_URL" '
  .mcpServers |= with_entries(
    if .key == "safeoutputs" then
      # Use direct URL and env var reference for Authorization (not gateway key)
      .value = {
        "type": "http",
        "url": $safeOutputsUrl,
        "headers": {
          "Authorization": "${GH_AW_SAFE_OUTPUTS_API_KEY}"
        }
      }
    else
      .value |= (
        (.type = "http") |
        # Fix the URL to use the correct domain
        .url |= (. | sub("^http://[^/]+/mcp/"; $urlPrefix + "/mcp/"))
      )
    end
  )
' "$MCP_GATEWAY_OUTPUT" > /tmp/gh-aw/mcp-config/mcp-servers.json

# Restrict permissions so only the runner process owner can read this file.
# mcp-servers.json contains the bearer token for the MCP gateway; an attacker
# who reads it could bypass the --allowed-tools constraint by issuing raw
# JSON-RPC calls directly to the gateway.
chmod 600 /tmp/gh-aw/mcp-config/mcp-servers.json

echo "Claude configuration written to /tmp/gh-aw/mcp-config/mcp-servers.json"
echo ""
echo "Converted configuration:"
cat /tmp/gh-aw/mcp-config/mcp-servers.json
