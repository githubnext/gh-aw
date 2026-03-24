#!/usr/bin/env bash
# Stop DIFC proxy for pre-agent gh CLI steps
# This script stops the awmg proxy container and clears the GH_HOST environment variable.
# The proxy must be stopped before the MCP gateway starts to avoid double-filtering traffic.
#
# Environment:
#   GITHUB_ENV - Path to GitHub Actions environment file

set -e

docker rm -f awmg-proxy 2>/dev/null || true
git remote remove proxy 2>/dev/null || true
echo "GH_HOST=" >> "$GITHUB_ENV"
echo "DIFC proxy stopped"
