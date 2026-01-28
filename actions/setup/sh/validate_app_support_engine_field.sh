#!/bin/bash
set -e

# validate_app_support_engine_field.sh - Validate GitHub App configuration for engine authentication
#
# Usage: validate_app_support_engine_field.sh ENGINE_NAME DOCS_URL
#
# Arguments:
#   ENGINE_NAME : Name of the engine requiring GitHub App authentication (e.g., "GitHub Copilot CLI")
#   DOCS_URL    : Documentation URL for GitHub App configuration
#
# Environment variables (required):
#   APP_ID         : GitHub App ID (e.g., "${{ vars.APP_ID }}")
#   APP_PRIVATE_KEY: GitHub App private key (e.g., "${{ secrets.APP_PRIVATE_KEY }}")
#
# Exit codes:
#   0 - GitHub App configuration is valid
#   1 - GitHub App configuration is invalid or missing

# Parse arguments
if [ "$#" -ne 2 ]; then
  echo "Usage: $0 ENGINE_NAME DOCS_URL" >&2
  exit 1
fi

ENGINE_NAME="$1"
DOCS_URL="$2"

echo "Validating GitHub App configuration for $ENGINE_NAME..."
echo ""

# Validate app-id variable
if [ -z "$APP_ID" ]; then
  echo "❌ ERROR: GitHub App ID is not set"
  echo ""
  echo "To use GitHub App authentication with $ENGINE_NAME, you need to configure:"
  echo "  - vars.APP_ID (GitHub App ID)"
  echo "  - secrets.APP_PRIVATE_KEY (GitHub App private key)"
  echo ""
  echo "Documentation: $DOCS_URL"
  exit 1
fi

# Validate private-key secret
if [ -z "$APP_PRIVATE_KEY" ]; then
  echo "❌ ERROR: GitHub App private key is not set"
  echo ""
  echo "To use GitHub App authentication with $ENGINE_NAME, you need to configure:"
  echo "  - vars.APP_ID (GitHub App ID)"
  echo "  - secrets.APP_PRIVATE_KEY (GitHub App private key)"
  echo ""
  echo "Documentation: $DOCS_URL"
  exit 1
fi

echo "✅ GitHub App configuration validated successfully"

# Set step output to indicate verification succeeded
if [ -n "$GITHUB_OUTPUT" ]; then
  echo "verification_result=success" >> "$GITHUB_OUTPUT"
fi
