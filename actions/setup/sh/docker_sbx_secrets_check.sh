#!/usr/bin/env bash
set +o histexpand

# docker_sbx_secrets_check.sh - Verify Docker Hub secrets before docker-sbx installation.
#
# docker-sbx requires DOCKER_PAT and DOCKER_USERNAME to pull the sandbox template image.
# This script fails fast when either secret is missing.
#
# Usage: docker_sbx_secrets_check.sh
#
# Environment variables (pass secrets via env, not inline):
#   DOCKER_PAT_VAL      - Value of secrets.DOCKER_PAT
#   DOCKER_USERNAME_VAL - Value of secrets.DOCKER_USERNAME

set -euo pipefail

echo "::group::Docker Hub secrets check"
if [[ -z "${DOCKER_PAT_VAL:-}" ]]; then
  echo "::error::secrets.DOCKER_PAT is empty. docker-sbx requires a Docker Hub personal access token to pull the sandbox template image. Add a DOCKER_PAT secret to your repository."
  exit 1
fi
if [[ -z "${DOCKER_USERNAME_VAL:-}" ]]; then
  echo "::error::secrets.DOCKER_USERNAME is empty. docker-sbx requires a Docker Hub username. Add a DOCKER_USERNAME secret to your repository."
  exit 1
fi
echo "Docker Hub secrets are present"
echo "::endgroup::"
