#!/usr/bin/env bash
# Tests for setup_cache_memory_git.sh — pre-agent sanitization block
# Run: bash setup_cache_memory_git_test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/setup_cache_memory_git.sh"

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Temporary workspace for all tests
WORKSPACE=$(mktemp -d)

cleanup() {
  rm -rf "${WORKSPACE}"
}
trap cleanup EXIT

# Helper: assert a condition
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

# Helper: create a fresh git cache dir with the given files already committed.
# Usage: make_cache_dir <dir> [<file> ...]
# Files are created and committed to the 'none' branch (the lowest-trust default).
make_cache_dir() {
  local dir="$1"
  shift
  mkdir -p "${dir}"
  pushd "${dir}" >/dev/null
  git init -b merged -q
  git config user.email "test@example.com"
  git config user.name "test"
  git config core.hooksPath /dev/null
  git commit --allow-empty -m "initial" -q
  for level in approved unapproved none; do
    git branch "${level}" 2>/dev/null || true
  done
  git checkout -q none
  for f in "$@"; do
    mkdir -p "$(dirname "${f}")"
    echo "content" > "${f}"
  done
  git add -A
  git commit --allow-empty -m "test-files" -q
  popd >/dev/null
}

# Run the script, capturing stdout and ignoring the exit code.
run_script() {
  local dir="$1"
  local integrity="${2:-none}"
  local allowed_exts="${3:-}"
  GH_AW_CACHE_DIR="${dir}" \
  GH_AW_MIN_INTEGRITY="${integrity}" \
  GH_AW_ALLOWED_EXTENSIONS="${allowed_exts}" \
    bash "${SCRIPT}" 2>&1 || true
}

echo "Testing setup_cache_memory_git.sh — pre-agent sanitization"
echo ""

# ── Test 1: Execute bits are stripped from restored files ────────────────────
echo "Test 1: Execute bits are stripped unconditionally"
D="${WORKSPACE}/test1"
make_cache_dir "${D}" "script.sh" "data.json"
# Make files executable before the script runs
chmod +x "${D}/script.sh" "${D}/data.json"
run_script "${D}" none >/dev/null
assert "script.sh is not executable"   "[ ! -x '${D}/script.sh' ]"
assert "data.json is not executable"   "[ ! -x '${D}/data.json' ]"
assert "script.sh still exists"        "[ -f '${D}/script.sh' ]"
assert "data.json still exists"        "[ -f '${D}/data.json' ]"
echo ""

# ── Test 2: .git directory files are NOT touched (sanity check) ──────────────
echo "Test 2: .git directory is not affected by chmod"
D="${WORKSPACE}/test2"
make_cache_dir "${D}" "file.txt"
HOOK_FILE="${D}/.git/hooks/pre-commit"
echo "#!/bin/bash" > "${HOOK_FILE}"
chmod +x "${HOOK_FILE}"
run_script "${D}" none >/dev/null
# The hook file cleanup happens earlier in the script but the .git dir itself is
# excluded from find. Verify find exclusion by checking the .git dir is intact.
assert ".git directory still exists"   "[ -d '${D}/.git' ]"
echo ""

# ── Test 3: No extension filter — all files kept when GH_AW_ALLOWED_EXTENSIONS is empty ─
echo "Test 3: No extension filter when GH_AW_ALLOWED_EXTENSIONS is unset"
D="${WORKSPACE}/test3"
make_cache_dir "${D}" "file.json" "file.md" "helper.sh" "binary"
run_script "${D}" none ""
assert "file.json kept"  "[ -f '${D}/file.json' ]"
assert "file.md kept"    "[ -f '${D}/file.md' ]"
assert "helper.sh kept"  "[ -f '${D}/helper.sh' ]"
assert "binary kept"     "[ -f '${D}/binary' ]"
echo ""

# ── Test 4: Extension filter removes disallowed files ────────────────────────
echo "Test 4: Extension filter removes disallowed file types"
D="${WORKSPACE}/test4"
make_cache_dir "${D}" "data.json" "notes.md" "helper.sh" "archive.zip"
run_script "${D}" none ".json:.md"
assert "data.json kept"     "[ -f '${D}/data.json' ]"
assert "notes.md kept"      "[ -f '${D}/notes.md' ]"
assert "helper.sh removed"  "[ ! -f '${D}/helper.sh' ]"
assert "archive.zip removed" "[ ! -f '${D}/archive.zip' ]"
echo ""

# ── Test 5: Extension filter removes files without any extension ─────────────
echo "Test 5: Extension filter removes files with no extension"
D="${WORKSPACE}/test5"
make_cache_dir "${D}" "data.json" "noext"
run_script "${D}" none ".json"
assert "data.json kept"  "[ -f '${D}/data.json' ]"
assert "noext removed"   "[ ! -f '${D}/noext' ]"
echo ""

# ── Test 6: Extension filter with single extension ───────────────────────────
echo "Test 6: Extension filter with a single allowed extension"
D="${WORKSPACE}/test6"
make_cache_dir "${D}" "report.json" "notes.txt" "image.png"
run_script "${D}" none ".json"
assert "report.json kept"  "[ -f '${D}/report.json' ]"
assert "notes.txt removed" "[ ! -f '${D}/notes.txt' ]"
assert "image.png removed" "[ ! -f '${D}/image.png' ]"
echo ""

# ── Test 7: Execute bits stripped AND disallowed files removed together ───────
echo "Test 7: Execute-bit stripping and extension filtering both apply"
D="${WORKSPACE}/test7"
make_cache_dir "${D}" "keep.json" "drop.sh"
chmod +x "${D}/keep.json" "${D}/drop.sh"
run_script "${D}" none ".json"
assert "keep.json exists"        "[ -f '${D}/keep.json' ]"
assert "keep.json not executable" "[ ! -x '${D}/keep.json' ]"
assert "drop.sh removed"         "[ ! -f '${D}/drop.sh' ]"
echo ""

# ── Test 8: Extension matching is case-insensitive ───────────────────────────
echo "Test 8: Extension matching is case-insensitive"
D="${WORKSPACE}/test8"
make_cache_dir "${D}" "data.json" "data.JSON" "notes.MD"
# Allow list uses lowercase; both .json and .JSON files, and .MD files, should be kept
run_script "${D}" none ".json:.md"
assert "data.json kept (exact match)"     "[ -f '${D}/data.json' ]"
assert "data.JSON kept (uppercase file)"  "[ -f '${D}/data.JSON' ]"
assert "notes.MD kept (uppercase file)"   "[ -f '${D}/notes.MD' ]"
echo ""

# ── Test 9: Whitespace in GH_AW_ALLOWED_EXTENSIONS is trimmed ────────────────
echo "Test 9: Whitespace in allowed extensions list is trimmed"
D="${WORKSPACE}/test9"
make_cache_dir "${D}" "data.json" "note.md" "drop.sh"
# Extensions with leading/trailing spaces should still match
run_script "${D}" none " .json : .md "
assert "data.json kept (trimmed .json)"  "[ -f '${D}/data.json' ]"
assert "note.md kept (trimmed .md)"      "[ -f '${D}/note.md' ]"
assert "drop.sh removed"                 "[ ! -f '${D}/drop.sh' ]"
echo ""

# ── Test 10: Symlinks are deleted unconditionally ────────────────────────────
echo "Test 10: Symlinks in working tree are deleted"
D="${WORKSPACE}/test10"
make_cache_dir "${D}" "real.json"
# Plant a symlink (simulating a compromised prior run)
ln -s /etc/passwd "${D}/evil-link"
assert "symlink exists before script"    "[ -L '${D}/evil-link' ]"
run_script "${D}" none >/dev/null
assert "symlink removed by script"       "[ ! -L '${D}/evil-link' ]"
assert "real file still exists"          "[ -f '${D}/real.json' ]"
echo ""

# ── Test 11: Files with spaces in name are handled correctly ─────────────────
echo "Test 11: Files with spaces in names are handled correctly"
D="${WORKSPACE}/test11"
make_cache_dir "${D}" "my data.json" "my script.sh"
run_script "${D}" none ".json"
assert "file with space and .json kept"    "[ -f '${D}/my data.json' ]"
assert "file with space and .sh removed"   "[ ! -f '${D}/my script.sh' ]"
echo ""

# ── Summary ──────────────────────────────────────────────────────────────────
echo "Tests passed: ${TESTS_PASSED}"
echo "Tests failed: ${TESTS_FAILED}"

if [ "${TESTS_FAILED}" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
