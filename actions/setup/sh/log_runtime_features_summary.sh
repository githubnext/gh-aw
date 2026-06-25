#!/usr/bin/env bash
set +o histexpand
set -euo pipefail

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
  } >> "$GITHUB_STEP_SUMMARY"
fi
