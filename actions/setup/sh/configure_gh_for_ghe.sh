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

  # If the host is github.com, no configuration is needed
  if [ "$github_host" = "github.com" ]; then
    echo "Using public GitHub (github.com) - no additional gh configuration needed"
    return 0
  fi

  echo "Configuring gh CLI for GitHub Enterprise host: ${github_host}"

  # Check if gh is installed
  if ! command -v gh &> /dev/null; then
    echo "::error::gh CLI is not installed. Please install gh CLI to use with GitHub Enterprise."
    exit 1
  fi

  # Check if GH_TOKEN is set
  if [ -z "${GH_TOKEN}" ]; then
    echo "::error::GH_TOKEN environment variable is not set. gh CLI requires authentication."
    exit 1
  fi

  # Configure gh to use the enterprise host
  # We use 'gh auth login' with the token to configure the host
  echo "Authenticating gh CLI with host: ${github_host}"

  # Use gh auth login with --with-token to configure the host
  # This sets up gh to use the correct API endpoint for the enterprise host
  echo "${GH_TOKEN}" | gh auth login --hostname "${github_host}" --with-token

  if [ $? -eq 0 ]; then
    echo "✓ Successfully configured gh CLI for ${github_host}"

    # Verify the configuration
    if gh auth status --hostname "${github_host}" &> /dev/null; then
      echo "✓ Verified gh CLI authentication for ${github_host}"
    else
      echo "::warning::gh CLI configured but authentication verification failed"
    fi
  else
    echo "::error::Failed to configure gh CLI for ${github_host}"
    exit 1
  fi

  # Set GH_HOST environment variable to ensure gh uses the correct host for subsequent commands
  export GH_HOST="${github_host}"
  if [ -n "${GITHUB_ENV:-}" ]; then
    echo "GH_HOST=${github_host}" >> "${GITHUB_ENV}"
  fi
  echo "✓ Set GH_HOST environment variable to ${github_host}"
}

# Run main function
main

# Restore original errexit state so sourcing this script does not leak set -e
case "$ORIGINAL_SHELL_FLAGS" in
  *e*) set -e ;;
  *) set +e ;;
esac
