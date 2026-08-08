#!/usr/bin/env bash
set +o histexpand

# docker_sbx_preflight.sh - End-to-end smoke test for the docker-sbx stack.
#
# Creates a throwaway sandbox, executes a command inside it, then removes it.
# Fails if any step of the smoke test fails, ensuring the sbx runtime is ready
# before the agent starts.
#
# Usage: docker_sbx_preflight.sh
# No arguments required.

set -euo pipefail

echo "::group::docker-sbx pre-flight smoke test"
sandbox_name="test-sandbox-direct"
cleanup() {
  sbx stop "${sandbox_name}" >/dev/null 2>&1 || true
  sbx rm --force "${sandbox_name}" >/dev/null 2>&1 || true
}
trap cleanup EXIT
echo "y" | sbx create shell --name "${sandbox_name}" "${GITHUB_WORKSPACE}"
sbx exec "${sandbox_name}" uname -a
sbx stop "${sandbox_name}"
echo "sbx ready"
echo "::endgroup::"
