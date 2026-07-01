#!/usr/bin/env bash
set +o histexpand

# Ensure the GitHub CLI (gh) meets the minimum required version.
# If gh is missing or below the minimum, installs or upgrades it via install_gh_cli.sh.
#
# Usage: ensure_gh_cli_min_version.sh REQUIRED_VERSION
#
# Arguments:
#   REQUIRED_VERSION - minimum acceptable gh version (e.g., 2.90.0)

set -euo pipefail

REQUIRED="${1:-}"
if [ -z "$REQUIRED" ]; then
  echo "::error::Usage: $0 REQUIRED_VERSION"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Returns 0 if gh is installed and meets or exceeds REQUIRED, 1 otherwise.
gh_meets_min_version() {
  if ! command -v gh &>/dev/null; then
    return 1
  fi
  local current
  current=$(gh --version | awk 'NR==1 {print $3}')
  printf '%s\n%s\n' "$REQUIRED" "$current" | sort -V -C
}

if gh_meets_min_version; then
  GH_VERSION=$(gh --version | awk 'NR==1 {print $3}')
  echo "gh ${GH_VERSION} meets minimum required ${REQUIRED}, skipping upgrade."
  exit 0
fi

if ! command -v gh &>/dev/null; then
  echo "gh CLI not found, installing..."
else
  GH_VERSION=$(gh --version | awk 'NR==1 {print $3}')
  echo "gh ${GH_VERSION} is older than ${REQUIRED}, upgrading..."
fi

bash "${SCRIPT_DIR}/install_gh_cli.sh"

GH_VERSION=$(gh --version | awk 'NR==1 {print $3}')
echo "gh version after upgrade: ${GH_VERSION}"
if ! printf '%s\n%s\n' "$REQUIRED" "$GH_VERSION" | sort -V -C; then
  echo "::error::gh ${GH_VERSION} is older than required ${REQUIRED} (gh skill support requires v${REQUIRED}+)"
  exit 1
fi

echo "✓ gh ${GH_VERSION} meets minimum required ${REQUIRED}"
