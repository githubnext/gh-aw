#!/usr/bin/env bash
set +o histexpand

# sudo_docker_sbx_install.sh - Install the docker-sbx package via the Docker apt repository.
#
# Requires sudo access to install the package and fix KVM device permissions.
#
# Usage: sudo_docker_sbx_install.sh
# No arguments required.

set -euo pipefail

echo "::group::Install docker-sbx"
# Add Docker apt repo without installing Docker Engine (already present).
curl -fsSL https://get.docker.com | sudo REPO_ONLY=1 sh
sudo apt-get install -y docker-sbx
sbx version
# Fix KVM permissions so the runner user can create microVMs.
sudo chmod 666 /dev/kvm
echo "docker-sbx installed successfully"
echo "::endgroup::"
