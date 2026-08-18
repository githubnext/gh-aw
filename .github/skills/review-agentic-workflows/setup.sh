#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../.." && pwd)"
cd "$REPO_ROOT"

for required_file in AGENTS.md Makefile; do
  if [ ! -f "$required_file" ]; then
    echo "Required file not found at repository root: $required_file" >&2
    exit 1
  fi
done

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI is required" >&2
  exit 1
fi

if ! gh aw --help >/dev/null 2>&1; then
  if [ -f ./install-gh-aw.sh ]; then
    bash ./install-gh-aw.sh
  else
    echo "install-gh-aw.sh not found; skipping local install attempt" >&2
  fi

  if gh aw --help >/dev/null 2>&1; then
    echo "Using installed gh aw CLI extension"
  elif [ -x ./gh-aw ]; then
    echo "gh aw extension is unavailable; setup is in degraded mode. Use ./gh-aw commands in this skill." >&2
  else
    echo "gh-aw CLI extension or local ./gh-aw binary is required" >&2
    exit 1
  fi
fi
