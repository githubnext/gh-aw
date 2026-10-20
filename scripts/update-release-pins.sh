#!/usr/bin/env bash
set +o histexpand
set -euo pipefail

# Usage: scripts/update-release-pins.sh REPOSITORY VERSION MANIFEST EXPECTED_SHA256
# Obtain EXPECTED_SHA256 from an independently trusted channel (for example,
# authenticated release provenance), not another download from the release origin.
# This adds a reviewed manifest pin, never downloads or executes release binaries,
# and refuses to replace a different digest. Review the resulting diff before commit.

fail() {
  echo "error: $*" >&2
  exit 1
}

if [ "$#" -ne 4 ]; then
  fail "Usage: $0 REPOSITORY VERSION MANIFEST EXPECTED_SHA256 (independently authenticated)"
fi
repository="$1" version="$2" asset="$3" expected="$4"
case "$repository/$asset" in
  github/gh-aw-firewall/checksums.txt|github/copilot-cli/SHA256SUMS.txt) ;;
  *) fail "supported repository/manifest pairs: github/gh-aw-firewall checksums.txt, github/copilot-cli SHA256SUMS.txt" ;;
esac
[[ "$version" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]] ||
  fail "VERSION must be a semantic release tag such as v1.0.98"
version="v${version#v}"
[[ "$expected" =~ ^[0-9a-f]{64}$ ]] ||
  fail "EXPECTED_SHA256 must be an independently authenticated lowercase hex64 digest"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
target="$script_dir/../actions/setup/sh/release_integrity.sh"
[ -f "$target" ] && [ ! -L "$target" ] || fail "release_integrity.sh must be a regular, non-symlink file"

# Keep scratch files private and on the target filesystem for atomic replacement.
work="$target.update-$$-$RANDOM"
(umask 077; mkdir "$work") || fail "cannot create private update directory"
trap 'rm -rf "$work"' EXIT
cp -p "$target" "$work/original"

curl --disable --proto '=https' --proto-redir '=https' --fail --silent --show-error --location \
  --output "$work/manifest" \
  "https://github.com/$repository/releases/download/$version/$asset"
[ -s "$work/manifest" ] || fail "downloaded manifest is empty"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$work/manifest")"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$work/manifest")"
else
  fail "neither sha256sum nor shasum is available"
fi
[ "${actual%% *}" = "$expected" ] || fail "manifest SHA256 mismatch; refusing to update pins"

cp -p "$work/original" "$work/updated"
awk -v repository="$repository" -v version="$version" -v asset="$asset" -v digest="$expected" '
  function invalid(message) {
    print "error: " message > "/dev/stderr"
    failed = 1
    exit 1
  }
  /<<.*PINS/ {
    if ($0 != "  \047 <<\047PINS\047" || starts++ || in_pins || ends)
      invalid("could not uniquely locate the release pin heredoc")
    in_pins = 1
    print
    next
  }
  $0 == "PINS" {
    if (!in_pins || ends++) invalid("unexpected release pin delimiter")
    if (!found) print repository, version, asset, digest
    in_pins = 0
    print
    next
  }
  in_pins {
    if (NF != 4 ||
        !(($1 == "github/gh-aw-firewall" && $3 == "checksums.txt") ||
          ($1 == "github/copilot-cli" && $3 == "SHA256SUMS.txt")) ||
        $2 !~ /^v[0-9]+\.[0-9]+\.[0-9]+$/ ||
        length($4) != 64 || $4 !~ /^[0-9a-f]+$/)
      invalid("malformed release pin row")
    key = $1 SUBSEP $2 SUBSEP $3
    if (seen[key]++) invalid("duplicate release pin row")
    if ($1 == repository && $2 == version && $3 == asset) {
      if ($4 != digest) invalid("refusing to replace a different reviewed digest")
      found = 1
    }
  }
  { print }
  END {
    if (!failed && (starts != 1 || ends != 1 || in_pins))
      invalid("expected exactly one complete release pin heredoc")
  }
' "$work/original" > "$work/updated" || fail "release pin validation failed; no changes made"

cmp -s "$target" "$work/original" || fail "release_integrity.sh changed during update; retry after review"
if cmp -s "$work/original" "$work/updated"; then
  echo "Reviewed release pin already present; no changes made."
else
  mv "$work/updated" "$target"
  echo "Added verified manifest pin for $repository $version $asset; review the diff before committing."
fi
