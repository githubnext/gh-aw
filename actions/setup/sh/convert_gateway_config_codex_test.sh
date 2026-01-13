#!/usr/bin/env bash
# Test script for convert_gateway_config_codex.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_PATH="$SCRIPT_DIR/convert_gateway_config_codex.sh"

# Setup test directory
TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

echo "=== Testing convert_gateway_config_codex.sh ==="
echo ""

# Set required environment variables for all tests
export MCP_GATEWAY_DOMAIN="host.docker.internal"
export MCP_GATEWAY_PORT="80"

# Create the output directory the script writes to
mkdir -p /tmp/gh-aw/mcp-config

# Test 1: Basic gateway output conversion
echo "Test 1: Convert basic gateway output with Authorization headers"
mkdir -p "$TEST_DIR/mcp-config"

cat > "$TEST_DIR/gateway-output.json" << 'EOF'
{
  "mcpServers": {
    "github": {
      "type": "http",
      "url": "http://localhost:8080/mcp/github",
      "headers": {
        "Authorization": "test-api-key-abc123"
      }
    },
    "playwright": {
      "type": "http",
      "url": "http://localhost:8080/mcp/playwright",
      "headers": {
        "Authorization": "test-api-key-xyz789"
      }
    }
  }
}
EOF

export MCP_GATEWAY_OUTPUT="$TEST_DIR/gateway-output.json"
bash "$SCRIPT_PATH" > /dev/null 2>&1

# Check that config.toml was created
if [ ! -f /tmp/gh-aw/mcp-config/config.toml ]; then
  echo "✗ FAIL: config.toml not created"
  exit 1
fi

# Check for correct TOML format with http_headers inline table
if ! grep -q 'http_headers = { Authorization = "test-api-key-abc123" }' /tmp/gh-aw/mcp-config/config.toml; then
  echo "✗ FAIL: Missing or incorrect http_headers for github"
  cat /tmp/gh-aw/mcp-config/config.toml
  exit 1
fi

if ! grep -q 'http_headers = { Authorization = "test-api-key-xyz789" }' /tmp/gh-aw/mcp-config/config.toml; then
  echo "✗ FAIL: Missing or incorrect http_headers for playwright"
  cat /tmp/gh-aw/mcp-config/config.toml
  exit 1
fi

# Check that it does NOT use the old [mcp_servers.*.headers] format
if grep -q '^\[mcp_servers\..*\.headers\]' /tmp/gh-aw/mcp-config/config.toml; then
  echo "✗ FAIL: Found old-style [mcp_servers.*.headers] section (should use http_headers inline table)"
  cat /tmp/gh-aw/mcp-config/config.toml
  exit 1
fi

# Check for history section
if ! grep -q '\[history\]' /tmp/gh-aw/mcp-config/config.toml; then
  echo "✗ FAIL: Missing [history] section"
  exit 1
fi

if ! grep -q 'persistence = "none"' /tmp/gh-aw/mcp-config/config.toml; then
  echo "✗ FAIL: Missing persistence = \"none\" in history section"
  exit 1
fi

# Check that URLs are correctly formatted with domain and port
if ! grep -q 'url = "http://host.docker.internal:80/mcp/github"' /tmp/gh-aw/mcp-config/config.toml; then
  echo "✗ FAIL: URL not correctly formatted for github server"
  cat /tmp/gh-aw/mcp-config/config.toml
  exit 1
fi

echo "✓ PASS: Basic gateway output conversion"
echo ""

# Test 2: Validate TOML syntax
echo "Test 2: Validate generated TOML syntax"
if command -v python3 &> /dev/null; then
  if python3 -c "import tomllib; tomllib.loads(open('/tmp/gh-aw/mcp-config/config.toml').read())" 2>&1 || \
     python3 -c "import tomli; tomli.loads(open('/tmp/gh-aw/mcp-config/config.toml').read())" 2>&1; then
    echo "✓ PASS: Generated TOML is valid"
  else
    echo "✗ FAIL: Generated TOML is invalid"
    cat /tmp/gh-aw/mcp-config/config.toml
    exit 1
  fi
else
  echo "⊘ SKIP: Python not available for TOML validation"
fi
echo ""

# Test 3: Error handling - missing MCP_GATEWAY_OUTPUT
echo "Test 3: Error handling for missing MCP_GATEWAY_OUTPUT"
unset MCP_GATEWAY_OUTPUT
if bash "$SCRIPT_PATH" > /dev/null 2>&1; then
  echo "✗ FAIL: Script should fail when MCP_GATEWAY_OUTPUT is not set"
  exit 1
else
  echo "✓ PASS: Script correctly fails when MCP_GATEWAY_OUTPUT is not set"
fi
echo ""

# Test 4: Error handling - missing gateway output file
echo "Test 4: Error handling for missing gateway output file"
export MCP_GATEWAY_OUTPUT="$TEST_DIR/nonexistent.json"
if bash "$SCRIPT_PATH" > /dev/null 2>&1; then
  echo "✗ FAIL: Script should fail when gateway output file doesn't exist"
  exit 1
else
  echo "✓ PASS: Script correctly fails when gateway output file doesn't exist"
fi
echo ""

# Test 5: Error handling - missing MCP_GATEWAY_DOMAIN
echo "Test 5: Error handling for missing MCP_GATEWAY_DOMAIN"
export MCP_GATEWAY_OUTPUT="$TEST_DIR/gateway-output.json"
unset MCP_GATEWAY_DOMAIN
if bash "$SCRIPT_PATH" > /dev/null 2>&1; then
  echo "✗ FAIL: Script should fail when MCP_GATEWAY_DOMAIN is not set"
  exit 1
else
  echo "✓ PASS: Script correctly fails when MCP_GATEWAY_DOMAIN is not set"
fi
export MCP_GATEWAY_DOMAIN="host.docker.internal"
echo ""

# Test 6: Error handling - missing MCP_GATEWAY_PORT
echo "Test 6: Error handling for missing MCP_GATEWAY_PORT"
unset MCP_GATEWAY_PORT
if bash "$SCRIPT_PATH" > /dev/null 2>&1; then
  echo "✗ FAIL: Script should fail when MCP_GATEWAY_PORT is not set"
  exit 1
else
  echo "✓ PASS: Script correctly fails when MCP_GATEWAY_PORT is not set"
fi
export MCP_GATEWAY_PORT="80"
echo ""

echo "=== All tests passed ==="
