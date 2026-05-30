#!/usr/bin/env bash
set +o histexpand

# Append the agent's step summary to the real $GITHUB_STEP_SUMMARY.
# The file was written by the agent and already redacted for secrets.
# This is a no-op when the file is empty (agent wrote nothing).
if [ -s /tmp/gh-aw/agent-step-summary.md ]; then
  cat /tmp/gh-aw/agent-step-summary.md >> "$GITHUB_STEP_SUMMARY"
fi
