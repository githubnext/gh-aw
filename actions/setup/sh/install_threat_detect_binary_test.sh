#!/usr/bin/env bash
set +o histexpand

# Tests for install_threat_detect_binary.sh platform detection and asset-name mapping.
# Run: bash install_threat_detect_binary_test.sh
#
# The tests run the real script with a stubbed `uname` (to fake the platform) and a
# stubbed `curl` (to record the requested asset URL and serve a fake binary plus a
# matching checksums.txt), so no network access is required.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_SCRIPT="${SCRIPT_DIR}/install_threat_detect_binary.sh"

TESTS_PASSED=0
TESTS_FAILED=0

pass() { echo "PASS: $1"; TESTS_PASSED=$((TESTS_PASSED + 1)); }
fail() { echo "FAIL: $1"; echo "  $2"; TESTS_FAILED=$((TESTS_FAILED + 1)); }

# run_installer OS ARCH VERSION -> runs the installer in an isolated sandbox.
# Sets globals: RUN_OUTPUT (stdout+stderr), RUN_STATUS (exit code), RUN_ASSET (downloaded asset name).
run_installer() {
  local fake_os="$1"
  local fake_arch="$2"
  local version="$3"

  local sandbox
  sandbox=$(mktemp -d)

  mkdir -p "${sandbox}/bin"

  # Stubbed uname reports the platform under test.
  cat >"${sandbox}/bin/uname" <<EOF
#!/usr/bin/env bash
case "\$1" in
  -s) echo "${fake_os}" ;;
  -m) echo "${fake_arch}" ;;
  *) echo "${fake_os}" ;;
esac
EOF

  # Stubbed curl records the requested asset and serves a fake binary + checksums.txt.
  cat >"${sandbox}/bin/curl" <<'EOF'
#!/usr/bin/env bash
out=""
url=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "-o" ]; then
    out="$arg"
  elif [ "${arg#-}" = "$arg" ]; then
    url="$arg"
  fi
  prev="$arg"
done

payload='#!/usr/bin/env bash
echo "fake threat-detect"'

name="${url##*/}"
echo "${url}" >>"${SANDBOX_URL_LOG}"
if [ "$name" = "checksums.txt" ]; then
  hash=$(printf '%s\n' "$payload" | sha256sum | awk '{print $1}')
  for asset in threat-detect-linux-amd64 threat-detect-linux-arm64; do
    echo "${hash}  ${asset}"
  done >"$out"
else
  echo "$name" >>"${SANDBOX_ASSET_LOG}"
  printf '%s\n' "$payload" >"$out"
fi
EOF

  chmod +x "${sandbox}/bin/uname" "${sandbox}/bin/curl"

  local asset_log="${sandbox}/asset.log"
  local url_log="${sandbox}/url.log"
  : >"${asset_log}"
  : >"${url_log}"

  RUN_OUTPUT=$(cd "${sandbox}" && env PATH="${sandbox}/bin:${PATH}" HOME="${sandbox}" \
    SANDBOX_ASSET_LOG="${asset_log}" SANDBOX_URL_LOG="${url_log}" GITHUB_PATH="" \
    bash "${INSTALL_SCRIPT}" "${version}" --rootless 2>&1)
  RUN_STATUS=$?
  RUN_ASSET=$(cat "${asset_log}")
  RUN_URLS=$(cat "${url_log}")

  rm -rf "${sandbox}"
}

assert_asset() {
  local description="$1"
  local fake_os="$2"
  local fake_arch="$3"
  local expected="$4"

  run_installer "${fake_os}" "${fake_arch}" v0.4.0
  if [ "${RUN_STATUS}" -ne 0 ]; then
    fail "${description}" "installer exited with ${RUN_STATUS}: ${RUN_OUTPUT}"
  elif [ "${RUN_ASSET}" = "${expected}" ]; then
    pass "${description}"
  else
    fail "${description}" "expected asset ${expected}, got '${RUN_ASSET}'"
  fi
}

assert_failure() {
  local description="$1"
  local fake_os="$2"
  local fake_arch="$3"
  local expected_msg="$4"

  run_installer "${fake_os}" "${fake_arch}" v0.4.0
  if [ "${RUN_STATUS}" -eq 0 ]; then
    fail "${description}" "installer unexpectedly succeeded: ${RUN_OUTPUT}"
  elif ! echo "${RUN_OUTPUT}" | grep -qF "${expected_msg}"; then
    fail "${description}" "expected message '${expected_msg}' in: ${RUN_OUTPUT}"
  elif [ -n "${RUN_ASSET}" ]; then
    fail "${description}" "installer attempted a binary download: ${RUN_ASSET}"
  else
    pass "${description}"
  fi
}

echo "Running install_threat_detect_binary.sh tests..."
echo

# Test 1-3: Linux architecture mapping
echo "Test 1: Linux/x86_64 maps to threat-detect-linux-amd64..."
assert_asset "Linux/x86_64 -> threat-detect-linux-amd64" Linux x86_64 threat-detect-linux-amd64

echo "Test 2: Linux/aarch64 maps to threat-detect-linux-arm64..."
assert_asset "Linux/aarch64 -> threat-detect-linux-arm64" Linux aarch64 threat-detect-linux-arm64

echo "Test 3: Linux/arm64 maps to threat-detect-linux-arm64..."
assert_asset "Linux/arm64 -> threat-detect-linux-arm64" Linux arm64 threat-detect-linux-arm64

# Test 4-5: macOS is not supported and must fail fast without downloading anything
echo "Test 4: Darwin/arm64 fails fast as unsupported..."
assert_failure "Darwin/arm64 is rejected" Darwin arm64 "macOS is not a supported platform"

echo "Test 5: Darwin/x86_64 fails fast as unsupported..."
assert_failure "Darwin/x86_64 is rejected" Darwin x86_64 "macOS is not a supported platform"

# Test 6: unknown OS fails fast
echo "Test 6: unknown operating system fails fast..."
assert_failure "Unknown OS is rejected" FreeBSD x86_64 "Unsupported operating system"

# Test 7: unknown architecture fails with an actionable message
echo "Test 7: unknown Linux architecture fails fast..."
assert_failure "Unknown Linux architecture is rejected" Linux riscv64 "Unsupported Linux architecture"

# Test 8: latest release assets must be downloaded directly without the GitHub API.
echo "Test 8: latest release assets use direct downloads without the GitHub API..."
run_installer Linux x86_64 latest
if [ "${RUN_STATUS}" -ne 0 ]; then
  fail "Latest release installer succeeds" "installer exited with ${RUN_STATUS}: ${RUN_OUTPUT}"
elif ! echo "${RUN_URLS}" | grep -qF "https://github.com/github/gh-aw-threat-detection/releases/latest/download/checksums.txt"; then
  fail "Latest release installer downloads checksums directly" "expected latest release checksum URL, got: ${RUN_URLS}"
elif ! echo "${RUN_URLS}" | grep -qF "https://github.com/github/gh-aw-threat-detection/releases/latest/download/threat-detect-linux-amd64"; then
  fail "Latest release installer downloads the binary directly" "expected latest release binary URL, got: ${RUN_URLS}"
elif echo "${RUN_URLS}" | grep -qF "api.github.com"; then
  fail "Latest release installer avoids GitHub API" "unexpected API URL: ${RUN_URLS}"
else
  pass "Latest release installer uses direct downloads without GitHub API"
fi

echo
echo "==============================="
echo "Tests passed: $TESTS_PASSED"
echo "Tests failed: $TESTS_FAILED"
echo "==============================="

if [ $TESTS_FAILED -gt 0 ]; then
  exit 1
fi

exit 0
