#!/bin/bash
set -e

# copilot_preflight_diagnostic.sh - Pre-flight diagnostic for Copilot engine on GHES
#
# This script performs diagnostic checks before executing Copilot CLI to provide
# clear error messages when Copilot is not properly configured on GHES instances.
#
# Checks performed:
# 1. Token exchange test - Validates COPILOT_GITHUB_TOKEN can exchange for Copilot access
# 2. GHES detection - Identifies GHES environments and validates configuration
# 3. API target validation - Ensures engine.api-target matches GITHUB_API_URL on GHES
#
# Exit codes:
#   0 - All checks passed, safe to proceed
#   1 - Critical failure, should fail the workflow

# Check if we're on GHES (non-GitHub.com environment)
IS_GHES=false
if [ "$GITHUB_SERVER_URL" != "https://github.com" ]; then
  IS_GHES=true
  echo "🔍 Detected GitHub Enterprise Server environment"
  echo "  Server URL: $GITHUB_SERVER_URL"
  echo "  API URL: $GITHUB_API_URL"
fi

# Test 1: Token exchange to Copilot inference API
echo ""
echo "🔍 Testing Copilot token exchange..."

# Construct the token exchange endpoint
TOKEN_EXCHANGE_URL="${GITHUB_API_URL}/copilot_internal/v2/token"

# Attempt token exchange using COPILOT_GITHUB_TOKEN
HTTP_STATUS=$(curl -s -o /tmp/copilot_token_exchange.json -w "%{http_code}" \
  -H "Authorization: Bearer ${COPILOT_GITHUB_TOKEN}" \
  -H "Accept: application/json" \
  "$TOKEN_EXCHANGE_URL" 2>&1 || echo "000")

if [ "$HTTP_STATUS" = "200" ]; then
  echo "✅ Token exchange successful (HTTP 200)"
  echo "   Copilot is licensed and accessible"
elif [ "$HTTP_STATUS" = "403" ]; then
  # Parse error message from response
  ERROR_MSG=$(cat /tmp/copilot_token_exchange.json 2>/dev/null | grep -o '"message":"[^"]*"' | cut -d'"' -f4 || echo "")

  echo "❌ Token exchange failed (HTTP 403)"
  echo ""

  # Check for specific error messages
  if echo "$ERROR_MSG" | grep -qi "not licensed"; then
    {
      echo "## ❌ Copilot Not Licensed"
      echo ""
      echo "The token exchange endpoint returned HTTP 403 with message:"
      echo "\`\`\`"
      echo "$ERROR_MSG"
      echo "\`\`\`"
      echo ""
      echo "**This means Copilot is not licensed for this user/organization on GHES.**"
      echo ""
      echo "### How to fix:"
      echo "1. Ask your GHES administrator to enable Copilot at the **enterprise level**"
      echo "2. Ensure a Copilot seat is assigned to your user account"
      echo "3. Verify your organization has Copilot enabled"
      echo ""
      echo "### GHES Admin Steps:"
      echo "- Navigate to Enterprise settings → Copilot"
      echo "- Enable Copilot for the enterprise"
      echo "- Assign licenses to organizations"
      echo "- Ensure users have seats assigned"
      echo ""
      echo "**Note:** This is a licensing issue, not a configuration problem with gh-aw."
    } >> "$GITHUB_STEP_SUMMARY"

    echo "Copilot is not licensed for this user/org on GHES." >&2
    echo "Ask your GHES admin to enable Copilot at the enterprise level and assign a seat." >&2
    exit 1

  elif echo "$ERROR_MSG" | grep -qi "not accessible by personal access token\|token type"; then
    {
      echo "## ❌ Incorrect Token Type"
      echo ""
      echo "The token exchange endpoint returned HTTP 403 with message:"
      echo "\`\`\`"
      echo "$ERROR_MSG"
      echo "\`\`\`"
      echo ""
      echo "**The token type is not supported for Copilot access.**"
      echo ""
      echo "### How to fix:"
      echo "- Ensure you're using a **fine-grained Personal Access Token** (starts with \`github_pat_\`)"
      echo "- Configure the token with **Copilot Requests: Read-only** permission"
      echo "- Do NOT use classic PATs (\`ghp_\`) or OAuth tokens (\`gho_\`)"
      echo ""
      echo "Create a fine-grained PAT at: https://${GITHUB_SERVER_URL#https://}/settings/personal-access-tokens/new"
    } >> "$GITHUB_STEP_SUMMARY"

    echo "Token type is not supported for Copilot." >&2
    echo "Use a fine-grained PAT with Copilot Requests permission." >&2
    exit 1

  else
    # Generic 403 error
    {
      echo "## ❌ Copilot Access Denied"
      echo ""
      echo "The token exchange endpoint returned HTTP 403:"
      echo "\`\`\`"
      echo "$ERROR_MSG"
      echo "\`\`\`"
      echo ""
      echo "**Common causes:**"
      echo "- Copilot not licensed for this user/organization"
      echo "- Incorrect token permissions"
      echo "- Token type not supported"
      echo ""
      echo "Contact your GHES administrator for assistance."
    } >> "$GITHUB_STEP_SUMMARY"

    echo "Token exchange failed with HTTP 403: $ERROR_MSG" >&2
    exit 1
  fi

elif [ "$HTTP_STATUS" = "401" ]; then
  {
    echo "## ❌ Invalid or Expired Token"
    echo ""
    echo "The token exchange endpoint returned HTTP 401 (Unauthorized)."
    echo ""
    echo "**This means COPILOT_GITHUB_TOKEN is invalid or expired.**"
    echo ""
    echo "### How to fix:"
    echo "1. Verify the secret is correctly configured in repository settings"
    echo "2. Check if the token has expired (fine-grained PATs have expiration dates)"
    echo "3. Regenerate the token if needed"
    echo "4. Ensure the token has **Copilot Requests: Read-only** permission"
  } >> "$GITHUB_STEP_SUMMARY"

  echo "COPILOT_GITHUB_TOKEN is invalid or expired (HTTP 401)" >&2
  exit 1

elif [ "$HTTP_STATUS" = "404" ]; then
  {
    echo "## ❌ Copilot Endpoint Not Found"
    echo ""
    echo "The token exchange endpoint returned HTTP 404 (Not Found)."
    echo ""
    echo "**This may indicate:**"
    echo "- GHES version does not support Copilot"
    echo "- Copilot infrastructure is not enabled on this instance"
    echo ""
    echo "### How to fix:"
    echo "- Verify GHES version supports GitHub Copilot"
    echo "- Ask your GHES admin to enable Copilot infrastructure"
    echo "- Check endpoint URL: \`$TOKEN_EXCHANGE_URL\`"
  } >> "$GITHUB_STEP_SUMMARY"

  echo "Copilot endpoint not found (HTTP 404) - GHES may not support Copilot" >&2
  exit 1

elif [ "$HTTP_STATUS" = "000" ] || [ -z "$HTTP_STATUS" ]; then
  echo "⚠️  Could not connect to token exchange endpoint"
  echo "   This may indicate network issues or firewall blocking"
  echo "   Proceeding with Copilot execution (will fail if endpoint is truly unavailable)"
  # Don't exit - let Copilot CLI fail with its own error if needed

else
  echo "⚠️  Unexpected response from token exchange endpoint (HTTP $HTTP_STATUS)"
  echo "   Proceeding with Copilot execution"
  # Don't exit - unexpected statuses should not block execution
fi

# Test 2: GHES-specific validation
if [ "$IS_GHES" = true ]; then
  echo ""
  echo "🔍 Running GHES-specific checks..."

  # Check if engine.api-target is set (should match GITHUB_API_URL)
  # This env var would be set by the compiler if engine.api-target is configured
  if [ -n "$COPILOT_API_TARGET" ]; then
    if [ "$COPILOT_API_TARGET" != "$GITHUB_API_URL" ]; then
      echo "⚠️  Warning: engine.api-target ($COPILOT_API_TARGET) does not match GITHUB_API_URL ($GITHUB_API_URL)"
      echo "   This may cause API routing issues"
    else
      echo "✅ engine.api-target matches GITHUB_API_URL"
    fi
  else
    echo "ℹ️  engine.api-target not configured (using default GITHUB_API_URL)"
  fi

  # Verify GHES API domain is accessible
  GHES_DOMAIN=$(echo "$GITHUB_API_URL" | sed -E 's|https?://([^/]+).*|\1|')
  if [ -n "$GHES_DOMAIN" ]; then
    echo "ℹ️  GHES API domain: $GHES_DOMAIN"
    echo "   Ensure this domain is in network.allowed if using firewall"
  fi
fi

echo ""
echo "✅ Pre-flight diagnostic completed"
echo "   Proceeding with Copilot CLI execution..."

# Clean up temporary files
rm -f /tmp/copilot_token_exchange.json

exit 0
