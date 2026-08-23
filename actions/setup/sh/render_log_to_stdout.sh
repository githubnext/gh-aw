#!/usr/bin/env bash
set -euo pipefail

# Emits stdin in a GitHub Actions log group with workflow commands disabled.
#
# Usage: render_log_to_stdout.sh <group-title>
#
# Input must be redacted before it is passed to this helper.

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <group-title>" >&2
  exit 2
fi

GROUP_TITLE="$1"
if [[ "$GROUP_TITLE" == *$'\n'* || "$GROUP_TITLE" == *$'\r'* ]]; then
  echo "Group title must not contain newlines" >&2
  exit 2
fi

STOP_TOKEN="render-$(od -An -N12 -tx1 /dev/urandom | tr -d ' \n')"

printf '::group::%s\n' "$GROUP_TITLE"
printf '::stop-commands::%s\n' "$STOP_TOKEN"
cat
printf '\n::%s::\n::endgroup::\n' "$STOP_TOKEN"
