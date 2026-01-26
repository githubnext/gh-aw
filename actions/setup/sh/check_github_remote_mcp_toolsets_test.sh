#!/usr/bin/env bash
# Test script for check_github_remote_mcp_toolsets.sh
set -e

TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

echo "Running check_github_remote_mcp_toolsets.sh tests..."
echo ""

# Test 1: Missing arguments
echo "Test 1: Missing arguments"
if bash actions/setup/sh/check_github_remote_mcp_toolsets.sh 2>&1 | grep -q "Usage:"; then
  echo "✓ Test 1 passed: Correctly rejects missing arguments"
else
  echo "✗ Test 1 failed: Should reject missing arguments"
  exit 1
fi
echo ""

# Test 2: Non-existent config file
echo "Test 2: Non-existent config file"
if bash actions/setup/sh/check_github_remote_mcp_toolsets.sh /nonexistent/config.json http://localhost:8080 test-key 2>&1 | grep -q "ERROR: Gateway configuration file not found"; then
  echo "✓ Test 2 passed: Correctly detects missing config file"
else
  echo "✗ Test 2 failed: Should detect missing config file"
  exit 1
fi
echo ""

# Test 3: Config without GitHub MCP server
echo "Test 3: Config without GitHub MCP server"
cat > $TEST_DIR/config-no-github.json <<EOF
{
  "mcpServers": {
    "safeoutputs": {
      "type": "http",
      "url": "http://localhost:3000"
    }
  }
}
EOF

if bash actions/setup/sh/check_github_remote_mcp_toolsets.sh $TEST_DIR/config-no-github.json http://localhost:8080 test-key 2>&1 | grep -q "GitHub MCP server not configured"; then
  echo "✓ Test 3 passed: Skips check when GitHub MCP not configured"
else
  echo "✗ Test 3 failed: Should skip check when GitHub MCP not configured"
  exit 1
fi
echo ""

# Test 4: GitHub MCP in local mode (no URL)
echo "Test 4: GitHub MCP in local mode"
cat > $TEST_DIR/config-local.json <<EOF
{
  "mcpServers": {
    "github": {
      "type": "docker",
      "image": "ghcr.io/githubnext/gh-aw-github-mcp:latest"
    }
  }
}
EOF

if bash actions/setup/sh/check_github_remote_mcp_toolsets.sh $TEST_DIR/config-local.json http://localhost:8080 test-key 2>&1 | grep -q "not using HTTP"; then
  echo "✓ Test 4 passed: Skips check for local mode"
else
  echo "✗ Test 4 failed: Should skip check for local mode"
  exit 1
fi
echo ""

# Test 5: GitHub MCP remote mode URL check
echo "Test 5: GitHub MCP remote mode with non-githubcopilot.com URL"
cat > $TEST_DIR/config-custom-remote.json <<EOF
{
  "mcpServers": {
    "github": {
      "type": "http",
      "url": "http://custom-server.example.com/mcp/"
    }
  }
}
EOF

if bash actions/setup/sh/check_github_remote_mcp_toolsets.sh $TEST_DIR/config-custom-remote.json http://localhost:8080 test-key 2>&1 | grep -q "not using remote mode"; then
  echo "✓ Test 5 passed: Skips check for non-githubcopilot.com URLs"
else
  echo "✗ Test 5 failed: Should skip check for non-githubcopilot.com URLs"
  exit 1
fi
echo ""

echo "All tests passed!"
