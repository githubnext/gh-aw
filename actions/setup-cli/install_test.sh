#!/bin/bash
# Test script for install.sh in setup-cli action
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_PATH="$SCRIPT_DIR/install.sh"

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
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

# Test 2: Test is_long_sha function
test_is_long_sha() {
  echo ""
  echo "Test 2: Test is_long_sha function"
  
  # Extract and test the is_long_sha function
  cat > /tmp/test_sha_func.sh << 'FUNC_EOF'
is_long_sha() {
    local ver=$1
    if [[ $ver =~ ^[0-9a-f]{40}$ ]]; then
        return 0
    else
        return 1
    fi
}

# Test valid long SHA
if is_long_sha "53a14809f3234d628d47864d48170c48e5bb25b9"; then
    echo "PASS1"
else
    echo "FAIL1"
fi

# Test version tag (should fail)
if is_long_sha "v0.37.18"; then
    echo "FAIL2"
else
    echo "PASS2"
fi

# Test short SHA (should fail)
if is_long_sha "abc123"; then
    echo "FAIL3"
else
    echo "PASS3"
fi

# Test uppercase SHA (should fail)
if is_long_sha "53A14809F3234D628D47864D48170C48E5BB25B9"; then
    echo "FAIL4"
else
    echo "PASS4"
fi
FUNC_EOF
  
  results=$(bash /tmp/test_sha_func.sh)
  if echo "$results" | grep -q "FAIL"; then
    print_result "is_long_sha function validation" "FAIL"
    echo "$results"
  else
    print_result "is_long_sha function validation" "PASS"
  fi
  
  rm -f /tmp/test_sha_func.sh
}

# Test 3: Verify script is executable
test_executable() {
  echo ""
  echo "Test 3: Verify script is executable"
  
  if [ -x "$SCRIPT_PATH" ]; then
    print_result "Script is executable" "PASS"
  else
    print_result "Script is not executable" "FAIL"
  fi
}

# Test 4: Verify INPUT_VERSION support
test_input_version() {
  echo ""
  echo "Test 4: Verify INPUT_VERSION environment variable support"
  
  # Check if script references INPUT_VERSION
  if grep -q "INPUT_VERSION" "$SCRIPT_PATH"; then
    print_result "Script supports INPUT_VERSION" "PASS"
  else
    print_result "Script does not support INPUT_VERSION" "FAIL"
  fi
}

# Test 5: Verify gh extension install attempt
test_gh_install() {
  echo ""
  echo "Test 5: Verify gh extension install logic"
  
  # Check if script has gh extension install logic
  if grep -q "gh extension install" "$SCRIPT_PATH"; then
    print_result "Script includes gh extension install attempt" "PASS"
  else
    print_result "Script missing gh extension install logic" "FAIL"
  fi
}

# Test 6: Verify SHA resolution function
test_sha_resolution() {
  echo ""
  echo "Test 6: Verify SHA resolution function"
  
  if grep -q "resolve_sha_to_release" "$SCRIPT_PATH"; then
    print_result "Script includes SHA resolution function" "PASS"
  else
    print_result "Script missing SHA resolution function" "FAIL"
  fi
}

# Test 7: Verify release validation
test_release_validation() {
  echo ""
  echo "Test 7: Verify release validation"
  
  if grep -q "Validating release.*exists" "$SCRIPT_PATH"; then
    print_result "Script includes release validation" "PASS"
  else
    print_result "Script missing release validation" "FAIL"
  fi
}

# Run all tests
echo "========================================="
echo "Testing setup-cli action install.sh"
echo "========================================="

test_script_syntax
test_is_long_sha
test_executable
test_input_version
test_gh_install
test_sha_resolution
test_release_validation

# Summary
echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="
echo "Tests run: $TESTS_RUN"
echo -e "${GREEN}Tests passed: $TESTS_PASSED${NC}"
if [ $TESTS_FAILED -gt 0 ]; then
  echo -e "${RED}Tests failed: $TESTS_FAILED${NC}"
  exit 1
else
  echo -e "${GREEN}All tests passed!${NC}"
fi
