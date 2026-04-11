#!/usr/bin/env bash
#
# detect_mcp_policy_error.sh - Detect MCP servers blocked by policy
#
# Checks the agent stdio log for the known error pattern that indicates
# Copilot CLI blocked MCP servers due to enterprise/organization policy
# (e.g., "MCP servers were blocked by policy: 'github', 'safeoutputs'").
#
# Sets the GitHub Actions output variable:
#   mcp_policy_error=true    if the error pattern is found
#   mcp_policy_error=false   otherwise
#
# Exit codes:
#   0 - Always succeeds (uses continue-on-error in the workflow step)

set -euo pipefail

LOG_FILE="/tmp/gh-aw/agent-stdio.log"

if [ -f "$LOG_FILE" ] && grep -q "MCP servers were blocked by policy:" "$LOG_FILE"; then
  echo "Detected MCP policy error in agent log"
  echo "mcp_policy_error=true" >> "$GITHUB_OUTPUT"
else
  echo "mcp_policy_error=false" >> "$GITHUB_OUTPUT"
fi
