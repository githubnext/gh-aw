#!/usr/bin/env bash
set +o histexpand

set -euo pipefail

RESULT_FILE="${1:-/tmp/gh-aw/threat-detection/detection_result.json}"
continue_on_error="${GH_AW_DETECTION_CONTINUE_ON_ERROR:-true}"
continue_on_error="$(echo "${continue_on_error}" | tr '[:upper:]' '[:lower:]')"

if [ "${RUN_DETECTION:-false}" = "true" ] && [ ! -f "${RESULT_FILE}" ]; then
  if [ "${continue_on_error}" = "true" ]; then
    echo "::warning::Detection result file not found at: ${RESULT_FILE} (execution outcome: ${DETECTION_AGENTIC_EXECUTION_OUTCOME:-unknown}); continuing because GH_AW_DETECTION_CONTINUE_ON_ERROR=true"
    echo "conclusion=warning" >> "${GITHUB_OUTPUT}"
    echo "success=false" >> "${GITHUB_OUTPUT}"
    echo "reason=agent_failure" >> "${GITHUB_OUTPUT}"
    echo "GH_AW_DETECTION_CONCLUSION=warning" >> "${GITHUB_ENV}"
    echo "GH_AW_DETECTION_REASON=agent_failure" >> "${GITHUB_ENV}"
    exit 0
  fi
  echo "ERR_SYSTEM: ❌ Detection result file not found at: ${RESULT_FILE}"
  exit 1
fi

threat-detect conclude --result-file "${RESULT_FILE}"
