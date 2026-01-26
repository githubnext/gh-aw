#!/usr/bin/env bash
# Check GitHub Remote MCP Toolsets Availability
# This script verifies that the GitHub Remote MCP server has the required toolsets loaded
# and the tools are actually available for use
#
# Exit codes:
#   0 - GitHub Remote MCP toolsets are available and working
#   1 - GitHub Remote MCP toolsets are not available or server is not accessible
#   2 - GitHub tool is not configured or not using remote mode (skip check)

set -e

# Usage: check_github_remote_mcp_toolsets.sh GATEWAY_CONFIG_PATH GATEWAY_URL GATEWAY_API_KEY
#
# Arguments:
#   GATEWAY_CONFIG_PATH : Path to the gateway output configuration file (gateway-output.json)
#   GATEWAY_URL         : The HTTP URL of the MCP gateway (e.g., http://localhost:8080)
#   GATEWAY_API_KEY     : API key for gateway authentication

if [ "$#" -ne 3 ]; then
  echo "Usage: $0 GATEWAY_CONFIG_PATH GATEWAY_URL GATEWAY_API_KEY" >&2
  exit 1
fi

GATEWAY_CONFIG_PATH="$1"
GATEWAY_URL="$2"
GATEWAY_API_KEY="$3"

echo "Checking GitHub Remote MCP toolsets availability..."
echo ""

# Validate configuration file exists
if [ ! -f "$GATEWAY_CONFIG_PATH" ]; then
  echo "ERROR: Gateway configuration file not found: $GATEWAY_CONFIG_PATH" >&2
  exit 1
fi

# Check if GitHub MCP server is configured
if ! jq -e '.mcpServers.github' "$GATEWAY_CONFIG_PATH" >/dev/null 2>&1; then
  echo "GitHub MCP server not configured in gateway, skipping check"
  exit 2
fi

# Extract GitHub MCP server configuration
GITHUB_CONFIG=$(jq -r '.mcpServers.github' "$GATEWAY_CONFIG_PATH")

# Check if it's using remote mode (HTTP URL)
GITHUB_URL=$(echo "$GITHUB_CONFIG" | jq -r '.url // empty')
if [ -z "$GITHUB_URL" ] || [ "$GITHUB_URL" = "null" ]; then
  echo "GitHub MCP is not using HTTP (likely local mode), skipping remote toolset check"
  exit 2
fi

# Check if URL points to remote server
if ! echo "$GITHUB_URL" | grep -q "githubcopilot.com"; then
  echo "GitHub MCP is not using remote mode (URL: $GITHUB_URL), skipping check"
  exit 2
fi

echo "GitHub Remote MCP detected at: $GITHUB_URL"
echo ""

# Extract expected toolsets from headers (X-MCP-Toolsets)
EXPECTED_TOOLSETS=$(echo "$GITHUB_CONFIG" | jq -r '.headers["X-MCP-Toolsets"] // empty')
if [ -z "$EXPECTED_TOOLSETS" ] || [ "$EXPECTED_TOOLSETS" = "null" ]; then
  echo "No toolsets specified in configuration, using default verification"
  EXPECTED_TOOLSETS="repos,issues,discussions"
fi

echo "Expected toolsets: $EXPECTED_TOOLSETS"
echo ""

# Extract tools array from configuration (if present)
ALLOWED_TOOLS=$(echo "$GITHUB_CONFIG" | jq -r '.tools // [] | join(",")')
if [ -n "$ALLOWED_TOOLS" ] && [ "$ALLOWED_TOOLS" != "null" ]; then
  echo "Allowed tools configured: $ALLOWED_TOOLS"
  echo ""
fi

# Test GitHub MCP server connectivity and tool availability
# Send MCP tools/list request to verify tools are actually available
TOOLS_LIST_PAYLOAD='{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

echo "Verifying GitHub MCP tools are available..."
echo "Sending tools/list request to gateway..."

# Make the request with authentication
AUTH_HEADER=$(echo "$GITHUB_CONFIG" | jq -r '.headers.Authorization // empty')

if [ -n "$AUTH_HEADER" ] && [ "$AUTH_HEADER" != "null" ]; then
  TOOLS_RESPONSE=$(curl -s -w "\n%{http_code}" --max-time 30 -X POST "$GITHUB_URL" \
    -H "Content-Type: application/json" \
    -H "Authorization: $AUTH_HEADER" \
    -d "$TOOLS_LIST_PAYLOAD" 2>&1 || echo -e "\n000")
else
  echo "ERROR: No Authorization header found in GitHub MCP configuration" >&2
  echo "GitHub Remote MCP requires authentication" >&2
  exit 1
fi

TOOLS_HTTP_CODE=$(echo "$TOOLS_RESPONSE" | tail -n 1)
TOOLS_BODY=$(echo "$TOOLS_RESPONSE" | head -n -1)

echo "HTTP Status: $TOOLS_HTTP_CODE"

# Check HTTP response code
if [ "$TOOLS_HTTP_CODE" != "200" ]; then
  echo ""
  echo "ERROR: GitHub Remote MCP server returned HTTP $TOOLS_HTTP_CODE" >&2
  echo "Response: $TOOLS_BODY" >&2
  echo "" >&2
  echo "This indicates that the GitHub Remote MCP server is not accessible or authentication failed." >&2
  echo "" >&2
  echo "Common causes:" >&2
  echo "  - GitHub Copilot MCP service (https://api.githubcopilot.com/mcp/) is unavailable" >&2
  echo "  - Authentication token is invalid or expired" >&2
  echo "  - Network connectivity issues" >&2
  echo "" >&2
  exit 1
fi

# Check for JSON-RPC error in response
if echo "$TOOLS_BODY" | jq -e '.error' >/dev/null 2>&1; then
  ERROR_MESSAGE=$(echo "$TOOLS_BODY" | jq -r '.error.message // .error' 2>/dev/null)
  echo ""
  echo "ERROR: GitHub Remote MCP returned JSON-RPC error: $ERROR_MESSAGE" >&2
  echo "" >&2
  echo "This indicates that the GitHub Remote MCP server is accessible but" >&2
  echo "encountered an error when listing available tools." >&2
  echo "" >&2
  exit 1
fi

# Extract tools list from response
if ! echo "$TOOLS_BODY" | jq -e '.result.tools' >/dev/null 2>&1; then
  echo ""
  echo "ERROR: GitHub Remote MCP response does not contain tools list" >&2
  echo "Response: $TOOLS_BODY" >&2
  echo "" >&2
  echo "This indicates that the GitHub Remote MCP server did not return" >&2
  echo "the expected tools/list response format." >&2
  echo "" >&2
  exit 1
fi

TOOLS_LIST=$(echo "$TOOLS_BODY" | jq -r '.result.tools[] | .name' 2>/dev/null || echo "")

if [ -z "$TOOLS_LIST" ]; then
  echo ""
  echo "ERROR: GitHub Remote MCP returned empty tools list" >&2
  echo "" >&2
  echo "This indicates that the GitHub Remote MCP server is accessible but" >&2
  echo "no toolsets are currently loaded in the runner environment." >&2
  echo "" >&2
  echo "Root Cause:" >&2
  echo "  The GitHub MCP toolsets (repos, issues, discussions, etc.) are not available." >&2
  echo "  This is not an authentication issue - the server is reachable but the" >&2
  echo "  tools themselves are not loaded." >&2
  echo "" >&2
  echo "Remediation:" >&2
  echo "  1. Verify GitHub Copilot MCP service status" >&2
  echo "  2. Check if the toolsets feature is enabled for your GitHub organization" >&2
  echo "  3. Consider using local mode (mode: local) as a fallback" >&2
  echo "  4. Contact GitHub support if the issue persists" >&2
  echo "" >&2
  exit 1
fi

# Count available tools
TOOLS_COUNT=$(echo "$TOOLS_LIST" | wc -l | tr -d ' ')

echo ""
echo "✓ GitHub Remote MCP is working correctly"
echo "✓ $TOOLS_COUNT tools are available"
echo ""
echo "Available tools:"
echo "$TOOLS_LIST" | sed 's/^/  - /'
echo ""

# Verify critical tools are available if allowed tools are specified
if [ -n "$ALLOWED_TOOLS" ] && [ "$ALLOWED_TOOLS" != "null" ]; then
  echo "Verifying allowed tools are available..."
  MISSING_TOOLS=""
  
  # Split comma-separated allowed tools
  IFS=',' read -ra TOOLS_ARRAY <<< "$ALLOWED_TOOLS"
  for TOOL in "${TOOLS_ARRAY[@]}"; do
    TOOL=$(echo "$TOOL" | tr -d ' ')
    if ! echo "$TOOLS_LIST" | grep -q "^${TOOL}$"; then
      MISSING_TOOLS="${MISSING_TOOLS}${TOOL},"
    fi
  done
  
  if [ -n "$MISSING_TOOLS" ]; then
    # Remove trailing comma
    MISSING_TOOLS="${MISSING_TOOLS%,}"
    echo ""
    echo "WARNING: Some allowed tools are not available: $MISSING_TOOLS" >&2
    echo "This may cause the agent to fail when trying to use these tools." >&2
    echo ""
    # Don't fail - just warn. The agent will handle missing tools via safe-outputs
  else
    echo "✓ All allowed tools are available"
    echo ""
  fi
fi

echo "GitHub Remote MCP toolsets check PASSED"
exit 0
