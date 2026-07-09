#!/usr/bin/env bash
set +o histexpand

set -euo pipefail

RESULT_FILE="${1:-/tmp/gh-aw/threat-detection/detection_result.json}"
RESULT_DIR="$(dirname "${RESULT_FILE}")"
DETECTION_LOG_FILE="${DETECTION_LOG_FILE:-${RESULT_DIR}/detection.log}"
DETECTION_STATUS_PREFIX="THREAT_DETECTION_STATUS:"
continue_on_error="${GH_AW_DETECTION_CONTINUE_ON_ERROR:-true}"
continue_on_error="$(echo "${continue_on_error}" | tr '[:upper:]' '[:lower:]')"

if [ "${RUN_DETECTION:-false}" = "true" ] && [ ! -f "${RESULT_FILE}" ]; then
  detection_status=""
  if [ -f "${DETECTION_LOG_FILE}" ]; then
    detection_status="$(grep "${DETECTION_STATUS_PREFIX}" "${DETECTION_LOG_FILE}" | tail -n 1 || true)"
  fi

  result_message="Detection result file not found at: ${RESULT_FILE} (execution outcome: ${DETECTION_AGENTIC_EXECUTION_OUTCOME:-unknown})"
  if [ -n "${detection_status}" ]; then
    result_message="${result_message}; detector status: ${detection_status}"
  elif [ -f "${DETECTION_LOG_FILE}" ]; then
    result_message="${result_message}; detection log exists at ${DETECTION_LOG_FILE} but did not include ${DETECTION_STATUS_PREFIX}"
  else
    result_message="${result_message}; detection log not found at ${DETECTION_LOG_FILE}"
  fi

  if [ "${continue_on_error}" = "true" ]; then
    echo "::warning::${result_message}; continuing because GH_AW_DETECTION_CONTINUE_ON_ERROR=true"
    echo "conclusion=warning" >> "${GITHUB_OUTPUT}"
    echo "success=false" >> "${GITHUB_OUTPUT}"
    echo "reason=agent_failure" >> "${GITHUB_OUTPUT}"
    echo "GH_AW_DETECTION_CONCLUSION=warning" >> "${GITHUB_ENV}"
    echo "GH_AW_DETECTION_REASON=agent_failure" >> "${GITHUB_ENV}"
    exit 0
  fi
  echo "ERR_SYSTEM: ❌ ${result_message}"
  exit 1
fi

threat-detect conclude --result-file "${RESULT_FILE}"
