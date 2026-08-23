#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_PATH="${SCRIPT_DIR}/render_log_to_stdout.sh"

PASS=0
FAIL=0

assert_matches() {
  local label="$1" pattern="$2" output="$3"
  if [[ "$output" =~ $pattern ]]; then
    echo "PASS: $label"
    PASS=$((PASS + 1))
  else
    echo "FAIL: $label"
    FAIL=$((FAIL + 1))
  fi
}

output=$(printf '%s' '::warning::untrusted output' | bash "$SCRIPT_PATH" "Gateway stderr")
assert_matches "starts a log group" '^::group::Gateway stderr' "$output"
assert_matches "disables workflow commands" $'\n::stop-commands::render-[[:xdigit:]]{24}' "$output"
assert_matches "preserves log content" '::warning::untrusted output' "$output"
assert_matches "restores workflow commands and closes group" $'\n::render-[[:xdigit:]]{24}::\n::endgroup::$' "$output"

set +e
bash "$SCRIPT_PATH" $'invalid\ntitle' </dev/null >/dev/null 2>&1
exit_code=$?
set -e
if [ "$exit_code" -eq 2 ]; then
  echo "PASS: rejects newline in group title"
  PASS=$((PASS + 1))
else
  echo "FAIL: rejects newline in group title"
  FAIL=$((FAIL + 1))
fi

echo "Tests passed: $PASS"
echo "Tests failed: $FAIL"
[ "$FAIL" -eq 0 ]
