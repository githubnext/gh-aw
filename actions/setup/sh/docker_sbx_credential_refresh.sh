#!/usr/bin/env bash
set +o histexpand

# docker_sbx_credential_refresh.sh - Re-authenticate sbx immediately before AWF runs.
#
# Docker Hub OAuth tokens obtained during daemon setup can expire or be invalidated
# by the policy-reset cycle. This step runs without sudo and keeps the service ready
# regardless of whether the installation steps were generated.
#
# Usage: docker_sbx_credential_refresh.sh
#
# Environment variables (pass secrets via env, not inline):
#   DOCKER_PAT_VAL      - Value of secrets.DOCKER_PAT
#   DOCKER_USERNAME_VAL - Value of secrets.DOCKER_USERNAME

set -euo pipefail

# Re-authenticate sbx immediately before AWF runs.
# Docker Hub OAuth tokens from sbx login can expire between steps.
printf '%s' "${DOCKER_PAT_VAL}" | sbx login --username "${DOCKER_USERNAME_VAL}" --password-stdin
echo "sbx credentials refreshed"
