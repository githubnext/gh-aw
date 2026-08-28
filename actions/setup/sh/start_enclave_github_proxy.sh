#!/usr/bin/env bash
set +o histexpand
set -euo pipefail

# Starts the compiler-owned mcpg issues-read-v1 proxy. AWF later attaches this
# container to its private enclave control network; no host port is published.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=proxy_env_lib.sh
source "${SCRIPT_DIR}/proxy_env_lib.sh"

CONTAINER_NAME="awmg-enclave-github-proxy"
PORT="18443"
RUN_LABEL="com.github.gh-aw.enclave-github.run"
MCP_LOG_DIR="${RUNNER_TEMP:-/tmp}/gh-aw/enclave-github-proxy-logs"
CA_CERT="${MCP_LOG_DIR}/proxy-tls/ca.crt"
CONTAINER_IMAGE="${ENCLAVE_GITHUB_PROXY_IMAGE:-}"
PROXY_ALIAS="${ENCLAVE_GITHUB_PROXY_ALIAS:-}"
POLICY_TEMPLATE="${ENCLAVE_GITHUB_PROXY_POLICY_TEMPLATE:-}"

if [[ -z "$CONTAINER_IMAGE" ]]; then
  echo "::error::Enclave GitHub proxy image is required"
  exit 1
fi
if [[ -z "$PROXY_ALIAS" ]]; then
  echo "::error::Enclave GitHub proxy alias is required"
  exit 1
fi
if [[ -z "$POLICY_TEMPLATE" ]]; then
  echo "::error::Enclave GitHub proxy policy is required"
  exit 1
fi
echo "::add-mask::${POLICY_TEMPLATE}"
if [[ -z "${GH_TOKEN:-}" ]]; then
  echo "::error::Enclave GitHub proxy requires an upstream GitHub token"
  exit 1
fi
if [[ -z "${GITHUB_ENV:-}" ]]; then
  echo "::error::GITHUB_ENV is required for the trusted AWF handoff"
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "::error::jq is required to bind the enclave GitHub policy to the workflow run"
  exit 1
fi

derive_proxy_upstream_env

MCP_GATEWAY_ENCLAVE_CAPABILITY_KEY="$(openssl rand -hex 32)"
if [[ ! "$MCP_GATEWAY_ENCLAVE_CAPABILITY_KEY" =~ ^[0-9a-f]{64}$ ]]; then
  echo "::error::Failed to generate enclave GitHub proxy capability key"
  exit 1
fi
echo "::add-mask::${MCP_GATEWAY_ENCLAVE_CAPABILITY_KEY}"
export MCP_GATEWAY_ENCLAVE_CAPABILITY_KEY

JOB_HASH="$(printf '%s' "$GITHUB_JOB" | openssl dgst -sha256 -r | cut -d' ' -f1 | cut -c1-12)"
PROXY_IDENTITY="gh-aw-egh-${GITHUB_RUN_ID}-${GITHUB_RUN_ATTEMPT}-${JOB_HASH}"
if [[ ! "$PROXY_IDENTITY" =~ ^[a-z0-9][a-z0-9-]{0,63}$ ]]; then
  echo "::error::Failed to derive a valid enclave GitHub proxy identity"
  exit 1
fi

MCP_GATEWAY_ENCLAVE_POLICY_JSON="$(
  jq -c --arg workflow_run_id "$PROXY_IDENTITY" \
    '.workflow_run_id = $workflow_run_id' <<<"$POLICY_TEMPLATE"
)"
echo "::add-mask::${MCP_GATEWAY_ENCLAVE_POLICY_JSON}"
export MCP_GATEWAY_ENCLAVE_POLICY_JSON

mkdir -p "$MCP_LOG_DIR"
chmod 700 "$MCP_LOG_DIR"
rm -rf "${MCP_LOG_DIR}/proxy-tls"
docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true

docker run -d --name "$CONTAINER_NAME" \
  --network bridge \
  --label "${RUN_LABEL}=${PROXY_IDENTITY}" \
  --user "$(id -u):$(id -g)" \
  -e GH_TOKEN \
  -e GH_HOST \
  -e GITHUB_HOST \
  -e GITHUB_ENTERPRISE_HOST \
  -e GITHUB_SERVER_URL \
  -e GITHUB_API_URL \
  -e GITHUB_GRAPHQL_URL \
  -e GITHUB_COPILOT_BASE_URL \
  -e MCP_GATEWAY_ENCLAVE_POLICY_JSON \
  -e MCP_GATEWAY_ENCLAVE_CAPABILITY_KEY \
  -v "$MCP_LOG_DIR:$MCP_LOG_DIR" \
  "$CONTAINER_IMAGE" proxy \
    --listen "0.0.0.0:${PORT}" \
    --log-dir "$MCP_LOG_DIR" \
    --tls-dns-name "$PROXY_ALIAS" \
    --tls

PROXY_READY=false
for ((attempt = 1; attempt <= 30; attempt++)); do
  if [[ -f "$CA_CERT" ]]; then
    PROXY_IP="$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$CONTAINER_NAME")"
    if [[ -n "$PROXY_IP" ]] && curl -sf --cacert "$CA_CERT" \
      --resolve "${PROXY_ALIAS}:${PORT}:${PROXY_IP}" \
      "https://${PROXY_ALIAS}:${PORT}/api/v3/health" -o /dev/null; then
      PROXY_READY=true
      break
    fi
  fi
  sleep 1
done

if [[ "$PROXY_READY" != "true" ]]; then
  echo "::error::Enclave GitHub proxy failed to become ready"
  docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
  exit 1
fi

{
  printf '%s=%s\n' AWF_ENCLAVE_GITHUB_PROXY_CONTAINER "$CONTAINER_NAME"
  printf '%s=%s\n' AWF_ENCLAVE_GITHUB_PROXY_IDENTITY "$PROXY_IDENTITY"
  printf '%s=%s\n' AWF_ENCLAVE_GITHUB_PROXY_CA_CERT "$CA_CERT"
  printf '%s=%s\n' MCP_GATEWAY_ENCLAVE_CAPABILITY_KEY "$MCP_GATEWAY_ENCLAVE_CAPABILITY_KEY"
} >>"$GITHUB_ENV"
