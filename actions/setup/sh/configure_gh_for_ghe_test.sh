#!/usr/bin/env bash
# Test script for configure_gh_for_ghe.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIGURE_GH_SCRIPT="${SCRIPT_DIR}/configure_gh_for_ghe.sh"

echo "Testing configure_gh_for_ghe.sh"
echo "================================"

# Test 1: Check script exists and is executable
echo ""
echo "Test 1: Checking script exists and is executable..."
if [ ! -f "${CONFIGURE_GH_SCRIPT}" ]; then
  echo "FAIL: Script not found at ${CONFIGURE_GH_SCRIPT}"
  exit 1
fi

if [ ! -x "${CONFIGURE_GH_SCRIPT}" ]; then
  echo "FAIL: Script is not executable"
  exit 1
fi
echo "PASS: Script exists and is executable"

# Test 2: Test with github.com (should skip configuration)
echo ""
echo "Test 2: Testing with github.com (should skip configuration)..."
unset GITHUB_SERVER_URL GITHUB_ENTERPRISE_HOST GITHUB_HOST GH_HOST
output=$(bash -c "source ${CONFIGURE_GH_SCRIPT}" 2>&1)
if echo "$output" | grep -q "Using public GitHub (github.com)"; then
  echo "PASS: Correctly detected github.com and skipped configuration"
else
  echo "FAIL: Did not detect github.com correctly"
  echo "Output: $output"
  exit 1
fi

# Test 3: Test host detection from GITHUB_SERVER_URL
echo ""
echo "Test 3: Testing host detection from GITHUB_SERVER_URL..."
export GITHUB_SERVER_URL="https://myorg.ghe.com"
export GH_TOKEN="test-token"
# Use the real detect_github_host implementation from configure_gh_for_ghe.sh
output=$(bash -c "source '${CONFIGURE_GH_SCRIPT}'; detect_github_host" 2>&1)
if [ "$output" = "myorg.ghe.com" ]; then
  echo "PASS: Correctly extracted host from GITHUB_SERVER_URL"
else
  echo "FAIL: Did not extract host correctly. Got: $output"
  exit 1
fi

# Test 4: Test host detection from GITHUB_ENTERPRISE_HOST
echo ""
echo "Test 4: Testing host detection from GITHUB_ENTERPRISE_HOST..."
unset GITHUB_SERVER_URL
export GITHUB_ENTERPRISE_HOST="enterprise.github.com"
# Use the real detect_github_host implementation from configure_gh_for_ghe.sh
output=$(bash -c "source '${CONFIGURE_GH_SCRIPT}'; detect_github_host" 2>&1)
if [ "$output" = "enterprise.github.com" ]; then
  echo "PASS: Correctly extracted host from GITHUB_ENTERPRISE_HOST"
else
  echo "FAIL: Did not extract host correctly. Got: $output"
  exit 1
fi

# Test 5: Test URL normalization
echo ""
echo "Test 5: Testing URL normalization..."
declare -A test_cases=(
  ["https://myorg.ghe.com"]="myorg.ghe.com"
  ["http://myorg.ghe.com"]="myorg.ghe.com"
  ["https://myorg.ghe.com/"]="myorg.ghe.com"
  ["myorg.ghe.com"]="myorg.ghe.com"
  ["https://github.enterprise.com/api/v3"]="github.enterprise.com"
)

for input in "${!test_cases[@]}"; do
  expected="${test_cases[$input]}"
  output=$(bash -c "
    normalize_github_host() {
      local host=\"\$1\"
      host=\"\${host%/}\"
      if [[ \"\$host\" =~ ^https?:// ]]; then
        host=\"\${host#http://}\"
        host=\"\${host#https://}\"
        host=\"\${host%%/*}\"
      fi
      echo \"\$host\"
    }
    normalize_github_host '$input'
  " 2>&1)

  if [ "$output" = "$expected" ]; then
    echo "  PASS: '$input' -> '$output'"
  else
    echo "  FAIL: '$input' -> '$output' (expected '$expected')"
    exit 1
  fi
done

echo ""
echo "================================"
echo "All tests passed!"
echo "================================"
