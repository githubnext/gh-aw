#!/usr/bin/env bash
set +o histexpand

# sudo_gvisor_install.sh - Download, verify, install, and register the gVisor (runsc) runtime.
#
# Requires sudo access to install binaries and restart Docker.
#
# Usage: sudo_gvisor_install.sh <version>
#
# Arguments:
#   version - Pinned gVisor release version (e.g. "20240101.0")
#
# Key notes:
#   - Each binary is verified against its official SHA-512 file before installation.
#   - Uses uname -m directly (x86_64, aarch64) — gVisor download URLs use raw arch names.
#   - Restarts Docker with systemctl restart (NOT reload): Docker's SIGHUP reload does
#     not call setHostGatewayIP(), which breaks --add-host host.docker.internal:host-gateway.
#   - Downloads both runsc and containerd-shim-runsc-v1; the shim is required for
#     Docker's containerd integration.

set -euo pipefail

VERSION="${1:?Usage: sudo_gvisor_install.sh <version>}"

echo "::group::Install gVisor (runsc)"
ARCH=$(uname -m)
URL="https://storage.googleapis.com/gvisor/releases/release/${VERSION}/${ARCH}"
echo "Downloading runsc ${VERSION} for ${ARCH}..."
# runner-guard:ignore RGS-012 -- pinned version + SHA-512-verified download of a fixed artifact into a file; no data leaves the runner.
curl -fsSL "${URL}/runsc" -o /tmp/runsc
curl -fsSL "${URL}/runsc.sha512" -o /tmp/runsc.sha512
echo "Verifying SHA-512 for runsc..."
(cd /tmp && sha512sum -c runsc.sha512)
# runner-guard:ignore RGS-012 -- pinned version + SHA-512-verified download of a fixed artifact into a file; no data leaves the runner.
curl -fsSL "${URL}/containerd-shim-runsc-v1" -o /tmp/containerd-shim-runsc-v1
curl -fsSL "${URL}/containerd-shim-runsc-v1.sha512" -o /tmp/containerd-shim-runsc-v1.sha512
echo "Verifying SHA-512 for containerd-shim-runsc-v1..."
(cd /tmp && sha512sum -c containerd-shim-runsc-v1.sha512)
sudo install -m 755 /tmp/runsc /usr/local/bin/runsc
sudo install -m 755 /tmp/containerd-shim-runsc-v1 /usr/local/bin/containerd-shim-runsc-v1
runsc --version
echo "::endgroup::"

echo "::group::Register runsc as Docker runtime"
sudo runsc install
# IMPORTANT: Must use restart (not reload).
# Docker's SIGHUP reload does NOT call setHostGatewayIP(), so
# --add-host host.docker.internal:host-gateway breaks for any
# container started after a reload-only config change.
sudo systemctl restart docker
echo "Docker runtimes:"
docker info --format '{{.Runtimes}}' || docker info | grep -i runtime
echo "::endgroup::"

echo "::group::Verify gVisor works"
docker pull hello-world
docker run --rm --runtime=runsc hello-world
echo "gVisor runtime verified"
echo "::endgroup::"
