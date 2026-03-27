#!/bin/bash
# rescue_noop_run.sh - Rescue a workflow run when the agent failed after producing only
# noop safe-outputs (transient AI model server error after meaningful work was captured).
#
# Exit codes:
#   0 - Agent produced only noop outputs; run is treated as a successful no-action
#   1 - Agent failed without meaningful outputs; original failure is propagated

OUTPUT_FILE="/tmp/gh-aw/agent_output.json"

echo "Checking if agent failure can be rescued (noop-only safe-outputs captured before failure)"

if [ ! -f "$OUTPUT_FILE" ]; then
  echo "No agent output file found - propagating agent failure"
  exit 1
fi

# Use node.js to safely parse the JSON agent output file.
# Node.js is guaranteed to be available on all GitHub Actions runners.
RESULT=$(GH_AW_RESCUE_OUTPUT_FILE="$OUTPUT_FILE" node -e "
const fs = require('fs');
const outputFile = process.env.GH_AW_RESCUE_OUTPUT_FILE;
try {
  const output = JSON.parse(fs.readFileSync(outputFile, 'utf8'));
  const items = output.items || [];
  const total = items.length;
  const noopCount = items.filter(i => i.type === 'noop').length;
  if (total > 0 && total === noopCount) {
    console.log('rescue');
  } else {
    console.log('propagate');
  }
} catch (e) {
  console.log('propagate');
}
" 2>/dev/null)

if [ "$RESULT" = "rescue" ]; then
  echo "Agent failed but captured only noop safe-output(s) before the transient error. Treating run as successful no-action."
  exit 0
else
  echo "Agent failed without noop-only outputs - propagating agent failure"
  exit 1
fi
