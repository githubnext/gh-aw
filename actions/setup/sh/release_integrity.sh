#!/usr/bin/env bash
set +o histexpand

# Sourced by installers from the trusted setup action, never from the download origin.
# Review new pins against authenticated release provenance before committing them.
# Each entry binds repository, immutable release version, asset name, and SHA256.
release_artifact_sha256() {
  local repository="$1" version="$2" asset="$3" digest
  version="v${version#v}"
  digest=$(awk -v repository="$repository" -v version="$version" -v asset="$asset" '
    $1 == repository && $2 == version && $3 == asset { count++; digest = $4 }
    END { if (count == 1) print digest; else exit 1 }
  ' <<'PINS'
github/gh-aw-firewall v0.28.62 checksums.txt 0b03f6ea5b638356a799fa541ade4fd7902a6ac8d7c3f0252ee09d8259a4ce8f
github/copilot-cli v1.0.98 SHA256SUMS.txt 372f0c8d3d50076455cd19ef080dc59dc2a32401d8b2d4f77ce190c8ab13bc59
github/copilot-cli v1.0.21 SHA256SUMS.txt fd9b2fb92a39aeb2a41815a6aa0bf3853307c812b3451a45f5a2217be003987d
PINS
  ) || {
    echo "ERROR: No reviewed SHA256 pin for ${repository} ${version} ${asset}; update the trusted setup action's release_integrity.sh before installing this release" >&2
    return 1
  }
  if [[ ! "$digest" =~ ^[0-9a-f]{64}$ ]]; then
    echo "ERROR: Invalid reviewed SHA256 pin for ${repository} ${version} ${asset}" >&2
    return 1
  fi
  printf '%s\n' "$digest"
}

verify_release_digest() {
  local file="$1" expected="$2" actual
  if [[ ! "$expected" =~ ^[0-9a-f]{64}$ ]]; then
    echo "ERROR: A trusted lowercase SHA256 digest is required" >&2
    return 1
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$file") || return 1
  elif command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "$file") || return 1
  else
    echo "ERROR: No sha256sum or shasum found" >&2
    return 1
  fi
  actual="${actual%% *}"
  if [ "$actual" != "$expected" ]; then
    echo "ERROR: Trusted SHA256 verification failed for $(basename "$file"); refusing installation" >&2
    return 1
  fi
}
