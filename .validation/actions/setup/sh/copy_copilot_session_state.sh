#!/usr/bin/env bash
set +o histexpand

# Copy the entire Copilot session-state directory to the agent logs folder
# for artifact collection. This ensures all session files (events.jsonl,
# session.db, plan.md, checkpoints, etc.) are in /tmp/gh-aw/ where secret
# redaction can scan them and they get uploaded as artifacts.
#
# Copilot CLI writes session data inside UUID-named subdirectories:
#   ~/.copilot/session-state/<session-uuid>/events.jsonl
#   ~/.copilot/session-state/<session-uuid>/session.db
#   ~/.copilot/session-state/<session-uuid>/plan.md
#   ~/.copilot/session-state/<session-uuid>/checkpoints/
#   ~/.copilot/session-state/<session-uuid>/files/

set -euo pipefail

SESSION_STATE_DIR="$HOME/.copilot/session-state"
LOGS_DIR="/tmp/gh-aw/sandbox/agent/logs/copilot-session-state"

if [ -d "$SESSION_STATE_DIR" ]; then
  echo "Copying Copilot session state from $SESSION_STATE_DIR to $LOGS_DIR"
  mkdir -p "$LOGS_DIR"
  cp -rv "$SESSION_STATE_DIR"/. "$LOGS_DIR/" 2>/dev/null || true
  echo "Session state directory copied successfully"
else
  echo "No session-state directory found at $SESSION_STATE_DIR"
fi
