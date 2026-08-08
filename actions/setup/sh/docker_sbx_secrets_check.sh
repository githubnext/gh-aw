#!/usr/bin/env bash
set +o histexpand

# docker_sbx_secrets_check.sh - Verify Docker Hub secrets before docker-sbx installation.
#
# docker-sbx requires DOCKER_PAT and DOCKER_USERNAME to pull the sandbox template image.
# This script fails fast when either secret is missing unless DOCKER_SBX_SECRETS_SOFT_FAIL=true.
#
# Usage: docker_sbx_secrets_check.sh
#
# Environment variables (pass secrets via env, not inline):
#   DOCKER_PAT_VAL      - Value of secrets.DOCKER_PAT
#   DOCKER_USERNAME_VAL - Value of secrets.DOCKER_USERNAME
#   DOCKER_SBX_SECRETS_SOFT_FAIL - When true, warn and set verification_result=failed without exiting non-zero

set -euo pipefail

echo "::group::Docker Hub secrets check"
missing=()
if [[ -z "${DOCKER_PAT_VAL:-}" ]]; then
  missing+=("DOCKER_PAT")
fi
if [[ -z "${DOCKER_USERNAME_VAL:-}" ]]; then
  missing+=("DOCKER_USERNAME")
fi

if [[ "${#missing[@]}" -gt 0 ]]; then
  if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    echo "verification_result=failed" >> "$GITHUB_OUTPUT"
  fi
  for secret_name in "${missing[@]}"; do
    message="secrets.${secret_name} is empty. docker-sbx requires DOCKER_PAT and DOCKER_USERNAME to pull the sandbox template image. Add both secrets to your repository to enable docker-sbx."
    if [[ "${DOCKER_SBX_SECRETS_SOFT_FAIL:-}" == "true" ]]; then
      echo "::warning::${message}"
    else
      echo "::error::${message}"
    fi
  done
  echo "::endgroup::"
  if [[ "${DOCKER_SBX_SECRETS_SOFT_FAIL:-}" == "true" ]]; then
    exit 0
  fi
  exit 1
fi
echo "Docker Hub secrets are present"
if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  echo "verification_result=success" >> "$GITHUB_OUTPUT"
fi
echo "::endgroup::"
