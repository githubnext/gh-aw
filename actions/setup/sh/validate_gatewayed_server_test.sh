#!/bin/bash
# Test script for validate_gatewayed_server.sh
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_PATH="$SCRIPT_DIR/validate_gatewayed_server.sh"

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
  if ! bash "$SCRIPT_PATH" "server" 2>/dev/null; then
    print_result "Script rejects 1 argument" "PASS"
  else
    print_result "Script should reject 1 argument" "FAIL"
  fi
  
  # Test with 2 arguments
  if ! bash "$SCRIPT_PATH" "server" "config.json" 2>/dev/null; then
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
  
  if ! bash "$SCRIPT_PATH" "github" "$nonexistent_config" "http://localhost:8080" 2>/dev/null; then
    print_result "Script rejects non-existent config file" "PASS"
  else
    print_result "Script should reject non-existent config file" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 4: Server not found in config
test_server_not_found() {
  echo ""
  echo "Test 4: Server not found in config"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create config with different server
  cat > "$config_file" <<'EOF'
{
  "mcpServers": {
    "playwright": {
      "type": "http",
      "url": "http://localhost:8080/mcp/playwright"
    }
  }
}
EOF
  
  if ! bash "$SCRIPT_PATH" "github" "$config_file" "http://localhost:8080" 2>/dev/null; then
    print_result "Script detects missing server" "PASS"
  else
    print_result "Script should detect missing server" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 5: Server missing URL field
test_server_missing_url() {
  echo ""
  echo "Test 5: Server missing URL field"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create config with server but no URL
  cat > "$config_file" <<'EOF'
{
  "mcpServers": {
    "github": {
      "type": "http"
    }
  }
}
EOF
  
  if ! bash "$SCRIPT_PATH" "github" "$config_file" "http://localhost:8080" 2>/dev/null; then
    print_result "Script detects missing URL" "PASS"
  else
    print_result "Script should detect missing URL" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 6: Server wrong type (not http)
test_server_wrong_type() {
  echo ""
  echo "Test 6: Server wrong type"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create config with non-http type
  cat > "$config_file" <<'EOF'
{
  "mcpServers": {
    "github": {
      "type": "stdio",
      "url": "http://localhost:8080/mcp/github",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"]
    }
  }
}
EOF
  
  if ! bash "$SCRIPT_PATH" "github" "$config_file" "http://localhost:8080" 2>/dev/null; then
    print_result "Script detects wrong type" "PASS"
  else
    print_result "Script should detect wrong type" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 7: URL doesn't point to gateway
test_url_wrong_gateway() {
  echo ""
  echo "Test 7: URL doesn't point to gateway"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create config with URL pointing to different gateway
  cat > "$config_file" <<'EOF'
{
  "mcpServers": {
    "github": {
      "type": "http",
      "url": "http://localhost:9999/mcp/github"
    }
  }
}
EOF
  
  if ! bash "$SCRIPT_PATH" "github" "$config_file" "http://localhost:8080" 2>/dev/null; then
    print_result "Script detects wrong gateway URL" "PASS"
  else
    print_result "Script should detect wrong gateway URL" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 8: Valid gatewayed server
test_valid_gatewayed_server() {
  echo ""
  echo "Test 8: Valid gatewayed server"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create valid config
  cat > "$config_file" <<'EOF'
{
  "mcpServers": {
    "github": {
      "type": "http",
      "url": "http://localhost:8080/mcp/github",
      "tools": ["*"]
    }
  }
}
EOF
  
  if bash "$SCRIPT_PATH" "github" "$config_file" "http://localhost:8080" 2>/dev/null; then
    print_result "Script validates correct gatewayed server" "PASS"
  else
    print_result "Script should validate correct gatewayed server" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 9: Valid gatewayed server with authentication
test_valid_gatewayed_server_with_auth() {
  echo ""
  echo "Test 9: Valid gatewayed server with authentication"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create valid config with auth header
  cat > "$config_file" <<'EOF'
{
  "mcpServers": {
    "github": {
      "type": "http",
      "url": "http://localhost:8080/mcp/github",
      "tools": ["*"],
      "headers": {
        "Authorization": "Bearer test-key"
      }
    }
  }
}
EOF
  
  if bash "$SCRIPT_PATH" "github" "$config_file" "http://localhost:8080" 2>/dev/null; then
    print_result "Script validates gatewayed server with auth" "PASS"
  else
    print_result "Script should validate gatewayed server with auth" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 10: Multiple servers with mixed config
test_mixed_servers() {
  echo ""
  echo "Test 10: Multiple servers with mixed config"
  
  local tmpdir
  tmpdir=$(mktemp -d)
  local config_file="$tmpdir/config.json"
  
  # Create config with multiple servers
  cat > "$config_file" <<'EOF'
{
  "mcpServers": {
    "safeinputs": {
      "command": "gh",
      "args": ["aw", "mcp-server", "--mode", "safe-inputs"]
    },
    "safeoutputs": {
      "command": "gh",
      "args": ["aw", "mcp-server", "--mode", "safe-outputs"]
    },
    "github": {
      "type": "http",
      "url": "http://localhost:8080/mcp/github",
      "tools": ["*"]
    },
    "playwright": {
      "type": "http",
      "url": "http://localhost:8080/mcp/playwright",
      "tools": ["*"]
    }
  }
}
EOF
  
  # Validate github (should pass)
  if bash "$SCRIPT_PATH" "github" "$config_file" "http://localhost:8080" 2>/dev/null; then
    print_result "Script validates github in mixed config" "PASS"
  else
    print_result "Script should validate github in mixed config" "FAIL"
  fi
  
  # Validate playwright (should pass)
  if bash "$SCRIPT_PATH" "playwright" "$config_file" "http://localhost:8080" 2>/dev/null; then
    print_result "Script validates playwright in mixed config" "PASS"
  else
    print_result "Script should validate playwright in mixed config" "FAIL"
  fi
  
  # Validate safeinputs (should fail - not gatewayed)
  if ! bash "$SCRIPT_PATH" "safeinputs" "$config_file" "http://localhost:8080" 2>/dev/null; then
    print_result "Script detects safeinputs not gatewayed" "PASS"
  else
    print_result "Script should detect safeinputs not gatewayed" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 11: Functions exist
test_functions_exist() {
  echo ""
  echo "Test 11: Verify functions exist"
  
  # Check for main validation functions
  if grep -q "validate_config_file_exists()" "$SCRIPT_PATH"; then
    print_result "validate_config_file_exists function exists" "PASS"
  else
    print_result "validate_config_file_exists function missing" "FAIL"
  fi
  
  if grep -q "validate_server_exists()" "$SCRIPT_PATH"; then
    print_result "validate_server_exists function exists" "PASS"
  else
    print_result "validate_server_exists function missing" "FAIL"
  fi
  
  if grep -q "validate_server_config()" "$SCRIPT_PATH"; then
    print_result "validate_server_config function exists" "PASS"
  else
    print_result "validate_server_config function missing" "FAIL"
  fi
  
  if grep -q "validate_server_url()" "$SCRIPT_PATH"; then
    print_result "validate_server_url function exists" "PASS"
  else
    print_result "validate_server_url function missing" "FAIL"
  fi
  
  if grep -q "validate_server_type()" "$SCRIPT_PATH"; then
    print_result "validate_server_type function exists" "PASS"
  else
    print_result "validate_server_type function missing" "FAIL"
  fi
  
  if grep -q "validate_gateway_url()" "$SCRIPT_PATH"; then
    print_result "validate_gateway_url function exists" "PASS"
  else
    print_result "validate_gateway_url function missing" "FAIL"
  fi
}

# Run all tests
echo "=== Testing validate_gatewayed_server.sh ==="
echo "Script: $SCRIPT_PATH"

test_script_syntax
test_argument_validation
test_config_not_found
test_server_not_found
test_server_missing_url
test_server_wrong_type
test_url_wrong_gateway
test_valid_gatewayed_server
test_valid_gatewayed_server_with_auth
test_mixed_servers
test_functions_exist

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
