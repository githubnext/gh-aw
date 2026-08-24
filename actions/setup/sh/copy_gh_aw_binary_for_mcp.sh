#!/usr/bin/env bash
set +o histexpand

set -euo pipefail

# Copy the gh-aw binary to ${RUNNER_TEMP}/gh-aw for MCP Server containerization
mkdir -p "${RUNNER_TEMP}/gh-aw"

GH_AW_BIN=""
GH_AW_BIN=$(command -v gh-aw 2>/dev/null) || true

if [ -z "$GH_AW_BIN" ] && [ -x "${RUNNER_TEMP}/gh-aw/bin/gh-aw" ]; then
  GH_AW_BIN="${RUNNER_TEMP}/gh-aw/bin/gh-aw"
fi

if [ -z "$GH_AW_BIN" ]; then
  GH_AW_BIN=$(find "${RUNNER_TEMP}/gh-aw/bin" -maxdepth 1 -type f -name 'gh-aw*' -perm -111 2>/dev/null | head -1) || true
fi

if [ -z "$GH_AW_BIN" ]; then
  GH_AW_BIN=$(find "${HOME}/.local/share/gh/extensions/gh-aw" -name 'gh-aw*' -type f -perm -111 2>/dev/null | head -1) || true
fi

if [ -z "$GH_AW_BIN" ] && [ -n "${GH_CONFIG_DIR:-}" ]; then
  GH_AW_BIN=$(find "${GH_CONFIG_DIR}/extensions/gh-aw" -name 'gh-aw*' -type f -perm -111 2>/dev/null | head -1) || true
fi

if [ -z "$GH_AW_BIN" ] && [ -f "${GITHUB_WORKSPACE}/gh-aw" ]; then
  GH_AW_BIN="${GITHUB_WORKSPACE}/gh-aw"
fi

if [ -n "$GH_AW_BIN" ] && [ -f "$GH_AW_BIN" ]; then
  cp "$GH_AW_BIN" "${RUNNER_TEMP}/gh-aw/gh-aw"
  chmod +x "${RUNNER_TEMP}/gh-aw/gh-aw"
  echo "Copied gh-aw binary to ${RUNNER_TEMP}/gh-aw/gh-aw"
else
  echo "::error::Failed to find gh-aw binary for MCP Server"
  exit 1
fi
