#!/usr/bin/env bash
set +o histexpand
set -euo pipefail

docker rm -f awmg-enclave-github-proxy >/dev/null 2>&1 || true
MCP_LOG_DIR="${RUNNER_TEMP:-/tmp}/gh-aw/enclave-github-proxy-logs"
rm -rf "$MCP_LOG_DIR"
printf 'MCP_GATEWAY_ENCLAVE_CAPABILITY_KEY=\n' >> "$GITHUB_ENV"
