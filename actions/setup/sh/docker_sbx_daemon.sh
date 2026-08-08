#!/usr/bin/env bash
set +o histexpand

# docker_sbx_daemon.sh - Start the sbx daemon and authenticate with Docker Hub.
#
# This script:
#   1. Starts the sbx daemon and waits for it to become ready.
#   2. Authenticates Docker and sbx CLIs with Docker Hub.
#   3. Resets and re-initialises the sbx allow-all policy (required for mount policy).
#   4. Restarts the daemon and re-authenticates after the policy reset.
#   5. Pre-pulls the sandbox template image.
#
# Usage: docker_sbx_daemon.sh
#
# Environment variables (pass secrets via env, not inline):
#   DOCKER_PAT_VAL      - Value of secrets.DOCKER_PAT
#   DOCKER_USERNAME_VAL - Value of secrets.DOCKER_USERNAME

set -euo pipefail

DOCKER_CONFIG="$(mktemp -d)"
export DOCKER_CONFIG
trap 'rm -rf "${DOCKER_CONFIG}"' EXIT

echo "::group::Start sbx daemon"
nohup sbx daemon start > /tmp/sbx-daemon.log 2>&1 &
# Poll until daemon is running (up to 10 s).
for _ in $(seq 1 10); do
  if sbx daemon status 2>/dev/null | grep -q -i running; then
    echo "sbx daemon is running"
    break
  fi
  sleep 1
done
echo "::endgroup::"

echo "::group::Authenticate with Docker Hub"
printf '%s' "${DOCKER_PAT_VAL}" | docker login --username "${DOCKER_USERNAME_VAL}" --password-stdin
printf '%s' "${DOCKER_PAT_VAL}" | sbx login --username "${DOCKER_USERNAME_VAL}" --password-stdin
echo "::endgroup::"

echo "::group::Reset and initialise sbx policy"
sbx daemon stop || true
sbx policy reset --force || true
sbx policy init allow-all
nohup sbx daemon start > /tmp/sbx-daemon.log 2>&1 &
for _ in $(seq 1 10); do
  if sbx daemon status 2>/dev/null | grep -q -i running; then
    echo "sbx daemon restarted"
    break
  fi
  sleep 1
done
printf '%s' "${DOCKER_PAT_VAL}" | docker login --username "${DOCKER_USERNAME_VAL}" --password-stdin
printf '%s' "${DOCKER_PAT_VAL}" | sbx login --username "${DOCKER_USERNAME_VAL}" --password-stdin
echo "::endgroup::"

echo "::group::Pre-pull sandbox template image"
docker pull docker/sandbox-templates:shell-docker
echo "Template image ready"
echo "::endgroup::"
