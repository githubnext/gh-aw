#!/bin/bash
set -e

# validate_app_support_engine_field.sh - Validate GitHub App configuration for engine authentication
#
# Usage: validate_app_support_engine_field.sh APP_ID_VAR_NAME APP_PRIVATE_KEY_VAR_NAME ENGINE_NAME DOCS_URL
#
# Arguments:
#   APP_ID_VAR_NAME         : Name of environment variable containing GitHub App ID (e.g., "APP_ID")
#   APP_PRIVATE_KEY_VAR_NAME: Name of environment variable containing GitHub App private key (e.g., "APP_PRIVATE_KEY")
#   ENGINE_NAME             : Name of the engine requiring GitHub App authentication (e.g., "GitHub Copilot CLI")
#   DOCS_URL                : Documentation URL for GitHub App configuration
#
# Environment variables:
#   The script expects the app ID and private key values to be available as environment variables
#   with the names specified in APP_ID_VAR_NAME and APP_PRIVATE_KEY_VAR_NAME
#
# Exit codes:
#   0 - GitHub App configuration is valid
#   1 - GitHub App configuration is invalid or missing

# Parse arguments
if [ "$#" -ne 4 ]; then
  echo "Usage: $0 APP_ID_VAR_NAME APP_PRIVATE_KEY_VAR_NAME ENGINE_NAME DOCS_URL" >&2
  exit 1
fi

APP_ID_VAR_NAME="$1"
APP_PRIVATE_KEY_VAR_NAME="$2"
ENGINE_NAME="$3"
DOCS_URL="$4"

echo "Validating GitHub App configuration for $ENGINE_NAME..."
echo ""

# Use indirect expansion to get the values of the variables
APP_ID_VALUE="${!APP_ID_VAR_NAME}"
APP_PRIVATE_KEY_VALUE="${!APP_PRIVATE_KEY_VAR_NAME}"

# Validate app-id variable
if [ -z "$APP_ID_VALUE" ]; then
  echo "❌ ERROR: GitHub App ID is not set (expected in environment variable: $APP_ID_VAR_NAME)"
  echo ""
  echo "To use GitHub App authentication with $ENGINE_NAME, you need to configure:"
  echo "  - GitHub App ID (via variable or secret)"
  echo "  - GitHub App private key (via secret)"
  echo ""
  echo "Documentation: $DOCS_URL"
  exit 1
fi

# Validate private-key secret
if [ -z "$APP_PRIVATE_KEY_VALUE" ]; then
  echo "❌ ERROR: GitHub App private key is not set (expected in environment variable: $APP_PRIVATE_KEY_VAR_NAME)"
  echo ""
  echo "To use GitHub App authentication with $ENGINE_NAME, you need to configure:"
  echo "  - GitHub App ID (via variable or secret)"
  echo "  - GitHub App private key (via secret)"
  echo ""
  echo "Documentation: $DOCS_URL"
  exit 1
fi

echo "✅ GitHub App configuration validated successfully"

# Set step output to indicate verification succeeded
if [ -n "$GITHUB_OUTPUT" ]; then
  echo "verification_result=success" >> "$GITHUB_OUTPUT"
fi
