#!/bin/sh

#
# start_safe_outputs_mcp.sh - Configure git and launch the safe-outputs MCP server
#
# This script is copied to ${RUNNER_TEMP}/gh-aw/safeoutputs/ by setup.sh
# and run via: sh "${RUNNER_TEMP}/gh-aw/safeoutputs/start_safe_outputs_mcp.sh"
#
# It runs configure_git_credentials.sh first (configures git identity and
# safe.directory, skips remote auth since GITHUB_SERVER_URL is not exposed
# to the safeoutputs container), then launches the safe-outputs MCP server.

set -e

sh "${RUNNER_TEMP}/gh-aw/safeoutputs/configure_git_credentials.sh" >&2

exec node "${RUNNER_TEMP}/gh-aw/safeoutputs/safe_outputs_mcp_server.cjs"
