#!/usr/bin/env bash
# Test script for convert_gateway_config_claude.sh
# This script validates that the Claude MCP configuration converter correctly
# transforms both HTTP and stdio server configurations to Claude Code format.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_FAILED=0

echo "Testing Claude MCP Configuration Converter..."
echo ""

# Create test directory
mkdir -p /tmp/gh-aw/mcp-config

# Test 1: HTTP servers (post-gateway transformation)
echo "Test 1: HTTP servers (gateway-transformed format)"
echo "=================================================="
cat > /tmp/test-http-gateway.json << 'EOF'
{
  "mcpServers": {
    "github": {
      "type": "http",
      "url": "http://localhost:8080/mcp/github",
      "headers": {
        "Authorization": "Bearer placeholder"
      },
      "tools": ["*"]
    },
    "playwright": {
      "type": "http",
      "url": "http://localhost:8080/mcp/playwright",
      "headers": {
        "Authorization": "Bearer placeholder"
      }
    }
  }
}
EOF

export MCP_GATEWAY_OUTPUT=/tmp/test-http-gateway.json
export MCP_GATEWAY_API_KEY="test-api-key-123"

if bash "$SCRIPT_DIR/convert_gateway_config_claude.sh" > /dev/null 2>&1; then
  echo "✓ Conversion succeeded"
  
  # Validate HTTP format
  if jq -e '.mcpServers.github.url' /tmp/gh-aw/mcp-config/mcp-servers.json >/dev/null 2>&1; then
    echo "✓ Has url field"
  else
    echo "✗ Missing url field"
    TEST_FAILED=1
  fi
  
  if ! jq -e '.mcpServers.github.type' /tmp/gh-aw/mcp-config/mcp-servers.json >/dev/null 2>&1; then
    echo "✓ No type field (removed)"
  else
    echo "✗ Has type field (should be removed)"
    TEST_FAILED=1
  fi
  
  if ! jq -e '.mcpServers.github.tools' /tmp/gh-aw/mcp-config/mcp-servers.json >/dev/null 2>&1; then
    echo "✓ No tools field (removed)"
  else
    echo "✗ Has tools field (should be removed)"
    TEST_FAILED=1
  fi
  
  AUTH_VALUE=$(jq -r '.mcpServers.github.headers.Authorization' /tmp/gh-aw/mcp-config/mcp-servers.json)
  if [ "$AUTH_VALUE" = "test-api-key-123" ]; then
    echo "✓ Authorization header updated"
  else
    echo "✗ Authorization header not updated (got: $AUTH_VALUE)"
    TEST_FAILED=1
  fi
else
  echo "✗ Conversion failed"
  TEST_FAILED=1
fi

echo ""

# Test 2: Stdio servers (gateway-spec format)
echo "Test 2: Stdio servers (gateway-spec format)"
echo "============================================"
cat > /tmp/test-stdio-gateway.json << 'EOF'
{
  "mcpServers": {
    "github": {
      "type": "stdio",
      "container": "ghcr.io/github/github-mcp-server:latest",
      "entrypointArgs": ["--toolsets", "default"],
      "mounts": ["/tmp/gh-aw:/tmp/gh-aw:rw"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "$GITHUB_MCP_SERVER_TOKEN",
        "GITHUB_TOOLSETS": "default"
      }
    },
    "playwright": {
      "type": "stdio",
      "container": "mcr.microsoft.com/playwright/mcp",
      "args": ["--init", "--network", "host"],
      "entrypointArgs": ["--output-dir", "/tmp/gh-aw/mcp-logs/playwright"],
      "mounts": ["/tmp/gh-aw/mcp-logs:/tmp/gh-aw/mcp-logs:rw"]
    },
    "safeoutputs": {
      "type": "stdio",
      "container": "node:lts-alpine",
      "entrypoint": "node",
      "entrypointArgs": ["/opt/gh-aw/safeoutputs/mcp-server.cjs"],
      "mounts": ["/opt/gh-aw:/opt/gh-aw:ro", "/tmp/gh-aw:/tmp/gh-aw:rw"],
      "env": {
        "GH_AW_SAFE_OUTPUTS": "/tmp/gh-aw/safeoutputs/outputs.jsonl"
      }
    }
  }
}
EOF

export MCP_GATEWAY_OUTPUT=/tmp/test-stdio-gateway.json
export MCP_GATEWAY_API_KEY="test-api-key-123"

if bash "$SCRIPT_DIR/convert_gateway_config_claude.sh" > /dev/null 2>&1; then
  echo "✓ Conversion succeeded"
  
  # Validate stdio format
  if jq -e '.mcpServers.github.command' /tmp/gh-aw/mcp-config/mcp-servers.json >/dev/null 2>&1; then
    CMD=$(jq -r '.mcpServers.github.command' /tmp/gh-aw/mcp-config/mcp-servers.json)
    if [ "$CMD" = "docker" ]; then
      echo "✓ Has docker command"
    else
      echo "✗ Wrong command (got: $CMD)"
      TEST_FAILED=1
    fi
  else
    echo "✗ Missing command field"
    TEST_FAILED=1
  fi
  
  if jq -e '.mcpServers.github.args' /tmp/gh-aw/mcp-config/mcp-servers.json >/dev/null 2>&1; then
    echo "✓ Has args field"
    
    # Check args contains key elements
    ARGS=$(jq -r '.mcpServers.github.args | join(" ")' /tmp/gh-aw/mcp-config/mcp-servers.json)
    if echo "$ARGS" | grep -q "run --rm -i"; then
      echo "  ✓ Args contain docker run flags"
    else
      echo "  ✗ Args missing docker run flags"
      TEST_FAILED=1
    fi
    
    if echo "$ARGS" | grep -q "ghcr.io/github/github-mcp-server:latest"; then
      echo "  ✓ Args contain container image"
    else
      echo "  ✗ Args missing container image"
      TEST_FAILED=1
    fi
    
    if echo "$ARGS" | grep -q "\-v /tmp/gh-aw:/tmp/gh-aw:rw"; then
      echo "  ✓ Args contain volume mount"
    else
      echo "  ✗ Args missing volume mount"
      TEST_FAILED=1
    fi
  else
    echo "✗ Missing args field"
    TEST_FAILED=1
  fi
  
  if ! jq -e '.mcpServers.github.type' /tmp/gh-aw/mcp-config/mcp-servers.json >/dev/null 2>&1; then
    echo "✓ No type field (removed)"
  else
    echo "✗ Has type field (should be removed)"
    TEST_FAILED=1
  fi
  
  if ! jq -e '.mcpServers.github.container' /tmp/gh-aw/mcp-config/mcp-servers.json >/dev/null 2>&1; then
    echo "✓ No container field (removed)"
  else
    echo "✗ Has container field (should be removed)"
    TEST_FAILED=1
  fi
  
  if ! jq -e '.mcpServers.github.entrypointArgs' /tmp/gh-aw/mcp-config/mcp-servers.json >/dev/null 2>&1; then
    echo "✓ No entrypointArgs field (removed)"
  else
    echo "✗ Has entrypointArgs field (should be removed)"
    TEST_FAILED=1
  fi
  
  if ! jq -e '.mcpServers.github.mounts' /tmp/gh-aw/mcp-config/mcp-servers.json >/dev/null 2>&1; then
    echo "✓ No mounts field (removed)"
  else
    echo "✗ Has mounts field (should be removed)"
    TEST_FAILED=1
  fi
  
  if jq -e '.mcpServers.github.env' /tmp/gh-aw/mcp-config/mcp-servers.json >/dev/null 2>&1; then
    echo "✓ Has env field (preserved)"
  else
    echo "✗ Missing env field"
    TEST_FAILED=1
  fi
  
  # Test entrypoint override handling
  if jq -e '.mcpServers.safeoutputs.args | index("--entrypoint")' /tmp/gh-aw/mcp-config/mcp-servers.json >/dev/null 2>&1; then
    echo "✓ Entrypoint override converted to --entrypoint flag"
  else
    echo "✗ Entrypoint override not converted"
    TEST_FAILED=1
  fi
else
  echo "✗ Conversion failed"
  TEST_FAILED=1
fi

echo ""

# Summary
if [ $TEST_FAILED -eq 0 ]; then
  echo "=========================================="
  echo "All tests passed! ✓"
  echo "=========================================="
  exit 0
else
  echo "=========================================="
  echo "Some tests failed! ✗"
  echo "=========================================="
  exit 1
fi
