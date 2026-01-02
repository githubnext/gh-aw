#!/usr/bin/env bash
# Verify MCP Gateway Health
# This script verifies that the MCP gateway is running and healthy

set -e

# Usage: verify_mcp_gateway_health.sh GATEWAY_URL MCP_CONFIG_PATH LOGS_FOLDER
#
# Arguments:
#   GATEWAY_URL      : The HTTP URL of the MCP gateway (e.g., http://localhost:8080)
#   MCP_CONFIG_PATH  : Path to the MCP configuration file
#   LOGS_FOLDER      : Path to the gateway logs folder
#
# Exit codes:
#   0 - Gateway is healthy and ready
#   1 - Gateway failed to start or configuration is invalid

if [ "$#" -ne 3 ]; then
  echo "Usage: $0 GATEWAY_URL MCP_CONFIG_PATH LOGS_FOLDER" >&2
  exit 1
fi

gateway_url="$1"
mcp_config_path="$2"
logs_folder="$3"

echo 'Waiting for MCP Gateway to be ready...'

# Show MCP config file content
echo 'MCP Configuration:'
cat "$mcp_config_path" || echo 'No MCP config file found'
echo ''

# Verify safeinputs and safeoutputs are present in config
if ! grep -q '"safeinputs"' "$mcp_config_path"; then
  echo 'ERROR: safeinputs server not found in MCP configuration'
  exit 1
fi
if ! grep -q '"safeoutputs"' "$mcp_config_path"; then
  echo 'ERROR: safeoutputs server not found in MCP configuration'
  exit 1
fi
echo 'Verified: safeinputs and safeoutputs are present in configuration'

max_retries=30
retry_count=0
while [ $retry_count -lt $max_retries ]; do
  if curl -s -o /dev/null -w "%{http_code}" "${gateway_url}/health" | grep -q "200\|204"; then
    echo "MCP Gateway is ready!"
    curl -s "${gateway_url}/servers" || echo "Could not fetch servers list"
    
    # Test MCP server connectivity through gateway
    echo ''
    echo 'Testing MCP server connectivity...'
    
    # Extract first external MCP server name from config (excluding safeinputs/safeoutputs)
    mcp_server=$(jq -r '.mcpServers | to_entries[] | select(.key != "safeinputs" and .key != "safeoutputs") | .key' "$mcp_config_path" | head -n 1)
    if [ -n "$mcp_server" ]; then
      echo "Testing connectivity to MCP server: $mcp_server"
      mcp_url="${gateway_url}/mcp/${mcp_server}"
      echo "MCP URL: $mcp_url"
      
      # Test with MCP initialize call
      response=$(curl -s -w "\n%{http_code}" -X POST "$mcp_url" \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}')
      
      http_code=$(echo "$response" | tail -n 1)
      body=$(echo "$response" | head -n -1)
      
      echo "HTTP Status: $http_code"
      echo "Response: $body"
      
      if [ "$http_code" = "200" ]; then
        echo "✓ MCP server connectivity test passed"
      else
        echo "⚠ MCP server returned HTTP $http_code (may need authentication or different request)"
      fi
    else
      echo "No external MCP servers configured for testing"
    fi
    
    exit 0
  fi
  retry_count=$((retry_count + 1))
  echo "Waiting for gateway... (attempt $retry_count/$max_retries)"
  sleep 1
done
echo "Error: MCP Gateway failed to start after $max_retries attempts"

# Show gateway logs for debugging
echo 'Gateway logs:'
cat "${logs_folder}/gateway.log" || echo 'No gateway logs found'
exit 1
