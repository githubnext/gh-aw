#!/bin/bash
# Test script for check_mcp_servers.sh
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_PATH="$SCRIPT_DIR/check_mcp_servers.sh"

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

# Test 2: Script requires 3 arguments
test_argument_validation() {
  echo ""
  echo "Test 2: Argument validation"
  
  # Test with no arguments
  if ! bash "$SCRIPT_PATH" 2>/dev/null; then
    print_result "Script rejects no arguments" "PASS"
  else
    print_result "Script should reject no arguments" "FAIL"
  fi
  
  # Test with 1 argument
  if ! bash "$SCRIPT_PATH" "config.json" 2>/dev/null; then
    print_result "Script rejects 1 argument" "PASS"
  else
    print_result "Script should reject 1 argument" "FAIL"
  fi
  
  # Test with 2 arguments
  if ! bash "$SCRIPT_PATH" "config.json" "http://localhost:8080" 2>/dev/null; then
    print_result "Script rejects 2 arguments" "PASS"
  else
    print_result "Script should reject 2 arguments" "FAIL"
  fi
}

# Test 3: Config file not found
test_config_not_found() {
  echo ""
  echo "Test 3: Config file not found"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local nonexistent_config="$tmpdir/nonexistent.json"
  
  if ! bash "$SCRIPT_PATH" "$nonexistent_config" "http://localhost:8080" "test-key" 2>/dev/null; then
    print_result "Script rejects non-existent config file" "PASS"
  else
    print_result "Script should reject non-existent config file" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 4: Invalid JSON configuration
test_invalid_json_config() {
  echo ""
  echo "Test 4: Invalid JSON configuration"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create invalid JSON
  echo "{ invalid json" > "$config_file"
  
  if ! bash "$SCRIPT_PATH" "$config_file" "http://localhost:8080" "test-key" 2>/dev/null; then
    print_result "Script rejects invalid JSON" "PASS"
  else
    print_result "Script should reject invalid JSON" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 5: Empty mcpServers object
test_empty_servers() {
  echo ""
  echo "Test 5: Empty mcpServers object"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create config with empty mcpServers
  cat > "$config_file" <<'EOF'
{
  "mcpServers": {},
  "gateway": {
    "port": 8080,
    "domain": "localhost",
    "apiKey": "test-key"
  }
}
EOF
  
  # Should exit 0 but indicate no servers
  if bash "$SCRIPT_PATH" "$config_file" "http://localhost:8080" "test-key" >/dev/null 2>&1; then
    print_result "Script handles empty mcpServers gracefully" "PASS"
  else
    print_result "Script should handle empty mcpServers gracefully" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 6: Configuration with null mcpServers
test_null_servers() {
  echo ""
  echo "Test 6: Null mcpServers"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create config with null mcpServers
  cat > "$config_file" <<'EOF'
{
  "mcpServers": null,
  "gateway": {
    "port": 8080,
    "domain": "localhost",
    "apiKey": "test-key"
  }
}
EOF
  
  # Should exit 0 but indicate no servers
  if bash "$SCRIPT_PATH" "$config_file" "http://localhost:8080" "test-key" >/dev/null 2>&1; then
    print_result "Script handles null mcpServers gracefully" "PASS"
  else
    print_result "Script should handle null mcpServers gracefully" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 7: Valid configuration with HTTP server
test_valid_http_server() {
  echo ""
  echo "Test 7: Valid configuration with HTTP server"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create valid config with HTTP server
  cat > "$config_file" <<'EOF'
{
  "mcpServers": {
    "github": {
      "type": "http",
      "url": "http://localhost:8080/mcp/github",
      "headers": {
        "Authorization": "Bearer test-token"
      }
    }
  },
  "gateway": {
    "port": 8080,
    "domain": "localhost",
    "apiKey": "test-key"
  }
}
EOF
  
  # Script should fail because no servers can be connected (no gateway running)
  if ! bash "$SCRIPT_PATH" "$config_file" "http://localhost:8080" "test-key" >/dev/null 2>&1; then
    print_result "Script fails when no servers can connect" "PASS"
  else
    print_result "Script should fail when no servers can connect" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 8: Server without URL (stdio server)
test_server_without_url() {
  echo ""
  echo "Test 8: Server without URL (stdio server)"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create config with stdio server (no URL)
  cat > "$config_file" <<'EOF'
{
  "mcpServers": {
    "safeinputs": {
      "type": "stdio",
      "command": "gh",
      "args": ["aw", "mcp-server", "--mode", "safe-inputs"]
    }
  },
  "gateway": {
    "port": 8080,
    "domain": "localhost",
    "apiKey": "test-key"
  }
}
EOF
  
  # Should fail because only stdio servers (which are skipped)
  if ! bash "$SCRIPT_PATH" "$config_file" "http://localhost:8080" "test-key" >/dev/null 2>&1; then
    print_result "Script fails when only stdio servers (no HTTP servers)" "PASS"
  else
    print_result "Script should fail when only stdio servers" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 9: Multiple servers with mixed types
test_mixed_servers() {
  echo ""
  echo "Test 9: Multiple servers with mixed types"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create config with multiple servers
  cat > "$config_file" <<'EOF'
{
  "mcpServers": {
    "safeinputs": {
      "type": "stdio",
      "command": "gh",
      "args": ["aw", "mcp-server", "--mode", "safe-inputs"]
    },
    "github": {
      "type": "http",
      "url": "http://localhost:8080/mcp/github",
      "headers": {
        "Authorization": "Bearer github-token"
      }
    },
    "playwright": {
      "type": "http",
      "url": "http://localhost:8080/mcp/playwright"
    }
  },
  "gateway": {
    "port": 8080,
    "domain": "localhost",
    "apiKey": "test-key"
  }
}
EOF
  
  # Should fail because HTTP servers cannot connect (no gateway running)
  if ! bash "$SCRIPT_PATH" "$config_file" "http://localhost:8080" "test-key" >/dev/null 2>&1; then
    print_result "Script fails when HTTP servers cannot connect" "PASS"
  else
    print_result "Script should fail when HTTP servers cannot connect" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 10: Key validation functions exist
test_validation_functions_exist() {
  echo ""
  echo "Test 10: Verify key validation logic exists"
  
  # Check for configuration file validation
  if grep -q "Gateway configuration file not found" "$SCRIPT_PATH"; then
    print_result "Config file validation exists" "PASS"
  else
    print_result "Config file validation missing" "FAIL"
  fi
  
  # Check for mcpServers parsing
  if grep -q "Failed to parse mcpServers" "$SCRIPT_PATH"; then
    print_result "mcpServers parsing validation exists" "PASS"
  else
    print_result "mcpServers parsing validation missing" "FAIL"
  fi
  
  # Check for ping request (replacing initialize and tools/list)
  if grep -q "method.*ping" "$SCRIPT_PATH"; then
    print_result "Ping request logic exists" "PASS"
  else
    print_result "Ping request logic missing" "FAIL"
  fi
  
  # Check for gateway config authentication logic
  if grep -q "Authorization" "$SCRIPT_PATH"; then
    print_result "Gateway config authentication logic exists" "PASS"
  else
    print_result "Gateway config authentication logic missing" "FAIL"
  fi
}

# Run all tests
echo "=== Testing check_mcp_servers.sh ==="
echo "Script: $SCRIPT_PATH"

test_script_syntax
test_argument_validation
test_config_not_found
test_invalid_json_config
test_empty_servers
test_null_servers
test_valid_http_server
test_server_without_url
test_mixed_servers
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
