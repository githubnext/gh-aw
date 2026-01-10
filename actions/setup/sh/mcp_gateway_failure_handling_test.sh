#!/bin/bash
# Test script for verifying MCP gateway failure handling
# This test verifies that start_mcp_gateway.sh correctly terminates the gateway
# when check_mcp_servers.sh detects server connection failures

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECK_SCRIPT="$SCRIPT_DIR/check_mcp_servers.sh"

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Print test result
print_result() {
  local test_name="$1"
  local result="$2"
  
  TESTS_RUN=$((TESTS_RUN + 1))
  
  if [ "$result" = "PASS" ]; then
    echo -e "${GREEN}✓ PASS${NC}: $test_name"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    echo -e "${RED}✗ FAIL${NC}: $test_name"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi
}

echo "========================================"
echo "MCP Gateway Failure Handling Tests"
echo "========================================"
echo ""

# Test 1: Server with connection failure returns exit code 1
test_server_connection_failure() {
  echo "Test 1: Server connection failure returns exit code 1"
  
  local tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create config with HTTP server that cannot be reached
  cat > "$config_file" <<'EOF'
{
  "mcpServers": {
    "unreachable-server": {
      "type": "http",
      "url": "http://localhost:9999/mcp/unreachable",
      "headers": {
        "Authorization": "Bearer test-token"
      }
    }
  }
}
EOF
  
  # Script should fail with exit code 1 (server cannot be connected)
  # Use timeout to avoid long waits, but check the actual script exit code
  timeout 20 bash "$CHECK_SCRIPT" "$config_file" "http://localhost:9999" "test-key" >/dev/null 2>&1
  local exit_code=$?
  
  # timeout returns 124 on timeout, but we want the script's exit code
  # Script should fail before timeout with exit code 1
  if [ $exit_code -eq 1 ]; then
    print_result "Connection failure returns exit code 1" "PASS"
  elif [ $exit_code -eq 124 ]; then
    echo "  Test timed out - script may be hanging"
    print_result "Connection failure test timed out" "FAIL"
  else
    echo "  Expected exit code 1, got $exit_code"
    print_result "Connection failure returns correct exit code" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 2: Multiple servers with at least one failure returns exit code 1
test_multiple_servers_with_failure() {
  echo ""
  echo "Test 2: Multiple servers with at least one failure returns exit code 1"
  
  local tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create config with multiple HTTP servers, all unreachable
  cat > "$config_file" <<'EOF'
{
  "mcpServers": {
    "server1": {
      "type": "http",
      "url": "http://localhost:9998/mcp/server1",
      "headers": {
        "Authorization": "Bearer token1"
      }
    },
    "server2": {
      "type": "http",
      "url": "http://localhost:9997/mcp/server2",
      "headers": {
        "Authorization": "Bearer token2"
      }
    }
  }
}
EOF
  
  # Script should fail because all servers are unreachable
  # Use timeout to avoid long waits (multiple servers = more retry time)
  timeout 30 bash "$CHECK_SCRIPT" "$config_file" "http://localhost:9998" "test-key" >/dev/null 2>&1
  local exit_code=$?
  
  if [ $exit_code -eq 1 ]; then
    print_result "Multiple server failures return exit code 1" "PASS"
  elif [ $exit_code -eq 124 ]; then
    echo "  Test timed out - script may be hanging"
    print_result "Multiple server failures test timed out" "FAIL"
  else
    echo "  Expected exit code 1, got $exit_code"
    print_result "Multiple server failures return correct exit code" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 3: Verify exit code 1 for initialize request failure
test_initialize_failure() {
  echo ""
  echo "Test 3: Initialize request failure returns exit code 1"
  
  local tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create config with unreachable server
  cat > "$config_file" <<'EOF'
{
  "mcpServers": {
    "test-server": {
      "type": "http",
      "url": "http://localhost:8765/mcp/test",
      "headers": {
        "Authorization": "Bearer test-token"
      }
    }
  }
}
EOF
  
  # Capture both exit code and output (with timeout)
  output=$(timeout 20 bash "$CHECK_SCRIPT" "$config_file" "http://localhost:8765" "test-key" 2>&1)
  exit_code=$?
  
  # Should fail with exit code 1
  if [ $exit_code -eq 1 ]; then
    # Verify error message mentions failure
    if echo "$output" | grep -q "failed"; then
      print_result "Initialize failure detected and reported" "PASS"
    else
      echo "  Exit code correct but error message not found"
      print_result "Initialize failure error message" "FAIL"
    fi
  else
    echo "  Expected exit code 1, got $exit_code"
    print_result "Initialize failure returns exit code 1" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 4: Verify no servers successfully checked returns exit code 1
test_no_successful_checks() {
  echo ""
  echo "Test 4: No successful server checks returns exit code 1"
  
  local tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create config with only stdio servers (which get skipped)
  cat > "$config_file" <<'EOF'
{
  "mcpServers": {
    "stdio-server": {
      "type": "stdio",
      "command": "node",
      "args": ["server.js"]
    }
  }
}
EOF
  
  # Should exit 1 because no HTTP servers were successfully checked
  timeout 5 bash "$CHECK_SCRIPT" "$config_file" "http://localhost:8080" "test-key" >/dev/null 2>&1
  local exit_code=$?
  
  if [ $exit_code -eq 1 ]; then
    print_result "No successful checks returns exit code 1" "PASS"
  elif [ $exit_code -eq 124 ]; then
    echo "  Test timed out unexpectedly"
    print_result "No successful checks test timed out" "FAIL"
  else
    echo "  Expected exit code 1, got $exit_code"
    print_result "No successful checks returns correct exit code" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Run all tests
test_server_connection_failure
test_multiple_servers_with_failure
test_initialize_failure
test_no_successful_checks

# Print summary
echo ""
echo "========================================"
echo "Test Summary"
echo "========================================"
echo "Tests run: $TESTS_RUN"
echo "Tests passed: $TESTS_PASSED"
echo "Tests failed: $TESTS_FAILED"
echo ""

if [ $TESTS_FAILED -gt 0 ]; then
  echo -e "${RED}Some tests failed${NC}"
  exit 1
else
  echo -e "${GREEN}All tests passed!${NC}"
  exit 0
fi
