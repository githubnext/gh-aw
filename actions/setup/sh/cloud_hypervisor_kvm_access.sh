#!/usr/bin/env bash
set +o histexpand

# Grant only the current runner user access to KVM. This avoids weakening the
# device permissions for unrelated users on the host.

set -euo pipefail

echo "::group::Configure cloud-hypervisor KVM access"

if [[ "${RUNNER_ENVIRONMENT:-}" != "github-hosted" || "${RUNNER_OS:-}" != "Linux" || "${RUNNER_ARCH:-}" != "X64" || "${ImageOS:-}" != ubuntu* ]]; then
  echo "::error::cloud-hypervisor KVM access is supported only on GitHub-hosted Ubuntu x86_64 runners."
  exit 1
fi

if [[ ! -e /dev/kvm ]]; then
  echo "::error::/dev/kvm is missing. cloud-hypervisor preview requires a KVM-capable runner."
  exit 1
fi

if [[ ! -c /dev/kvm ]]; then
  echo "::error::/dev/kvm must be a character device."
  exit 1
fi

if ! command -v setfacl >/dev/null 2>&1; then
  echo "::error::setfacl is required to grant scoped access to /dev/kvm."
  exit 1
fi

runner_uid="$(id -u)"
if [[ ! "${runner_uid}" =~ ^[0-9]+$ ]]; then
  echo "::error::failed to resolve a numeric runner UID."
  exit 1
fi
sudo setfacl -m "u:${runner_uid}:rw" /dev/kvm

if [[ ! -r /dev/kvm || ! -w /dev/kvm ]]; then
  echo "::error::failed to grant the runner user read/write access to /dev/kvm."
  exit 1
fi

acl_output="$(getfacl -ncp /dev/kvm | sed 's/[[:space:]]#effective:.*$//' || true)"
if [[ -z "${acl_output}" ]]; then
  echo "::error::failed to read /dev/kvm ACLs for verification."
  exit 1
fi
if ! grep -Eq "^user:${runner_uid}:rw-$" <<<"${acl_output}"; then
  echo "::error::failed to verify scoped ACL entry for the runner user on /dev/kvm."
  exit 1
fi

echo "runner user has scoped read/write access to /dev/kvm"
echo "::endgroup::"
