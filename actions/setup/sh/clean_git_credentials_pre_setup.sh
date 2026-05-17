#!/usr/bin/env bash
set +o histexpand

set -euo pipefail

cleaned=0
while IFS= read -r git_config; do
  git config --file "${git_config}" --remove-section credential 2>/dev/null || true
  git config --file "${git_config}" --unset-all http.extraheader 2>/dev/null || true
  git config --file "${git_config}" --get-regexp '^http\..*\.extraheader$' 2>/dev/null | while read -r key _; do
    git config --file "${git_config}" --unset-all "${key}" || true
  done || true
  cleaned=$((cleaned + 1))
done < <(find "${GITHUB_WORKSPACE}" /tmp -maxdepth 15 -type f -name "config" \( -path "*/.git/config" -o -path "*/.git/modules/*/config" \) 2>/dev/null | sort -u)

if [ "${cleaned}" -eq 0 ]; then
  echo "No git config files found for cleanup"
fi
