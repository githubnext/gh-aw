#!/usr/bin/env bash
set +o histexpand

# docker_sbx_kvm_check.sh - Verify KVM availability before docker-sbx installation.
#
# docker-sbx requires a KVM-capable runner with nested virtualisation enabled.
# This script fails fast when the KVM kernel module is not loaded or /dev/kvm is missing.
#
# Usage: docker_sbx_kvm_check.sh
# No arguments required.

set -euo pipefail

echo "::group::KVM availability check"
if ! lsmod | grep -q kvm; then
  echo "::error::KVM kernel module is not loaded. docker-sbx requires a KVM-capable runner with nested virtualisation enabled."
  exit 1
fi
if ! test -e /dev/kvm; then
  echo "::error::/dev/kvm is missing. docker-sbx requires the KVM device to be present on the runner."
  exit 1
fi
echo "KVM is available and /dev/kvm is present"
echo "::endgroup::"
