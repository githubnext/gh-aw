#!/bin/bash
# Test script for convert_gateway_config_codex.sh
# Validates that MCP gateway JSON output is properly converted to Codex TOML format

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_PATH="$SCRIPT_DIR/convert_gateway_config_codex.sh"

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

# Test 2: Missing MCP_GATEWAY_OUTPUT environment variable
test_missing_env_var() {
  echo ""
  echo "Test 2: Missing MCP_GATEWAY_OUTPUT environment variable"
  
  if ! bash "$SCRIPT_PATH" 2>/dev/null; then
    print_result "Script rejects missing MCP_GATEWAY_OUTPUT" "PASS"
  else
    print_result "Script should reject missing MCP_GATEWAY_OUTPUT" "FAIL"
  fi
}

# Test 3: Gateway output file not found
test_file_not_found() {
  echo ""
  echo "Test 3: Gateway output file not found"
  
  local tmpdir=$(mktemp -d)
  local nonexistent_file="$tmpdir/nonexistent.json"
  
  if ! MCP_GATEWAY_OUTPUT="$nonexistent_file" bash "$SCRIPT_PATH" 2>/dev/null; then
    print_result "Script rejects nonexistent gateway output file" "PASS"
  else
    print_result "Script should reject nonexistent gateway output file" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 4: Valid gateway output conversion - single server
test_valid_conversion_single() {
  echo ""
  echo "Test 4: Valid gateway output conversion - single server"
  
  local tmpdir=$(mktemp -d)
  local gateway_output="$tmpdir/gateway-output.json"
  local toml_output="/tmp/gh-aw/mcp-config/config.toml"
  
  # Create test directories
  mkdir -p /tmp/gh-aw/mcp-config
  
  # Create valid gateway JSON output with single server
  cat > "$gateway_output" << 'EOF'
{
  "mcpServers": {
    "github": {
      "type": "http",
      "url": "http://localhost:8080/mcp/github",
      "headers": {
        "Authorization": "Bearer test-api-key-123"
      }
    }
  }
}
EOF
  
  # Run conversion
  if MCP_GATEWAY_OUTPUT="$gateway_output" bash "$SCRIPT_PATH" >/dev/null 2>&1; then
    # Check if TOML file was created
    if [ -f "$toml_output" ]; then
      # Verify TOML content
      local toml_content=$(cat "$toml_output")
      
      # Check for required sections
      if echo "$toml_content" | grep -q '\[history\]' && \
         echo "$toml_content" | grep -q 'persistence = "none"' && \
         echo "$toml_content" | grep -q '\[mcp_servers.github\]' && \
         echo "$toml_content" | grep -q 'url = "http://localhost:8080/mcp/github"' && \
         echo "$toml_content" | grep -q '\[mcp_servers.github.headers\]' && \
         echo "$toml_content" | grep -q 'Authorization = "Bearer test-api-key-123"'; then
        print_result "Valid single server conversion produces correct TOML" "PASS"
      else
        echo "TOML content:"
        cat "$toml_output"
        print_result "TOML content is missing required sections" "FAIL"
      fi
    else
      print_result "TOML file was not created" "FAIL"
    fi
  else
    print_result "Conversion script failed on valid input" "FAIL"
  fi
  
  rm -rf "$tmpdir"
  rm -f "$toml_output"
}

# Test 5: Valid gateway output conversion - multiple servers
test_valid_conversion_multiple() {
  echo ""
  echo "Test 5: Valid gateway output conversion - multiple servers"
  
  local tmpdir=$(mktemp -d)
  local gateway_output="$tmpdir/gateway-output.json"
  local toml_output="/tmp/gh-aw/mcp-config/config.toml"
  
  # Create test directories
  mkdir -p /tmp/gh-aw/mcp-config
  
  # Create valid gateway JSON output with multiple servers
  cat > "$gateway_output" << 'EOF'
{
  "mcpServers": {
    "github": {
      "type": "http",
      "url": "http://localhost:8080/mcp/github",
      "headers": {
        "Authorization": "Bearer github-token"
      }
    },
    "playwright": {
      "type": "http",
      "url": "http://localhost:8080/mcp/playwright",
      "headers": {
        "Authorization": "Bearer playwright-token"
      }
    },
    "safe-outputs": {
      "type": "http",
      "url": "http://localhost:8080/mcp/safe-outputs",
      "headers": {
        "Authorization": "Bearer safeoutputs-token"
      }
    }
  }
}
EOF
  
  # Run conversion
  if MCP_GATEWAY_OUTPUT="$gateway_output" bash "$SCRIPT_PATH" >/dev/null 2>&1; then
    # Check if TOML file was created
    if [ -f "$toml_output" ]; then
      # Verify TOML content for all servers
      local toml_content=$(cat "$toml_output")
      
      # Check for all server sections
      if echo "$toml_content" | grep -q '\[mcp_servers.github\]' && \
         echo "$toml_content" | grep -q '\[mcp_servers.playwright\]' && \
         echo "$toml_content" | grep -q '\[mcp_servers.safe-outputs\]' && \
         echo "$toml_content" | grep -q 'url = "http://localhost:8080/mcp/github"' && \
         echo "$toml_content" | grep -q 'url = "http://localhost:8080/mcp/playwright"' && \
         echo "$toml_content" | grep -q 'url = "http://localhost:8080/mcp/safe-outputs"'; then
        print_result "Valid multiple servers conversion produces correct TOML" "PASS"
      else
        echo "TOML content:"
        cat "$toml_output"
        print_result "TOML content is missing server sections" "FAIL"
      fi
    else
      print_result "TOML file was not created" "FAIL"
    fi
  else
    print_result "Conversion script failed on valid input" "FAIL"
  fi
  
  rm -rf "$tmpdir"
  rm -f "$toml_output"
}

# Test 6: Gateway output with host.docker.internal domain
test_docker_internal_domain() {
  echo ""
  echo "Test 6: Gateway output with host.docker.internal domain"
  
  local tmpdir=$(mktemp -d)
  local gateway_output="$tmpdir/gateway-output.json"
  local toml_output="/tmp/gh-aw/mcp-config/config.toml"
  
  # Create test directories
  mkdir -p /tmp/gh-aw/mcp-config
  
  # Create gateway JSON output with host.docker.internal
  cat > "$gateway_output" << 'EOF'
{
  "mcpServers": {
    "github": {
      "type": "http",
      "url": "http://host.docker.internal:8080/mcp/github",
      "headers": {
        "Authorization": "Bearer test-key"
      }
    }
  }
}
EOF
  
  # Run conversion
  if MCP_GATEWAY_OUTPUT="$gateway_output" bash "$SCRIPT_PATH" >/dev/null 2>&1; then
    if [ -f "$toml_output" ]; then
      local toml_content=$(cat "$toml_output")
      
      # Verify host.docker.internal is preserved
      if echo "$toml_content" | grep -q 'url = "http://host.docker.internal:8080/mcp/github"'; then
        print_result "Docker internal domain is preserved in TOML" "PASS"
      else
        echo "TOML content:"
        cat "$toml_output"
        print_result "Docker internal domain not found in TOML" "FAIL"
      fi
    else
      print_result "TOML file was not created" "FAIL"
    fi
  else
    print_result "Conversion script failed" "FAIL"
  fi
  
  rm -rf "$tmpdir"
  rm -f "$toml_output"
}

# Test 7: TOML format structure validation
test_toml_format_structure() {
  echo ""
  echo "Test 7: TOML format structure validation"
  
  local tmpdir=$(mktemp -d)
  local gateway_output="$tmpdir/gateway-output.json"
  local toml_output="/tmp/gh-aw/mcp-config/config.toml"
  
  # Create test directories
  mkdir -p /tmp/gh-aw/mcp-config
  
  # Create valid gateway JSON output
  cat > "$gateway_output" << 'EOF'
{
  "mcpServers": {
    "test-server": {
      "type": "http",
      "url": "http://example.com:8080/mcp/test",
      "headers": {
        "Authorization": "Bearer test-token"
      }
    }
  }
}
EOF
  
  # Run conversion
  if MCP_GATEWAY_OUTPUT="$gateway_output" bash "$SCRIPT_PATH" >/dev/null 2>&1; then
    if [ -f "$toml_output" ]; then
      local toml_content=$(cat "$toml_output")
      
      # Verify TOML structure order: history section first, then server sections
      # Extract line numbers for each section
      local history_line=$(echo "$toml_content" | grep -n '\[history\]' | cut -d: -f1)
      local server_line=$(echo "$toml_content" | grep -n '\[mcp_servers.test-server\]' | cut -d: -f1)
      local headers_line=$(echo "$toml_content" | grep -n '\[mcp_servers.test-server.headers\]' | cut -d: -f1)
      
      if [ ! -z "$history_line" ] && [ ! -z "$server_line" ] && [ ! -z "$headers_line" ]; then
        if [ "$history_line" -lt "$server_line" ] && [ "$server_line" -lt "$headers_line" ]; then
          print_result "TOML structure follows correct section order" "PASS"
        else
          echo "Section order: history=$history_line, server=$server_line, headers=$headers_line"
          print_result "TOML sections are not in correct order" "FAIL"
        fi
      else
        print_result "TOML sections are missing" "FAIL"
      fi
    else
      print_result "TOML file was not created" "FAIL"
    fi
  else
    print_result "Conversion script failed" "FAIL"
  fi
  
  rm -rf "$tmpdir"
  rm -f "$toml_output"
}

# Test 8: Empty mcpServers object
test_empty_servers() {
  echo ""
  echo "Test 8: Empty mcpServers object"
  
  local tmpdir=$(mktemp -d)
  local gateway_output="$tmpdir/gateway-output.json"
  local toml_output="/tmp/gh-aw/mcp-config/config.toml"
  
  # Create test directories
  mkdir -p /tmp/gh-aw/mcp-config
  
  # Create gateway JSON output with empty servers
  cat > "$gateway_output" << 'EOF'
{
  "mcpServers": {}
}
EOF
  
  # Run conversion
  if MCP_GATEWAY_OUTPUT="$gateway_output" bash "$SCRIPT_PATH" >/dev/null 2>&1; then
    if [ -f "$toml_output" ]; then
      local toml_content=$(cat "$toml_output")
      
      # Should still have history section
      if echo "$toml_content" | grep -q '\[history\]'; then
        print_result "Empty servers produces valid TOML with history section" "PASS"
      else
        print_result "History section missing in empty servers TOML" "FAIL"
      fi
    else
      print_result "TOML file was not created" "FAIL"
    fi
  else
    print_result "Conversion script failed on empty servers" "FAIL"
  fi
  
  rm -rf "$tmpdir"
  rm -f "$toml_output"
}

# Test 9: Server with special characters in name
test_special_chars_in_name() {
  echo ""
  echo "Test 9: Server with special characters in name"
  
  local tmpdir=$(mktemp -d)
  local gateway_output="$tmpdir/gateway-output.json"
  local toml_output="/tmp/gh-aw/mcp-config/config.toml"
  
  # Create test directories
  mkdir -p /tmp/gh-aw/mcp-config
  
  # Create gateway JSON output with special chars
  cat > "$gateway_output" << 'EOF'
{
  "mcpServers": {
    "custom-server_v2": {
      "type": "http",
      "url": "http://localhost:8080/mcp/custom-server_v2",
      "headers": {
        "Authorization": "Bearer token-123"
      }
    }
  }
}
EOF
  
  # Run conversion
  if MCP_GATEWAY_OUTPUT="$gateway_output" bash "$SCRIPT_PATH" >/dev/null 2>&1; then
    if [ -f "$toml_output" ]; then
      local toml_content=$(cat "$toml_output")
      
      # Verify server name with special characters
      if echo "$toml_content" | grep -q '\[mcp_servers.custom-server_v2\]'; then
        print_result "Server name with special characters is handled correctly" "PASS"
      else
        echo "TOML content:"
        cat "$toml_output"
        print_result "Server name with special characters not found" "FAIL"
      fi
    else
      print_result "TOML file was not created" "FAIL"
    fi
  else
    print_result "Conversion script failed" "FAIL"
  fi
  
  rm -rf "$tmpdir"
  rm -f "$toml_output"
}

# Test 10: Verify TOML values are properly quoted
test_toml_value_quoting() {
  echo ""
  echo "Test 10: Verify TOML values are properly quoted"
  
  local tmpdir=$(mktemp -d)
  local gateway_output="$tmpdir/gateway-output.json"
  local toml_output="/tmp/gh-aw/mcp-config/config.toml"
  
  # Create test directories
  mkdir -p /tmp/gh-aw/mcp-config
  
  # Create gateway JSON output
  cat > "$gateway_output" << 'EOF'
{
  "mcpServers": {
    "github": {
      "type": "http",
      "url": "http://localhost:8080/mcp/github",
      "headers": {
        "Authorization": "Bearer token"
      }
    }
  }
}
EOF
  
  # Run conversion
  if MCP_GATEWAY_OUTPUT="$gateway_output" bash "$SCRIPT_PATH" >/dev/null 2>&1; then
    if [ -f "$toml_output" ]; then
      local toml_content=$(cat "$toml_output")
      
      # Check that string values are properly quoted
      if echo "$toml_content" | grep -q 'persistence = "none"' && \
         echo "$toml_content" | grep -q 'url = "http://localhost:8080/mcp/github"' && \
         echo "$toml_content" | grep -q 'Authorization = "Bearer token"'; then
        print_result "TOML string values are properly quoted" "PASS"
      else
        echo "TOML content:"
        cat "$toml_output"
        print_result "TOML values are not properly quoted" "FAIL"
      fi
    else
      print_result "TOML file was not created" "FAIL"
    fi
  else
    print_result "Conversion script failed" "FAIL"
  fi
  
  rm -rf "$tmpdir"
  rm -f "$toml_output"
}

# Run all tests
echo "=========================================="
echo "Testing convert_gateway_config_codex.sh"
echo "=========================================="

test_script_syntax
test_missing_env_var
test_file_not_found
test_valid_conversion_single
test_valid_conversion_multiple
test_docker_internal_domain
test_toml_format_structure
test_empty_servers
test_special_chars_in_name
test_toml_value_quoting

# Print summary
echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo "Tests run:    $TESTS_RUN"
echo -e "${GREEN}Tests passed: $TESTS_PASSED${NC}"
if [ $TESTS_FAILED -gt 0 ]; then
  echo -e "${RED}Tests failed: $TESTS_FAILED${NC}"
  exit 1
else
  echo "Tests failed: $TESTS_FAILED"
  echo ""
  echo -e "${GREEN}All tests passed!${NC}"
  exit 0
fi
