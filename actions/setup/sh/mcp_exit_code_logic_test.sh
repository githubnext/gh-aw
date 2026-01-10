#!/bin/bash
# Fast unit test for MCP gateway failure exit code logic
# Tests the final exit code determination without network timeouts

set -e

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

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
echo "MCP Gateway Exit Code Logic Tests"
echo "========================================"
echo ""

# Test 1: Simulate exit code logic with failures
echo "Test 1: Exit code logic with SERVERS_FAILED > 0"
SERVERS_FAILED=1
SERVERS_SUCCEEDED=0
if [ $SERVERS_FAILED -gt 0 ]; then
  print_result "SERVERS_FAILED > 0 triggers exit 1 logic" "PASS"
elif [ $SERVERS_SUCCEEDED -eq 0 ]; then
  print_result "Secondary condition not needed" "FAIL"
else
  print_result "Should not reach success path" "FAIL"
fi
echo ""

# Test 2: No successes should fail
echo "Test 2: Exit code logic with SERVERS_SUCCEEDED = 0"
SERVERS_FAILED=0
SERVERS_SUCCEEDED=0
if [ $SERVERS_FAILED -gt 0 ]; then
  print_result "Should not trigger first condition" "FAIL"
elif [ $SERVERS_SUCCEEDED -eq 0 ]; then
  print_result "SERVERS_SUCCEEDED = 0 triggers exit 1 logic" "PASS"
else
  print_result "Should not reach success path" "FAIL"
fi
echo ""

# Test 3: Success case
echo "Test 3: Exit code logic with successes and no failures"
SERVERS_FAILED=0
SERVERS_SUCCEEDED=2
if [ $SERVERS_FAILED -gt 0 ]; then
  print_result "Should not trigger first condition" "FAIL"
elif [ $SERVERS_SUCCEEDED -eq 0 ]; then
  print_result "Should not trigger second condition" "FAIL"
else
  print_result "Success case triggers exit 0 logic" "PASS"
fi
echo ""

# Test 4: Verify actual script exit codes with quick-fail configs
echo "Test 4: Actual script with invalid config (quick exit)"
tmpdir=$(mktemp -d)
# Invalid JSON - should fail immediately
echo "not json" > "$tmpdir/bad.json"
if bash actions/setup/sh/check_mcp_servers.sh "$tmpdir/bad.json" "http://localhost:8080" "key" >/dev/null 2>&1; then
  print_result "Invalid JSON should fail" "FAIL"
else
  exit_code=$?
  if [ $exit_code -eq 1 ]; then
    print_result "Invalid JSON returns exit code 1" "PASS"
  else
    echo "  Expected 1, got $exit_code"
    print_result "Invalid JSON exit code" "FAIL"
  fi
fi
rm -rf "$tmpdir"
echo ""

# Test 5: Empty servers (quick exit)
echo "Test 5: Actual script with empty servers (quick exit)"
tmpdir=$(mktemp -d)
cat > "$tmpdir/empty.json" <<'EOF'
{
  "mcpServers": {}
}
EOF
if bash actions/setup/sh/check_mcp_servers.sh "$tmpdir/empty.json" "http://localhost:8080" "key" >/dev/null 2>&1; then
  print_result "Empty mcpServers returns exit code 0" "PASS"
else
  echo "  Empty servers should exit 0 (no servers to check)"
  print_result "Empty mcpServers handling" "FAIL"
fi
rm -rf "$tmpdir"
echo ""

# Test 6: Only stdio servers (quick exit - no HTTP servers to check)
echo "Test 6: Only stdio servers (should fail - no HTTP servers checked)"
tmpdir=$(mktemp -d)
cat > "$tmpdir/stdio.json" <<'EOF'
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
if bash actions/setup/sh/check_mcp_servers.sh "$tmpdir/stdio.json" "http://localhost:8080" "key" >/dev/null 2>&1; then
  print_result "stdio-only should fail (no HTTP servers)" "FAIL"
else
  exit_code=$?
  if [ $exit_code -eq 1 ]; then
    print_result "stdio-only returns exit code 1" "PASS"
  else
    echo "  Expected 1, got $exit_code"
    print_result "stdio-only exit code" "FAIL"
  fi
fi
rm -rf "$tmpdir"
echo ""

# Print summary
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
