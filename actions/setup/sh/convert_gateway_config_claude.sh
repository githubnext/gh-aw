#!/usr/bin/env bash
# Convert MCP Gateway Configuration to Claude Format
# This script converts the gateway's standard HTTP-based MCP configuration
# to the JSON format expected by Claude (without Copilot-specific fields)

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

echo "Converting gateway configuration to Claude format..."
echo "Input: $MCP_GATEWAY_OUTPUT"

# Convert gateway output to Claude format
#
# Gateway can output two formats:
#
# 1. HTTP format (post-transformation):
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
# 2. Gateway-spec format (pre-transformation):
# {
#   "mcpServers": {
#     "server-name": {
#       "type": "stdio",
#       "container": "image:tag",
#       "entrypointArgs": ["arg1", "arg2"],
#       "mounts": ["src:dest:mode"],
#       "env": {
#         "VAR": "value"
#       }
#     }
#   }
# }
#
# Claude Code format (per Claude Code MCP schema):
# {
#   "mcpServers": {
#     "server-name": {
#       "url": "http://domain:port/mcp/server-name",  // For HTTP servers
#       "headers": {
#         "Authorization": "apiKey"
#       }
#     }
#   }
# }
# OR for stdio servers:
# {
#   "mcpServers": {
#     "server-name": {
#       "command": "docker",
#       "args": ["run", "--rm", "-i", "-e", "VAR", "image:tag", "arg1", "arg2"],
#       "env": {
#         "VAR": "value"
#       }
#     }
#   }
# }
#
# Claude Code doesn't use "type", "tools", "container", "entrypointArgs", "mounts",
# "entrypoint", "registry", or "proxy-args" fields.
# It expects either HTTP servers (url + headers) or stdio servers (command + args + env).

jq --arg apiKey "$MCP_GATEWAY_API_KEY" '
  .mcpServers |= with_entries(
    .value |= (
      # Transform gateway-spec stdio servers to Claude-compatible format
      # If server has "container" field, transform to "command" + "args" format
      if .container then
        (
          # Build Docker command from container spec
          .command = "docker" |
          # Build args array: docker run flags + container image + entrypoint args
          .args = (
            # Start with docker run base flags
            ["run", "--rm", "-i"] +
            # Add environment variables as -e flags
            (if .env then 
              [.env | to_entries | .[] | "-e", .key] | flatten
            else [] end) +
            # Add volume mounts as -v flags  
            (if .mounts then
              [.mounts | .[] | "-v", .] | flatten
            else [] end) +
            # Add custom args if present (additional docker flags)
            (if .args then .args else [] end) +
            # Add entrypoint override if specified
            (if .entrypoint then ["--entrypoint", .entrypoint] else [] end) +
            # Add container image
            [.container] +
            # Add entrypoint args after container image
            (if .entrypointArgs then .entrypointArgs else [] end)
          ) |
          # Remove gateway-specific fields
          del(.container, .entrypointArgs, .mounts, .entrypoint, .registry, .["proxy-args"])
        )
      else . end |
      # Remove type field if present (Claude doesn'\''t use it)
      del(.type) |
      # Remove tools field if present (Claude doesn'\''t use it)
      del(.tools) |
      # Ensure headers Authorization uses actual API key for HTTP servers
      if .headers and .headers.Authorization then
        .headers.Authorization = $apiKey
      else
        .
      end
    )
  )
' "$MCP_GATEWAY_OUTPUT" > /tmp/gh-aw/mcp-config/mcp-servers.json

echo "Claude configuration written to /tmp/gh-aw/mcp-config/mcp-servers.json"
echo ""
echo "Converted configuration:"
cat /tmp/gh-aw/mcp-config/mcp-servers.json
