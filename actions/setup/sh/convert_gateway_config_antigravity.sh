#!/usr/bin/env bash
set +o histexpand

# Convert MCP Gateway Configuration to Antigravity Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the JSON format expected by Antigravity CLI (.antigravity/settings.json)
#
# Antigravity CLI reads MCP server configuration from settings.json files:
# - Global: ~/.antigravity/settings.json
# - Project: .antigravity/settings.json (used here)
#
# See: https://antigravitycli.com/docs/tools/mcp-server/

set -e

# Restrict default file creation mode to owner-only (rw-------) for all new files.
# This prevents the race window between file creation via output redirection and
# a subsequent chmod, which would leave credential-bearing files world-readable
# (mode 0644) with a typical umask of 022.
umask 077

# Required environment variables:
# - MCP_GATEWAY_OUTPUT: Path to gateway output configuration file
# - MCP_GATEWAY_PORT: Port for MCP gateway (e.g., 80)
# - GITHUB_WORKSPACE: Workspace directory for project-level settings
#
# Optional environment variables:
# - MCP_GATEWAY_HOST_DOMAIN: Host-side domain for Antigravity MCP URLs (default: localhost)

if [ -z "$MCP_GATEWAY_OUTPUT" ]; then
  echo "ERROR: MCP_GATEWAY_OUTPUT environment variable is required"
  exit 1
fi

if [ ! -f "$MCP_GATEWAY_OUTPUT" ]; then
  echo "ERROR: Gateway output file not found: $MCP_GATEWAY_OUTPUT"
  exit 1
fi

if [ -z "$MCP_GATEWAY_HOST_DOMAIN" ]; then
  echo "WARNING: MCP_GATEWAY_HOST_DOMAIN environment variable not set, defaulting to localhost"
  MCP_GATEWAY_HOST_DOMAIN="localhost"
fi

if [ -z "$MCP_GATEWAY_PORT" ]; then
  echo "ERROR: MCP_GATEWAY_PORT environment variable is required"
  exit 1
fi

if [ -z "$GITHUB_WORKSPACE" ]; then
  echo "ERROR: GITHUB_WORKSPACE environment variable is required"
  exit 1
fi

echo "Converting gateway configuration to Antigravity format..."
echo "Input: $MCP_GATEWAY_OUTPUT"
echo "Target domain: $MCP_GATEWAY_HOST_DOMAIN:$MCP_GATEWAY_PORT"

# Convert gateway output to Antigravity settings.json format
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
# Antigravity settings.json format:
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
# 1. Remove "type" field (Antigravity uses transport auto-detection from url/httpUrl)
# 2. The "tools" field is preserved from the gateway config to enforce the tool allowlist
#    at the gateway layer (not removed, unlike older versions that treated it as Copilot-specific)
# 3. URLs must use localhost (MCP_GATEWAY_HOST_DOMAIN) since Antigravity runs on the host runner

# Build the correct URL prefix using the host-side domain and port.
# Antigravity CLI runs directly on the host runner (not inside a Docker container), so use
# MCP_GATEWAY_HOST_DOMAIN (localhost) instead of MCP_GATEWAY_DOMAIN (host.docker.internal).
# host.docker.internal does not resolve on the host runner on Linux.
URL_PREFIX="http://${MCP_GATEWAY_HOST_DOMAIN}:${MCP_GATEWAY_PORT}"

# Create .antigravity directory in the workspace (project-level settings)
ANTIGRAVITY_SETTINGS_DIR="${GITHUB_WORKSPACE}/.antigravity"
ANTIGRAVITY_SETTINGS_FILE="${ANTIGRAVITY_SETTINGS_DIR}/settings.json"

mkdir -p "$ANTIGRAVITY_SETTINGS_DIR"

jq --arg urlPrefix "$URL_PREFIX" --argjson cliServers "${GH_AW_MCP_CLI_SERVERS:-[]}" '
  .mcpServers |= with_entries(
    select(.key | IN($cliServers[]) | not) |
    .value |= (
      (del(.type)) |
      # Fix the URL to use the correct domain
      .url |= (. | sub("^http://[^/]+/mcp/"; $urlPrefix + "/mcp/"))
    )
  ) |
  # Allow Antigravity CLI to read/write files from /tmp/ (e.g. MCP payload files, cache-memory, agent outputs)
  .context.includeDirectories = ["/tmp/"]
' "$MCP_GATEWAY_OUTPUT" > "$ANTIGRAVITY_SETTINGS_FILE"

# Restrict permissions so only the runner process owner can read this file.
# settings.json contains the bearer token for the MCP gateway; an attacker
# who reads it could bypass the --allowed-tools constraint by issuing raw
# JSON-RPC calls directly to the gateway.
chmod 600 "$ANTIGRAVITY_SETTINGS_FILE"

echo "Antigravity configuration written to $ANTIGRAVITY_SETTINGS_FILE"
echo ""
echo "Converted configuration:"
cat "$ANTIGRAVITY_SETTINGS_FILE"
