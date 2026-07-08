#!/bin/bash
set +o histexpand

# check_oauth_tokens.sh - Check that automation tokens are not OAuth tokens.
#
# OAuth tokens (gho_...) should not be used for automation as they:
#   - Are typically over-provisioned and grant broad access
#   - Expire when the user logs out or changes their password
#   - Cannot be scoped to specific repositories or permissions
#
# This script checks the following tokens:
#   COPILOT_GITHUB_TOKEN          - Token for GitHub Copilot API access
#   GH_AW_GITHUB_TOKEN            - Token for GitHub API access in agentic workflows
#   GH_AW_GITHUB_MCP_SERVER_TOKEN - Token for the GitHub MCP server
#
# Environment variables (set to the actual token values):
#   COPILOT_GITHUB_TOKEN
#   GH_AW_GITHUB_TOKEN
#   GH_AW_GITHUB_MCP_SERVER_TOKEN

set -e

found_error=false

check_token_not_oauth() {
  local token_name="$1"
  local token_value="${!token_name}"

  if [ -z "$token_value" ]; then
    # Token not configured - nothing to check
    return 0
  fi

  if [[ "$token_value" == gho_* ]]; then
    {
      echo "❌ Error: $token_name is an OAuth token (gho_...)"
      echo ""
      echo "OAuth tokens are not suitable for automation:"
      echo "- They are typically over-provisioned and grant broad access to user resources"
      echo "- They may expire when the user logs out or changes their password"
      echo "- They cannot be scoped to specific repositories or permissions"
      echo ""
      echo "**How to fix:** Replace $token_name with a fine-grained Personal Access Token."
      echo "Create one at: https://github.com/settings/personal-access-tokens/new"
    } >> "$GITHUB_STEP_SUMMARY"

    echo "Error: $token_name is an OAuth token (gho_...)" >&2
    echo "OAuth tokens are not suitable for automation." >&2
    echo "Replace $token_name with a fine-grained PAT (github_pat_...) at: https://github.com/settings/personal-access-tokens/new" >&2
    found_error=true
  fi
}

check_token_not_oauth COPILOT_GITHUB_TOKEN
check_token_not_oauth GH_AW_GITHUB_TOKEN
check_token_not_oauth GH_AW_GITHUB_MCP_SERVER_TOKEN

if [ "$found_error" = true ]; then
  exit 1
fi
