#!/usr/bin/env bash
set +o histexpand

# Start DIFC proxy on the host for AWF CLI proxy sidecar
# This script starts the awmg proxy container so AWF's cli-proxy container
# can connect to it via host.docker.internal:18443 for gh CLI access.
#
# Unlike start_difc_proxy.sh (which is for pre-agent steps), this proxy
# runs alongside AWF and does NOT modify GH_HOST or GITHUB_ENV.
#
# Environment:
#   CLI_PROXY_POLICY    - JSON guard policy string
#   CLI_PROXY_IMAGE     - Container image to use (e.g., ghcr.io/github/gh-aw-mcpg:v0.2.2)
#   GH_TOKEN            - GitHub token passed to the proxy container
#   GITHUB_SERVER_URL   - GitHub server URL for upstream routing

set -e

POLICY="${CLI_PROXY_POLICY:-}"
CONTAINER_IMAGE="${CLI_PROXY_IMAGE:-}"
CLI_PROXY_LISTEN_ADDR="${CLI_PROXY_LISTEN_ADDR:-[::]:18443}"
CLI_PROXY_DIAL_HOST="${CLI_PROXY_DIAL_HOST:-host.docker.internal}"
CLI_PROXY_DIAL_TARGET="${CLI_PROXY_DIAL_HOST}:18443"
LOCAL_HEALTH_URL="https://127.0.0.1:18443/api/v3/health"
SIDECAR_HEALTH_URL="https://${CLI_PROXY_DIAL_TARGET}/api/v3/health"

if [ -z "$CONTAINER_IMAGE" ]; then
  echo "::warning::CLI proxy container image not specified, skipping proxy start"
  exit 0
fi

TLS_DIR=/tmp/gh-aw/difc-proxy-tls
MCP_LOG_DIR=/tmp/gh-aw/mcp-logs

mkdir -p "$TLS_DIR" "$MCP_LOG_DIR"

# Remove any leftover container from a prior run (e.g., cancelled job on a self-hosted runner)
docker rm -f awmg-cli-proxy 2>/dev/null || true

echo "Starting CLI proxy container: $CONTAINER_IMAGE"
echo "CLI proxy listen address: $CLI_PROXY_LISTEN_ADDR"
echo "CLI proxy sidecar dial target: $CLI_PROXY_DIAL_TARGET"

# Build docker run command arguments
POLICY_ARGS=()
if [ -n "$POLICY" ]; then
  POLICY_ARGS=(--policy "$POLICY")
fi

docker run -d --name awmg-cli-proxy --network host \
  --user "$(id -u):$(id -g)" \
  -e GH_TOKEN \
  -e GITHUB_SERVER_URL \
  -e DEBUG='*' \
  -v "$TLS_DIR:$TLS_DIR" \
  -v "$MCP_LOG_DIR:$MCP_LOG_DIR" \
  "$CONTAINER_IMAGE" proxy \
    "${POLICY_ARGS[@]}" \
    --listen "$CLI_PROXY_LISTEN_ADDR" \
    --log-dir "$MCP_LOG_DIR" \
    --tls --tls-dir "$TLS_DIR" \
    --guards-mode filter \
    --trusted-bots github-actions[bot],github-actions,dependabot[bot],copilot

# Wait for TLS cert + health checks (up to 60s).
# Require consecutive passing checks to survive brief startup blips.
PROXY_READY=false
CONSECUTIVE_READY=0
for i in $(seq 1 60); do
  if [ -f "$TLS_DIR/ca.crt" ]; then
    if curl -sf --cacert "$TLS_DIR/ca.crt" "$LOCAL_HEALTH_URL" -o /dev/null 2>/dev/null && \
       curl -sf --cacert "$TLS_DIR/ca.crt" --resolve "${CLI_PROXY_DIAL_TARGET}:127.0.0.1" "$SIDECAR_HEALTH_URL" -o /dev/null 2>/dev/null; then
      CONSECUTIVE_READY=$((CONSECUTIVE_READY + 1))
    else
      CONSECUTIVE_READY=0
    fi

    if [ "$CONSECUTIVE_READY" -ge 3 ]; then
      echo "CLI proxy ready on port 18443"
      PROXY_READY=true
      break
    fi
  fi
  sleep 1
done

if [ "$PROXY_READY" = "false" ]; then
  echo "::error::CLI proxy failed to start within 60s (listen=${CLI_PROXY_LISTEN_ADDR}, dial=${CLI_PROXY_DIAL_TARGET})"
  docker logs awmg-cli-proxy 2>&1 | tail -20 || true
  docker rm -f awmg-cli-proxy 2>/dev/null || true
  exit 1
fi

if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  {
    echo "### CLI proxy startup diagnostics"
    echo "- Listen address: \`$CLI_PROXY_LISTEN_ADDR\`"
    echo "- Sidecar dial target: \`$CLI_PROXY_DIAL_TARGET\`"
    echo "- Local readiness URL: \`$LOCAL_HEALTH_URL\`"
    echo "- Sidecar-equivalent readiness URL: \`$SIDECAR_HEALTH_URL\` (checked via curl --resolve)"
  } >> "$GITHUB_STEP_SUMMARY"
fi
