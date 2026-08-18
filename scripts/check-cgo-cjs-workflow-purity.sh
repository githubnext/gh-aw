#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -eq 0 ]; then
  set -- .github/workflows/cgo.yml .github/workflows/cjs.yml
fi

failed=0
for workflow in "$@"; do
  echo "Checking $workflow"
  if [ ! -f "$workflow" ]; then
    echo "Missing workflow file: $workflow"
    failed=1
    continue
  fi

  disallowed_secrets="$(grep -Eno '\$\{\{[[:space:]]*secrets\.[A-Za-z_][A-Za-z0-9_]*' "$workflow" || true)"
  disallowed_secrets="$(printf '%s\n' "$disallowed_secrets" | grep -Ev 'secrets\.(GITHUB_TOKEN|SCIENCE)$' || true)"
  if [ -n "$disallowed_secrets" ]; then
    echo "Disallowed secrets expressions found in $workflow:"
    echo "$disallowed_secrets"
    failed=1
  fi

  write_permissions="$(awk '
    FNR == 1 {
      in_permissions = 0
      perm_indent = -1
    }
    /^[[:space:]]*permissions:[[:space:]]*$/ {
      perm_indent = match($0, /[^[:space:]]/) - 1
      in_permissions = 1
      next
    }
    in_permissions {
      if ($0 ~ /^[[:space:]]*($|#)/) next
      indent = match($0, /[^[:space:]]/) - 1
      if (indent <= perm_indent) in_permissions = 0
    }
    in_permissions && $0 ~ /^[[:space:]]*[A-Za-z0-9_-]+:[[:space:]]*write([[:space:]]*(#.*)?)?$/ {
      print FILENAME ":" FNR ":" $0
    }
  ' "$workflow")"
  if [ -n "$write_permissions" ]; then
    echo "Write permissions found in $workflow:"
    echo "$write_permissions"
    failed=1
  fi
done

exit "$failed"
