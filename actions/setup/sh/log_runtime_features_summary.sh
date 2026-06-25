#!/usr/bin/env bash
set +o histexpand
set -euo pipefail

# Writes a collapsed Runtime features section to $GITHUB_STEP_SUMMARY.
# The step is only run when GH_AW_RUNTIME_FEATURES is present in the vars context
# (guarded by the step's `if:` condition), so we only need to check for non-empty here.
# A variable that exists in vars as an empty string produces no summary output — this
# is intentional: an empty value has no meaningful content to surface.
if [[ -n "${GH_AW_RUNTIME_FEATURES:-}" ]]; then
  {
    echo "### Runtime features"
    echo
    echo "<details>"
    echo "<summary>Show configured runtime features</summary>"
    echo
    echo '```text'
    printf '%s\n' "$GH_AW_RUNTIME_FEATURES"
    echo '```'
    echo
    echo "</details>"
  } >> "${GITHUB_STEP_SUMMARY:-/dev/null}"
fi
