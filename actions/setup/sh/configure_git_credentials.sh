#!/bin/sh

#
# configure_git_credentials.sh - Configure Git identity, safe directory, and remote authentication
#
# Sets up Git global configuration for use in GitHub Actions workflows and the gh-aw-node
# container. Always configures git identity and safe.directory trust. Configures the remote
# URL for authentication when credentials are provided; silently skips auth when any required
# credential variable (GITHUB_REPOSITORY, GITHUB_SERVER_URL, GITHUB_TOKEN) is absent.
#
# Required environment variables:
#   GITHUB_WORKSPACE     - Workspace directory path (for safe.directory)
#
# Optional environment variables for remote authentication:
#   GITHUB_REPOSITORY    - Repository slug (e.g., "org/repo")
#   GITHUB_SERVER_URL    - GitHub server URL (with or without https:// prefix)
#   GITHUB_TOKEN         - Authentication token; falls back to GIT_TOKEN
#
# Exit codes:
#   0 - Success
#   1 - Error

set -eu

# Configure git identity
git config --global user.email "github-actions[bot]@users.noreply.github.com"
git config --global user.name "github-actions[bot]"
git config --global am.keepcr true

# Trust the workspace directory to avoid "dubious ownership" errors
# when the repository is owned by a different user (e.g., in mounted containers)
if [ -n "${GITHUB_WORKSPACE:-}" ]; then
  git config --global --add safe.directory "${GITHUB_WORKSPACE}"
fi

# Configure remote URL authentication when all required credentials are present.
# Silently skips when any variable is absent (e.g., inside the safeoutputs container
# where GITHUB_SERVER_URL is intentionally not exposed).
REPO="${GITHUB_REPOSITORY:-}"
URL="${GITHUB_SERVER_URL:-}"
TOKEN="${GITHUB_TOKEN:-${GIT_TOKEN:-}}"

if [ -n "${REPO}" ] && [ -n "${URL}" ] && [ -n "${TOKEN}" ]; then
  URL_STRIPPED="${URL#https://}"
  git remote set-url origin "https://x-access-token:${TOKEN}@${URL_STRIPPED}/${REPO}.git"
fi

echo "Git configured with standard GitHub Actions identity" >&2
