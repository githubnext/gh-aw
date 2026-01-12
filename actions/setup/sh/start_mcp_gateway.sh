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
# - MCP_GATEWAY_API_KEY: API key for gateway authentication (required for converter scripts)

# Validate that container is specified (command execution is not supported per spec)
if [ -z "$MCP_GATEWAY_DOCKER_COMMAND" ]; then
  echo "ERROR: MCP_GATEWAY_DOCKER_COMMAND must be set (command-based execution is not supported per MCP Gateway Specification v1.0.0)"
  exit 1
fi

# Create logs directory for gateway
mkdir -p /tmp/gh-aw/mcp-logs
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

# Set MCP_GATEWAY_LOG_DIR environment variable for use by the gateway
export MCP_GATEWAY_LOG_DIR="/tmp/gh-aw/mcp-logs/"

# Start gateway process with container
echo "Starting gateway with container: $MCP_GATEWAY_DOCKER_COMMAND"
echo "Full docker command: $MCP_GATEWAY_DOCKER_COMMAND"
echo ""

# Security requirement: Gateway configuration must never be stored in a file
# We use a named pipe (FIFO) to keep configuration in memory only
GATEWAY_OUTPUT_PIPE="/tmp/gh-aw/mcp-config/gateway-output.fifo"
mkfifo "$GATEWAY_OUTPUT_PIPE"

# Note: MCP_GATEWAY_DOCKER_COMMAND is the full docker command with all flags, mounts, and image
# Pass MCP_GATEWAY_LOG_DIR to the container via -e flag
echo "$MCP_CONFIG" | MCP_GATEWAY_LOG_DIR="$MCP_GATEWAY_LOG_DIR" $MCP_GATEWAY_DOCKER_COMMAND \
  > "$GATEWAY_OUTPUT_PIPE" 2> /tmp/gh-aw/mcp-logs/stderr.log &

GATEWAY_PID=$!
echo "Gateway started with PID: $GATEWAY_PID"
echo "Verifying gateway process is running..."
if ps -p $GATEWAY_PID > /dev/null 2>&1; then
  echo "Gateway process confirmed running (PID: $GATEWAY_PID)"
else
  echo "ERROR: Gateway process exited immediately after start"
  echo ""
  echo "Gateway stderr logs:"
  cat /tmp/gh-aw/mcp-logs/stderr.log 2>/dev/null || echo "No stderr logs available"
  rm -f "$GATEWAY_OUTPUT_PIPE"
  exit 1
fi
echo ""

# Wait for gateway to be ready using /health endpoint
# Note: Gateway may take 40-50 seconds when starting multiple MCP servers
# (e.g., serena alone takes ~22 seconds to start)
echo "Waiting for gateway to be ready..."
# Use localhost for health check since:
# 1. This script runs on the host (not in a container)
# 2. The gateway uses --network host, so it's accessible on localhost
# Note: MCP_GATEWAY_DOMAIN may be set to host.docker.internal for use by containers,
# but the health check should always use localhost since we're running on the host.
HEALTH_CHECK_HOST="localhost"
echo "Health endpoint: http://${HEALTH_CHECK_HOST}:${MCP_GATEWAY_PORT}/health"
echo "(Note: MCP_GATEWAY_DOMAIN is '${MCP_GATEWAY_DOMAIN}' for container access)"
echo "Retrying up to 120 times with 1s delay (120s total timeout)"

# Check health endpoint using localhost (since we're running on the host)
# Per MCP Gateway Specification v1.3.0, /health must return HTTP 200 with JSON body containing specVersion and gatewayVersion
# Use curl retry: 120 attempts with 1 second delay = 120s total
RESPONSE=$(curl -s --retry 120 --retry-delay 1 --retry-connrefused --retry-all-errors -w "\n%{http_code}" "http://${HEALTH_CHECK_HOST}:${MCP_GATEWAY_PORT}/health" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
HEALTH_RESPONSE=$(echo "$RESPONSE" | head -n -1)

# Always log the health response for debugging
echo "Health endpoint HTTP code: $HTTP_CODE"
if [ -n "$HEALTH_RESPONSE" ]; then
  echo "Health response body: $HEALTH_RESPONSE"
else
  echo "Health response body: (empty)"
fi

if [ "$HTTP_CODE" = "200" ] && [ -n "$HEALTH_RESPONSE" ]; then
  echo "Gateway is ready!"
else
  echo ""
  echo "ERROR: Gateway failed to become ready"
  echo "Last HTTP code: $HTTP_CODE"
  echo "Last health response: ${HEALTH_RESPONSE:-(empty)}"
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
  echo "Gateway stderr logs (debug output):"
  cat /tmp/gh-aw/mcp-logs/stderr.log || echo "No stderr logs available"
  echo ""
  echo "Checking network connectivity to gateway port..."
  netstat -tlnp 2>/dev/null | grep ":${MCP_GATEWAY_PORT}" || ss -tlnp 2>/dev/null | grep ":${MCP_GATEWAY_PORT}" || echo "Port ${MCP_GATEWAY_PORT} does not appear to be listening"
  kill $GATEWAY_PID 2>/dev/null || true
  exit 1
fi
echo ""

# Wait for gateway output (rewritten configuration)
echo "Reading gateway output configuration from named pipe (in-memory)..."

# Read gateway configuration into memory from the named pipe
# This blocks until the gateway writes its output
# Timeout after 30 seconds
if ! GATEWAY_CONFIG=$(timeout 30 cat "$GATEWAY_OUTPUT_PIPE" 2>/dev/null); then
  echo "ERROR: Timeout or error reading gateway output"
  echo ""
  echo "Gateway stderr logs:"
  cat /tmp/gh-aw/mcp-logs/stderr.log || echo "No stderr logs available"
  rm -f "$GATEWAY_OUTPUT_PIPE"
  kill $GATEWAY_PID 2>/dev/null || true
  exit 1
fi

# Remove the named pipe after reading
rm -f "$GATEWAY_OUTPUT_PIPE"

# Verify output was received
if [ -z "$GATEWAY_CONFIG" ]; then
  echo "ERROR: Gateway did not write output configuration"
  echo ""
  echo "Gateway stderr logs:"
  cat /tmp/gh-aw/mcp-logs/stderr.log || echo "No stderr logs available"
  kill $GATEWAY_PID 2>/dev/null || true
  exit 1
fi

echo "Gateway output received (in-memory)!"

# Check if output contains an error payload instead of valid configuration
# Per MCP Gateway Specification v1.0.0 section 9.1, errors are written to stdout as error payloads
if echo "$GATEWAY_CONFIG" | jq -e '.error' >/dev/null 2>&1; then
  echo "ERROR: Gateway returned an error payload instead of configuration"
  echo ""
  echo "Gateway error details:"
  echo "$GATEWAY_CONFIG"
  echo ""
  echo "Gateway stderr logs:"
  cat /tmp/gh-aw/mcp-logs/stderr.log || echo "No stderr logs available"
  kill $GATEWAY_PID 2>/dev/null || true
  exit 1
fi

# Convert gateway output to agent-specific format
echo "Converting gateway configuration to agent format..."

# Gateway configuration is now in-memory in GATEWAY_CONFIG variable
# Pass it via stdin to converter scripts (no files created for gateway output)

# Validate MCP_GATEWAY_API_KEY is set (required by converter scripts)
if [ -z "$MCP_GATEWAY_API_KEY" ]; then
  echo "ERROR: MCP_GATEWAY_API_KEY environment variable must be set for converter scripts"
  echo "This variable should be set in the workflow before calling start_mcp_gateway.sh"
  exit 1
fi

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
    echo "$GATEWAY_CONFIG" | bash /opt/gh-aw/actions/convert_gateway_config_copilot.sh
    ;;
  codex)
    echo "Using Codex converter..."
    echo "$GATEWAY_CONFIG" | bash /opt/gh-aw/actions/convert_gateway_config_codex.sh
    ;;
  claude)
    echo "Using Claude converter..."
    echo "$GATEWAY_CONFIG" | bash /opt/gh-aw/actions/convert_gateway_config_claude.sh
    ;;
  *)
    echo "No agent-specific converter found for engine: $ENGINE_TYPE"
    echo "Using gateway output directly"
    # Default fallback - write to most common location (but only the final converted config)
    mkdir -p /home/runner/.copilot
    echo "$GATEWAY_CONFIG" | jq '.' > /home/runner/.copilot/mcp-config.json
    cat /home/runner/.copilot/mcp-config.json
    ;;
esac
echo ""

# Check MCP server functionality
echo "Checking MCP server functionality..."
if [ -f /opt/gh-aw/actions/check_mcp_servers.sh ]; then
  echo "Running MCP server checks..."
  # Store check diagnostic logs in /tmp/gh-aw/mcp-logs/start-gateway.log for artifact upload
  # Use tee to output to both stdout and the log file
  # Enable pipefail so the exit code comes from check_mcp_servers.sh, not tee
  # Use process substitution to pass in-memory configuration without creating a file
  set -o pipefail
  if ! bash /opt/gh-aw/actions/check_mcp_servers.sh \
    <(echo "$GATEWAY_CONFIG") \
    "http://localhost:${MCP_GATEWAY_PORT}" \
    "${MCP_GATEWAY_API_KEY}" 2>&1 | tee /tmp/gh-aw/mcp-logs/start-gateway.log; then
    echo "ERROR: MCP server checks failed - no servers could be connected"
    echo "Gateway process will be terminated"
    kill $GATEWAY_PID 2>/dev/null || true
    exit 1
  fi
  set +o pipefail
else
  echo "WARNING: MCP server check script not found at /opt/gh-aw/actions/check_mcp_servers.sh"
  echo "Skipping MCP server functionality checks"
fi
echo ""

echo "MCP gateway is running:"
echo "  - From host: http://localhost:${MCP_GATEWAY_PORT}"
echo "  - From containers: http://${MCP_GATEWAY_DOMAIN}:${MCP_GATEWAY_PORT}"
echo "Gateway PID: $GATEWAY_PID"

# Output PID as GitHub Actions step output for use in cleanup
echo "gateway-pid=$GATEWAY_PID" >> $GITHUB_OUTPUT
# Output port and API key for use in stop script (per MCP Gateway Specification v1.1.0)
echo "gateway-port=${MCP_GATEWAY_PORT}" >> $GITHUB_OUTPUT
echo "gateway-api-key=${MCP_GATEWAY_API_KEY}" >> $GITHUB_OUTPUT
