#!/bin/bash
set +o histexpand

# Updates copilotSetupStepsStaticSHA and copilotSetupStepsStaticSHA256 in
# pkg/cli/copilot_setup.go to the current HEAD commit of the main branch.
#
# Usage:
#   scripts/update-install-script-hashes.sh
#
# Requirements: gh (GitHub CLI, authenticated), sha256sum (coreutils), curl, sed
#
# Environment Variables:
#   GITHUB_TOKEN - Optional. GitHub token for authentication (used by gh CLI).
set -euo pipefail

REPO="github/gh-aw"
FILE="pkg/cli/copilot_setup.go"

# Accept a GITHUB_TOKEN forwarded from CI
if [ -n "${GITHUB_TOKEN:-}" ]; then
  export GH_TOKEN="$GITHUB_TOKEN"
fi

if ! command -v gh &>/dev/null; then
  echo "error: GitHub CLI (gh) not found. Install it from https://cli.github.com" >&2
  exit 1
fi

echo "Resolving HEAD SHA of ${REPO} main branch..."
NEW_SHA=$(gh api "repos/${REPO}/commits/main" --jq '.sha')
if [ -z "$NEW_SHA" ] || [ "${#NEW_SHA}" -ne 40 ]; then
  echo "error: could not resolve main HEAD SHA (got: '${NEW_SHA}')" >&2
  exit 1
fi
echo "  HEAD SHA: ${NEW_SHA}"

SCRIPT_URL="https://raw.githubusercontent.com/${REPO}/${NEW_SHA}/install-gh-aw.sh"
echo "Downloading install-gh-aw.sh at ${NEW_SHA}..."
TMPFILE=$(mktemp /tmp/install-gh-aw-XXXXXX.sh)
trap 'rm -f "$TMPFILE"' EXIT
curl -fsSL "$SCRIPT_URL" -o "$TMPFILE"

NEW_SHA256=$(sha256sum "$TMPFILE" | awk '{print $1}')
if [ -z "$NEW_SHA256" ] || [ "${#NEW_SHA256}" -ne 64 ]; then
  echo "error: could not compute SHA256 (got: '${NEW_SHA256}')" >&2
  exit 1
fi
echo "  SHA256:   ${NEW_SHA256}"

OLD_SHA=$(grep 'copilotSetupStepsStaticSHA = ' "$FILE" | sed 's/.*"\([^"]*\)".*/\1/')
OLD_SHA256=$(grep 'copilotSetupStepsStaticSHA256 = ' "$FILE" | sed 's/.*"\([^"]*\)".*/\1/')

if [ "$OLD_SHA" = "$NEW_SHA" ] && [ "$OLD_SHA256" = "$NEW_SHA256" ]; then
  echo "No changes needed — values are already up to date."
  exit 0
fi

echo "Updating ${FILE}..."
sed -i \
  -e "s|copilotSetupStepsStaticSHA = \"[0-9a-f]*\"|copilotSetupStepsStaticSHA = \"${NEW_SHA}\"|" \
  -e "s|copilotSetupStepsStaticSHA256 = \"[0-9a-f]*\"|copilotSetupStepsStaticSHA256 = \"${NEW_SHA256}\"|" \
  "$FILE"

echo "Done. Review the diff and commit:"
echo "  git diff ${FILE}"
