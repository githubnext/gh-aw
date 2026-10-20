#!/usr/bin/env bash
set +o histexpand
set -euo pipefail

# Check the actual compiler, not just the module directive or setup-go request.
version="$(go env GOVERSION)"
if [[ ! "$version" =~ ^go1\.([0-9]+)\.([0-9]+)$ ]]; then
  echo "ERROR: Release builds require a stable patched Go toolchain; found ${version}" >&2
  exit 1
fi
minor="${BASH_REMATCH[1]}"
patch="${BASH_REMATCH[2]}"
if (( minor < 26 || (minor == 26 && patch < 10) || (minor == 27 && patch < 3) )); then
  echo "ERROR: ${version} lacks the required CA distrust fix (CVE-2026-111943); use Go 1.26.10, 1.27.3, or a newer supported release" >&2
  exit 1
fi
printf '%s\n' "$version"
