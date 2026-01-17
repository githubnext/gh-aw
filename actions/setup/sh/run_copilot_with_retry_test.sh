#!/usr/bin/env bash
#
# run_copilot_with_retry_test.sh - Tests for run_copilot_with_retry.sh
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="$SCRIPT_DIR/run_copilot_with_retry.sh"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test counters
tests_passed=0
tests_failed=0

# Test helper functions
assert_success() {
    local test_name="$1"
    local command="$2"
    
    echo -e "${YELLOW}Testing: $test_name${NC}"
    if eval "$command" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASS${NC}"
        tests_passed=$((tests_passed + 1))
    else
        echo -e "${RED}✗ FAIL${NC}"
        tests_failed=$((tests_failed + 1))
    fi
    echo
}

assert_failure() {
    local test_name="$1"
    local command="$2"
    
    echo -e "${YELLOW}Testing: $test_name${NC}"
    if ! eval "$command" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASS (expected failure)${NC}"
        tests_passed=$((tests_passed + 1))
    else
        echo -e "${RED}✗ FAIL (should have failed)${NC}"
        tests_failed=$((tests_failed + 1))
    fi
    echo
}

# Create mock commands for testing
create_mock_success() {
    cat > /tmp/mock_copilot_success.sh <<'EOF'
#!/bin/bash
echo "Copilot executing successfully"
exit 0
EOF
    chmod +x /tmp/mock_copilot_success.sh
}

create_mock_quick_finish_reason_fail() {
    cat > /tmp/mock_copilot_finish_reason_fail.sh <<'EOF'
#!/bin/bash
echo "Execution failed: Error: missing finish_reason for choice 0" >&2
exit 1
EOF
    chmod +x /tmp/mock_copilot_finish_reason_fail.sh
}

create_mock_slow_fail() {
    cat > /tmp/mock_copilot_slow_fail.sh <<'EOF'
#!/bin/bash
sleep 6
echo "Execution failed: Some other error" >&2
exit 1
EOF
    chmod +x /tmp/mock_copilot_slow_fail.sh
}

create_mock_fail_then_success() {
    cat > /tmp/mock_copilot_fail_then_success.sh <<'EOF'
#!/bin/bash
FLAG_FILE="/tmp/mock_copilot_state"
if [ -f "$FLAG_FILE" ]; then
    echo "Copilot executing successfully on retry"
    rm -f "$FLAG_FILE"
    exit 0
else
    touch "$FLAG_FILE"
    echo "Execution failed: Error: missing finish_reason for choice 0" >&2
    exit 1
fi
EOF
    chmod +x /tmp/mock_copilot_fail_then_success.sh
}

# Clean up function
cleanup() {
    rm -f /tmp/mock_copilot_*.sh /tmp/mock_copilot_state
}

trap cleanup EXIT

# Run tests
echo "======================================"
echo "Testing run_copilot_with_retry.sh"
echo "======================================"
echo

# Test 1: Script exists and is executable
assert_success "Script exists and is executable" "test -x '$SCRIPT'"

# Test 2: No arguments provided
assert_failure "No arguments (should fail)" "'$SCRIPT'"

# Test 3: Successful command
create_mock_success
assert_success "Successful command" "'$SCRIPT' /tmp/mock_copilot_success.sh"

# Test 4: Quick failure with finish_reason error (should retry and eventually fail)
create_mock_quick_finish_reason_fail
export COPILOT_RETRY_MAX_ATTEMPTS=2
export COPILOT_RETRY_DELAY=1
assert_failure "Quick finish_reason failure (retries then fails)" "'$SCRIPT' /tmp/mock_copilot_finish_reason_fail.sh"

# Test 5: Slow failure (should not retry)
create_mock_slow_fail
assert_failure "Slow failure (no retry)" "'$SCRIPT' /tmp/mock_copilot_slow_fail.sh"

# Test 6: Fail then success (should retry and succeed)
create_mock_fail_then_success
rm -f /tmp/mock_copilot_state
export COPILOT_RETRY_MAX_ATTEMPTS=3
export COPILOT_RETRY_DELAY=1
assert_success "Fail with finish_reason then succeed on retry" "'$SCRIPT' /tmp/mock_copilot_fail_then_success.sh"

# Summary
echo "======================================"
echo "Test Results"
echo "======================================"
echo -e "Passed: ${GREEN}$tests_passed${NC}"
echo -e "Failed: ${RED}$tests_failed${NC}"
echo

if [ $tests_failed -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
