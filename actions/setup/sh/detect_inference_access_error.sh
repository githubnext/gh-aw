#!/usr/bin/env bash
#
# detect_inference_access_error.sh - Detect Copilot CLI inference access errors
#
# Checks the agent stdio log for known error patterns that indicate the
# COPILOT_GITHUB_TOKEN does not have valid access to inference (e.g., the
# "Access denied by policy settings" error emitted by the Copilot CLI).
#
# Sets the GitHub Actions output variable:
#   inference_access_error=true   if the error pattern is found
#   inference_access_error=false  otherwise
#
# Exit codes:
#   0 - Always succeeds (uses continue-on-error in the workflow step)

set -euo pipefail

LOG_FILE="/tmp/gh-aw/agent-stdio.log"

if [ -f "$LOG_FILE" ] && grep -qE "Access denied by policy settings|invalid access to inference" "$LOG_FILE"; then
  echo "Detected inference access error in agent log"
  echo "inference_access_error=true" >> "$GITHUB_OUTPUT"
else
  echo "inference_access_error=false" >> "$GITHUB_OUTPUT"
fi
