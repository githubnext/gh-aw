#!/usr/bin/env bash
# Check MCP Server Functionality
# This script performs basic functionality checks on MCP servers configured by the MCP gateway
# It connects to each server, retrieves tools list, and displays available tools
#
# Resilience Features:
# - Progressive timeout: 10s, 20s, 30s across retry attempts
# - Progressive delay: 2s, 4s between retry attempts
# - Up to 3 retry attempts per server request (initialize and tools/list)
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
#   0 - All MCP server checks succeeded (some may be skipped if not HTTP-based)
#   1 - Invalid arguments, configuration file issues, or any MCP server failed to connect

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
SERVERS_SKIPPED=0

# Retry configuration for slow-starting servers
# Gateway may take 40-50 seconds to start all MCP servers (per start_mcp_gateway.sh)
MAX_RETRIES=3

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
    SERVERS_SKIPPED=$((SERVERS_SKIPPED + 1))
    echo ""
    continue
  fi
  
  echo "Server URL: $SERVER_URL"
  
  # Extract authentication headers from gateway configuration
  # Per MCP Gateway Specification v1.2.0 section 5.4:
  # "The gateway is responsible for generating and including appropriate 
  # authentication credentials - client-side configuration converters MUST NOT 
  # modify or supplement the headers provided by the gateway."
  AUTH_HEADER=""
  if echo "$SERVER_CONFIG" | jq -e '.headers.Authorization' >/dev/null 2>&1; then
    AUTH_HEADER=$(echo "$SERVER_CONFIG" | jq -r '.headers.Authorization' 2>/dev/null)
    echo "Authentication: From gateway config (${AUTH_HEADER:0:20}...)"
  else
    echo "WARNING: No Authorization header in gateway configuration for: $SERVER_NAME"
    echo "The gateway should have included authentication headers in its output."
    echo "Skipping authentication check..."
  fi
  echo ""
  
  # Step 1: Send MCP initialize request with retry logic
  echo "Step 1: Sending MCP initialize request..."
  INIT_PAYLOAD='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"gh-aw-check","version":"1.0.0"}}}'
  
  echo "Request payload:"
  echo "$INIT_PAYLOAD" | jq '.' 2>/dev/null || echo "$INIT_PAYLOAD"
  echo ""
  
  # Retry logic for slow-starting servers
  RETRY_COUNT=0
  INIT_SUCCESS=false
  
  while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    # Calculate timeout based on retry attempt (10s, 20s, 30s)
    TIMEOUT=$((10 + RETRY_COUNT * 10))
    
    if [ $RETRY_COUNT -gt 0 ]; then
      echo "Attempt $((RETRY_COUNT + 1)) of $MAX_RETRIES (with ${TIMEOUT}s timeout)..."
      # Progressive delay between retries (2s, 4s)
      DELAY=$((2 * RETRY_COUNT))
      echo "Waiting ${DELAY}s before retry..."
      sleep $DELAY
    fi
    
    # Make the request with proper headers and progressive timeout
    echo "Sending HTTP POST to: $SERVER_URL (timeout: ${TIMEOUT}s)"
    if [ -n "$AUTH_HEADER" ]; then
      INIT_RESPONSE=$(curl -s -w "\n%{http_code}" --max-time $TIMEOUT -X POST "$SERVER_URL" \
        -H "Content-Type: application/json" \
        -H "Authorization: $AUTH_HEADER" \
        -d "$INIT_PAYLOAD" 2>&1 || echo -e "\n000")
    else
      INIT_RESPONSE=$(curl -s -w "\n%{http_code}" --max-time $TIMEOUT -X POST "$SERVER_URL" \
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
    if [ "$INIT_HTTP_CODE" = "200" ]; then
      # Check for JSON-RPC error in response
      if ! echo "$INIT_BODY" | jq -e '.error' >/dev/null 2>&1; then
        INIT_SUCCESS=true
        break
      else
        echo "WARNING: Initialize request returned JSON-RPC error:"
        echo "$INIT_BODY" | jq '.error' 2>/dev/null
      fi
    else
      echo "WARNING: Initialize request failed with HTTP $INIT_HTTP_CODE"
      echo "Server may still be initializing or require different authentication"
    fi
    
    RETRY_COUNT=$((RETRY_COUNT + 1))
  done
  
  if [ "$INIT_SUCCESS" = false ]; then
    echo "WARNING: Initialize request failed after $MAX_RETRIES attempts"
    echo "Server may require different authentication or be unavailable"
    SERVERS_FAILED=$((SERVERS_FAILED + 1))
    echo ""
    continue
  fi
  
  echo "✓ Initialize request succeeded"
  echo ""
  
  # Step 2: Send tools/list request with retry logic
  echo "Step 2: Sending tools/list request..."
  TOOLS_PAYLOAD='{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
  
  echo "Request payload:"
  echo "$TOOLS_PAYLOAD" | jq '.' 2>/dev/null || echo "$TOOLS_PAYLOAD"
  echo ""
  
  # Retry logic for slow-starting servers
  RETRY_COUNT=0
  TOOLS_SUCCESS=false
  
  while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    # Calculate timeout based on retry attempt (10s, 20s, 30s)
    TIMEOUT=$((10 + RETRY_COUNT * 10))
    
    if [ $RETRY_COUNT -gt 0 ]; then
      echo "Attempt $((RETRY_COUNT + 1)) of $MAX_RETRIES (with ${TIMEOUT}s timeout)..."
      # Progressive delay between retries (2s, 4s)
      DELAY=$((2 * RETRY_COUNT))
      echo "Waiting ${DELAY}s before retry..."
      sleep $DELAY
    fi
    
    echo "Sending HTTP POST to: $SERVER_URL (timeout: ${TIMEOUT}s)"
    if [ -n "$AUTH_HEADER" ]; then
      TOOLS_RESPONSE=$(curl -s -w "\n%{http_code}" --max-time $TIMEOUT -X POST "$SERVER_URL" \
        -H "Content-Type: application/json" \
        -H "Authorization: $AUTH_HEADER" \
        -d "$TOOLS_PAYLOAD" 2>&1 || echo -e "\n000")
    else
      TOOLS_RESPONSE=$(curl -s -w "\n%{http_code}" --max-time $TIMEOUT -X POST "$SERVER_URL" \
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
    if [ "$TOOLS_HTTP_CODE" = "200" ]; then
      # Check for JSON-RPC error in response
      if ! echo "$TOOLS_BODY" | jq -e '.error' >/dev/null 2>&1; then
        TOOLS_SUCCESS=true
        break
      else
        echo "WARNING: tools/list request returned JSON-RPC error:"
        echo "$TOOLS_BODY" | jq '.error' 2>/dev/null
      fi
    else
      echo "WARNING: tools/list request failed with HTTP $TOOLS_HTTP_CODE"
    fi
    
    RETRY_COUNT=$((RETRY_COUNT + 1))
  done
  
  if [ "$TOOLS_SUCCESS" = false ]; then
    echo "WARNING: tools/list request failed after $MAX_RETRIES attempts"
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
echo "Servers failed: $SERVERS_FAILED"
echo "Servers skipped: $SERVERS_SKIPPED"
echo ""

if [ $SERVERS_FAILED -gt 0 ]; then
  echo "ERROR: One or more MCP servers failed to respond"
  echo "Failed servers: $SERVERS_FAILED"
  exit 1
elif [ $SERVERS_SUCCEEDED -eq 0 ]; then
  echo "ERROR: No HTTP servers were successfully checked"
  exit 1
else
  echo "✓ All HTTP server checks succeeded ($SERVERS_SUCCEEDED succeeded, $SERVERS_SKIPPED skipped)"
  exit 0
fi
