#!/usr/bin/env bash
# Start MCP Gateway
# This script starts the MCP gateway process that proxies MCP servers through a unified HTTP endpoint
# Following the MCP Gateway Specification: https://github.com/githubnext/gh-aw/blob/main/docs/src/content/docs/reference/mcp-gateway.md
# Per MCP Gateway Specification v1.0.0: Only container-based execution is supported.
#
# This script reads the MCP configuration from stdin and pipes it to the gateway container.

set -e

# Required environment variables:
# - MCP_GATEWAY_DOCKER_COMMAND: Container image to run (required)

# Validate that container is specified (command execution is not supported per spec)
if [ -z "$MCP_GATEWAY_DOCKER_COMMAND" ]; then
  echo "ERROR: MCP_GATEWAY_DOCKER_COMMAND must be set (command-based execution is not supported per MCP Gateway Specification v1.0.0)"
  exit 1
fi

# Create logs directory for gateway
mkdir -p /tmp/gh-aw/mcp-logs/gateway
mkdir -p /tmp/gh-aw/mcp-config

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

# Read MCP configuration from stdin
echo "Reading MCP configuration from stdin..."
MCP_CONFIG=$(cat)

# Log the configuration for debugging
echo "-------START MCP CONFIG-----------"
echo "$MCP_CONFIG"
echo "-------END MCP CONFIG-----------"
echo ""

# Validate configuration is valid JSON
if ! echo "$MCP_CONFIG" | jq empty 2>/tmp/gh-aw/mcp-config/jq-error.log; then
  echo "ERROR: Configuration is not valid JSON"
  echo ""
  echo "JSON validation error:"
  if [ -f /tmp/gh-aw/mcp-config/jq-error.log ]; then
    cat /tmp/gh-aw/mcp-config/jq-error.log
  fi
  echo ""
  echo "Configuration content:"
  echo "$MCP_CONFIG" | head -50
  if [ $(echo "$MCP_CONFIG" | wc -l) -gt 50 ]; then
    echo "... (truncated, showing first 50 lines)"
  fi
  exit 1
fi

# Validate gateway section exists and has required fields
echo "Validating gateway configuration..."
if ! echo "$MCP_CONFIG" | jq -e '.gateway' >/dev/null 2>&1; then
  echo "ERROR: Configuration is missing required 'gateway' section"
  echo "Per MCP Gateway Specification v1.0.0 section 4.1.3, the gateway section is required"
  exit 1
fi

if ! echo "$MCP_CONFIG" | jq -e '.gateway.port' >/dev/null 2>&1; then
  echo "ERROR: Gateway configuration is missing required 'port' field"
  exit 1
fi

if ! echo "$MCP_CONFIG" | jq -e '.gateway.domain' >/dev/null 2>&1; then
  echo "ERROR: Gateway configuration is missing required 'domain' field"
  exit 1
fi

if ! echo "$MCP_CONFIG" | jq -e '.gateway.apiKey' >/dev/null 2>&1; then
  echo "ERROR: Gateway configuration is missing required 'apiKey' field"
  exit 1
fi

echo "Configuration validated successfully"
echo ""

# Start gateway process with container
echo "Starting gateway with container: $MCP_GATEWAY_DOCKER_COMMAND"
echo "Full docker command: $MCP_GATEWAY_DOCKER_COMMAND"
echo ""
# Note: MCP_GATEWAY_DOCKER_COMMAND is the full docker command with all flags, mounts, and image
echo "$MCP_CONFIG" | $MCP_GATEWAY_DOCKER_COMMAND \
  > /tmp/gh-aw/mcp-config/gateway-output.json 2> /tmp/gh-aw/mcp-logs/gateway/stderr.log &

GATEWAY_PID=$!
echo "Gateway started with PID: $GATEWAY_PID"
echo "Verifying gateway process is running..."
if ps -p $GATEWAY_PID > /dev/null 2>&1; then
  echo "Gateway process confirmed running (PID: $GATEWAY_PID)"
else
  echo "ERROR: Gateway process exited immediately after start"
  echo ""
  echo "Gateway stdout output:"
  cat /tmp/gh-aw/mcp-config/gateway-output.json 2>/dev/null || echo "No stdout output available"
  echo ""
  echo "Gateway stderr logs:"
  cat /tmp/gh-aw/mcp-logs/gateway/stderr.log 2>/dev/null || echo "No stderr logs available"
  exit 1
fi
echo ""

# Wait for gateway to be ready using /health endpoint
# Note: Gateway may take 40-50 seconds when starting multiple MCP servers
# (e.g., serena alone takes ~22 seconds to start)
echo "Waiting for gateway to be ready..."
echo "Health endpoint: http://${MCP_GATEWAY_DOMAIN}:${MCP_GATEWAY_PORT}/health"
MAX_ATTEMPTS=60
ATTEMPT=0
while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
  # First check if the gateway process is still running
  if ! ps -p $GATEWAY_PID > /dev/null 2>&1; then
    echo "ERROR: Gateway process (PID: $GATEWAY_PID) has exited unexpectedly!"
    echo ""
    echo "Gateway stdout output:"
    cat /tmp/gh-aw/mcp-config/gateway-output.json 2>/dev/null || echo "No stdout output available"
    echo ""
    echo "Gateway stderr logs:"
    cat /tmp/gh-aw/mcp-logs/gateway/stderr.log 2>/dev/null || echo "No stderr logs available"
    exit 1
  fi
  
  # Check health endpoint
  HEALTH_RESPONSE=$(curl -f -s "http://${MCP_GATEWAY_DOMAIN}:${MCP_GATEWAY_PORT}/health" 2>&1) && {
    echo "Gateway is ready!"
    echo "Health response: $HEALTH_RESPONSE"
    break
  }
  ATTEMPT=$((ATTEMPT + 1))
  if [ $ATTEMPT -lt $MAX_ATTEMPTS ]; then
    echo "Attempt $ATTEMPT/$MAX_ATTEMPTS: Gateway not ready yet (curl response: $HEALTH_RESPONSE), waiting 1 second..."
    sleep 1
  fi
done

if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
  echo "ERROR: Gateway failed to become ready after $MAX_ATTEMPTS attempts"
  echo ""
  echo "Checking if gateway process is still alive..."
  if ps -p $GATEWAY_PID > /dev/null 2>&1; then
    echo "Gateway process (PID: $GATEWAY_PID) is still running"
  else
    echo "Gateway process (PID: $GATEWAY_PID) has exited"
    WAIT_STATUS=$(wait $GATEWAY_PID 2>/dev/null; echo $?)
    echo "Gateway exit status: $WAIT_STATUS"
  fi
  echo ""
  echo "Docker container status:"
  docker ps -a 2>/dev/null | head -20 || echo "Could not list docker containers"
  echo ""
  echo "Gateway stdout (errors are written here per MCP Gateway Specification):"
  cat /tmp/gh-aw/mcp-config/gateway-output.json 2>/dev/null || echo "No stdout output available"
  echo ""
  echo "Gateway stderr logs (debug output):"
  cat /tmp/gh-aw/mcp-logs/gateway/stderr.log || echo "No stderr logs available"
  echo ""
  echo "Checking network connectivity to gateway port..."
  netstat -tlnp 2>/dev/null | grep ":${MCP_GATEWAY_PORT}" || ss -tlnp 2>/dev/null | grep ":${MCP_GATEWAY_PORT}" || echo "Port ${MCP_GATEWAY_PORT} does not appear to be listening"
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
  echo ""
  echo "Gateway stdout (should contain error or config):"
  cat /tmp/gh-aw/mcp-config/gateway-output.json 2>/dev/null || echo "No stdout output available"
  echo ""
  echo "Gateway stderr logs:"
  cat /tmp/gh-aw/mcp-logs/gateway/stderr.log || echo "No stderr logs available"
  kill $GATEWAY_PID 2>/dev/null || true
  exit 1
fi

# Check if output contains an error payload instead of valid configuration
# Per MCP Gateway Specification v1.0.0 section 9.1, errors are written to stdout as error payloads
if jq -e '.error' /tmp/gh-aw/mcp-config/gateway-output.json >/dev/null 2>&1; then
  echo "ERROR: Gateway returned an error payload instead of configuration"
  echo ""
  echo "Gateway error details:"
  cat /tmp/gh-aw/mcp-config/gateway-output.json
  echo ""
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
