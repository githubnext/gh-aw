#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

cat > "$TMP_DIR/example.md" <<'EOF'
---
description: |
  nested_in_block: ignored
on:
  schedule: ignored
  workflow_dispatch:
permissions:
  contents: ignored
engine:
  id: ignored
max-ai-credits: 10
---

# Body

body_key: ignored
EOF

EXPECTED_KEYS=$(cat <<'EOF'
description
engine
max-ai-credits
on
permissions
EOF
)

ACTUAL_KEYS="$(bash "$ROOT_DIR/scripts/extract-workflow-frontmatter-keys.sh" "$TMP_DIR"/*.md)"

if [ "$ACTUAL_KEYS" != "$EXPECTED_KEYS" ]; then
  echo "extract-workflow-frontmatter-keys produced unexpected keys" >&2
  diff -u <(printf '%s\n' "$EXPECTED_KEYS") <(printf '%s\n' "$ACTUAL_KEYS") >&2 || true
  exit 1
fi

echo "extract-workflow-frontmatter-keys test passed"
