#!/bin/bash
# Test script for generate_git_patch.sh
# This script validates the patch generation logic in different scenarios

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PATCH_SCRIPT="$SCRIPT_DIR/generate_git_patch.sh"

# Test helper function
assert_contains() {
  local output="$1"
  local expected="$2"
  if echo "$output" | grep -q "$expected"; then
    echo "✓ Output contains: $expected"
  else
    echo "✗ Expected output to contain: $expected"
    echo "Actual output:"
    echo "$output"
    exit 1
  fi
}

echo "=== Testing generate_git_patch.sh ==="

# Test 1: Script should handle missing branch gracefully
echo "Test 1: Handle missing branch name from safe-outputs"
cat > /tmp/test-safe-outputs.jsonl << 'EOF'
{"type":"push_to_pull_request_branch","branch":"nonexistent-branch","message":"test commit"}
EOF

# Create a simple git repo for testing
TEST_DIR=$(mktemp -d)
cd "$TEST_DIR"
git init
git config user.email "test@example.com"
git config user.name "Test User"
echo "test" > file.txt
git add file.txt
git commit -m "Initial commit"

export GH_AW_SAFE_OUTPUTS="/tmp/test-safe-outputs.jsonl"
export GITHUB_SHA="HEAD"

# Source the script (capturing output)
OUTPUT=$(bash "$PATCH_SCRIPT" 2>&1)

assert_contains "$OUTPUT" "Branch name from safe-outputs: nonexistent-branch"
assert_contains "$OUTPUT" "does not exist locally, falling back to current HEAD"

echo "✓ Test 1 passed"

# Clean up
cd /
rm -rf "$TEST_DIR"
rm -f /tmp/test-safe-outputs.jsonl

echo "=== All tests passed ==="
