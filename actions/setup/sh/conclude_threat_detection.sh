#!/usr/bin/env bash
set +o histexpand

set -euo pipefail

RESULT_FILE="${1:-/tmp/gh-aw/threat-detection/detection_result.json}"
RESULT_DIR="$(dirname "${RESULT_FILE}")"
DETECTION_LOG_FILE="${DETECTION_LOG_FILE:-${RESULT_DIR}/detection.log}"

# threat-detect conclude handles every branch of the conclusion contract
# (skipped run, missing/malformed result file, warn-mode vs strict-mode
# hard-fail rules, status-reason mapping, diagnostics, and step summary).
# The only failure it cannot report on itself is its own absence from PATH.
if ! command -v threat-detect >/dev/null 2>&1; then
  message="threat-detect binary not found on PATH"
  continue_on_error="${GH_AW_DETECTION_CONTINUE_ON_ERROR:-true}"
  if [ "${continue_on_error,,}" != "false" ]; then
    echo "::warning::${message}; continuing because GH_AW_DETECTION_CONTINUE_ON_ERROR != false"
    echo "conclusion=warning" >> "${GITHUB_OUTPUT}"
    echo "success=false" >> "${GITHUB_OUTPUT}"
    echo "reason=agent_failure" >> "${GITHUB_OUTPUT}"
    exit 0
  fi
  echo "conclusion=failure" >> "${GITHUB_OUTPUT}"
  echo "success=false" >> "${GITHUB_OUTPUT}"
  echo "reason=agent_failure" >> "${GITHUB_OUTPUT}"
  echo "ERR_SYSTEM: ${message}"
  exit 1
fi

exec threat-detect conclude \
  --result-file "${RESULT_FILE}" \
  --detection-log "${DETECTION_LOG_FILE}"
