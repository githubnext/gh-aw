#!/usr/bin/env bash
# Start MCP Gateway
# This script starts the MCP gateway process that proxies MCP servers through a unified HTTP endpoint
# Following the MCP Gateway Specification: https://github.com/githubnext/gh-aw/blob/main/docs/src/content/docs/reference/mcp-gateway.md
# Per MCP Gateway Specification v1.0.0: Only container-based execution is supported.

set -e

# Required environment variables:
# - MCP_GATEWAY_PORT: Port for the gateway HTTP server
# - MCP_GATEWAY_DOMAIN: Domain for gateway URL (localhost or host.docker.internal)
# - MCP_GATEWAY_API_KEY: API key for gateway authentication
# - MCP_GATEWAY_DOCKER_COMMAND: Container image to run (required)

# Validate required environment variables
if [ -z "$MCP_GATEWAY_PORT" ]; then
  echo "ERROR: MCP_GATEWAY_PORT environment variable is required"
  exit 1
fi

if [ -z "$MCP_GATEWAY_DOMAIN" ]; then
  echo "ERROR: MCP_GATEWAY_DOMAIN environment variable is required"
  exit 1
fi

if [ -z "$MCP_GATEWAY_API_KEY" ]; then
  echo "ERROR: MCP_GATEWAY_API_KEY environment variable is required"
  exit 1
fi

# Validate that container is specified (command execution is not supported per spec)
if [ -z "$MCP_GATEWAY_DOCKER_COMMAND" ]; then
  echo "ERROR: MCP_GATEWAY_DOCKER_COMMAND must be set (command-based execution is not supported per MCP Gateway Specification v1.0.0)"
  exit 1
fi

# Create logs directory for gateway
mkdir -p /tmp/gh-aw/mcp-logs/gateway

# Validate container syntax first (before accessing files)
# Container should be a valid docker command starting with "docker run"
if ! echo "$MCP_GATEWAY_DOCKER_COMMAND" | grep -qE '^docker run'; then
  echo "ERROR: MCP_GATEWAY_DOCKER_COMMAND has incorrect syntax"
  echo "Expected: docker run command with image and arguments"
  echo "Got: $MCP_GATEWAY_DOCKER_COMMAND"
  exit 1
fi

# Validate container command includes required flags
if ! echo "$MCP_GATEWAY_DOCKER_COMMAND" | grep -qE -- '-i'; then
  echo "ERROR: MCP_GATEWAY_DOCKER_COMMAND must include -i flag for interactive mode"
  exit 1
fi

if ! echo "$MCP_GATEWAY_DOCKER_COMMAND" | grep -qE -- '--rm'; then
  echo "ERROR: MCP_GATEWAY_DOCKER_COMMAND must include --rm flag for cleanup"
  exit 1
fi

if ! echo "$MCP_GATEWAY_DOCKER_COMMAND" | grep -qE -- '--network host'; then
  echo "ERROR: MCP_GATEWAY_DOCKER_COMMAND must include --network host flag"
  exit 1
fi

# Determine MCP config file location based on engine type
# The gateway expects JSON input conforming to MCP Gateway Specification v1.0.0
if [ -n "$GH_AW_ENGINE" ]; then
  ENGINE_TYPE="$GH_AW_ENGINE"
elif [ -f "/home/runner/.copilot" ] || [ -n "$GITHUB_COPILOT_CLI_MODE" ]; then
  ENGINE_TYPE="copilot"
elif [ -f "/tmp/gh-aw/mcp-config/config.toml" ]; then
  ENGINE_TYPE="codex"
elif [ -f "/tmp/gh-aw/mcp-config/mcp-servers.json" ]; then
  ENGINE_TYPE="claude"
else
  ENGINE_TYPE="unknown"
fi

# Set config path based on engine type
case "$ENGINE_TYPE" in
  copilot)
    MCP_CONFIG_PATH="/home/runner/.copilot/mcp-config.json"
    ;;
  codex)
    MCP_CONFIG_PATH="/tmp/gh-aw/mcp-config/mcp-servers.json"
    ;;
  claude)
    MCP_CONFIG_PATH="/tmp/gh-aw/mcp-config/mcp-servers.json"
    ;;
  *)
    # Default to Copilot location for unknown engines
    MCP_CONFIG_PATH="/home/runner/.copilot/mcp-config.json"
    ;;
esac

echo "Engine type: $ENGINE_TYPE"
echo "MCP config path: $MCP_CONFIG_PATH"

# Validate configuration file exists
if [ ! -f "$MCP_CONFIG_PATH" ]; then
  echo "ERROR: Configuration file not found at $MCP_CONFIG_PATH"
  echo "The MCP configuration file must be created before starting the gateway"
  exit 1
fi

# Validate configuration file is valid JSON
if ! jq empty "$MCP_CONFIG_PATH" 2>/dev/null; then
  echo "ERROR: Configuration file $MCP_CONFIG_PATH is not valid JSON"
  exit 1
fi

# Build gateway configuration with runtime values
echo "Building gateway configuration..."
cat "$MCP_CONFIG_PATH" | jq --arg port "$MCP_GATEWAY_PORT" --arg apiKey "$MCP_GATEWAY_API_KEY" --arg domain "$MCP_GATEWAY_DOMAIN" \
  '.gateway = { port: ($port | tonumber), apiKey: $apiKey, domain: $domain }' > /tmp/gh-aw/mcp-config/gateway-input.json

echo "Gateway input configuration:"
cat /tmp/gh-aw/mcp-config/gateway-input.json
echo ""

# Start gateway process with container
echo "Starting gateway with container: $MCP_GATEWAY_DOCKER_COMMAND"
# Note: MCP_GATEWAY_DOCKER_COMMAND is the full docker command with all flags and image
cat /tmp/gh-aw/mcp-config/gateway-input.json | $MCP_GATEWAY_DOCKER_COMMAND \
  > /tmp/gh-aw/mcp-config/gateway-output.json 2> /tmp/gh-aw/mcp-logs/gateway/stderr.log &

GATEWAY_PID=$!
echo "Gateway started with PID: $GATEWAY_PID"
echo ""

# Wait for gateway to be ready using /health endpoint
echo "Waiting for gateway to be ready..."
MAX_ATTEMPTS=30
ATTEMPT=0
while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
  if curl -f -s "http://${MCP_GATEWAY_DOMAIN}:${MCP_GATEWAY_PORT}/health" > /dev/null 2>&1; then
    echo "Gateway is ready!"
    break
  fi
  ATTEMPT=$((ATTEMPT + 1))
  if [ $ATTEMPT -lt $MAX_ATTEMPTS ]; then
    echo "Attempt $ATTEMPT/$MAX_ATTEMPTS: Gateway not ready yet, waiting 1 second..."
    sleep 1
  fi
done

if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
  echo "ERROR: Gateway failed to become ready after $MAX_ATTEMPTS attempts"
  echo "Gateway stderr logs:"
  cat /tmp/gh-aw/mcp-logs/gateway/stderr.log || echo "No stderr logs available"
  kill $GATEWAY_PID 2>/dev/null || true
  exit 1
fi
echo ""

# Wait for gateway output (rewritten configuration)
echo "Reading gateway output configuration..."
WAIT_ATTEMPTS=10
WAIT_ATTEMPT=0
while [ $WAIT_ATTEMPT -lt $WAIT_ATTEMPTS ]; do
  if [ -s /tmp/gh-aw/mcp-config/gateway-output.json ]; then
    echo "Gateway output received!"
    break
  fi
  WAIT_ATTEMPT=$((WAIT_ATTEMPT + 1))
  if [ $WAIT_ATTEMPT -lt $WAIT_ATTEMPTS ]; then
    sleep 1
  fi
done

# Verify output was written
if [ ! -s /tmp/gh-aw/mcp-config/gateway-output.json ]; then
  echo "ERROR: Gateway did not write output configuration"
  echo "Gateway stderr logs:"
  cat /tmp/gh-aw/mcp-logs/gateway/stderr.log || echo "No stderr logs available"
  kill $GATEWAY_PID 2>/dev/null || true
  exit 1
fi

# Convert gateway output to agent-specific format
echo "Converting gateway configuration to agent format..."
export MCP_GATEWAY_OUTPUT=/tmp/gh-aw/mcp-config/gateway-output.json

# Determine which agent-specific converter to use based on engine type
# Check for engine-specific indicators and call appropriate converter
if [ -n "$GH_AW_ENGINE" ]; then
  ENGINE_TYPE="$GH_AW_ENGINE"
elif [ -f "/home/runner/.copilot" ] || [ -n "$GITHUB_COPILOT_CLI_MODE" ]; then
  ENGINE_TYPE="copilot"
elif [ -f "/tmp/gh-aw/mcp-config/config.toml" ]; then
  ENGINE_TYPE="codex"
elif [ -f "/tmp/gh-aw/mcp-config/mcp-servers.json" ]; then
  ENGINE_TYPE="claude"
else
  ENGINE_TYPE="unknown"
fi

echo "Detected engine type: $ENGINE_TYPE"

case "$ENGINE_TYPE" in
  copilot)
    echo "Using Copilot converter..."
    bash /tmp/gh-aw/actions/convert_gateway_config_copilot.sh
    ;;
  codex)
    echo "Using Codex converter..."
    bash /tmp/gh-aw/actions/convert_gateway_config_codex.sh
    ;;
  claude)
    echo "Using Claude converter..."
    bash /tmp/gh-aw/actions/convert_gateway_config_claude.sh
    ;;
  *)
    echo "No agent-specific converter found for engine: $ENGINE_TYPE"
    echo "Using gateway output directly"
    # Default fallback - copy to most common location
    mkdir -p /home/runner/.copilot
    cp /tmp/gh-aw/mcp-config/gateway-output.json /home/runner/.copilot/mcp-config.json
    cat /home/runner/.copilot/mcp-config.json
    ;;
esac
echo ""

echo "MCP gateway is running on http://${MCP_GATEWAY_DOMAIN}:${MCP_GATEWAY_PORT}"
echo "Gateway PID: $GATEWAY_PID"

# Store PID for cleanup
echo $GATEWAY_PID > /tmp/gh-aw/mcp-logs/gateway/gateway.pid
