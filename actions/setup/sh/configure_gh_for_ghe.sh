#!/usr/bin/env bash
# Configure gh CLI for GitHub Enterprise
#
# This script configures the gh CLI to work with GitHub Enterprise environments
# by detecting the GitHub host from environment variables and setting up gh to
# authenticate with that host.
#
# Environment variables checked (in priority order):
# 1. GITHUB_SERVER_URL - GitHub Actions standard (e.g., https://MYORG.ghe.com)
# 2. GITHUB_ENTERPRISE_HOST - GitHub Enterprise standard (e.g., MYORG.ghe.com)
# 3. GITHUB_HOST - GitHub Enterprise standard (e.g., MYORG.ghe.com)
# 4. GH_HOST - GitHub CLI standard (e.g., MYORG.ghe.com)
#
# If none are set, defaults to github.com (public GitHub).

ORIGINAL_SHELL_FLAGS="$-"
set -e

# Function to normalize GitHub host URL
normalize_github_host() {
  local host="$1"

  # Remove trailing slashes
  host="${host%/}"

  # Extract hostname from URL if it's a full URL
  if [[ "$host" =~ ^https?:// ]]; then
    host="${host#http://}"
    host="${host#https://}"
    host="${host%%/*}"
  fi

  echo "$host"
}

# Detect GitHub host from environment variables
detect_github_host() {
  local host=""

  if [ -n "${GITHUB_SERVER_URL}" ]; then
    host=$(normalize_github_host "${GITHUB_SERVER_URL}")
    echo "Detected GitHub host from GITHUB_SERVER_URL: ${host}" >&2
  elif [ -n "${GITHUB_ENTERPRISE_HOST}" ]; then
    host=$(normalize_github_host "${GITHUB_ENTERPRISE_HOST}")
    echo "Detected GitHub host from GITHUB_ENTERPRISE_HOST: ${host}" >&2
  elif [ -n "${GITHUB_HOST}" ]; then
    host=$(normalize_github_host "${GITHUB_HOST}")
    echo "Detected GitHub host from GITHUB_HOST: ${host}" >&2
  elif [ -n "${GH_HOST}" ]; then
    host=$(normalize_github_host "${GH_HOST}")
    echo "Detected GitHub host from GH_HOST: ${host}" >&2
  else
    host="github.com"
    echo "No GitHub host environment variable set, using default: ${host}" >&2
  fi

  echo "$host"
}

# Main configuration
main() {
  local github_host
  github_host=$(detect_github_host)

  # If the host is github.com, no GHE configuration is needed.
  # However, we must clear any stale GH_HOST value to prevent gh CLI
  # from targeting the wrong host (e.g., a leftover localhost:18443
  # from a prior DIFC proxy run). See #24208.
  if [ "$github_host" = "github.com" ]; then
    if [ -n "${GH_HOST:-}" ] && [ "${GH_HOST}" != "github.com" ]; then
      echo "Clearing stale GH_HOST=${GH_HOST} (expected github.com)"
      unset GH_HOST
      if [ -n "${GITHUB_ENV:-}" ]; then
        echo "GH_HOST=github.com" >> "${GITHUB_ENV}"
      fi
    fi
    echo "Using public GitHub (github.com) - no additional gh configuration needed"
    # Clear any stale GH_HOST to prevent gh CLI mismatches
    if [ -n "${GH_HOST:-}" ] && [ "${GH_HOST}" != "github.com" ]; then
      echo "Clearing stale GH_HOST" >&2
      unset GH_HOST
      if [ -n "${GITHUB_ENV:-}" ]; then
        echo "GH_HOST=" >> "${GITHUB_ENV}"
      fi
    fi
    return 0
  fi

  echo "Configuring gh CLI for GitHub Enterprise host: ${github_host}"

  # Check if gh is installed
  if ! command -v gh &> /dev/null; then
    echo "::error::gh CLI is not installed. Please install gh CLI to use with GitHub Enterprise."
    exit 1
  fi

  # When GH_TOKEN is already set in the environment, running 'gh auth login' would fail with:
  #   "The value of the GH_TOKEN environment variable is being used for authentication.
  #    To have GitHub CLI store credentials instead, first clear the value from the environment."
  # In this case, gh CLI will already authenticate via GH_TOKEN. This script still requires gh
  # to be installed (checked above); here we only need to set GH_HOST so gh knows which host
  # to target.
  if [ -n "${GH_TOKEN}" ]; then
    echo "GH_TOKEN is set — skipping gh auth login and exporting GH_HOST (gh CLI must already be installed)"
    export GH_HOST="${github_host}"
    if [ -n "${GITHUB_ENV:-}" ]; then
      echo "GH_HOST=${github_host}" >> "${GITHUB_ENV}"
    fi
    echo "✓ Set GH_HOST=${github_host}"
    return 0
  fi

  echo "::error::GH_TOKEN environment variable is not set. gh CLI requires authentication."
  exit 1
}

# Run main function
main

# Restore original errexit state so sourcing this script does not leak set -e
case "$ORIGINAL_SHELL_FLAGS" in
  *e*) set -e ;;
  *) set +e ;;
esac
