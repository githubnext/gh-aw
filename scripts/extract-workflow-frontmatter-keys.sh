#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -eq 0 ]; then
  exit 0
fi

awk '
  FNR == 1 {
    in_frontmatter = 0
  }
  FNR == 1 && $0 ~ /^---[[:space:]]*$/ {
    in_frontmatter = 1
    next
  }
  in_frontmatter && $0 ~ /^---[[:space:]]*$/ {
    in_frontmatter = 0
    next
  }
  in_frontmatter && $0 ~ /^[a-z][a-z0-9_-]*:/ {
    key = $0
    sub(/:.*/, "", key)
    print key
  }
' "$@" | sort -u
