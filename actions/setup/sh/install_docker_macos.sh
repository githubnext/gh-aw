#!/usr/bin/env bash
# Install Docker on macOS GitHub Actions runners via colima
# Usage: install_docker_macos.sh
#
# macOS GHA runners (macos-latest / macos-15-arm64) do not include Docker.
# This script installs colima (a lightweight Docker runtime for macOS) and the
# Docker CLI via Homebrew, then starts colima with Apple's Virtualization
# framework (vz) for native ARM64 performance.
#
# On Linux runners, Docker is pre-installed so this script exits early.

set -euo pipefail

# Skip on non-macOS systems (Linux runners already have Docker)
if [ "$(uname -s)" != "Darwin" ]; then
  echo "Not macOS — skipping Docker installation (Docker is pre-installed on Linux runners)"
  exit 0
fi

# Check if Docker is already available and running
if command -v docker &>/dev/null && docker info &>/dev/null; then
  echo "Docker is already installed and running — skipping installation"
  exit 0
fi

echo "Installing Docker on macOS via colima..."

# Install colima and docker CLI via Homebrew
echo "Installing colima and docker CLI..."
brew install colima docker

# Start colima with ARM64 architecture and Apple Virtualization framework
# - --arch aarch64: native ARM64 for Apple Silicon runners
# - --vm-type vz: Apple's Virtualization.framework (faster than QEMU)
# - --memory 4: 4GB RAM for the VM (sufficient for agent containers)
echo "Starting colima..."
colima start --arch aarch64 --vm-type=vz --memory 4

# Verify Docker is working
echo "Verifying Docker installation..."
if ! docker info &>/dev/null; then
  echo "ERROR: Docker is not responding after colima start"
  colima status || true
  exit 1
fi

docker version
echo "Docker installation complete"
