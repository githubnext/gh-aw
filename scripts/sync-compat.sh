#!/bin/bash

# sync-compat.sh - Keep .github/aw/compat.json in sync with DefaultCopilotVersion
#
# Reads DefaultCopilotVersion from pkg/constants/version_constants.go and updates
# the max-agent field of the latest interval (the entry with "open": true) in
# .github/aw/compat.json.
#
# Usage:
#   sync-compat.sh [--check]
#
# Options:
#   --check   Exit with code 1 if compat.json is out of sync instead of updating it.
#
# Exit codes:
#   0 - compat.json is already in sync, or was updated successfully
#   1 - compat.json is out of sync (only when --check is passed), or an error occurred

set -euo pipefail

# Script must be run from the repository root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

CONSTANTS_FILE="$REPO_ROOT/pkg/constants/version_constants.go"
COMPAT_FILE="$REPO_ROOT/.github/aw/compat.json"

CHECK_ONLY=0
for arg in "$@"; do
  if [ "$arg" = "--check" ]; then
    CHECK_ONLY=1
  fi
done

# Extract DefaultCopilotVersion from Go constants file
COPILOT_VERSION=$(grep -oP '^\s*const DefaultCopilotVersion\s+Version\s*=\s*"\K[^"]+' "$CONSTANTS_FILE")
if [ -z "$COPILOT_VERSION" ]; then
  echo "Error: could not extract DefaultCopilotVersion from $CONSTANTS_FILE" >&2
  exit 1
fi

# Read the current max-agent of the latest interval (entry with "open": true)
CURRENT_MAX_AGENT=$(jq -r '."agent-compat-v1".copilot[] | select(.open == true) | ."max-agent"' "$COMPAT_FILE")
if [ -z "$CURRENT_MAX_AGENT" ] || [ "$CURRENT_MAX_AGENT" = "null" ]; then
  echo "Error: could not find latest interval (\"open\": true) in $COMPAT_FILE" >&2
  exit 1
fi

if [ "$CURRENT_MAX_AGENT" = "$COPILOT_VERSION" ]; then
  echo "compat.json is already in sync (max-agent=$COPILOT_VERSION)"
  exit 0
fi

if [ "$CHECK_ONLY" = "1" ]; then
  echo "Error: compat.json is out of sync: max-agent=$CURRENT_MAX_AGENT, DefaultCopilotVersion=$COPILOT_VERSION" >&2
  echo "Run 'make sync-compat' to update." >&2
  exit 1
fi

# Update max-agent in the latest interval
UPDATED=$(jq --arg version "$COPILOT_VERSION" \
  '."agent-compat-v1".copilot = [."agent-compat-v1".copilot[] | if .open == true then ."max-agent" = $version else . end]' \
  "$COMPAT_FILE")

echo "$UPDATED" > "$COMPAT_FILE"
echo "Updated compat.json: max-agent $CURRENT_MAX_AGENT -> $COPILOT_VERSION"
