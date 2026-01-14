#!/bin/bash
# Test script for start_mcp_gateway.sh
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_PATH="$SCRIPT_DIR/start_mcp_gateway.sh"

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

# Test 1: Script syntax is valid
test_script_syntax() {
  echo ""
  echo "Test 1: Verify script syntax"
  
  if bash -n "$SCRIPT_PATH" 2>/dev/null; then
    print_result "Script syntax is valid" "PASS"
  else
    print_result "Script has syntax errors" "FAIL"
  fi
}

# Test 2: Required environment variables validation
test_env_var_validation() {
  echo ""
  echo "Test 2: Required environment variables validation"
  
  # Test missing MCP_GATEWAY_PORT
  if ! MCP_GATEWAY_DOMAIN="localhost" MCP_GATEWAY_API_KEY="test-key" MCP_GATEWAY_DOCKER_COMMAND="docker run -i --rm --network host test-image" bash "$SCRIPT_PATH" 2>/dev/null; then
    print_result "Script rejects missing MCP_GATEWAY_PORT" "PASS"
  else
    print_result "Script should reject missing MCP_GATEWAY_PORT" "FAIL"
  fi
  
  # Test missing MCP_GATEWAY_DOMAIN
  if ! MCP_GATEWAY_PORT="8080" MCP_GATEWAY_API_KEY="test-key" MCP_GATEWAY_DOCKER_COMMAND="docker run -i --rm --network host test-image" bash "$SCRIPT_PATH" 2>/dev/null; then
    print_result "Script rejects missing MCP_GATEWAY_DOMAIN" "PASS"
  else
    print_result "Script should reject missing MCP_GATEWAY_DOMAIN" "FAIL"
  fi
  
  # Test missing MCP_GATEWAY_API_KEY
  if ! MCP_GATEWAY_PORT="8080" MCP_GATEWAY_DOMAIN="localhost" MCP_GATEWAY_DOCKER_COMMAND="docker run -i --rm --network host test-image" bash "$SCRIPT_PATH" 2>/dev/null; then
    print_result "Script rejects missing MCP_GATEWAY_API_KEY" "PASS"
  else
    print_result "Script should reject missing MCP_GATEWAY_API_KEY" "FAIL"
  fi
  
  # Test missing MCP_GATEWAY_DOCKER_COMMAND
  if ! MCP_GATEWAY_PORT="8080" MCP_GATEWAY_DOMAIN="localhost" MCP_GATEWAY_API_KEY="test-key" bash "$SCRIPT_PATH" 2>/dev/null; then
    print_result "Script rejects missing MCP_GATEWAY_DOCKER_COMMAND" "PASS"
  else
    print_result "Script should reject missing MCP_GATEWAY_DOCKER_COMMAND" "FAIL"
  fi
}

# Test 3: Configuration file not found
test_config_not_found() {
  echo ""
  echo "Test 3: Configuration file not found"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local fake_home="$tmpdir/home"
  mkdir -p "$fake_home/.copilot"
  
  # Create a modified script that uses our fake home
  local test_script="$tmpdir/test_script.sh"
  sed "s|/home/runner|$fake_home|g" "$SCRIPT_PATH" > "$test_script"
  
  # Test without config file
  if ! MCP_GATEWAY_PORT="8080" MCP_GATEWAY_DOMAIN="localhost" MCP_GATEWAY_API_KEY="test-key" MCP_GATEWAY_DOCKER_COMMAND="docker run -i --rm --network host test-image" bash "$test_script" 2>/dev/null; then
    print_result "Script rejects non-existent config file" "PASS"
  else
    print_result "Script should reject non-existent config file" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 4: Configuration file is invalid JSON
test_invalid_json_config() {
  echo ""
  echo "Test 4: Configuration file is invalid JSON"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local fake_home="$tmpdir/home"
  mkdir -p "$fake_home/.copilot"
  
  # Create invalid JSON config
  echo "{ invalid json" > "$fake_home/.copilot/mcp-config.json"
  
  # Create a modified script that uses our fake home
  local test_script="$tmpdir/test_script.sh"
  sed "s|/home/runner|$fake_home|g" "$SCRIPT_PATH" > "$test_script"
  
  if ! MCP_GATEWAY_PORT="8080" MCP_GATEWAY_DOMAIN="localhost" MCP_GATEWAY_API_KEY="test-key" MCP_GATEWAY_DOCKER_COMMAND="docker run -i --rm --network host test-image" bash "$test_script" 2>/dev/null; then
    print_result "Script rejects invalid JSON config" "PASS"
  else
    print_result "Script should reject invalid JSON config" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 5: Container missing 'docker run' prefix
test_container_missing_docker_run() {
  echo ""
  echo "Test 5: Container missing 'docker run' prefix"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local fake_home="$tmpdir/home"
  mkdir -p "$fake_home/.copilot"
  
  # Create valid JSON config with required gateway section
  echo '{"mcpServers":{},"gateway":{"port":8080,"domain":"localhost","apiKey":"test-key"}}' > "$fake_home/.copilot/mcp-config.json"
  
  # Create a modified script that uses our fake home
  local test_script="$tmpdir/test_script.sh"
  sed "s|/home/runner|$fake_home|g" "$SCRIPT_PATH" > "$test_script"
  
  # Test with container that doesn't start with "docker run"
  if ! MCP_GATEWAY_PORT="8080" MCP_GATEWAY_DOMAIN="localhost" MCP_GATEWAY_API_KEY="test-key" MCP_GATEWAY_DOCKER_COMMAND="test-image" bash "$test_script" 2>/dev/null; then
    print_result "Script rejects container without 'docker run'" "PASS"
  else
    print_result "Script should reject container without 'docker run'" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 6: Container missing required -i flag
test_container_missing_i_flag() {
  echo ""
  echo "Test 6: Container missing required -i flag"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local fake_home="$tmpdir/home"
  mkdir -p "$fake_home/.copilot"
  
  # Create valid JSON config with required gateway section
  echo '{"mcpServers":{},"gateway":{"port":8080,"domain":"localhost","apiKey":"test-key"}}' > "$fake_home/.copilot/mcp-config.json"
  
  # Create a modified script that uses our fake home
  local test_script="$tmpdir/test_script.sh"
  sed "s|/home/runner|$fake_home|g" "$SCRIPT_PATH" > "$test_script"
  
  # Test with container missing -i flag
  if ! MCP_GATEWAY_PORT="8080" MCP_GATEWAY_DOMAIN="localhost" MCP_GATEWAY_API_KEY="test-key" MCP_GATEWAY_DOCKER_COMMAND="docker run --rm --network host test-image" bash "$test_script" 2>/dev/null; then
    print_result "Script rejects container without -i flag" "PASS"
  else
    print_result "Script should reject container without -i flag" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 7: Container missing required --rm flag
test_container_missing_rm_flag() {
  echo ""
  echo "Test 7: Container missing required --rm flag"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local fake_home="$tmpdir/home"
  mkdir -p "$fake_home/.copilot"
  
  # Create valid JSON config with required gateway section
  echo '{"mcpServers":{},"gateway":{"port":8080,"domain":"localhost","apiKey":"test-key"}}' > "$fake_home/.copilot/mcp-config.json"
  
  # Create a modified script that uses our fake home
  local test_script="$tmpdir/test_script.sh"
  sed "s|/home/runner|$fake_home|g" "$SCRIPT_PATH" > "$test_script"
  
  # Test with container missing --rm flag
  if ! MCP_GATEWAY_PORT="8080" MCP_GATEWAY_DOMAIN="localhost" MCP_GATEWAY_API_KEY="test-key" MCP_GATEWAY_DOCKER_COMMAND="docker run -i --network host test-image" bash "$test_script" 2>/dev/null; then
    print_result "Script rejects container without --rm flag" "PASS"
  else
    print_result "Script should reject container without --rm flag" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 8: Container missing required --network host flag
test_container_missing_network_flag() {
  echo ""
  echo "Test 8: Container missing required --network host flag"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local fake_home="$tmpdir/home"
  mkdir -p "$fake_home/.copilot"
  
  # Create valid JSON config with required gateway section
  echo '{"mcpServers":{},"gateway":{"port":8080,"domain":"localhost","apiKey":"test-key"}}' > "$fake_home/.copilot/mcp-config.json"
  
  # Create a modified script that uses our fake home
  local test_script="$tmpdir/test_script.sh"
  sed "s|/home/runner|$fake_home|g" "$SCRIPT_PATH" > "$test_script"
  
  # Test with container missing --network host flag
  if ! MCP_GATEWAY_PORT="8080" MCP_GATEWAY_DOMAIN="localhost" MCP_GATEWAY_API_KEY="test-key" MCP_GATEWAY_DOCKER_COMMAND="docker run -i --rm test-image" bash "$test_script" 2>/dev/null; then
    print_result "Script rejects container without --network host flag" "PASS"
  else
    print_result "Script should reject container without --network host flag" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 9: Validation functions exist
test_validation_functions_exist() {
  echo ""
  echo "Test 9: Verify validation logic exists"
  
  # Check for config file validation
  if grep -q "Configuration file not found" "$SCRIPT_PATH"; then
    print_result "Config file validation exists" "PASS"
  else
    print_result "Config file validation missing" "FAIL"
  fi
  
  # Check for JSON validation
  if grep -q "not valid JSON" "$SCRIPT_PATH"; then
    print_result "JSON validation exists" "PASS"
  else
    print_result "JSON validation missing" "FAIL"
  fi
  
  # Check for container syntax validation
  if grep -q "incorrect syntax" "$SCRIPT_PATH"; then
    print_result "Container syntax validation exists" "PASS"
  else
    print_result "Container syntax validation missing" "FAIL"
  fi
  
  # Check for -i flag validation
  if grep -q "must include -i flag" "$SCRIPT_PATH"; then
    print_result "-i flag validation exists" "PASS"
  else
    print_result "-i flag validation missing" "FAIL"
  fi
  
  # Check for --rm flag validation
  if grep -q "must include --rm flag" "$SCRIPT_PATH"; then
    print_result "--rm flag validation exists" "PASS"
  else
    print_result "--rm flag validation missing" "FAIL"
  fi
  
  # Check for --network host validation
  if grep -q "must include --network host flag" "$SCRIPT_PATH"; then
    print_result "--network host flag validation exists" "PASS"
  else
    print_result "--network host flag validation missing" "FAIL"
  fi
}

# Run all tests
echo "=== Testing start_mcp_gateway.sh ==="
echo "Script: $SCRIPT_PATH"

test_script_syntax
test_env_var_validation
test_config_not_found
test_invalid_json_config
test_container_missing_docker_run
test_container_missing_i_flag
test_container_missing_rm_flag
test_container_missing_network_flag
test_validation_functions_exist

# Print summary
echo ""
echo "=== Test Summary ==="
echo "Tests run: $TESTS_RUN"
echo -e "${GREEN}Tests passed: $TESTS_PASSED${NC}"
if [ $TESTS_FAILED -gt 0 ]; then
  echo -e "${RED}Tests failed: $TESTS_FAILED${NC}"
  exit 1
else
  echo -e "${GREEN}All tests passed!${NC}"
  exit 0
fi
