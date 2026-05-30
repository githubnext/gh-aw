#!/bin/bash
set +o histexpand

# Test script for validate_prompt_placeholders.sh

set -e

# Setup test environment
TEST_DIR=$(mktemp -d)
SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/validate_prompt_placeholders.sh"

cleanup() {
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

echo "Testing validate_prompt_placeholders.sh..."
echo ""

# Test 1: Valid prompt with no placeholders
echo "Test 1: Valid prompt with no placeholders"
cat > "$TEST_DIR/prompt.txt" << 'EOF'
<system>
# System Instructions
You are a helpful assistant.
</system>

# User Task
Please help me with this task.
Repository: github/gh-aw
Actor: octocat
EOF

export GH_AW_PROMPT="$TEST_DIR/prompt.txt"
if bash "$SCRIPT_PATH"; then
    echo "✅ Test 1 passed: Valid prompt accepted"
else
    echo "❌ Test 1 failed: Valid prompt rejected"
    exit 1
fi
echo ""

# Test 2: Prompt with unreplaced placeholders (should fail)
echo "Test 2: Prompt with unreplaced placeholders (should fail)"
cat > "$TEST_DIR/prompt_bad.txt" << 'EOF'
<system>
# System Instructions
You are a helpful assistant.
</system>

# User Task
Repository: __GH_AW_GITHUB_REPOSITORY__
Actor: __GH_AW_GITHUB_ACTOR__
EOF

export GH_AW_PROMPT="$TEST_DIR/prompt_bad.txt"
if bash "$SCRIPT_PATH" 2>&1; then
    echo "❌ Test 2 failed: Invalid prompt accepted"
    exit 1
else
    echo "✅ Test 2 passed: Invalid prompt rejected"
fi
echo ""

# Test 3: Missing prompt file (should fail)
echo "Test 3: Missing prompt file (should fail)"
export GH_AW_PROMPT="$TEST_DIR/nonexistent.txt"
if bash "$SCRIPT_PATH" 2>&1; then
    echo "❌ Test 3 failed: Missing file not detected"
    exit 1
else
    echo "✅ Test 3 passed: Missing file detected"
fi
echo ""

# Test 4: Prompt with GitHub expressions (warning but not error)
echo "Test 4: Prompt with GitHub expressions (warning)"
cat > "$TEST_DIR/prompt_expr.txt" << 'EOF'
<system>
# System Instructions
{{#if something}}
  Check: ${{ github.event.issue.number }}
{{/if}}
</system>

# User Task
Do something useful.
EOF

export GH_AW_PROMPT="$TEST_DIR/prompt_expr.txt"
OUTPUT=$(bash "$SCRIPT_PATH" 2>&1)
if echo "$OUTPUT" | grep -q "Warning"; then
    echo "✅ Test 4 passed: Warning shown for GitHub expressions"
else
    echo "⚠️  Test 4: No warning for GitHub expressions (may be acceptable)"
fi
echo ""

# Test 5: Prompt with placeholder name in backtick code span (should pass - it's documentation)
echo "Test 5: Prompt with placeholder in backtick code span (should pass)"
cat > "$TEST_DIR/prompt_backtick.txt" << 'EOF'
<system>
# System Instructions
You are a helpful assistant.
</system>

# User Task
This is a PR description that mentions `__GH_AW_MCP_CLI_SERVERS_LIST__` as documentation.
The value has already been substituted but the *name* appears in code formatting.
Also mentions `__GH_AW_GITHUB_ACTOR__` in inline code (safe).
EOF

export GH_AW_PROMPT="$TEST_DIR/prompt_backtick.txt"
if bash "$SCRIPT_PATH"; then
    echo "✅ Test 5 passed: Backtick-quoted placeholder names accepted"
else
    echo "❌ Test 5 failed: Backtick-quoted placeholder names incorrectly rejected"
    exit 1
fi
echo ""

echo "🎉 All validation tests passed!"
