#!/usr/bin/env bash
# Check MCP Server Functionality
# This script performs basic functionality checks on MCP servers configured by the MCP gateway
# It connects to each server, retrieves tools list, and displays available tools

set -e

# Usage: check_mcp_servers.sh GATEWAY_CONFIG_PATH GATEWAY_URL GATEWAY_API_KEY
#
# Arguments:
#   GATEWAY_CONFIG_PATH : Path to the gateway output configuration file (gateway-output.json)
#   GATEWAY_URL         : The HTTP URL of the MCP gateway (e.g., http://localhost:8080)
#   GATEWAY_API_KEY     : API key for gateway authentication
#
# Exit codes:
#   0 - All server checks completed (warnings logged for failures)
#   1 - Invalid arguments or configuration file issues

if [ "$#" -ne 3 ]; then
  echo "Usage: $0 GATEWAY_CONFIG_PATH GATEWAY_URL GATEWAY_API_KEY" >&2
  exit 1
fi

GATEWAY_CONFIG_PATH="$1"
GATEWAY_URL="$2"
GATEWAY_API_KEY="$3"

echo "=========================================="
echo "MCP Server Functionality Check"
echo "=========================================="
echo ""
echo "Configuration:"
echo "  Gateway Config: $GATEWAY_CONFIG_PATH"
echo "  Gateway URL: $GATEWAY_URL"
echo "  API Key: ${GATEWAY_API_KEY:0:8}..." # Show only first 8 chars for security
echo ""

# Validate configuration file exists
if [ ! -f "$GATEWAY_CONFIG_PATH" ]; then
  echo "ERROR: Gateway configuration file not found: $GATEWAY_CONFIG_PATH" >&2
  exit 1
fi

echo "Reading gateway configuration..."
# Parse the mcpServers section from gateway-output.json
if ! MCP_SERVERS=$(jq -r '.mcpServers' "$GATEWAY_CONFIG_PATH" 2>/dev/null); then
  echo "ERROR: Failed to parse mcpServers from configuration file" >&2
  echo "Configuration file content:" >&2
  cat "$GATEWAY_CONFIG_PATH" >&2
  exit 1
fi

# Check if mcpServers is null or empty
if [ "$MCP_SERVERS" = "null" ] || [ "$MCP_SERVERS" = "{}" ]; then
  echo "No MCP servers configured in gateway output"
  echo "Configuration appears to be empty or invalid"
  exit 0
fi

echo "Gateway configuration loaded successfully"
echo ""

# Get list of server names
SERVER_NAMES=$(echo "$MCP_SERVERS" | jq -r 'keys[]' 2>/dev/null)

if [ -z "$SERVER_NAMES" ]; then
  echo "No MCP servers found in configuration"
  exit 0
fi

# Count servers
SERVER_COUNT=$(echo "$SERVER_NAMES" | wc -l | tr -d ' ')
echo "Found $SERVER_COUNT MCP server(s) to check"
echo ""

# Track overall results
SERVERS_CHECKED=0
SERVERS_SUCCEEDED=0
SERVERS_FAILED=0

# Iterate through each server
while IFS= read -r SERVER_NAME; do
  echo "=========================================="
  echo "Checking server: $SERVER_NAME"
  echo "=========================================="
  
  SERVERS_CHECKED=$((SERVERS_CHECKED + 1))
  
  # Extract server configuration
  echo "Extracting server configuration..."
  SERVER_CONFIG=$(echo "$MCP_SERVERS" | jq -r ".\"$SERVER_NAME\"" 2>/dev/null)
  
  if [ "$SERVER_CONFIG" = "null" ]; then
    echo "WARNING: Server configuration is null for: $SERVER_NAME"
    SERVERS_FAILED=$((SERVERS_FAILED + 1))
    echo ""
    continue
  fi
  
  echo "Server configuration:"
  echo "$SERVER_CONFIG" | jq '.' 2>/dev/null || echo "$SERVER_CONFIG"
  echo ""
  
  # Extract server URL (should be HTTP URL pointing to gateway)
  SERVER_URL=$(echo "$SERVER_CONFIG" | jq -r '.url // empty' 2>/dev/null)
  
  if [ -z "$SERVER_URL" ] || [ "$SERVER_URL" = "null" ]; then
    echo "WARNING: Server does not have HTTP URL (not gatewayed): $SERVER_NAME"
    echo "Skipping functionality check..."
    SERVERS_FAILED=$((SERVERS_FAILED + 1))
    echo ""
    continue
  fi
  
  echo "Server URL: $SERVER_URL"
  
  # Extract authentication headers if present
  AUTH_HEADER=""
  if echo "$SERVER_CONFIG" | jq -e '.headers.Authorization' >/dev/null 2>&1; then
    AUTH_HEADER=$(echo "$SERVER_CONFIG" | jq -r '.headers.Authorization' 2>/dev/null)
    echo "Authentication: Configured (${AUTH_HEADER:0:20}...)"
  else
    echo "Authentication: None"
  fi
  echo ""
  
  # Step 1: Send MCP initialize request
  echo "Step 1: Sending MCP initialize request..."
  INIT_PAYLOAD='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"gh-aw-check","version":"1.0.0"}}}'
  
  echo "Request payload:"
  echo "$INIT_PAYLOAD" | jq '.' 2>/dev/null || echo "$INIT_PAYLOAD"
  echo ""
  
  # Make the request with proper headers (5 second timeout)
  echo "Sending HTTP POST to: $SERVER_URL"
  if [ -n "$AUTH_HEADER" ]; then
    INIT_RESPONSE=$(curl -s -w "\n%{http_code}" --max-time 5 -X POST "$SERVER_URL" \
      -H "Content-Type: application/json" \
      -H "Authorization: $AUTH_HEADER" \
      -d "$INIT_PAYLOAD" 2>&1 || echo -e "\n000")
  else
    INIT_RESPONSE=$(curl -s -w "\n%{http_code}" --max-time 5 -X POST "$SERVER_URL" \
      -H "Content-Type: application/json" \
      -d "$INIT_PAYLOAD" 2>&1 || echo -e "\n000")
  fi
  
  INIT_HTTP_CODE=$(echo "$INIT_RESPONSE" | tail -n 1)
  INIT_BODY=$(echo "$INIT_RESPONSE" | head -n -1)
  
  echo "HTTP Status: $INIT_HTTP_CODE"
  echo "Response:"
  echo "$INIT_BODY" | jq '.' 2>/dev/null || echo "$INIT_BODY"
  echo ""
  
  # Check if initialize succeeded
  if [ "$INIT_HTTP_CODE" != "200" ]; then
    echo "WARNING: Initialize request failed with HTTP $INIT_HTTP_CODE"
    echo "Server may require different authentication or be unavailable"
    SERVERS_FAILED=$((SERVERS_FAILED + 1))
    echo ""
    continue
  fi
  
  # Check for JSON-RPC error in response
  if echo "$INIT_BODY" | jq -e '.error' >/dev/null 2>&1; then
    echo "WARNING: Initialize request returned JSON-RPC error:"
    echo "$INIT_BODY" | jq '.error' 2>/dev/null
    SERVERS_FAILED=$((SERVERS_FAILED + 1))
    echo ""
    continue
  fi
  
  echo "✓ Initialize request succeeded"
  echo ""
  
  # Step 2: Send tools/list request
  echo "Step 2: Sending tools/list request..."
  TOOLS_PAYLOAD='{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
  
  echo "Request payload:"
  echo "$TOOLS_PAYLOAD" | jq '.' 2>/dev/null || echo "$TOOLS_PAYLOAD"
  echo ""
  
  echo "Sending HTTP POST to: $SERVER_URL"
  if [ -n "$AUTH_HEADER" ]; then
    TOOLS_RESPONSE=$(curl -s -w "\n%{http_code}" --max-time 5 -X POST "$SERVER_URL" \
      -H "Content-Type: application/json" \
      -H "Authorization: $AUTH_HEADER" \
      -d "$TOOLS_PAYLOAD" 2>&1 || echo -e "\n000")
  else
    TOOLS_RESPONSE=$(curl -s -w "\n%{http_code}" --max-time 5 -X POST "$SERVER_URL" \
      -H "Content-Type: application/json" \
      -d "$TOOLS_PAYLOAD" 2>&1 || echo -e "\n000")
  fi
  
  TOOLS_HTTP_CODE=$(echo "$TOOLS_RESPONSE" | tail -n 1)
  TOOLS_BODY=$(echo "$TOOLS_RESPONSE" | head -n -1)
  
  echo "HTTP Status: $TOOLS_HTTP_CODE"
  echo "Response:"
  echo "$TOOLS_BODY" | jq '.' 2>/dev/null || echo "$TOOLS_BODY"
  echo ""
  
  # Check if tools/list succeeded
  if [ "$TOOLS_HTTP_CODE" != "200" ]; then
    echo "WARNING: tools/list request failed with HTTP $TOOLS_HTTP_CODE"
    SERVERS_FAILED=$((SERVERS_FAILED + 1))
    echo ""
    continue
  fi
  
  # Check for JSON-RPC error in response
  if echo "$TOOLS_BODY" | jq -e '.error' >/dev/null 2>&1; then
    echo "WARNING: tools/list request returned JSON-RPC error:"
    echo "$TOOLS_BODY" | jq '.error' 2>/dev/null
    SERVERS_FAILED=$((SERVERS_FAILED + 1))
    echo ""
    continue
  fi
  
  echo "✓ tools/list request succeeded"
  echo ""
  
  # Step 3: Display available tools
  echo "Available tools from $SERVER_NAME:"
  echo "---"
  
  # Extract tools array and display
  if TOOLS_ARRAY=$(echo "$TOOLS_BODY" | jq -r '.result.tools[]?' 2>/dev/null); then
    if [ -n "$TOOLS_ARRAY" ]; then
      TOOL_COUNT=0
      while IFS= read -r TOOL; do
        TOOL_COUNT=$((TOOL_COUNT + 1))
        TOOL_NAME=$(echo "$TOOL" | jq -r '.name // "unknown"' 2>/dev/null)
        TOOL_DESC=$(echo "$TOOL" | jq -r '.description // "No description"' 2>/dev/null)
        
        echo "  [$TOOL_COUNT] $TOOL_NAME"
        echo "      Description: $TOOL_DESC"
        
        # Show input schema if available
        if echo "$TOOL" | jq -e '.inputSchema' >/dev/null 2>&1; then
          echo "      Input schema: $(echo "$TOOL" | jq -c '.inputSchema' 2>/dev/null)"
        fi
        echo ""
      done <<< "$(echo "$TOOLS_BODY" | jq -c '.result.tools[]' 2>/dev/null)"
      
      echo "Total tools available: $TOOL_COUNT"
    else
      echo "  No tools available from this server"
    fi
  else
    echo "  Could not parse tools array from response"
    SERVERS_FAILED=$((SERVERS_FAILED + 1))
    echo ""
    continue
  fi
  
  echo "---"
  echo ""
  
  SERVERS_SUCCEEDED=$((SERVERS_SUCCEEDED + 1))
  echo "✓ Server check completed successfully: $SERVER_NAME"
  echo ""
  
done <<< "$SERVER_NAMES"

# Print summary
echo "=========================================="
echo "MCP Server Check Summary"
echo "=========================================="
echo "Servers checked: $SERVERS_CHECKED"
echo "Servers succeeded: $SERVERS_SUCCEEDED"
echo "Servers failed/skipped: $SERVERS_FAILED"
echo ""

if [ $SERVERS_SUCCEEDED -gt 0 ]; then
  echo "✓ At least one server check succeeded"
else
  echo "⚠ No servers were successfully checked"
fi

# Always exit 0 since failures are warnings
exit 0
