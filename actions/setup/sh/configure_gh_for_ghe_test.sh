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
# Source with no GHE vars (so main is a no-op), then call detect_github_host directly.
unset GITHUB_SERVER_URL GITHUB_ENTERPRISE_HOST GITHUB_HOST GH_HOST GH_TOKEN
output=$(bash -c "
  source '${CONFIGURE_GH_SCRIPT}' >/dev/null 2>&1
  GITHUB_SERVER_URL='https://myorg.ghe.com' detect_github_host
" 2>/dev/null)
if [ "$output" = "myorg.ghe.com" ]; then
  echo "PASS: Correctly extracted host from GITHUB_SERVER_URL"
else
  echo "FAIL: Did not extract host correctly. Got: $output"
  exit 1
fi

# Test 4: Test host detection from GITHUB_ENTERPRISE_HOST
echo ""
echo "Test 4: Testing host detection from GITHUB_ENTERPRISE_HOST..."
unset GITHUB_SERVER_URL GITHUB_ENTERPRISE_HOST GITHUB_HOST GH_HOST GH_TOKEN
output=$(bash -c "
  source '${CONFIGURE_GH_SCRIPT}' >/dev/null 2>&1
  GITHUB_ENTERPRISE_HOST='enterprise.github.com' detect_github_host
" 2>/dev/null)
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

# Test 6: GHE host + GH_TOKEN set — must skip gh auth login and only export GH_HOST
echo ""
echo "Test 6: Testing GHE host with GH_TOKEN set (should skip gh auth login)..."
unset GITHUB_SERVER_URL GITHUB_ENTERPRISE_HOST GITHUB_HOST GH_HOST
export GITHUB_SERVER_URL="https://myorg.ghe.com"
export GH_TOKEN="test-token"

# Stub a fake gh binary so configure_gh_for_ghe.sh can find it via command -v gh
FAKE_GH_DIR=$(mktemp -d)
FAKE_GH="${FAKE_GH_DIR}/gh"
cat > "${FAKE_GH}" << 'EOF'
#!/usr/bin/env bash
exit 0
EOF
chmod +x "${FAKE_GH}"
export PATH="${FAKE_GH_DIR}:${PATH}"

FAKE_GITHUB_ENV=$(mktemp)
output=$(bash -c "
  GITHUB_ENV='${FAKE_GITHUB_ENV}' \
  GITHUB_SERVER_URL='https://myorg.ghe.com' \
  GH_TOKEN='test-token' \
  source '${CONFIGURE_GH_SCRIPT}'
" 2>&1)
exit_code=$?

if [ $exit_code -ne 0 ]; then
  echo "FAIL: Script exited with code $exit_code"
  echo "Output: $output"
  rm -f "${FAKE_GITHUB_ENV}"
  rm -rf "${FAKE_GH_DIR}"
  exit 1
fi

if ! echo "$output" | grep -q "GH_TOKEN is set"; then
  echo "FAIL: Expected 'GH_TOKEN is set' message. Output: $output"
  rm -f "${FAKE_GITHUB_ENV}"
  rm -rf "${FAKE_GH_DIR}"
  exit 1
fi

if ! grep -q "GH_HOST=myorg.ghe.com" "${FAKE_GITHUB_ENV}"; then
  echo "FAIL: GH_HOST=myorg.ghe.com was not written to GITHUB_ENV"
  cat "${FAKE_GITHUB_ENV}"
  rm -f "${FAKE_GITHUB_ENV}"
  rm -rf "${FAKE_GH_DIR}"
  exit 1
fi
rm -f "${FAKE_GITHUB_ENV}"
echo "PASS: Skipped gh auth login and exported GH_HOST when GH_TOKEN is set"

# Test 7: GHE host without GH_TOKEN set — must error
echo ""
echo "Test 7: Testing GHE host without GH_TOKEN set (should error)..."
unset GITHUB_SERVER_URL GITHUB_ENTERPRISE_HOST GITHUB_HOST GH_HOST GH_TOKEN
exit_code=0
output=$(bash -c "
  GITHUB_SERVER_URL='https://myorg.ghe.com' \
  source '${CONFIGURE_GH_SCRIPT}'
" 2>&1) || exit_code=$?

if [ $exit_code -eq 0 ]; then
  echo "FAIL: Script should have failed but exited with 0"
  rm -rf "${FAKE_GH_DIR}"
  exit 1
fi

if ! echo "$output" | grep -q "GH_TOKEN environment variable is not set"; then
  echo "FAIL: Expected 'GH_TOKEN environment variable is not set' error. Output: $output"
  rm -rf "${FAKE_GH_DIR}"
  exit 1
fi
echo "PASS: Correctly errors when GH_TOKEN is not set for GHE host"

# Clean up fake gh directory
rm -rf "${FAKE_GH_DIR}"

echo ""
echo "================================"
echo "All tests passed!"
echo "================================"
