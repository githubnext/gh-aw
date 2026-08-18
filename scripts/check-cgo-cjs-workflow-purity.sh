#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -eq 0 ]; then
  set -- .github/workflows/cgo.yml .github/workflows/cjs.yml
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

failed=0
for workflow in "$@"; do
  echo "Checking $workflow"
  if [ ! -f "$workflow" ]; then
    echo "Missing workflow file: $workflow"
    failed=1
    continue
  fi

  # These workflows should stay pure test workflows: allow only the built-in
  # GITHUB_TOKEN and the repository's SCIENCE telemetry secret.
  disallowed_secrets_file="$tmp_dir/disallowed-secrets.txt"
  if ! perl -ne '
    while (/\$\{\{\s*secrets\.([A-Za-z_][A-Za-z0-9_]*)\b/g) {
      print "$ARGV:$.: secrets.$1\n" unless $1 eq "GITHUB_TOKEN" || $1 eq "SCIENCE";
    }
    close ARGV if eof;
  ' "$workflow" >"$disallowed_secrets_file"; then
    echo "Failed to scan secrets expressions in $workflow"
    failed=1
    continue
  fi
  if [ -s "$disallowed_secrets_file" ]; then
    echo "Disallowed secrets expressions found in $workflow:"
    cat "$disallowed_secrets_file"
    failed=1
  fi

  write_permissions_file="$tmp_dir/write-permissions.txt"
  if ! awk '
    FNR == 1 {
      in_permissions = 0
      perm_indent = -1
    }
    /^[[:space:]]*permissions:[[:space:]]*write-all([[:space:]]*(#.*)?)?$/ {
      print FILENAME ":" FNR ":" $0
    }
    /^[[:space:]]*permissions:[[:space:]]*$/ {
      perm_indent = match($0, /[^[:space:]]/) - 1
      in_permissions = 1
      next
    }
    in_permissions {
      if ($0 ~ /^[[:space:]]*$/ || $0 ~ /^[[:space:]]*#/) next
      indent = match($0, /[^[:space:]]/) - 1
      if (indent <= perm_indent) {
        in_permissions = 0
        next
      }
    }
    in_permissions && $0 ~ /^[[:space:]]*[A-Za-z0-9_-]+:[[:space:]]*write([[:space:]]*(#.*)?)?$/ {
      print FILENAME ":" FNR ":" $0
    }
  ' "$workflow" >"$write_permissions_file"; then
    echo "Failed to scan permissions in $workflow"
    failed=1
    continue
  fi
  if [ -s "$write_permissions_file" ]; then
    echo "Write permissions found in $workflow:"
    cat "$write_permissions_file"
    failed=1
  fi
done

exit "$failed"
