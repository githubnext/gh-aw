#!/usr/bin/env bash
set +o histexpand
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
work=".release-pins-test-$$-$RANDOM"
(umask 077; mkdir "$work")
trap 'rm -rf "$work"' EXIT
mkdir -p "$work/repo/scripts" "$work/repo/actions/setup/sh" "$work/bin"
cp scripts/update-release-pins.sh "$work/repo/scripts/"
cp actions/setup/sh/release_integrity.sh "$work/baseline"
target="$work/repo/actions/setup/sh/release_integrity.sh"
updater="$work/repo/scripts/update-release-pins.sh"
printf '%064d  fixture-asset.tar.gz\n' 0 > "$work/manifest"
export MOCK_MANIFEST="$PWD/$work/manifest"
export MOCK_CURL_LOG="$PWD/$work/curl.log"
export MOCK_EXPECTED_URL="https://github.com/github/gh-aw-firewall/releases/download/v9.8.7/checksums.txt"
cat > "$work/bin/curl" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
echo called >> "$MOCK_CURL_LOG"
[ "$#" -eq 12 ]
[ "$1" = --disable ]
[ "$2" = --proto ] && [ "$3" = '=https' ]
[ "$4" = --proto-redir ] && [ "$5" = '=https' ]
[ "$6" = --fail ] && [ "$7" = --silent ] && [ "$8" = --show-error ]
[ "$9" = --location ] && [ "${10}" = --output ]
[ "${12}" = "$MOCK_EXPECTED_URL" ]
[ "${MOCK_CURL_FAIL:-0}" = 0 ] || exit 22
cp "$MOCK_MANIFEST" "${11}"
MOCK
chmod +x "$work/bin/curl"
export PATH="$PWD/$work/bin:$PATH"

hash_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}
digest="$(hash_file "$work/manifest")"
wrong_digest="$(printf '%064d' 1)"
passed=0

reject() {
  local name="$1"
  shift
  cp "$target" "$work/before"
  if bash "$updater" "$@" > "$work/output" 2>&1; then
    echo "FAIL: $name unexpectedly succeeded" >&2
    exit 1
  fi
  cmp -s "$target" "$work/before" || {
    echo "FAIL: $name modified the pin helper" >&2
    exit 1
  }
  passed=$((passed + 1))
}

cp "$work/baseline" "$target"
reject "missing arguments"
reject "unsupported repository" evil/repository v9.8.7 checksums.txt "$digest"
reject "wrong manifest" github/gh-aw-firewall v9.8.7 SHA256SUMS.txt "$digest"
reject "asset path traversal" github/gh-aw-firewall v9.8.7 ../checksums.txt "$digest"
reject "version URL injection" github/gh-aw-firewall 'v9.8.7?token=secret' checksums.txt "$digest"
reject "prerelease tag" github/gh-aw-firewall v9.8.7-rc1 checksums.txt "$digest"
reject "invalid expected digest" github/gh-aw-firewall v9.8.7 checksums.txt BAD
reject "uppercase expected digest" github/gh-aw-firewall v9.8.7 checksums.txt "$(printf '%064d' 0 | tr 0 A)"
[ ! -e "$MOCK_CURL_LOG" ] || { echo "FAIL: unsupported input caused a download" >&2; exit 1; }
reject "manifest hash mismatch" github/gh-aw-firewall v9.8.7 checksums.txt "$wrong_digest"
export MOCK_CURL_FAIL=1
reject "download failure" github/gh-aw-firewall v9.8.7 checksums.txt "$digest"
unset MOCK_CURL_FAIL
cp "$work/manifest" "$work/nonempty-manifest"
: > "$work/manifest"
reject "empty manifest" github/gh-aw-firewall v9.8.7 checksums.txt "$(hash_file "$work/manifest")"
cp "$work/nonempty-manifest" "$work/manifest"

for defect in short uppercase extra-field bad-version bad-repository bad-asset duplicate blank; do
  awk -v defect="$defect" '
    /^github\/gh-aw-firewall / && !changed++ {
      if (defect == "short") $4 = "abc"
      if (defect == "uppercase") $4 = toupper($4)
      if (defect == "extra-field") $5 = "extra"
      if (defect == "bad-version") $2 = "latest"
      if (defect == "bad-repository") $1 = "other/repo"
      if (defect == "bad-asset") $3 = "other.txt"
      if (defect == "duplicate") print
      if (defect == "blank") print ""
    }
    { print }
  ' "$work/baseline" > "$target"
  reject "$defect table row" github/gh-aw-firewall v9.8.7 checksums.txt "$digest"
done
awk '$0 != "PINS" { print }' "$work/baseline" > "$target"
reject "unterminated table" github/gh-aw-firewall v9.8.7 checksums.txt "$digest"
awk '!/<<.*PINS/ { print }' "$work/baseline" > "$target"
reject "missing table start" github/gh-aw-firewall v9.8.7 checksums.txt "$digest"
cp "$work/baseline" "$target"
cat "$work/baseline" >> "$target"
reject "multiple pin tables" github/gh-aw-firewall v9.8.7 checksums.txt "$digest"

cp "$work/baseline" "$target"
bash "$updater" github/gh-aw-firewall 9.8.7 checksums.txt "$digest"
awk -v digest="$digest" '
  $0 == "PINS" { print "github/gh-aw-firewall v9.8.7 checksums.txt " digest }
  { print }
' "$work/baseline" > "$work/expected"
cmp "$target" "$work/expected"
passed=$((passed + 1))
bash "$updater" github/gh-aw-firewall v9.8.7 checksums.txt "$digest"
cmp "$target" "$work/expected"
passed=$((passed + 1))
printf 'another independently authenticated manifest\n' > "$work/manifest"
reject "replacement of different digest" github/gh-aw-firewall v9.8.7 checksums.txt "$(hash_file "$work/manifest")"

export MOCK_EXPECTED_URL="https://github.com/github/copilot-cli/releases/download/v9.8.7/SHA256SUMS.txt"
digest="$(hash_file "$work/manifest")"
bash "$updater" github/copilot-cli v9.8.7 SHA256SUMS.txt "$digest"
awk -v digest="$digest" '
  $0 == "PINS" { print "github/copilot-cli v9.8.7 SHA256SUMS.txt " digest }
  { print }
' "$work/expected" > "$work/expected-both"
cmp "$target" "$work/expected-both"
passed=$((passed + 1))

# A concurrent edit during the fetch must not be overwritten.
cat > "$work/bin/curl" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
printf '\n# concurrent edit\n' >> "$MOCK_TARGET"
cp "$MOCK_MANIFEST" "${11}"
MOCK
export MOCK_TARGET="$PWD/$target"
cp "$target" "$work/before"
if bash "$updater" github/copilot-cli v9.8.7 SHA256SUMS.txt "$digest" > "$work/output" 2>&1; then
  echo "FAIL: concurrent edit unexpectedly succeeded" >&2
  exit 1
fi
printf '\n# concurrent edit\n' >> "$work/before"
cmp "$target" "$work/before"
passed=$((passed + 1))
echo "PASS: $passed offline release pin authoring checks"
