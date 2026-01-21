#!/bin/sh
# Start Safe Outputs MCP Server with Git Configuration
# This script ensures git is properly configured in the container before starting the MCP server

set -e

# Install git if not present (Alpine package manager)
if ! command -v git >/dev/null 2>&1; then
    echo "Installing git..." >&2
    apk add --no-cache git >/dev/null 2>&1
fi

# Configure git user (required for git operations)
git config --global user.email "github-actions[bot]@users.noreply.github.com"
git config --global user.name "github-actions[bot]"

# Mark workspace as safe directory for git
# This is required because the container user may differ from the workspace owner
if [ -n "$GITHUB_WORKSPACE" ]; then
    git config --global --add safe.directory "$GITHUB_WORKSPACE"
fi

# Start the MCP server
exec node /opt/gh-aw/safeoutputs/mcp-server.cjs
