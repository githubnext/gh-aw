#!/usr/bin/env bash
# Clean Action Post Script
# Mirror of actions/setup/post.js for script mode (run: bash steps).
# Sends an OTLP conclusion span then removes the /tmp/gh-aw/ directory.
#
# Must be called from an `if: always()` step so it runs even when job steps fail,
# ensuring both the trace span and the cleanup always complete.
#
# Usage (script mode):
#   - name: Clean Scripts
#     if: always()
#     run: |
#       bash /tmp/gh-aw/actions-source/actions/setup/clean.sh
#     env:
#       INPUT_DESTINATION: ${{ runner.temp }}/gh-aw/actions

set -e

DESTINATION="${INPUT_DESTINATION:-${RUNNER_TEMP}/gh-aw/actions}"

# Send OTLP job conclusion span (non-fatal, mirrors actions/setup/post.js).
if command -v node &>/dev/null && [ -f "${DESTINATION}/send_otlp_span.cjs" ]; then
  DESTINATION="${DESTINATION}" node - 2>/dev/null <<'OTLP_CLEAN_EOF' || true
;(async () => {
  const { sendJobConclusionSpan } = require(process.env.DESTINATION + '/send_otlp_span.cjs');
  const spanName = process.env.INPUT_JOB_NAME ? 'gh-aw.job.' + process.env.INPUT_JOB_NAME : 'gh-aw.job.conclusion';
  await sendJobConclusionSpan(spanName);
})().catch(() => {});
OTLP_CLEAN_EOF
fi

# Remove /tmp/gh-aw/ (mirrors post.js cleanup).
tmpDir="/tmp/gh-aw"
if [ -d "${tmpDir}" ]; then
  if sudo rm -rf "${tmpDir}" 2>/dev/null; then
    echo "Cleaned up ${tmpDir} (sudo)"
  elif rm -rf "${tmpDir}" 2>/dev/null; then
    echo "Cleaned up ${tmpDir}"
  else
    echo "Warning: failed to clean up ${tmpDir}" >&2
  fi
fi
