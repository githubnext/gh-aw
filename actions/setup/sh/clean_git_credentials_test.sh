#!/usr/bin/env bash
# Test script for clean_git_credentials.sh
# Run: bash clean_git_credentials_test.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLEAN_SCRIPT="${SCRIPT_DIR}/clean_git_credentials.sh"

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Temporary workspace for tests
TEST_WORKSPACE=$(mktemp -d)

cleanup() {
  rm -rf "${TEST_WORKSPACE}"
}
trap cleanup EXIT

# Helper: create a minimal git repo with a .git/config file
make_git_config() {
  local dir="$1"
  local config="$2"
  mkdir -p "${dir}/.git"
  echo "${config}" >"${dir}/.git/config"
}

# Helper: assert a condition
assert() {
  local name="$1"
  local condition="$2"
  if eval "${condition}"; then
    echo "✓ ${name}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    echo "✗ ${name}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi
}

echo "Testing clean_git_credentials.sh..."
echo ""

# ── Test 1: No .git/config anywhere (no-op, exit 0) ─────────────────────────
echo "Test 1: No .git/config files → exit 0 with informational message"
EMPTY_WS=$(mktemp -d)
OUTPUT=$(GITHUB_WORKSPACE="${EMPTY_WS}" bash "${CLEAN_SCRIPT}" 2>&1)
EXIT_CODE=$?
rmdir "${EMPTY_WS}"
assert "exits 0 when no .git/config found" "[ ${EXIT_CODE} -eq 0 ]"
assert "prints informational message" "echo '${OUTPUT}' | grep -q 'No .git/config'"
echo ""

# ── Test 2: Removes [credential] section ────────────────────────────────────
echo "Test 2: Removes [credential] section from workspace .git/config"
REPO="${TEST_WORKSPACE}/repo2"
make_git_config "${REPO}" "[core]
	repositoryformatversion = 0
[credential]
	helper = store
[remote \"origin\"]
	url = https://github.com/org/repo.git"
GITHUB_WORKSPACE="${REPO}" bash "${CLEAN_SCRIPT}" >/dev/null 2>&1
assert "credential section removed" "! grep -q '\[credential\]' '${REPO}/.git/config'"
assert "core section preserved"    "grep -q '\[core\]' '${REPO}/.git/config'"
assert "remote section preserved"  "grep -q '\[remote' '${REPO}/.git/config'"
echo ""

# ── Test 3: Removes http.extraheader ────────────────────────────────────────
echo "Test 3: Removes http.extraheader from git config"
REPO="${TEST_WORKSPACE}/repo3"
make_git_config "${REPO}" "[core]
	repositoryformatversion = 0
[http]
	extraheader = Authorization: Basic dXNlcjpwYXNz"
GITHUB_WORKSPACE="${REPO}" bash "${CLEAN_SCRIPT}" >/dev/null 2>&1
assert "http.extraheader removed" "! git config --file '${REPO}/.git/config' http.extraheader 2>/dev/null"
assert "config still valid"       "git config --file '${REPO}/.git/config' --list >/dev/null 2>&1"
echo ""

# ── Test 4: Strips credentials from remote URL ──────────────────────────────
echo "Test 4: Strips credentials from authenticated remote URL"
REPO="${TEST_WORKSPACE}/repo4"
make_git_config "${REPO}" "[core]
	repositoryformatversion = 0
[remote \"origin\"]
	url = https://x-access-token:ghs_abc123@github.com/org/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*"
GITHUB_WORKSPACE="${REPO}" bash "${CLEAN_SCRIPT}" >/dev/null 2>&1
CLEANED_URL=$(git config --file "${REPO}/.git/config" remote.origin.url)
assert "credentials stripped from URL" "[ '${CLEANED_URL}' = 'https://github.com/org/repo.git' ]"
echo ""

# ── Test 5: Recursively finds repo nested inside workspace ──────────────────
echo "Test 5: Recursively cleans nested git repo inside workspace"
OUTER="${TEST_WORKSPACE}/outer5"
INNER="${OUTER}/vendor/dep"
make_git_config "${OUTER}" "[core]
	repositoryformatversion = 0
[credential]
	helper = store"
make_git_config "${INNER}" "[core]
	repositoryformatversion = 0
[http]
	extraheader = Authorization: Basic dXNlcjpwYXNz"
GITHUB_WORKSPACE="${OUTER}" bash "${CLEAN_SCRIPT}" >/dev/null 2>&1
assert "outer credential section cleaned" "! grep -q '\[credential\]' '${OUTER}/.git/config'"
assert "inner extraheader cleaned"         "! git config --file '${INNER}/.git/config' http.extraheader 2>/dev/null"
echo ""

# ── Test 6: Finds git repo in /tmp ──────────────────────────────────────────
echo "Test 6: Cleans git repo located in /tmp"
TMP_REPO=$(mktemp -d)
make_git_config "${TMP_REPO}" "[core]
	repositoryformatversion = 0
[credential]
	helper = store"
# Use a workspace that does NOT contain the repo so it is found only via /tmp
GITHUB_WORKSPACE="${TEST_WORKSPACE}/workspace6"
mkdir -p "${GITHUB_WORKSPACE}"
GITHUB_WORKSPACE="${GITHUB_WORKSPACE}" bash "${CLEAN_SCRIPT}" >/dev/null 2>&1
assert "tmp repo credential section cleaned" "! grep -q '\[credential\]' '${TMP_REPO}/.git/config'"
rm -rf "${TMP_REPO}"
echo ""

# ── Test 7: Config file remains valid after all cleanups ────────────────────
echo "Test 7: Config file is still valid after cleaning"
REPO="${TEST_WORKSPACE}/repo7"
make_git_config "${REPO}" "[core]
	repositoryformatversion = 0
[credential]
	helper = /usr/lib/git-credential-gnome-keyring
[http]
	extraheader = Authorization: Bearer sometoken
[remote \"origin\"]
	url = https://oauth2:tok@github.com/org/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*"
GITHUB_WORKSPACE="${REPO}" bash "${CLEAN_SCRIPT}" >/dev/null 2>&1
assert "config still valid" "git config --file '${REPO}/.git/config' --list >/dev/null 2>&1"
assert "core settings intact" "git config --file '${REPO}/.git/config' core.repositoryformatversion >/dev/null 2>&1"
echo ""

# ── Summary ──────────────────────────────────────────────────────────────────
echo "Tests passed: ${TESTS_PASSED}"
echo "Tests failed: ${TESTS_FAILED}"

if [ "${TESTS_FAILED}" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
