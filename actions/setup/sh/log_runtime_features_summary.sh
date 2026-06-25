#!/usr/bin/env bash
set +o histexpand
set -euo pipefail

# Writes a collapsed Runtime features section to $GITHUB_STEP_SUMMARY only when the
# variable is both declared in the vars context (IS_SET=true) and non-empty.
# A variable that exists in vars as an empty string produces no summary output — this
# is intentional: an empty value has no meaningful content to surface.
if [[ "${GH_AW_RUNTIME_FEATURES_IS_SET:-}" == "true" && -n "${GH_AW_RUNTIME_FEATURES:-}" ]]; then
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
