#!/bin/bash
set +o histexpand

# Test script for resolve_docker_socket_gid.sh
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_PATH="$SCRIPT_DIR/resolve_docker_socket_gid.sh"

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Print test result
print_result() {
  local test_name="$1"
  local result="$2"
  local message="${3:-}"
  
  TESTS_RUN=$((TESTS_RUN + 1))
  
  if [ "$result" = "PASS" ]; then
    echo -e "${GREEN}✓ PASS${NC}: $test_name"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    echo -e "${RED}✗ FAIL${NC}: $test_name"
    if [ -n "$message" ]; then
      echo -e "  ${YELLOW}Message:${NC} $message"
    fi
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi
}

# Test 1: Script syntax is valid
test_script_syntax() {
  echo ""
  echo "═══════════════════════════════════════════════════════════"
  echo "Test 1: Verify script syntax"
  echo "═══════════════════════════════════════════════════════════"
  
  if bash -n "$SCRIPT_PATH" 2>/dev/null; then
    print_result "Script syntax is valid" "PASS"
  else
    print_result "Script has syntax errors" "FAIL"
  fi
}

# Test 2: GH_AW_DOCKER_SOCK_PATH override takes precedence
test_docker_sock_path_override() {
  echo ""
  echo "═══════════════════════════════════════════════════════════"
  echo "Test 2: GH_AW_DOCKER_SOCK_PATH override takes precedence"
  echo "═══════════════════════════════════════════════════════════"
  
  # Create a temporary test socket
  local test_dir=$(mktemp -d)
  local test_socket="$test_dir/docker.sock"
  touch "$test_socket"
  
  # Set socket group to current user's group
  local test_gid=$(id -g)
  chgrp "$test_gid" "$test_socket" 2>/dev/null || true
  
  # Test with override
  local output
  output=$(
    export GH_AW_DOCKER_SOCK_PATH="$test_socket"
    export DOCKER_HOST="unix:///some/other/socket.sock"
    source "$SCRIPT_PATH" 2>&1
    echo "$DOCKER_SOCK_PATH|$DOCKER_SOCK_GID"
  )
  
  local sock_path=$(echo "$output" | tail -1 | cut -d'|' -f1)
  local sock_gid=$(echo "$output" | tail -1 | cut -d'|' -f2)
  
  if [ "$sock_path" = "$test_socket" ]; then
    print_result "Override path is used" "PASS"
  else
    print_result "Override path is NOT used" "FAIL" "Expected: $test_socket, Got: $sock_path"
  fi
  
  # Cleanup
  rm -rf "$test_dir"
}

# Test 3: GH_AW_DOCKER_SOCK_GID override takes precedence
test_docker_sock_gid_override() {
  echo ""
  echo "═══════════════════════════════════════════════════════════"
  echo "Test 3: GH_AW_DOCKER_SOCK_GID override takes precedence"
  echo "═══════════════════════════════════════════════════════════"
  
  # Create a temporary test socket
  local test_dir=$(mktemp -d)
  local test_socket="$test_dir/docker.sock"
  touch "$test_socket"
  
  # Test with GID override
  local output
  output=$(
    export GH_AW_DOCKER_SOCK_PATH="$test_socket"
    export GH_AW_DOCKER_SOCK_GID="999"
    source "$SCRIPT_PATH" 2>&1
    echo "$DOCKER_SOCK_GID"
  )
  
  local sock_gid=$(echo "$output" | tail -1)
  
  if [ "$sock_gid" = "999" ]; then
    print_result "Override GID is used" "PASS"
  else
    print_result "Override GID is NOT used" "FAIL" "Expected: 999, Got: $sock_gid"
  fi
  
  # Cleanup
  rm -rf "$test_dir"
}

# Test 4: unix:// DOCKER_HOST is parsed correctly
test_unix_docker_host_parsing() {
  echo ""
  echo "═══════════════════════════════════════════════════════════"
  echo "Test 4: unix:// DOCKER_HOST is parsed correctly"
  echo "═══════════════════════════════════════════════════════════"
  
  # Create a temporary test socket
  local test_dir=$(mktemp -d)
  local test_socket="$test_dir/docker.sock"
  touch "$test_socket"
  
  # Set socket group to current user's group
  local test_gid=$(id -g)
  chgrp "$test_gid" "$test_socket" 2>/dev/null || true
  
  # Test with unix:// scheme
  local output
  output=$(
    unset GH_AW_DOCKER_SOCK_PATH
    unset GH_AW_DOCKER_SOCK_GID
    export DOCKER_HOST="unix://$test_socket"
    source "$SCRIPT_PATH" 2>&1
    echo "$DOCKER_SOCK_PATH"
  )
  
  local sock_path=$(echo "$output" | tail -1)
  
  if [ "$sock_path" = "$test_socket" ]; then
    print_result "unix:// scheme is parsed correctly" "PASS"
  else
    print_result "unix:// scheme parsing failed" "FAIL" "Expected: $test_socket, Got: $sock_path"
  fi
  
  # Cleanup
  rm -rf "$test_dir"
}

# Test 5: unix:// with authority is parsed correctly
test_unix_with_authority_parsing() {
  echo ""
  echo "═══════════════════════════════════════════════════════════"
  echo "Test 5: unix:// with authority (hostname) is parsed correctly"
  echo "═══════════════════════════════════════════════════════════"
  
  # Create a temporary test socket
  local test_dir=$(mktemp -d)
  local test_socket="$test_dir/docker.sock"
  touch "$test_socket"
  
  # Set socket group to current user's group
  local test_gid=$(id -g)
  chgrp "$test_gid" "$test_socket" 2>/dev/null || true
  
  # Test with unix://hostname/path scheme
  local output
  output=$(
    unset GH_AW_DOCKER_SOCK_PATH
    unset GH_AW_DOCKER_SOCK_GID
    export DOCKER_HOST="unix://somehost$test_socket"
    source "$SCRIPT_PATH" 2>&1
    echo "$DOCKER_SOCK_PATH"
  )
  
  local sock_path=$(echo "$output" | tail -1)
  
  if [ "$sock_path" = "$test_socket" ]; then
    print_result "unix:// with authority is parsed correctly" "PASS"
  else
    print_result "unix:// with authority parsing failed" "FAIL" "Expected: $test_socket, Got: $sock_path"
  fi
  
  # Cleanup
  rm -rf "$test_dir"
}

# Test 6: Absolute path DOCKER_HOST is used directly
test_absolute_path_docker_host() {
  echo ""
  echo "═══════════════════════════════════════════════════════════"
  echo "Test 6: Absolute path DOCKER_HOST is used directly"
  echo "═══════════════════════════════════════════════════════════"
  
  # Create a temporary test socket
  local test_dir=$(mktemp -d)
  local test_socket="$test_dir/docker.sock"
  touch "$test_socket"
  
  # Set socket group to current user's group
  local test_gid=$(id -g)
  chgrp "$test_gid" "$test_socket" 2>/dev/null || true
  
  # Test with absolute path
  local output
  output=$(
    unset GH_AW_DOCKER_SOCK_PATH
    unset GH_AW_DOCKER_SOCK_GID
    export DOCKER_HOST="$test_socket"
    source "$SCRIPT_PATH" 2>&1
    echo "$DOCKER_SOCK_PATH"
  )
  
  local sock_path=$(echo "$output" | tail -1)
  
  if [ "$sock_path" = "$test_socket" ]; then
    print_result "Absolute path DOCKER_HOST is used" "PASS"
  else
    print_result "Absolute path DOCKER_HOST failed" "FAIL" "Expected: $test_socket, Got: $sock_path"
  fi
  
  # Cleanup
  rm -rf "$test_dir"
}

# Test 7: Non-unix schemes fall back to /var/run/docker.sock
test_fallback_to_default() {
  echo ""
  echo "═══════════════════════════════════════════════════════════"
  echo "Test 7: Non-unix schemes fall back to /var/run/docker.sock"
  echo "═══════════════════════════════════════════════════════════"
  
  local output
  output=$(
    unset GH_AW_DOCKER_SOCK_PATH
    export DOCKER_HOST="tcp://localhost:2375"
    source "$SCRIPT_PATH" 2>&1 || true
    echo "$DOCKER_SOCK_PATH"
  )
  
  local sock_path=$(echo "$output" | grep -v "::error::" | grep -v "::warning::" | tail -1)
  
  if [ "$sock_path" = "/var/run/docker.sock" ]; then
    print_result "Falls back to /var/run/docker.sock for tcp://" "PASS"
  else
    print_result "Fallback to default failed" "FAIL" "Expected: /var/run/docker.sock, Got: $sock_path"
  fi
}

# Test 8: stat -Lc follows symlinks
test_symlink_following() {
  echo ""
  echo "═══════════════════════════════════════════════════════════"
  echo "Test 8: stat -Lc follows symlinks to resolve real socket GID"
  echo "═══════════════════════════════════════════════════════════"
  
  # Create a temporary test socket and symlink
  local test_dir=$(mktemp -d)
  local real_socket="$test_dir/real_docker.sock"
  local link_socket="$test_dir/docker.sock"
  touch "$real_socket"
  ln -s "$real_socket" "$link_socket"
  
  # Set socket group to current user's group
  local test_gid=$(id -g)
  chgrp "$test_gid" "$real_socket" 2>/dev/null || true
  
  # Test with symlink
  local output
  output=$(
    export GH_AW_DOCKER_SOCK_PATH="$link_socket"
    unset GH_AW_DOCKER_SOCK_GID
    source "$SCRIPT_PATH" 2>&1
    echo "$DOCKER_SOCK_GID"
  )
  
  local sock_gid=$(echo "$output" | tail -1)
  
  if [ "$sock_gid" = "$test_gid" ]; then
    print_result "Symlink is followed to get real socket GID" "PASS"
  else
    print_result "Symlink following failed" "FAIL" "Expected: $test_gid, Got: $sock_gid"
  fi
  
  # Cleanup
  rm -rf "$test_dir"
}

# Test 9: Non-numeric GID is rejected
test_non_numeric_gid_rejected() {
  echo ""
  echo "═══════════════════════════════════════════════════════════"
  echo "Test 9: Non-numeric GID is rejected with error"
  echo "═══════════════════════════════════════════════════════════"
  
  # Test with non-numeric override
  local exit_code=0
  local output
  output=$(
    export GH_AW_DOCKER_SOCK_GID="abc"
    source "$SCRIPT_PATH" 2>&1 || exit_code=$?
    echo "EXIT:$?"
  ) || exit_code=$?
  
  if [ $exit_code -ne 0 ] && echo "$output" | grep -q "::error::.*not a valid numeric group ID"; then
    print_result "Non-numeric GID is rejected" "PASS"
  else
    print_result "Non-numeric GID validation failed" "FAIL" "Exit code: $exit_code"
  fi
}

# Test 10: Empty GID is rejected
test_empty_gid_rejected() {
  echo ""
  echo "═══════════════════════════════════════════════════════════"
  echo "Test 10: Empty GID is rejected with error"
  echo "═══════════════════════════════════════════════════════════"
  
  # Test with empty override
  local exit_code=0
  local output
  output=$(
    export GH_AW_DOCKER_SOCK_GID=""
    export GH_AW_DOCKER_SOCK_PATH="/nonexistent/socket"
    source "$SCRIPT_PATH" 2>&1 || exit_code=$?
    echo "EXIT:$?"
  ) || exit_code=$?
  
  if [ $exit_code -ne 0 ] && echo "$output" | grep -q "::error::Cannot determine Docker socket group"; then
    print_result "Empty GID from failed stat is rejected" "PASS"
  else
    print_result "Empty GID validation failed" "FAIL" "Exit code: $exit_code"
  fi
}

# Test 11: Missing socket file produces error
test_missing_socket_error() {
  echo ""
  echo "═══════════════════════════════════════════════════════════"
  echo "Test 11: Missing socket file produces error and warning"
  echo "═══════════════════════════════════════════════════════════"
  
  # Test with non-existent socket
  local exit_code=0
  local output
  output=$(
    unset GH_AW_DOCKER_SOCK_GID
    export GH_AW_DOCKER_SOCK_PATH="/nonexistent/socket.sock"
    source "$SCRIPT_PATH" 2>&1 || exit_code=$?
    echo "EXIT:$?"
  ) || exit_code=$?
  
  if [ $exit_code -ne 0 ] && \
     echo "$output" | grep -q "::error::Cannot determine Docker socket group" && \
     echo "$output" | grep -q "::warning::.*does not exist"; then
    print_result "Missing socket produces error and warning" "PASS"
  else
    print_result "Missing socket error handling failed" "FAIL" "Exit code: $exit_code"
  fi
}

# Test 12: Successful GID resolution from stat
test_successful_gid_resolution() {
  echo ""
  echo "═══════════════════════════════════════════════════════════"
  echo "Test 12: Successful GID resolution from stat"
  echo "═══════════════════════════════════════════════════════════"
  
  # Create a temporary test socket
  local test_dir=$(mktemp -d)
  local test_socket="$test_dir/docker.sock"
  touch "$test_socket"
  
  # Set socket group to current user's group
  local expected_gid=$(id -g)
  chgrp "$expected_gid" "$test_socket" 2>/dev/null || true
  
  # Test with auto-detection
  local output
  output=$(
    export GH_AW_DOCKER_SOCK_PATH="$test_socket"
    unset GH_AW_DOCKER_SOCK_GID
    source "$SCRIPT_PATH" 2>&1
    echo "$DOCKER_SOCK_GID"
  )
  
  local sock_gid=$(echo "$output" | tail -1)
  
  if [ "$sock_gid" = "$expected_gid" ]; then
    print_result "GID is resolved correctly from stat" "PASS"
  else
    print_result "GID resolution from stat failed" "FAIL" "Expected: $expected_gid, Got: $sock_gid"
  fi
  
  # Cleanup
  rm -rf "$test_dir"
}

# Test 13: Variables are exported
test_variables_exported() {
  echo ""
  echo "═══════════════════════════════════════════════════════════"
  echo "Test 13: DOCKER_SOCK_PATH and DOCKER_SOCK_GID are exported"
  echo "═══════════════════════════════════════════════════════════"
  
  # Create a temporary test socket
  local test_dir=$(mktemp -d)
  local test_socket="$test_dir/docker.sock"
  touch "$test_socket"
  
  # Set socket group to current user's group
  local test_gid=$(id -g)
  chgrp "$test_gid" "$test_socket" 2>/dev/null || true
  
  # Test that variables are exported
  local output
  output=$(
    export GH_AW_DOCKER_SOCK_PATH="$test_socket"
    unset GH_AW_DOCKER_SOCK_GID
    source "$SCRIPT_PATH" 2>&1
    env | grep -E "^DOCKER_SOCK_(PATH|GID)=" | sort
  )
  
  if echo "$output" | grep -q "^DOCKER_SOCK_GID=" && \
     echo "$output" | grep -q "^DOCKER_SOCK_PATH="; then
    print_result "Variables are exported" "PASS"
  else
    print_result "Variables export failed" "FAIL"
  fi
  
  # Cleanup
  rm -rf "$test_dir"
}

# Test 14: GID with leading zeros is rejected
test_gid_with_leading_zeros() {
  echo ""
  echo "═══════════════════════════════════════════════════════════"
  echo "Test 14: GID with special characters is rejected"
  echo "═══════════════════════════════════════════════════════════"
  
  # Test with GID containing special characters
  local exit_code=0
  local output
  output=$(
    export GH_AW_DOCKER_SOCK_GID="123-456"
    source "$SCRIPT_PATH" 2>&1 || exit_code=$?
    echo "EXIT:$?"
  ) || exit_code=$?
  
  if [ $exit_code -ne 0 ] && echo "$output" | grep -q "::error::.*not a valid numeric group ID"; then
    print_result "GID with special characters is rejected" "PASS"
  else
    print_result "GID validation for special chars failed" "FAIL" "Exit code: $exit_code"
  fi
}

# Test 15: Empty DOCKER_HOST defaults to /var/run/docker.sock
test_empty_docker_host_default() {
  echo ""
  echo "═══════════════════════════════════════════════════════════"
  echo "Test 15: Empty DOCKER_HOST defaults to /var/run/docker.sock"
  echo "═══════════════════════════════════════════════════════════"
  
  local output
  output=$(
    unset GH_AW_DOCKER_SOCK_PATH
    unset DOCKER_HOST
    source "$SCRIPT_PATH" 2>&1 || true
    echo "$DOCKER_SOCK_PATH"
  )
  
  local sock_path=$(echo "$output" | grep -v "::error::" | grep -v "::warning::" | tail -1)
  
  if [ "$sock_path" = "/var/run/docker.sock" ]; then
    print_result "Empty DOCKER_HOST defaults to /var/run/docker.sock" "PASS"
  else
    print_result "Default path logic failed" "FAIL" "Expected: /var/run/docker.sock, Got: $sock_path"
  fi
}

# Run all tests
echo "════════════════════════════════════════════════════════════════════"
echo "Running tests for resolve_docker_socket_gid.sh"
echo "════════════════════════════════════════════════════════════════════"

test_script_syntax
test_docker_sock_path_override
test_docker_sock_gid_override
test_unix_docker_host_parsing
test_unix_with_authority_parsing
test_absolute_path_docker_host
test_fallback_to_default
test_symlink_following
test_non_numeric_gid_rejected
test_empty_gid_rejected
test_missing_socket_error
test_successful_gid_resolution
test_variables_exported
test_gid_with_leading_zeros
test_empty_docker_host_default

# Print summary
echo ""
echo "════════════════════════════════════════════════════════════════════"
echo "Test Summary"
echo "════════════════════════════════════════════════════════════════════"
echo "Total tests run: $TESTS_RUN"
echo -e "${GREEN}Tests passed: $TESTS_PASSED${NC}"
echo -e "${RED}Tests failed: $TESTS_FAILED${NC}"
echo "════════════════════════════════════════════════════════════════════"

if [ $TESTS_FAILED -eq 0 ]; then
  echo -e "${GREEN}All tests passed!${NC}"
  exit 0
else
  echo -e "${RED}Some tests failed!${NC}"
  exit 1
fi
