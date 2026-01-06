#!/usr/bin/env bash
# Verify MCP Gateway (gh-aw-mcpg) Health
# This script verifies that the gh-aw-mcpg gateway is running and healthy

set -e

# Usage: verify_mcp_gateway_health.sh GATEWAY_URL MCP_CONFIG_PATH LOGS_FOLDER [SESSION_TOKEN]
#
# Arguments:
#   GATEWAY_URL      : The HTTP URL of the MCP gateway (e.g., http://localhost:80)
#   MCP_CONFIG_PATH  : Path to the MCP configuration file
#   LOGS_FOLDER      : Path to the gateway logs folder
#   SESSION_TOKEN    : Bearer token for MCP client auth (optional, defaults to awf-session)
#
# Exit codes:
#   0 - Gateway is healthy and ready
#   1 - Gateway failed to start or configuration is invalid

if [ "$#" -lt 3 ]; then
  echo "Usage: $0 GATEWAY_URL MCP_CONFIG_PATH LOGS_FOLDER [SESSION_TOKEN]" >&2
  exit 1
fi

gateway_url="$1"
mcp_config_path="$2"
logs_folder="$3"
session_token="${4:-awf-session}"

echo 'Verifying gh-aw-mcpg (MCP Gateway) health...'
echo ''
echo '=== File Locations ==='
echo "Gateway URL: $gateway_url"
echo "MCP Config Path: $mcp_config_path"
echo "Logs Folder: $logs_folder"
echo "Gateway Log: ${logs_folder}/gateway.log"
echo ''

# Check for gateway logs early
echo '=== Gateway Logs Check ==='
if [ -f "${logs_folder}/gateway.log" ]; then
  echo "✓ Gateway log file exists at: ${logs_folder}/gateway.log"
  echo "Log file size: $(stat -c%s "${logs_folder}/gateway.log" 2>/dev/null || stat -f%z "${logs_folder}/gateway.log" 2>/dev/null || echo 'unknown') bytes"
  echo "Last few lines of gateway log:"
  tail -10 "${logs_folder}/gateway.log" 2>/dev/null || echo "Could not read log tail"
else
  echo "⚠ Gateway log file NOT found at: ${logs_folder}/gateway.log"
fi
echo ''

# Check if gh-aw-mcpg container is running
echo '=== Docker Container Check ==='
if docker ps | grep -q gh-aw-mcpg; then
  echo "✓ gh-aw-mcpg container is running"
  docker ps --filter "name=gh-aw-mcpg" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
else
  echo "⚠ gh-aw-mcpg container not found in running containers"
  echo "Available containers:"
  docker ps --format "table {{.Names}}\t{{.Status}}"
fi
echo ''

# Wait for gateway to be ready FIRST before checking config
echo '=== Testing Gateway Health ==='
max_retries=30
retry_count=0
gateway_ready=false

while [ $retry_count -lt $max_retries ]; do
  if curl -s -o /dev/null -w "%{http_code}" "${gateway_url}/health" | grep -q "200\|204"; then
    echo "✓ MCP Gateway is ready!"
    gateway_ready=true
    break
  fi
  retry_count=$((retry_count + 1))
  echo "Waiting for gateway... (attempt $retry_count/$max_retries)"
  sleep 1
done

if [ "$gateway_ready" = false ]; then
  echo "✗ Error: MCP Gateway failed to start after $max_retries attempts"
  echo ''
  echo '=== Gateway Logs (Full) ==='
  cat "${logs_folder}/gateway.log" 2>/dev/null || echo 'No gateway logs found'
  echo ''
  echo '=== Docker Logs ==='
  docker logs gh-aw-mcpg 2>&1 || echo 'Could not get docker logs'
  exit 1
fi
echo ''

# Now that gateway is ready, check the config file
echo '=== MCP Configuration File ==='
if [ -f "$mcp_config_path" ]; then
  echo "✓ Config file exists at: $mcp_config_path"
  echo "File size: $(stat -c%s "$mcp_config_path" 2>/dev/null || stat -f%z "$mcp_config_path" 2>/dev/null || echo 'unknown') bytes"
  echo "Last modified: $(stat -c%y "$mcp_config_path" 2>/dev/null || stat -f%Sm "$mcp_config_path" 2>/dev/null || echo 'unknown')"
else
  echo "✗ Config file NOT found at: $mcp_config_path"
  exit 1
fi
echo ''

# Show MCP config file content
echo '=== MCP Configuration Content ==='
cat "$mcp_config_path" || { echo 'ERROR: Failed to read MCP config file'; exit 1; }
echo ''

# Verify required servers are present in config (if applicable)
echo '=== Verifying Required Servers ==='
# Check for safeinputs and safeoutputs if they're expected
if grep -q '"safeinputs"' "$mcp_config_path" 2>/dev/null; then
  echo '✓ safeinputs server found in configuration'
else
  echo '⚠ safeinputs server not found (may not be required)'
fi

if grep -q '"safeoutputs"' "$mcp_config_path" 2>/dev/null; then
  echo '✓ safeoutputs server found in configuration'
else
  echo '⚠ safeoutputs server not found (may not be required)'
fi
echo ''

# List registered MCP servers via sys endpoint
echo '=== Gateway MCP System Info ==='
echo "Querying: ${gateway_url}/mcp/sys"
sys_response=$(curl -sf "${gateway_url}/mcp/sys" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${session_token}" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_servers","arguments":{}}}' 2>/dev/null) || true

if [ -n "$sys_response" ]; then
  echo "Response:"
  echo "$sys_response" | jq -r '.result.content[0].text' 2>/dev/null || echo "$sys_response"
else
  echo "⚠ Could not query sys endpoint (this may be expected if sys server is not configured)"
fi
echo ''

# Test MCP server connectivity through gateway
echo '=== Testing MCP Server Connectivity ==='

# Extract first MCP server name from config (excluding safeinputs/safeoutputs)
mcp_server=$(jq -r '.mcpServers | to_entries[] | select(.key != "safeinputs" and .key != "safeoutputs") | .key' "$mcp_config_path" 2>/dev/null | head -n 1) || true

if [ -n "$mcp_server" ]; then
  echo "Testing connectivity to MCP server: $mcp_server"
  mcp_url="${gateway_url}/mcp/${mcp_server}"
  echo "MCP URL: $mcp_url"
  echo ''

  # Check if server config uses HTTP transport (gateway-proxied)
  echo "Checking '$mcp_server' configuration..."
  server_config=$(jq -r ".mcpServers.\"$mcp_server\"" "$mcp_config_path" 2>/dev/null) || true
  if [ -n "$server_config" ]; then
    echo "Server config:"
    echo "$server_config" | jq '.' 2>/dev/null || echo "$server_config"

    if echo "$server_config" | grep -q '"type": "http"' 2>/dev/null; then
      echo "✓ Server is configured for HTTP transport (gateway-proxied)"
    else
      echo "⚠ Server may use local transport (not gateway-proxied)"
    fi
  fi
  echo ''

  # Test with MCP initialize call
  echo "Sending MCP initialize request..."
  response=$(curl -s -w "\n%{http_code}" -X POST "$mcp_url" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${session_token}" \
    -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' 2>/dev/null) || true

  http_code=$(echo "$response" | tail -n 1)
  body=$(echo "$response" | head -n -1)

  echo "HTTP Status: $http_code"
  echo "Response: $body"
  echo ''

  if [ "$http_code" = "200" ]; then
    echo "✓ MCP server connectivity test passed"
  else
    echo "⚠ MCP server returned HTTP $http_code (may need authentication or different request)"
    echo ''
    echo "Gateway logs (last 20 lines):"
    tail -20 "${logs_folder}/gateway.log" 2>/dev/null || echo "Could not read gateway logs"
  fi
else
  echo "No external MCP servers configured for testing"
fi

echo ''
echo '=== Health Check Complete ==='
exit 0
