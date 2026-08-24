#!/usr/bin/env bash
set +o histexpand
set -euo pipefail

docker rm -f awmg-enclave-github-proxy >/dev/null 2>&1 || true
rm -rf /tmp/gh-aw/enclave-github-proxy-logs
printf 'MCP_GATEWAY_ENCLAVE_CAPABILITY_KEY=\n' >> "$GITHUB_ENV"
