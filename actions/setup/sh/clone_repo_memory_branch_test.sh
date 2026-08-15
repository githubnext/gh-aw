#!/usr/bin/env bash
set +o histexpand

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/clone_repo_memory_branch.sh"

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

run_script() {
  local memory_dir="$1"
  GH_TOKEN="test-token" \
  BRANCH_NAME="repo-memory-test" \
  TARGET_REPO="octo/test" \
  MEMORY_DIR="${memory_dir}" \
  CREATE_ORPHAN="true" \
  GITHUB_SERVER_URL="https://127.0.0.1:9" \
  bash "${SCRIPT}" 2>&1 || true
}

echo "Testing clone_repo_memory_branch.sh"
echo ""

echo "Test 1: Script syntax is valid"
assert "script passes bash -n" "bash -n '${SCRIPT}'"
echo ""

echo "Test 2: Missing branch creates hardened orphan repo"
D="${WORKSPACE}/test2"
mkdir -p "${D}"
OUTPUT="$(run_script "${D}")"
assert "orphan branch message emitted" "printf '%s' \"${OUTPUT}\" | grep -q 'creating orphan branch'"
assert "git metadata exists" "[ -d '${D}/.git' ]"
assert "hooks path hardened" "[ \"\$(git -C '${D}' config --default '' core.hooksPath)\" = '/dev/null' ]"
assert "fsmonitor hardened" "[ \"\$(git -C '${D}' config --default '' core.fsmonitor)\" = 'false' ]"
echo ""

echo "Test 3: Scrubs persisted config/info command surfaces"
D="${WORKSPACE}/test3"
mkdir -p "${D}"
git -C "${D}" init -q
git -C "${D}" config core.attributesFile /tmp/evil-attributes
git -C "${D}" config core.fsmonitor /tmp/evil-fsmonitor
git -C "${D}" config core.sshCommand "ssh -F /tmp/evil-ssh"
git -C "${D}" config core.hooksPath /tmp/evil-hooks
git -C "${D}" config include.path /tmp/evil-include
git -C "${D}" config includeIf.onbranch:main.path /tmp/evil-include-if
git -C "${D}" config credential.helper /tmp/evil-cred-helper
git -C "${D}" config alias.pwn '!echo owned'
git -C "${D}" config filter.pwn.process '/tmp/evil-filter'
git -C "${D}" config merge.pwn.driver '/tmp/evil-merge'
mkdir -p "${D}/.git/info"
echo "*.tmp" > "${D}/.git/info/exclude"
echo "*.tmp -text" > "${D}/.git/info/attributes"
echo "0000000000000000000000000000000000000000 1111111111111111111111111111111111111111" > "${D}/.git/info/grafts"
echo "/tmp" > "${D}/.git/info/sparse-checkout"
OUTPUT="$(run_script "${D}")"
assert "attributes file unset" "[ -z \"\$(git -C '${D}' config --default '' core.attributesFile)\" ]"
assert "fsmonitor reset" "[ \"\$(git -C '${D}' config --default '' core.fsmonitor)\" = 'false' ]"
assert "ssh command unset" "[ -z \"\$(git -C '${D}' config --default '' core.sshCommand)\" ]"
assert "hooks path reset" "[ \"\$(git -C '${D}' config --default '' core.hooksPath)\" = '/dev/null' ]"
assert "include entries scrubbed" "! git -C '${D}' config --local --name-only --get-regexp '^include\\.' >/dev/null 2>&1"
assert "includeIf entries scrubbed" "! git -C '${D}' config --local --name-only --get-regexp '^includeif\\.' >/dev/null 2>&1"
assert "credential entries scrubbed" "! git -C '${D}' config --local --name-only --get-regexp '^credential\\.' >/dev/null 2>&1"
assert "alias entries scrubbed" "! git -C '${D}' config --local --name-only --get-regexp '^alias\\.' >/dev/null 2>&1"
assert "filter entries scrubbed" "! git -C '${D}' config --local --name-only --get-regexp '^filter\\.' >/dev/null 2>&1"
assert "merge entries scrubbed" "! git -C '${D}' config --local --name-only --get-regexp '^merge\\.' >/dev/null 2>&1"
assert "info/exclude removed" "[ ! -f '${D}/.git/info/exclude' ]"
assert "info/attributes removed" "[ ! -f '${D}/.git/info/attributes' ]"
assert "info/grafts removed" "[ ! -f '${D}/.git/info/grafts' ]"
assert "info/sparse-checkout removed" "[ ! -f '${D}/.git/info/sparse-checkout' ]"
assert "config remains valid" "git -C '${D}' config --local --list >/dev/null 2>&1"
echo ""

echo "Test 4: Symlinked metadata is reinitialized safely"
D="${WORKSPACE}/test4"
mkdir -p "${D}" "${D}/outside-info"
git -C "${D}" init -q
rm -rf "${D}/.git/info"
ln -s "${D}/outside-info" "${D}/.git/info"
echo "persist" > "${D}/outside-info/exclude"
OUTPUT="$(run_script "${D}")"
assert "symlink warning emitted" "printf '%s' \"${OUTPUT}\" | grep -qi 'symlinked repo-memory git metadata'"
assert "local .git/info rebuilt as directory" "[ -d '${D}/.git/info' ] && [ ! -L '${D}/.git/info' ]"
assert "outside target untouched" "[ -f '${D}/outside-info/exclude' ]"
echo ""

echo "Test 5: Config/hooks symlink metadata is reinitialized safely"
D="${WORKSPACE}/test5"
mkdir -p "${D}" "${D}/outside-hooks"
git -C "${D}" init -q
rm -f "${D}/.git/config"
cat > "${D}/outside-config" <<'EOF'
[core]
	repositoryformatversion = 0
EOF
ln -s "${D}/outside-config" "${D}/.git/config"
rm -rf "${D}/.git/hooks"
ln -s "${D}/outside-hooks" "${D}/.git/hooks"
touch "${D}/outside-hooks/post-checkout"
OUTPUT="$(run_script "${D}")"
assert "symlink warning emitted for config/hooks" "printf '%s' \"${OUTPUT}\" | grep -qi 'symlinked repo-memory git metadata'"
assert "local .git/config rebuilt as regular file" "[ -f '${D}/.git/config' ] && [ ! -L '${D}/.git/config' ]"
assert "local .git/hooks rebuilt as directory" "[ -d '${D}/.git/hooks' ] && [ ! -L '${D}/.git/hooks' ]"
assert "outside config untouched" "grep -q 'repositoryformatversion' '${D}/outside-config'"
assert "outside hooks untouched" "[ -f '${D}/outside-hooks/post-checkout' ]"
echo ""

echo "Tests passed: ${TESTS_PASSED}"
echo "Tests failed: ${TESTS_FAILED}"

if [ "${TESTS_FAILED}" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
