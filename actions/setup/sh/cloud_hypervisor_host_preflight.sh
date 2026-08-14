#!/usr/bin/env bash
set +o histexpand

# cloud_hypervisor_host_preflight.sh - Validate runner eligibility for AWF's
# preview cloud-hypervisor runtime.
#
# Supported scope is intentionally narrow:
# - GitHub-hosted runners only
# - Ubuntu Linux x86_64 only
# - /dev/kvm must be present

set -euo pipefail

echo "::group::cloud-hypervisor host preflight"

if [[ "${RUNNER_ENVIRONMENT:-}" != "github-hosted" ]]; then
  echo "::error::cloud-hypervisor preview is supported only on GitHub-hosted runners."
  exit 1
fi

if [[ "${RUNNER_OS:-}" != "Linux" ]]; then
  echo "::error::cloud-hypervisor preview requires Linux runners."
  exit 1
fi

if [[ "${RUNNER_ARCH:-}" != "X64" ]]; then
  echo "::error::cloud-hypervisor preview requires x86_64 (RUNNER_ARCH=X64) runners."
  exit 1
fi

if [[ "${ImageOS:-}" != ubuntu* ]]; then
  echo "::error::cloud-hypervisor preview requires GitHub-hosted Ubuntu images (ImageOS starts with 'ubuntu')."
  exit 1
fi

if ! test -e /dev/kvm; then
  echo "::error::/dev/kvm is missing. cloud-hypervisor preview requires KVM-capable GitHub-hosted Ubuntu x86_64 runners."
  exit 1
fi

if ! test -c /dev/kvm; then
  echo "::error::/dev/kvm must be a character device."
  exit 1
fi

echo "runner is eligible for cloud-hypervisor preview"
echo "::endgroup::"
