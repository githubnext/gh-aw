#!/usr/bin/env bash
set +o histexpand

# Tests for start_difc_proxy.sh retry behavior.
# Run: bash start_difc_proxy_test.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
START_SCRIPT="${SCRIPT_DIR}/start_difc_proxy.sh"

TESTS_PASSED=0
TESTS_FAILED=0

pass() { echo "PASS: $1"; TESTS_PASSED=$((TESTS_PASSED + 1)); }
fail() { echo "FAIL: $1"; echo "  $2"; TESTS_FAILED=$((TESTS_FAILED + 1)); }

run_start_proxy() {
  local docker_mode="$1"
  local sandbox
  sandbox=$(mktemp -d)

  rm -rf /tmp/gh-aw/proxy-logs /tmp/gh-aw/mcp-logs
  mkdir -p "${sandbox}/bin"

  cat >"${sandbox}/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "$*" >>"${DOCKER_LOG}"

case "$1" in
  image)
    exit 1
    ;;
  pull)
    count=0
    if [ -f "${PULL_COUNT_FILE}" ]; then
      count=$(cat "${PULL_COUNT_FILE}")
    fi
    count=$((count + 1))
    echo "$count" >"${PULL_COUNT_FILE}"
    if { [ "${DOCKER_MODE}" = "pull-succeeds-second" ] && [ "$count" -ge 2 ]; } || [ "${DOCKER_MODE}" = "run-succeeds-second" ]; then
      echo "mock pull success"
      exit 0
    fi
    echo "mock pull failure" >&2
    exit 1
    ;;
  run)
    count=0
    if [ -f "${RUN_COUNT_FILE}" ]; then
      count=$(cat "${RUN_COUNT_FILE}")
    fi
    count=$((count + 1))
    echo "$count" >"${RUN_COUNT_FILE}"
    if [ "${DOCKER_MODE}" = "run-succeeds-second" ] && [ "$count" -eq 1 ]; then
      echo "mock run failure" >&2
      exit 125
    fi
    mkdir -p /tmp/gh-aw/proxy-logs/proxy-tls
    printf 'mock ca\n' >/tmp/gh-aw/proxy-logs/proxy-tls/ca.crt
    echo "mock-container-id"
    exit 0
    ;;
  rm)
    exit 0
    ;;
  logs)
    echo "mock proxy logs"
    exit 0
    ;;
  *)
    echo "unexpected docker command: $*" >&2
    exit 1
    ;;
esac
EOF

  cat >"${sandbox}/bin/curl" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF

  cat >"${sandbox}/bin/git" <<'EOF'
#!/usr/bin/env bash
echo "$*" >>"${GIT_LOG}"
exit 0
EOF

  cat >"${sandbox}/bin/sleep" <<'EOF'
#!/usr/bin/env bash
echo "$*" >>"${SLEEP_LOG}"
exit 0
EOF

  cat >"${sandbox}/bin/sudo" <<'EOF'
#!/usr/bin/env bash
case "$1" in
  cp)
    shift
    cp "$@"
    ;;
  update-ca-certificates)
    exit 0
    ;;
  *)
    "$@"
    ;;
esac
EOF

  chmod +x "${sandbox}/bin/docker" "${sandbox}/bin/curl" "${sandbox}/bin/git" "${sandbox}/bin/sleep" "${sandbox}/bin/sudo"

  DOCKER_LOG="${sandbox}/docker.log"
  GIT_LOG="${sandbox}/git.log"
  SLEEP_LOG="${sandbox}/sleep.log"
  PULL_COUNT_FILE="${sandbox}/pull-count"
  RUN_COUNT_FILE="${sandbox}/run-count"
  RUN_OUTPUT_FILE="${sandbox}/run-output.log"
  : >"${DOCKER_LOG}"
  : >"${GIT_LOG}"
  : >"${SLEEP_LOG}"

  set +e
  env PATH="${sandbox}/bin:${PATH}" \
    DOCKER_MODE="${docker_mode}" \
    DOCKER_LOG="${DOCKER_LOG}" \
    GIT_LOG="${GIT_LOG}" \
    SLEEP_LOG="${SLEEP_LOG}" \
    PULL_COUNT_FILE="${PULL_COUNT_FILE}" \
    RUN_COUNT_FILE="${RUN_COUNT_FILE}" \
    DIFC_PROXY_POLICY='{"rules":[]}' \
    DIFC_PROXY_IMAGE='ghcr.io/github/gh-aw-mcpg:v0.4.9' \
    GH_TOKEN='test-token' \
    GITHUB_SERVER_URL='https://github.com' \
    GITHUB_REPOSITORY='github/gh-aw' \
    bash "${START_SCRIPT}" >"${RUN_OUTPUT_FILE}" 2>&1
  RUN_STATUS=$?
  set -e

  RUN_OUTPUT=$(cat "${RUN_OUTPUT_FILE}")
  RUN_DOCKER_LOG=$(cat "${DOCKER_LOG}")
  RUN_GIT_LOG=$(cat "${GIT_LOG}")
  RUN_SLEEP_LOG=$(cat "${SLEEP_LOG}")

  rm -rf "${sandbox}"
}

echo "Running start_difc_proxy.sh tests..."
echo

echo "Test 1: image pull is retried before starting the proxy..."
run_start_proxy pull-succeeds-second
if [ "${RUN_STATUS}" -ne 0 ]; then
  fail "Proxy start succeeds after a retried pull" "script exited with ${RUN_STATUS}: ${RUN_OUTPUT}"
elif ! echo "${RUN_OUTPUT}" | grep -qF "DIFC proxy image pull attempt 1 of 3 failed with exit code 1"; then
  fail "Pull retry failure is logged" "missing retry failure log: ${RUN_OUTPUT}"
elif ! echo "${RUN_OUTPUT}" | grep -qF "Retrying DIFC proxy image pull in 5s"; then
  fail "Pull retry backoff is logged" "missing backoff log: ${RUN_OUTPUT}"
elif ! echo "${RUN_OUTPUT}" | grep -qF "Successfully pulled DIFC proxy image"; then
  fail "Successful pull is logged" "missing successful pull log: ${RUN_OUTPUT}"
elif ! echo "${RUN_DOCKER_LOG}" | grep -qF "run --pull=never -d --name awmg-proxy"; then
  fail "Docker run uses the pre-pulled image" "docker log: ${RUN_DOCKER_LOG}"
elif ! echo "${RUN_GIT_LOG}" | grep -qF "remote add proxy https://localhost:18443/github/gh-aw.git"; then
  fail "Proxy remote is added after health check" "git log: ${RUN_GIT_LOG}"
else
  pass "Proxy start succeeds after a retried pull"
fi

echo "Test 2: container start is retried after a transient docker run failure..."
run_start_proxy run-succeeds-second
if [ "${RUN_STATUS}" -ne 0 ]; then
  fail "Proxy start succeeds after a retried docker run" "script exited with ${RUN_STATUS}: ${RUN_OUTPUT}"
elif ! echo "${RUN_OUTPUT}" | grep -qF "DIFC proxy container start attempt 1 of 3 failed with exit code 125"; then
  fail "Docker run retry failure is logged" "missing run retry failure log: ${RUN_OUTPUT}"
elif ! echo "${RUN_OUTPUT}" | grep -qF "mock run failure"; then
  fail "Docker run stderr is surfaced" "missing docker run stderr: ${RUN_OUTPUT}"
elif ! echo "${RUN_OUTPUT}" | grep -qF "Retrying DIFC proxy container start in 5s"; then
  fail "Docker run retry backoff is logged" "missing run backoff log: ${RUN_OUTPUT}"
else
  pass "Proxy start succeeds after a retried docker run"
fi

echo "Test 3: exhausted image pull retries fail with Docker output..."
run_start_proxy pull-always-fails
if [ "${RUN_STATUS}" -eq 0 ]; then
  fail "Proxy start fails after exhausted pull retries" "script unexpectedly succeeded: ${RUN_OUTPUT}"
elif ! echo "${RUN_OUTPUT}" | grep -qF "Failed to pull DIFC proxy image after 3 attempts"; then
  fail "Final pull failure is logged" "missing final failure log: ${RUN_OUTPUT}"
elif ! echo "${RUN_OUTPUT}" | grep -qF "mock pull failure"; then
  fail "Docker pull stderr is surfaced" "missing docker stderr: ${RUN_OUTPUT}"
elif echo "${RUN_DOCKER_LOG}" | grep -qF "run "; then
  fail "Docker run is skipped after pull failure" "docker log: ${RUN_DOCKER_LOG}"
else
  pass "Proxy start fails after exhausted pull retries"
fi

echo
echo "==============================="
echo "Tests passed: $TESTS_PASSED"
echo "Tests failed: $TESTS_FAILED"
echo "==============================="

rm -rf /tmp/gh-aw/proxy-logs /tmp/gh-aw/mcp-logs

if [ $TESTS_FAILED -gt 0 ]; then
  exit 1
fi

exit 0
