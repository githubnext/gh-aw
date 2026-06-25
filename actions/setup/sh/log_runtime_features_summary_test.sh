#!/usr/bin/env bash
set +o histexpand

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/log_runtime_features_summary.sh"
SUMMARY_FILE="$(mktemp)"
trap 'rm -f "$SUMMARY_FILE"' EXIT

echo "Testing log_runtime_features_summary.sh..."
echo ""

# Case 1: non-empty GH_AW_RUNTIME_FEATURES -> writes heading + details block
echo "Test 1: non-empty value — should write heading and details"
export GH_AW_RUNTIME_FEATURES="feature1=on"
export GITHUB_STEP_SUMMARY="$SUMMARY_FILE"
bash "$SCRIPT"
if grep -q "### Runtime features" "$SUMMARY_FILE"; then
  echo "✅ Test 1a passed: heading written"
else
  echo "❌ Test 1a failed: missing heading"
  exit 1
fi
if grep -q "<details>" "$SUMMARY_FILE"; then
  echo "✅ Test 1b passed: wrapped in details block"
else
  echo "❌ Test 1b failed: missing details block"
  exit 1
fi
if grep -q "feature1=on" "$SUMMARY_FILE"; then
  echo "✅ Test 1c passed: feature value written"
else
  echo "❌ Test 1c failed: missing feature value"
  exit 1
fi
echo ""

# Case 2: empty GH_AW_RUNTIME_FEATURES -> no output written
echo "Test 2: empty value — should suppress output"
> "$SUMMARY_FILE"
export GH_AW_RUNTIME_FEATURES=""
bash "$SCRIPT"
if [[ ! -s "$SUMMARY_FILE" ]]; then
  echo "✅ Test 2 passed: no output when value is empty"
else
  echo "❌ Test 2 failed: unexpectedly wrote output for empty value"
  exit 1
fi
echo ""

# Case 3: unset GH_AW_RUNTIME_FEATURES -> no output written
echo "Test 3: unset GH_AW_RUNTIME_FEATURES — should suppress output"
> "$SUMMARY_FILE"
unset GH_AW_RUNTIME_FEATURES
bash "$SCRIPT"
if [[ ! -s "$SUMMARY_FILE" ]]; then
  echo "✅ Test 3 passed: no output when GH_AW_RUNTIME_FEATURES is unset"
else
  echo "❌ Test 3 failed: unexpectedly wrote output when GH_AW_RUNTIME_FEATURES is unset"
  exit 1
fi
echo ""

echo "🎉 All tests passed!"
