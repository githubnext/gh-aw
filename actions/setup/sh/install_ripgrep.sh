#!/usr/bin/env bash
set +o histexpand

set -euo pipefail

if command -v rg >/dev/null 2>&1; then
  rg --version
  exit 0
fi

echo "ripgrep not found; installing with apt-get..."
sudo apt-get update -qq
sudo apt-get install -y -qq ripgrep
