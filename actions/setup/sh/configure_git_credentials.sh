#!/usr/bin/env bash
set +o histexpand

#
# configure_git_credentials.sh - Configure Git identity, safe directory, and remote authentication
#
# Sets up Git global configuration for use in GitHub Actions workflows and the gh-aw-node
# container. Always configures git identity and safe.directory trust. Optionally configures
# the remote URL for authentication when credentials are provided.
#
# Required environment variables:
#   GITHUB_WORKSPACE     - Workspace directory path (for safe.directory)
#
# Optional environment variables for remote authentication:
#   REPO_NAME            - Repository slug (e.g., "org/repo"); falls back to GITHUB_REPOSITORY
#   SERVER_URL           - GitHub server URL; falls back to GITHUB_SERVER_URL
#   GITHUB_TOKEN         - Authentication token; falls back to GIT_TOKEN
#
# Exit codes:
#   0 - Success
#   1 - Error

set -euo pipefail

# Configure git identity
git config --global user.email "github-actions[bot]@users.noreply.github.com"
git config --global user.name "github-actions[bot]"
git config --global am.keepcr true

# Trust the workspace directory to avoid "dubious ownership" errors
# when the repository is owned by a different user (e.g., in mounted containers)
if [ -n "${GITHUB_WORKSPACE:-}" ]; then
  git config --global --add safe.directory "${GITHUB_WORKSPACE}"
fi

# Configure remote URL authentication when credentials are provided
REPO="${REPO_NAME:-${GITHUB_REPOSITORY:-}}"
URL="${SERVER_URL:-${GITHUB_SERVER_URL:-}}"
TOKEN="${GITHUB_TOKEN:-${GIT_TOKEN:-}}"

if [ -n "${REPO}" ] && [ -n "${URL}" ] && [ -n "${TOKEN}" ]; then
  URL_STRIPPED="${URL#https://}"
  git remote set-url origin "https://x-access-token:${TOKEN}@${URL_STRIPPED}/${REPO}.git"
fi

echo "Git configured with standard GitHub Actions identity"
