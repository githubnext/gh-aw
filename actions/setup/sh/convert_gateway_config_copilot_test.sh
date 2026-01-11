#!/bin/bash
# Test script for convert_gateway_config_copilot.sh
# Validates that the script correctly converts MCP gateway output to Copilot CLI format
# per MCP Gateway Specification v1.3.0 Section 5.4
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_PATH="$SCRIPT_DIR/convert_gateway_config_copilot.sh"

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

# Test 2: Script requires MCP_GATEWAY_OUTPUT environment variable
test_env_var_required() {
  echo ""
  echo "Test 2: MCP_GATEWAY_OUTPUT environment variable required"
  
  if ! bash "$SCRIPT_PATH" 2>/dev/null; then
    print_result "Script rejects missing MCP_GATEWAY_OUTPUT" "PASS"
  else
    print_result "Script should reject missing MCP_GATEWAY_OUTPUT" "FAIL"
  fi
}

# Test 3: Script rejects non-existent gateway output file
test_file_not_found() {
  echo ""
  echo "Test 3: Gateway output file not found"
  
  if ! MCP_GATEWAY_OUTPUT="/nonexistent/file.json" bash "$SCRIPT_PATH" 2>/dev/null; then
    print_result "Script rejects non-existent file" "PASS"
  else
    print_result "Script should reject non-existent file" "FAIL"
  fi
}

# Test 4: Converts gateway output with all required fields (per MCP Gateway Spec Section 5.4)
test_gateway_output_format() {
  echo ""
  echo "Test 4: Gateway output format conversion"
  
  local tmpdir=$(mktemp -d)
  local gateway_output="$tmpdir/gateway-output.json"
  local copilot_config="/home/runner/.copilot/mcp-config.json"
  
  # Create realistic gateway output per MCP Gateway Specification Section 5.4
  # Gateway MUST output servers with type=http, url, and headers with Authorization
  cat > "$gateway_output" <<'EOF'
{
  "mcpServers": {
    "github": {
      "type": "http",
      "url": "http://host.docker.internal:8080/mcp/github",
      "headers": {
        "Authorization": "test-api-key-12345"
      }
    },
    "playwright": {
      "type": "http",
      "url": "http://host.docker.internal:8080/mcp/playwright",
      "headers": {
        "Authorization": "test-api-key-12345"
      }
    }
  }
}
EOF
  
  # Run conversion
  mkdir -p /home/runner/.copilot
  if MCP_GATEWAY_OUTPUT="$gateway_output" bash "$SCRIPT_PATH" >/dev/null 2>&1; then
    # Verify output exists
    if [ -f "$copilot_config" ]; then
      # Verify it's valid JSON
      if jq empty "$copilot_config" 2>/dev/null; then
        print_result "Conversion produces valid JSON" "PASS"
      else
        print_result "Conversion should produce valid JSON" "FAIL"
      fi
    else
      print_result "Conversion should create output file" "FAIL"
    fi
  else
    print_result "Conversion should succeed" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 5: Preserves type field (should remain "http" from gateway)
test_preserves_type_field() {
  echo ""
  echo "Test 5: Preserves type=http from gateway output"
  
  local tmpdir=$(mktemp -d)
  local gateway_output="$tmpdir/gateway-output.json"
  local copilot_config="/home/runner/.copilot/mcp-config.json"
  
  cat > "$gateway_output" <<'EOF'
{
  "mcpServers": {
    "server1": {
      "type": "http",
      "url": "http://host.docker.internal:8080/mcp/server1",
      "headers": {
        "Authorization": "test-key"
      }
    }
  }
}
EOF
  
  mkdir -p /home/runner/.copilot
  MCP_GATEWAY_OUTPUT="$gateway_output" bash "$SCRIPT_PATH" >/dev/null 2>&1
  
  local type_value=$(jq -r '.mcpServers.server1.type' "$copilot_config")
  if [ "$type_value" = "http" ]; then
    print_result "Type field preserved as 'http'" "PASS"
  else
    print_result "Type field should be 'http', got: $type_value" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 6: Preserves url field exactly
test_preserves_url_field() {
  echo ""
  echo "Test 6: Preserves url field from gateway output"
  
  local tmpdir=$(mktemp -d)
  local gateway_output="$tmpdir/gateway-output.json"
  local copilot_config="/home/runner/.copilot/mcp-config.json"
  
  local expected_url="http://host.docker.internal:8080/mcp/testserver"
  
  cat > "$gateway_output" <<EOF
{
  "mcpServers": {
    "testserver": {
      "type": "http",
      "url": "$expected_url",
      "headers": {
        "Authorization": "test-key"
      }
    }
  }
}
EOF
  
  mkdir -p /home/runner/.copilot
  MCP_GATEWAY_OUTPUT="$gateway_output" bash "$SCRIPT_PATH" >/dev/null 2>&1
  
  local actual_url=$(jq -r '.mcpServers.testserver.url' "$copilot_config")
  if [ "$actual_url" = "$expected_url" ]; then
    print_result "URL field preserved exactly" "PASS"
  else
    print_result "URL should be '$expected_url', got: $actual_url" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 7: Preserves headers WITHOUT modification (per MCP Gateway Spec Section 5.4)
test_preserves_headers() {
  echo ""
  echo "Test 7: Preserves headers object without modification"
  
  local tmpdir=$(mktemp -d)
  local gateway_output="$tmpdir/gateway-output.json"
  local copilot_config="/home/runner/.copilot/mcp-config.json"
  
  cat > "$gateway_output" <<'EOF'
{
  "mcpServers": {
    "server1": {
      "type": "http",
      "url": "http://localhost:8080/mcp/server1",
      "headers": {
        "Authorization": "gateway-api-key-xyz",
        "X-Custom-Header": "custom-value",
        "Content-Type": "application/json"
      }
    }
  }
}
EOF
  
  mkdir -p /home/runner/.copilot
  MCP_GATEWAY_OUTPUT="$gateway_output" bash "$SCRIPT_PATH" >/dev/null 2>&1
  
  # Verify Authorization header preserved
  local auth_header=$(jq -r '.mcpServers.server1.headers.Authorization' "$copilot_config")
  if [ "$auth_header" = "gateway-api-key-xyz" ]; then
    print_result "Authorization header preserved" "PASS"
  else
    print_result "Authorization header should be preserved, got: $auth_header" "FAIL"
  fi
  
  # Verify custom header preserved
  local custom_header=$(jq -r '.mcpServers.server1.headers."X-Custom-Header"' "$copilot_config")
  if [ "$custom_header" = "custom-value" ]; then
    print_result "Custom headers preserved" "PASS"
  else
    print_result "Custom header should be preserved, got: $custom_header" "FAIL"
  fi
  
  # Verify Content-Type preserved
  local content_type=$(jq -r '.mcpServers.server1.headers."Content-Type"' "$copilot_config")
  if [ "$content_type" = "application/json" ]; then
    print_result "Content-Type header preserved" "PASS"
  else
    print_result "Content-Type should be preserved, got: $content_type" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 8: Adds tools field with wildcard (required for Copilot CLI)
test_adds_tools_field() {
  echo ""
  echo "Test 8: Adds tools field for Copilot CLI compatibility"
  
  local tmpdir=$(mktemp -d)
  local gateway_output="$tmpdir/gateway-output.json"
  local copilot_config="/home/runner/.copilot/mcp-config.json"
  
  # Gateway output without tools field
  cat > "$gateway_output" <<'EOF'
{
  "mcpServers": {
    "server1": {
      "type": "http",
      "url": "http://localhost:8080/mcp/server1",
      "headers": {
        "Authorization": "test-key"
      }
    }
  }
}
EOF
  
  mkdir -p /home/runner/.copilot
  MCP_GATEWAY_OUTPUT="$gateway_output" bash "$SCRIPT_PATH" >/dev/null 2>&1
  
  # Verify tools field was added
  local tools=$(jq -r '.mcpServers.server1.tools' "$copilot_config")
  if [ "$tools" != "null" ]; then
    # Verify it's an array with "*"
    local tools_array=$(jq -r '.mcpServers.server1.tools | type' "$copilot_config")
    local tools_content=$(jq -r '.mcpServers.server1.tools[0]' "$copilot_config")
    
    if [ "$tools_array" = "array" ] && [ "$tools_content" = "*" ]; then
      print_result "Tools field added correctly as ['*']" "PASS"
    else
      print_result "Tools field should be ['*'], got: $tools" "FAIL"
    fi
  else
    print_result "Tools field should be added" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 9: Preserves existing tools field if present
test_preserves_existing_tools() {
  echo ""
  echo "Test 9: Preserves existing tools field if present"
  
  local tmpdir=$(mktemp -d)
  local gateway_output="$tmpdir/gateway-output.json"
  local copilot_config="/home/runner/.copilot/mcp-config.json"
  
  # Gateway output WITH tools field
  cat > "$gateway_output" <<'EOF'
{
  "mcpServers": {
    "server1": {
      "type": "http",
      "url": "http://localhost:8080/mcp/server1",
      "headers": {
        "Authorization": "test-key"
      },
      "tools": ["tool1", "tool2"]
    }
  }
}
EOF
  
  mkdir -p /home/runner/.copilot
  MCP_GATEWAY_OUTPUT="$gateway_output" bash "$SCRIPT_PATH" >/dev/null 2>&1
  
  # Verify tools field was preserved
  local tool1=$(jq -r '.mcpServers.server1.tools[0]' "$copilot_config")
  local tool2=$(jq -r '.mcpServers.server1.tools[1]' "$copilot_config")
  
  if [ "$tool1" = "tool1" ] && [ "$tool2" = "tool2" ]; then
    print_result "Existing tools field preserved" "PASS"
  else
    print_result "Existing tools should be preserved, got: [$tool1, $tool2]" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 10: Handles multiple servers correctly
test_multiple_servers() {
  echo ""
  echo "Test 10: Handles multiple servers"
  
  local tmpdir=$(mktemp -d)
  local gateway_output="$tmpdir/gateway-output.json"
  local copilot_config="/home/runner/.copilot/mcp-config.json"
  
  cat > "$gateway_output" <<'EOF'
{
  "mcpServers": {
    "github": {
      "type": "http",
      "url": "http://localhost:8080/mcp/github",
      "headers": {
        "Authorization": "key1"
      }
    },
    "playwright": {
      "type": "http",
      "url": "http://localhost:8080/mcp/playwright",
      "headers": {
        "Authorization": "key2"
      }
    },
    "custom": {
      "type": "http",
      "url": "http://localhost:8080/mcp/custom",
      "headers": {
        "Authorization": "key3"
      }
    }
  }
}
EOF
  
  mkdir -p /home/runner/.copilot
  MCP_GATEWAY_OUTPUT="$gateway_output" bash "$SCRIPT_PATH" >/dev/null 2>&1
  
  # Count servers in output
  local server_count=$(jq -r '.mcpServers | keys | length' "$copilot_config")
  
  if [ "$server_count" = "3" ]; then
    # Verify all have tools field
    local github_tools=$(jq -r '.mcpServers.github.tools[0]' "$copilot_config")
    local playwright_tools=$(jq -r '.mcpServers.playwright.tools[0]' "$copilot_config")
    local custom_tools=$(jq -r '.mcpServers.custom.tools[0]' "$copilot_config")
    
    if [ "$github_tools" = "*" ] && [ "$playwright_tools" = "*" ] && [ "$custom_tools" = "*" ]; then
      print_result "All servers processed correctly" "PASS"
    else
      print_result "All servers should have tools field" "FAIL"
    fi
  else
    print_result "Should have 3 servers, got: $server_count" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Test 11: Output matches Copilot CLI required format
test_copilot_cli_format() {
  echo ""
  echo "Test 11: Output matches Copilot CLI required format"
  
  local tmpdir=$(mktemp -d)
  local gateway_output="$tmpdir/gateway-output.json"
  local copilot_config="/home/runner/.copilot/mcp-config.json"
  
  cat > "$gateway_output" <<'EOF'
{
  "mcpServers": {
    "test-server": {
      "type": "http",
      "url": "http://host.docker.internal:8080/mcp/test-server",
      "headers": {
        "Authorization": "test-api-key"
      }
    }
  }
}
EOF
  
  mkdir -p /home/runner/.copilot
  MCP_GATEWAY_OUTPUT="$gateway_output" bash "$SCRIPT_PATH" >/dev/null 2>&1
  
  # Verify all required Copilot CLI fields present
  local has_type=$(jq -e '.mcpServers."test-server".type' "$copilot_config" >/dev/null 2>&1 && echo "yes" || echo "no")
  local has_url=$(jq -e '.mcpServers."test-server".url' "$copilot_config" >/dev/null 2>&1 && echo "yes" || echo "no")
  local has_headers=$(jq -e '.mcpServers."test-server".headers' "$copilot_config" >/dev/null 2>&1 && echo "yes" || echo "no")
  local has_tools=$(jq -e '.mcpServers."test-server".tools' "$copilot_config" >/dev/null 2>&1 && echo "yes" || echo "no")
  
  if [ "$has_type" = "yes" ] && [ "$has_url" = "yes" ] && [ "$has_headers" = "yes" ] && [ "$has_tools" = "yes" ]; then
    print_result "All required Copilot CLI fields present" "PASS"
  else
    print_result "Missing required fields - type:$has_type url:$has_url headers:$has_headers tools:$has_tools" "FAIL"
  fi
  
  rm -rf "$tmpdir"
}

# Run all tests
echo "=== Testing convert_gateway_config_copilot.sh ==="
echo "Script: $SCRIPT_PATH"
echo ""
echo "Per MCP Gateway Specification v1.3.0 Section 5.4:"
echo "- Gateway outputs HTTP-type servers with url and Authorization header"
echo "- Copilot CLI requires 'tools' field to be added"
echo "- All other fields (type, url, headers) must be preserved"

test_script_syntax
test_env_var_required
test_file_not_found
test_gateway_output_format
test_preserves_type_field
test_preserves_url_field
test_preserves_headers
test_adds_tools_field
test_preserves_existing_tools
test_multiple_servers
test_copilot_cli_format

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
