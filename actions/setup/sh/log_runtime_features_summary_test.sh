#!/usr/bin/env bash
set +o histexpand

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/log_runtime_features_summary.sh"
SUMMARY_FILE="$(mktemp)"
trap 'rm -f "$SUMMARY_FILE"' EXIT

echo "Testing log_runtime_features_summary.sh..."
echo ""

# Case 1: IS_SET=false -> no output written
echo "Test 1: IS_SET=false — should suppress output"
export GH_AW_RUNTIME_FEATURES_IS_SET=false
export GH_AW_RUNTIME_FEATURES=""
export GITHUB_STEP_SUMMARY="$SUMMARY_FILE"
bash "$SCRIPT"
if [[ ! -s "$SUMMARY_FILE" ]]; then
  echo "✅ Test 1 passed: no output when IS_SET=false"
else
  echo "❌ Test 1 failed: unexpectedly wrote output when IS_SET=false"
  exit 1
fi
echo ""

# Case 2: IS_SET=true, non-empty value -> writes heading + details block
echo "Test 2: IS_SET=true, non-empty value — should write heading and details"
> "$SUMMARY_FILE"
export GH_AW_RUNTIME_FEATURES_IS_SET=true
export GH_AW_RUNTIME_FEATURES="feature1=on"
bash "$SCRIPT"
if grep -q "### Runtime features" "$SUMMARY_FILE"; then
  echo "✅ Test 2a passed: heading written"
else
  echo "❌ Test 2a failed: missing heading"
  exit 1
fi
if grep -q "<details>" "$SUMMARY_FILE"; then
  echo "✅ Test 2b passed: wrapped in details block"
else
  echo "❌ Test 2b failed: missing details block"
  exit 1
fi
if grep -q "feature1=on" "$SUMMARY_FILE"; then
  echo "✅ Test 2c passed: feature value written"
else
  echo "❌ Test 2c failed: missing feature value"
  exit 1
fi
echo ""

# Case 3: IS_SET=true, empty value -> no output written
echo "Test 3: IS_SET=true, empty value — should suppress output"
> "$SUMMARY_FILE"
export GH_AW_RUNTIME_FEATURES=""
bash "$SCRIPT"
if [[ ! -s "$SUMMARY_FILE" ]]; then
  echo "✅ Test 3 passed: no output when value is empty"
else
  echo "❌ Test 3 failed: unexpectedly wrote output for empty value"
  exit 1
fi
echo ""

# Case 4: IS_SET unset -> no output written (unset treated same as false)
echo "Test 4: IS_SET unset — should suppress output"
> "$SUMMARY_FILE"
unset GH_AW_RUNTIME_FEATURES_IS_SET
export GH_AW_RUNTIME_FEATURES="feature1=on"
bash "$SCRIPT"
if [[ ! -s "$SUMMARY_FILE" ]]; then
  echo "✅ Test 4 passed: no output when IS_SET is unset"
else
  echo "❌ Test 4 failed: unexpectedly wrote output when IS_SET is unset"
  exit 1
fi
echo ""

echo "🎉 All tests passed!"
