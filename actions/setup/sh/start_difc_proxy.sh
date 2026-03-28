#!/usr/bin/env bash
# Start DIFC proxy for pre-agent gh CLI steps
# This script starts the awmg proxy container that routes gh CLI calls
# through DIFC integrity filtering before the agent runs.
#
# Arguments:
#   $1 - POLICY: JSON guard policy string
#   $2 - CONTAINER_IMAGE: Container image to use (e.g., ghcr.io/github/gh-aw-mcpg:v0.2.2)
#
# Environment:
#   GH_TOKEN - GitHub token passed to the proxy container
#   GITHUB_SERVER_URL - GitHub server URL for upstream routing (e.g. https://github.com or https://TENANT.ghe.com)
#   GITHUB_REPOSITORY - Repository name (owner/repo) for git remote
#   GITHUB_ENV - Path to GitHub Actions environment file

set -e

POLICY="$1"
CONTAINER_IMAGE="$2"

if [ -z "$POLICY" ]; then
  echo "::warning::DIFC proxy policy not specified, skipping proxy start"
  exit 0
fi

if [ -z "$CONTAINER_IMAGE" ]; then
  echo "::warning::DIFC proxy container image not specified, skipping proxy start"
  exit 0
fi

PROXY_LOG_DIR=/tmp/gh-aw/proxy-logs
MCP_LOG_DIR=/tmp/gh-aw/mcp-logs

mkdir -p "$PROXY_LOG_DIR" "$MCP_LOG_DIR"

echo "Starting DIFC proxy container: $CONTAINER_IMAGE"

docker run -d --name awmg-proxy --network host \
  -e GH_TOKEN \
  -e GITHUB_SERVER_URL \
  -e DEBUG='*' \
  -v "$PROXY_LOG_DIR:$PROXY_LOG_DIR" \
  -v "$MCP_LOG_DIR:$MCP_LOG_DIR" \
  "$CONTAINER_IMAGE" proxy \
    --policy "$POLICY" \
    --listen 0.0.0.0:18443 \
    --log-dir "$MCP_LOG_DIR" \
    --tls --tls-dir "$PROXY_LOG_DIR/proxy-tls" \
    --guards-mode filter \
    --trusted-bots github-actions[bot],github-actions,dependabot[bot],copilot

# Wait for TLS cert + health check (up to 30s)
CA_INSTALLED=false
PROXY_READY=false
for i in $(seq 1 30); do
  if [ -f "$PROXY_LOG_DIR/proxy-tls/ca.crt" ]; then
    if [ "$CA_INSTALLED" = "false" ]; then
      if command -v sudo >/dev/null 2>&1 && command -v update-ca-certificates >/dev/null 2>&1; then
        if sudo cp "$PROXY_LOG_DIR/proxy-tls/ca.crt" /usr/local/share/ca-certificates/awmg-proxy-difc.crt && \
           sudo update-ca-certificates; then
          CA_INSTALLED=true
        else
          echo "::warning::Failed to install DIFC proxy CA into system trust store; continuing without system CA update"
        fi
      else
        echo "::warning::Cannot install DIFC proxy CA (missing sudo or update-ca-certificates); continuing without system CA update"
      fi
    fi
    if curl -sf "https://localhost:18443/api/v3/health" -o /dev/null 2>/dev/null; then
      echo "DIFC proxy ready on port 18443"
      # Route gh CLI calls through the proxy.
      echo "GH_HOST=localhost:18443" >> "$GITHUB_ENV"
      git remote add proxy "https://localhost:18443/${GITHUB_REPOSITORY}.git" || true
      # Route actions/github-script Octokit calls through the proxy.
      # Save the originals so stop_difc_proxy.sh can restore them.
      echo "GH_AW_ORIGINAL_GITHUB_API_URL=${GITHUB_API_URL:-}" >> "$GITHUB_ENV"
      echo "GH_AW_ORIGINAL_GITHUB_GRAPHQL_URL=${GITHUB_GRAPHQL_URL:-}" >> "$GITHUB_ENV"
      echo "GITHUB_API_URL=https://localhost:18443/api/v3" >> "$GITHUB_ENV"
      echo "GITHUB_GRAPHQL_URL=https://localhost:18443/api/graphql" >> "$GITHUB_ENV"
      # Trust the proxy TLS certificate from Node.js (used by actions/github-script).
      echo "NODE_EXTRA_CA_CERTS=$PROXY_LOG_DIR/proxy-tls/ca.crt" >> "$GITHUB_ENV"
      PROXY_READY=true
      break
    fi
  fi
  sleep 1
done

if [ "$PROXY_READY" = "false" ]; then
  echo "::warning::DIFC proxy failed to start, falling back to direct API access"
  docker logs awmg-proxy 2>&1 | tail -20 || true
  docker rm -f awmg-proxy 2>/dev/null || true
fi
