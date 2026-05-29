#!/usr/bin/env bash
set +o histexpand

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/check_cache_memory_git_integrity.sh"

TESTS_PASSED=0
TESTS_FAILED=0
WORKSPACE="$(mktemp -d)"

cleanup() {
  rm -rf "${WORKSPACE}"
}
trap cleanup EXIT

assert() {
  local name="$1"
  local condition="$2"
  if eval "${condition}" 2>/dev/null; then
    echo "  ✓ ${name}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    echo "  ✗ ${name}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi
}

make_repo() {
  local dir="$1"
  mkdir -p "${dir}"
  pushd "${dir}" >/dev/null
  git init -q
  git config user.email "test@example.com"
  git config user.name "test"
  echo "data" > data.json
  git add data.json
  git commit -m "init" -q
  popd >/dev/null
}

run_script() {
  local dir="$1"
  GH_AW_CACHE_DIR="${dir}" bash "${SCRIPT}" 2>&1 || true
}

echo "Testing check_cache_memory_git_integrity.sh"
echo ""

echo "Test 1: Script syntax is valid"
assert "script passes bash -n" "bash -n '${SCRIPT}'"
echo ""

echo "Test 2: Missing .git repo is a no-op"
D="${WORKSPACE}/test2"
mkdir -p "${D}"
echo "data" > "${D}/data.json"
set +e
OUTPUT="$(run_script "${D}")"
EXIT_CODE=$?
set -e
assert "script exits successfully" "[ '${EXIT_CODE}' -eq 0 ]"
assert "warning not emitted" "! printf '%s' \"${OUTPUT}\" | grep -q 'cache-memory git integrity'"
assert "non-git files preserved" "[ -f '${D}/data.json' ]"
echo ""

echo "Test 3: Healthy git repo remains healthy"
D="${WORKSPACE}/test3"
make_repo "${D}"
set +e
OUTPUT="$(run_script "${D}")"
EXIT_CODE=$?
set -e
assert "script exits successfully" "[ '${EXIT_CODE}' -eq 0 ]"
assert "no corruption warning for healthy repo" "! printf '%s' \"${OUTPUT}\" | grep -q 'Detected git corruption'"
assert "git metadata still exists" "[ -d '${D}/.git' ]"
assert "git repo still readable" "git -C '${D}' rev-parse --verify HEAD >/dev/null 2>&1"
echo ""

echo "Test 4: Corrupted repo is reseeded"
D="${WORKSPACE}/test4"
make_repo "${D}"
pushd "${D}" >/dev/null
TREE_OBJ="$(git rev-parse HEAD^{tree})"
TREE_OBJ_PATH=".git/objects/${TREE_OBJ:0:2}/${TREE_OBJ:2}"
rm -f "${TREE_OBJ_PATH}"
popd >/dev/null
set +e
OUTPUT="$(run_script "${D}")"
EXIT_CODE=$?
set -e
assert "script exits successfully after corruption" "[ '${EXIT_CODE}' -eq 0 ]"
assert "corruption warning emitted" "printf '%s' \"${OUTPUT}\" | grep -q 'Detected git corruption; reseeding cache-memory git repository'"
assert "git metadata recreated" "[ -d '${D}/.git' ]"
assert "new empty commit exists" "git -C '${D}' rev-parse --verify HEAD >/dev/null 2>&1"
echo ""

echo "Tests passed: ${TESTS_PASSED}"
echo "Tests failed: ${TESTS_FAILED}"

if [ "${TESTS_FAILED}" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
