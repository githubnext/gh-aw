#!/usr/bin/env bash
# Stop DIFC proxy for pre-agent gh CLI steps
# This script stops the awmg proxy container, removes the proxy CA certificate from the
# system trust store (if it was installed), and clears the GH_HOST environment variable.
# The proxy must be stopped before the MCP gateway starts to avoid double-filtering traffic.
#
# Environment:
#   GITHUB_ENV - Path to GitHub Actions environment file

set -e

docker rm -f awmg-proxy 2>/dev/null || true
git remote remove proxy 2>/dev/null || true

# Remove the DIFC proxy CA certificate from the system trust store if present,
# to avoid permanently expanding the trusted root set on persistent/self-hosted runners.
DIFC_PROXY_CA_CERT="/usr/local/share/ca-certificates/awmg-proxy-difc.crt"
if [ -f "$DIFC_PROXY_CA_CERT" ]; then
  rm -f "$DIFC_PROXY_CA_CERT" || true
  if command -v update-ca-certificates >/dev/null 2>&1; then
    sudo update-ca-certificates || true
  fi
fi

# Only clear GH_HOST if it was set to the proxy address; preserve any pre-existing
# GH_HOST value (e.g., from configure_gh_for_ghe.sh on GitHub Enterprise runners).
if [ "${GH_HOST:-}" = "localhost:18443" ]; then
  echo "GH_HOST=" >> "$GITHUB_ENV"
fi

# Restore GITHUB_API_URL and GITHUB_GRAPHQL_URL to their original values.
if [ "${GITHUB_API_URL:-}" = "https://localhost:18443/api/v3" ]; then
  echo "GITHUB_API_URL=${GH_AW_ORIGINAL_GITHUB_API_URL:-}" >> "$GITHUB_ENV"
fi
if [ "${GITHUB_GRAPHQL_URL:-}" = "https://localhost:18443/api/graphql" ]; then
  echo "GITHUB_GRAPHQL_URL=${GH_AW_ORIGINAL_GITHUB_GRAPHQL_URL:-}" >> "$GITHUB_ENV"
fi

# Clear the Node.js CA certs override set by start_difc_proxy.sh.
echo "NODE_EXTRA_CA_CERTS=" >> "$GITHUB_ENV"

echo "DIFC proxy stopped"
