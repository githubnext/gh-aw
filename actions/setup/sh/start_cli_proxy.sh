#!/usr/bin/env bash
set +o histexpand

# Start DIFC proxy on the host for AWF CLI proxy sidecar
# This script starts the awmg proxy container so AWF's cli-proxy container
# can connect to it via host.docker.internal:18443 for gh CLI access.
#
# This script exports GH_HOST (and related vars) within the script for use when
# launching the proxy container, but does NOT write to $GITHUB_ENV and the
# exports do not persist beyond this script.
#
# Environment:
#   CLI_PROXY_POLICY    - JSON guard policy string
#   CLI_PROXY_IMAGE     - Container image to use (e.g., ghcr.io/github/gh-aw-mcpg:v0.2.2)
#   GH_TOKEN            - GitHub token passed to the proxy container
#   GITHUB_SERVER_URL   - GitHub server URL for upstream routing

set -e

normalize_github_host() {
  local host="$1"

  host="${host%/}"
  if [[ "$host" =~ ^https?:// ]]; then
    host="${host#http://}"
    host="${host#https://}"
    host="${host%%/*}"
  fi

  echo "$host"
}

derive_proxy_upstream_env() {
  local server_url="${GITHUB_SERVER_URL:-https://github.com}"
  local server_host
  local github_host="${GH_HOST:-${GITHUB_HOST:-${GITHUB_ENTERPRISE_HOST:-}}}"

  server_url="${server_url%/}"
  server_host="$(normalize_github_host "$server_url")"
  if [ -z "$github_host" ] || { [ "$server_host" != "github.com" ] && [ "$github_host" = "github.com" ]; }; then
    github_host="$server_host"
  fi
  if [ -z "$github_host" ]; then
    github_host="github.com"
  fi

  export GH_HOST="$github_host"

  if [ "$github_host" != "github.com" ]; then
    export GITHUB_HOST="${GITHUB_HOST:-$github_host}"
    export GITHUB_ENTERPRISE_HOST="${GITHUB_ENTERPRISE_HOST:-$github_host}"
  fi

  if [ -z "${GITHUB_API_URL:-}" ] || { [ "$github_host" != "github.com" ] && [ "${GITHUB_API_URL}" = "https://api.github.com" ]; }; then
    if [ "$github_host" = "github.com" ]; then
      export GITHUB_API_URL="https://api.github.com"
    elif [[ "$github_host" == *.ghe.com ]]; then
      export GITHUB_API_URL="https://api.${github_host}"
    else
      export GITHUB_API_URL="${server_url}/api/v3"
    fi
  fi

  if [ -z "${GITHUB_GRAPHQL_URL:-}" ] || { [ "$github_host" != "github.com" ] && [ "${GITHUB_GRAPHQL_URL}" = "https://api.github.com/graphql" ]; }; then
    if [ "$github_host" = "github.com" ]; then
      export GITHUB_GRAPHQL_URL="https://api.github.com/graphql"
    elif [[ "$github_host" == *.ghe.com ]]; then
      export GITHUB_GRAPHQL_URL="https://api.${github_host}/graphql"
    else
      export GITHUB_GRAPHQL_URL="${server_url}/api/graphql"
    fi
  fi

  if [ -z "${GITHUB_COPILOT_BASE_URL:-}" ] && [[ "$github_host" == *.ghe.com ]]; then
    export GITHUB_COPILOT_BASE_URL="https://copilot-api.${github_host}"
  fi
}

POLICY="${CLI_PROXY_POLICY:-}"
CONTAINER_IMAGE="${CLI_PROXY_IMAGE:-}"

if [ -z "$CONTAINER_IMAGE" ]; then
  echo "::warning::CLI proxy container image not specified, skipping proxy start"
  exit 0
fi

TLS_DIR=/tmp/gh-aw/difc-proxy-tls
MCP_LOG_DIR=/tmp/gh-aw/mcp-logs

mkdir -p "$TLS_DIR" "$MCP_LOG_DIR"

derive_proxy_upstream_env

# Remove any leftover container from a prior run (e.g., cancelled job on a self-hosted runner)
docker rm -f awmg-cli-proxy 2>/dev/null || true

echo "Starting CLI proxy container: $CONTAINER_IMAGE"
echo "Using CLI proxy upstream host: ${GH_HOST} (API: ${GITHUB_API_URL})"

# Build docker run command arguments
POLICY_ARGS=()
if [ -n "$POLICY" ]; then
  POLICY_ARGS=(--policy "$POLICY")
fi

DOCKER_NETWORK_ARGS=(--network host)
if [ "${GH_AW_NETWORK_ISOLATION:-false}" = "true" ]; then
  DOCKER_NETWORK_ARGS=(--network bridge -p 127.0.0.1:18443:18443)
fi

docker run -d --name awmg-cli-proxy "${DOCKER_NETWORK_ARGS[@]}" \
  --user "$(id -u):$(id -g)" \
  -e GH_TOKEN \
  -e GH_HOST \
  -e GITHUB_HOST \
  -e GITHUB_ENTERPRISE_HOST \
  -e GITHUB_SERVER_URL \
  -e GITHUB_API_URL \
  -e GITHUB_GRAPHQL_URL \
  -e GITHUB_COPILOT_BASE_URL \
  -e DEBUG='*' \
  -v "$TLS_DIR:$TLS_DIR" \
  -v "$MCP_LOG_DIR:$MCP_LOG_DIR" \
  "$CONTAINER_IMAGE" proxy \
    "${POLICY_ARGS[@]}" \
    --listen 0.0.0.0:18443 \
    --log-dir "$MCP_LOG_DIR" \
    --tls --tls-dir "$TLS_DIR" \
    --guards-mode filter \
    --trusted-bots github-actions[bot],github-actions,dependabot[bot],copilot

# Wait for TLS cert + health check (up to 30s)
PROXY_READY=false
for i in $(seq 1 30); do
  if [ -f "$TLS_DIR/ca.crt" ]; then
    if curl -sf --cacert "$TLS_DIR/ca.crt" "https://localhost:18443/api/v3/health" -o /dev/null 2>/dev/null; then
      echo "CLI proxy ready on port 18443"
      PROXY_READY=true
      break
    fi
  fi
  sleep 1
done

if [ "$PROXY_READY" = "false" ]; then
  echo "::error::CLI proxy failed to start within 30s"
  docker logs awmg-cli-proxy 2>&1 | tail -20 || true
  docker rm -f awmg-cli-proxy 2>/dev/null || true
  exit 1
fi
