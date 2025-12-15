#!/usr/bin/env bash
set -euo pipefail

# Record a checkpoint during agent execution
# Usage: record_checkpoint.sh <type> <description>
# Types: tool_call, patch, eval, decision, safe_output

TRACE_DIR="${TRACE_DIR:-trace}"
CHECKPOINT_TYPE="${1:-tool_call}"
CHECKPOINT_DESC="${2:-Agent action}"

# Auto-increment checkpoint ID
CHECKPOINT_COUNTER_FILE="$TRACE_DIR/.checkpoint_counter"
if [[ -f "$CHECKPOINT_COUNTER_FILE" ]]; then
    COUNTER=$(cat "$CHECKPOINT_COUNTER_FILE")
else
    COUNTER=0
fi

COUNTER=$((COUNTER + 1))
echo "$COUNTER" > "$CHECKPOINT_COUNTER_FILE"

CHECKPOINT_ID=$(printf "c%03d" "$COUNTER")

# Get repo SHA
REPO_SHA="$(git rev-parse HEAD 2>/dev/null || echo 'unknown')"

# Build checkpoint JSON and append to checkpoints.jsonl
cat >> "$TRACE_DIR/checkpoints.jsonl" <<EOF
{"id":"$CHECKPOINT_ID","ts":"$(date -u +%Y-%m-%dT%H:%M:%SZ)","kind":"$CHECKPOINT_TYPE","name":"$CHECKPOINT_DESC","repo_sha":"$REPO_SHA"}
EOF

echo "✅ Checkpoint $CHECKPOINT_ID: $CHECKPOINT_TYPE - $CHECKPOINT_DESC"
