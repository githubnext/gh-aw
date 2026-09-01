#!/bin/bash
set +o histexpand
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_SCRIPT="$SCRIPT_DIR/check-skill-file-paths.sh"

pass() { echo "PASS: $1"; }
fail() { echo "FAIL: $1"; echo "  $2"; exit 1; }

TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMP_ROOT"' EXIT

# Valid skill references should pass.
VALID_ROOT="$TMP_ROOT/valid"
mkdir -p "$VALID_ROOT/.github/skills/example-skill" "$VALID_ROOT/pkg/workflow/js"
cat > "$VALID_ROOT/.github/skills/example-skill/SKILL.md" <<'EOF'
Use `pkg/workflow/js/messages_core.cjs` and `./README.md`.
EOF
printf '%s\n' 'example' > "$VALID_ROOT/README.md"
: > "$VALID_ROOT/pkg/workflow/js/messages_core.cjs"
if (cd "$VALID_ROOT" && bash "$TEST_SCRIPT" --repo-root "$VALID_ROOT" >/tmp/check-skill-valid.out 2>&1); then
    pass "valid skill file paths pass"
else
    fail "valid skill file paths should pass" "$(cat /tmp/check-skill-valid.out)"
fi

# Invalid repo paths should fail.
INVALID_ROOT="$TMP_ROOT/invalid"
mkdir -p "$INVALID_ROOT/.github/skills/example-skill" "$INVALID_ROOT/pkg/workflow/js"
cat > "$INVALID_ROOT/.github/skills/example-skill/SKILL.md" <<'EOF'
This doc references `pkg/workflow/nope.cjs` and `github/gh-aw`, which should be caught.
EOF
if (cd "$INVALID_ROOT" && bash "$TEST_SCRIPT" --repo-root "$INVALID_ROOT" >/tmp/check-skill-invalid.out 2>&1); then
    fail "invalid skill path should fail" "$(cat /tmp/check-skill-invalid.out)"
elif grep -q "pkg/workflow/nope.cjs" /tmp/check-skill-invalid.out; then
    pass "invalid skill file paths fail with the offending path"
else
    fail "invalid skill path output did not name the stale file" "$(cat /tmp/check-skill-invalid.out)"
fi

# Missing skill directory should error.
MISSING_ROOT="$TMP_ROOT/missing"
mkdir -p "$MISSING_ROOT"
if (cd "$MISSING_ROOT" && bash "$TEST_SCRIPT" --repo-root "$MISSING_ROOT" >/tmp/check-skill-missing.out 2>&1); then
    fail "missing skill directory should fail" "$(cat /tmp/check-skill-missing.out)"
elif grep -qi "skill directory not found" /tmp/check-skill-missing.out; then
    pass "missing skill directory exits with an error"
else
    fail "missing skill directory output was unexpected" "$(cat /tmp/check-skill-missing.out)"
fi

echo "All skill path checks passed."
