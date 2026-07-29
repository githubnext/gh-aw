#!/bin/bash
# Upgrade Go and the GitHub CLI (gh) to their latest released versions.
# Intended to run on devcontainer post-create so the environment always
# picks up the newest toolchain.
set -euo pipefail

ARCH="$(dpkg --print-architecture 2>/dev/null || uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
esac

# --- Upgrade Go to the latest stable version ---
echo "Determining latest Go version..."
LATEST_GO="$(curl -fsSL https://go.dev/VERSION?m=text | head -n1)"
if [ -z "${LATEST_GO:-}" ]; then
  echo "Failed to determine latest Go version; skipping Go upgrade." >&2
else
  CURRENT_GO="$(go version 2>/dev/null | awk '{print $3}' || echo "none")"
  if [ "$CURRENT_GO" = "$LATEST_GO" ]; then
    echo "Go is already up to date ($CURRENT_GO)."
  else
    echo "Upgrading Go from ${CURRENT_GO} to ${LATEST_GO}..."
    TARBALL="${LATEST_GO}.linux-${ARCH}.tar.gz"
    TMP="$(mktemp -d)"
    curl -fsSL "https://go.dev/dl/${TARBALL}" -o "${TMP}/${TARBALL}"
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "${TMP}/${TARBALL}"
    rm -rf "${TMP}"
    echo "Go upgraded to $(/usr/local/go/bin/go version | awk '{print $3}')."
  fi
fi

# --- Upgrade the GitHub CLI (gh) to the latest release ---
echo "Upgrading GitHub CLI (gh)..."
sudo apt-get update
sudo apt-get install --only-upgrade -y gh || sudo apt-get install -y gh
echo "gh is now $(gh --version | head -n1)."
