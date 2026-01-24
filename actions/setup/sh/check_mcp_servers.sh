#!/usr/bin/env bash
# Check MCP Server Functionality
# This script performs basic functionality checks on MCP servers configured by the MCP gateway
# It sends a ping message to each server to verify connectivity
#
# Resilience Features:
# - Progressive timeout: 10s, 20s, 30s across retry attempts
# - Progressive delay: 2s, 4s between retry attempts
# - Up to 3 retry attempts per server ping request
# - Accommodates slow-starting MCP servers (gateway may take 40-50 seconds to start)

set -e

# Usage: check_mcp_servers.sh GATEWAY_CONFIG_PATH GATEWAY_URL GATEWAY_API_KEY
#
# Arguments:
#   GATEWAY_CONFIG_PATH : Path to the gateway output configuration file (gateway-output.json)
#   GATEWAY_URL         : The HTTP URL of the MCP gateway (e.g., http://localhost:8080)
#   GATEWAY_API_KEY     : API key for gateway authentication
#
# Exit codes:
#   0 - At least one HTTP server successfully checked (failed servers logged as warnings)
#   1 - Invalid arguments, configuration file issues, or NO servers connected successfully
#
# Note: This script allows optional MCP servers to fail gracefully. As long as at least
# one HTTP server connects successfully, the script exits 0. This enables workflows with
# optional servers (e.g., Datadog, Sentry) that require secrets not always configured.

if [ "$#" -ne 3 ]; then
  echo "Usage: $0 GATEWAY_CONFIG_PATH GATEWAY_URL GATEWAY_API_KEY" >&2
  exit 1
fi

GATEWAY_CONFIG_PATH="$1"
GATEWAY_URL="$2"
GATEWAY_API_KEY="$3"

echo "Checking MCP servers..."
echo ""

# Validate configuration file exists
if [ ! -f "$GATEWAY_CONFIG_PATH" ]; then
  echo "ERROR: Gateway configuration file not found: $GATEWAY_CONFIG_PATH" >&2
  exit 1
fi

# Parse the mcpServers section from gateway-output.json
if ! MCP_SERVERS=$(jq -r '.mcpServers' "$GATEWAY_CONFIG_PATH" 2>/dev/null); then
  echo "ERROR: Failed to parse mcpServers from configuration file" >&2
  exit 1
fi

# Check if mcpServers is null or empty
if [ "$MCP_SERVERS" = "null" ] || [ "$MCP_SERVERS" = "{}" ]; then
  echo "No MCP servers configured"
  exit 0
fi

# Get list of server names
SERVER_NAMES=$(echo "$MCP_SERVERS" | jq -r 'keys[]' 2>/dev/null)

if [ -z "$SERVER_NAMES" ]; then
  echo "No MCP servers found"
  exit 0
fi

# Track overall results
SERVERS_CHECKED=0
SERVERS_SUCCEEDED=0
SERVERS_FAILED=0
SERVERS_SKIPPED=0

# Retry configuration for slow-starting servers
# Gateway may take 40-50 seconds to start all MCP servers (per start_mcp_gateway.sh)
MAX_RETRIES=3

# Iterate through each server
while IFS= read -r SERVER_NAME; do
  SERVERS_CHECKED=$((SERVERS_CHECKED + 1))
  
  # Extract server configuration
  SERVER_CONFIG=$(echo "$MCP_SERVERS" | jq -r ".\"$SERVER_NAME\"" 2>/dev/null)
  
  if [ "$SERVER_CONFIG" = "null" ]; then
    echo "⚠ $SERVER_NAME: configuration is null"
    SERVERS_FAILED=$((SERVERS_FAILED + 1))
    continue
  fi
  
  # Extract server URL (should be HTTP URL pointing to gateway)
  SERVER_URL=$(echo "$SERVER_CONFIG" | jq -r '.url // empty' 2>/dev/null)
  
  if [ -z "$SERVER_URL" ] || [ "$SERVER_URL" = "null" ]; then
    echo "⚠ $SERVER_NAME: skipped (not HTTP)"
    SERVERS_SKIPPED=$((SERVERS_SKIPPED + 1))
    continue
  fi
  
  # Extract authentication headers from gateway configuration
  AUTH_HEADER=""
  if echo "$SERVER_CONFIG" | jq -e '.headers.Authorization' >/dev/null 2>&1; then
    AUTH_HEADER=$(echo "$SERVER_CONFIG" | jq -r '.headers.Authorization' 2>/dev/null)
  fi
  
  # Send MCP ping request with retry logic
  PING_PAYLOAD='{"jsonrpc":"2.0","id":1,"method":"ping"}'
  
  # Retry logic for slow-starting servers
  RETRY_COUNT=0
  PING_SUCCESS=false
  
  while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    # Calculate timeout based on retry attempt (10s, 20s, 30s)
    TIMEOUT=$((10 + RETRY_COUNT * 10))
    
    if [ $RETRY_COUNT -gt 0 ]; then
      # Progressive delay between retries (2s, 4s)
      DELAY=$((2 * RETRY_COUNT))
      sleep $DELAY
    fi
    
    # Make the request with proper headers and progressive timeout
    if [ -n "$AUTH_HEADER" ]; then
      PING_RESPONSE=$(curl -s -w "\n%{http_code}" --max-time $TIMEOUT -X POST "$SERVER_URL" \
        -H "Content-Type: application/json" \
        -H "Authorization: $AUTH_HEADER" \
        -d "$PING_PAYLOAD" 2>&1 || echo -e "\n000")
    else
      PING_RESPONSE=$(curl -s -w "\n%{http_code}" --max-time $TIMEOUT -X POST "$SERVER_URL" \
        -H "Content-Type: application/json" \
        -d "$PING_PAYLOAD" 2>&1 || echo -e "\n000")
    fi
    
    PING_HTTP_CODE=$(echo "$PING_RESPONSE" | tail -n 1)
    PING_BODY=$(echo "$PING_RESPONSE" | head -n -1)
    
    # Check if ping succeeded
    if [ "$PING_HTTP_CODE" = "200" ]; then
      # Check for JSON-RPC error in response
      if ! echo "$PING_BODY" | jq -e '.error' >/dev/null 2>&1; then
        PING_SUCCESS=true
        break
      fi
    fi
    
    RETRY_COUNT=$((RETRY_COUNT + 1))
  done
  
  if [ "$PING_SUCCESS" = true ]; then
    echo "✓ $SERVER_NAME: connected"
    SERVERS_SUCCEEDED=$((SERVERS_SUCCEEDED + 1))
  else
    echo "✗ $SERVER_NAME: failed to connect"
    SERVERS_FAILED=$((SERVERS_FAILED + 1))
  fi
  
done <<< "$SERVER_NAMES"

# Print summary
echo ""
echo "MCP Server Check Summary:"
echo "  Succeeded: $SERVERS_SUCCEEDED"
echo "  Failed: $SERVERS_FAILED"
echo "  Skipped: $SERVERS_SKIPPED (non-HTTP servers)"
echo ""

# Only fail if NO servers succeeded (all servers are broken)
# This allows optional servers (datadog, sentry, etc.) to fail gracefully
# when their required secrets are not configured
if [ $SERVERS_SUCCEEDED -eq 0 ]; then
  echo "ERROR: No HTTP servers were successfully checked"
  echo "At least one MCP server must be operational for the workflow to function"
  exit 1
else
  if [ $SERVERS_FAILED -gt 0 ]; then
    echo "⚠ WARNING: $SERVERS_FAILED server(s) failed to connect"
    echo "This is expected for optional servers without configured secrets (e.g., Datadog, Sentry)"
    echo "The workflow will continue with the $SERVERS_SUCCEEDED available server(s)"
  else
    echo "✓ All checks passed ($SERVERS_SUCCEEDED succeeded, $SERVERS_SKIPPED skipped)"
  fi
  exit 0
fi
