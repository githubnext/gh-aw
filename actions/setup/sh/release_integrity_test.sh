#!/usr/bin/env bash
set +o histexpand
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TEMP_DIR"' EXIT
# shellcheck source=actions/setup/sh/release_integrity.sh
source "$SCRIPT_DIR/release_integrity.sh"

mkdir -p "$TEMP_DIR/bin" "$TEMP_DIR/home" "$TEMP_DIR/scripts" "$TEMP_DIR/copilot-fixture"
cp "$SCRIPT_DIR/install_awf_binary.sh" "$TEMP_DIR/scripts/"
cp "$SCRIPT_DIR/install_copilot_cli.sh" "$TEMP_DIR/scripts/"
printf '#!/usr/bin/env bash\necho EXECUTED >>"$EXECUTION_LOG"\n' >"$TEMP_DIR/payload"
payload_digest="$(shasum -a 256 "$TEMP_DIR/payload" | awk '{print $1}')"
for asset in awf-bundle.js awf-linux-x64 awf-linux-arm64 awf-darwin-x64 awf-darwin-arm64; do
  printf '%s  %s\n' "$payload_digest" "$asset"
done >"$TEMP_DIR/checksums.txt"
cp "$TEMP_DIR/checksums.txt" "$TEMP_DIR/valid-checksums.txt"
manifest_digest="$(shasum -a 256 "$TEMP_DIR/checksums.txt" | awk '{print $1}')"
cp "$TEMP_DIR/payload" "$TEMP_DIR/copilot-fixture/copilot"
chmod +x "$TEMP_DIR/copilot-fixture/copilot"
tar -czf "$TEMP_DIR/copilot-linux-x64.tar.gz" -C "$TEMP_DIR/copilot-fixture" copilot
cp "$TEMP_DIR/copilot-linux-x64.tar.gz" "$TEMP_DIR/copilot-linuxmusl-x64.tar.gz"
archive_digest="$(shasum -a 256 "$TEMP_DIR/copilot-linux-x64.tar.gz" | awk '{print $1}')"
printf '%s  copilot-linux-x64.tar.gz\n%s  copilot-linuxmusl-x64.tar.gz\n' "$archive_digest" "$archive_digest" >"$TEMP_DIR/SHA256SUMS.txt"
copilot_digest="$(shasum -a 256 "$TEMP_DIR/SHA256SUMS.txt" | awk '{print $1}')"
# Test pins live only beside copies of the real installers, never in production inputs.
sed -e "s/0b03f6ea5b638356a799fa541ade4fd7902a6ac8d7c3f0252ee09d8259a4ce8f/$manifest_digest/" \
    -e "s/372f0c8d3d50076455cd19ef080dc59dc2a32401d8b2d4f77ce190c8ab13bc59/$copilot_digest/" \
  "$SCRIPT_DIR/release_integrity.sh" >"$TEMP_DIR/scripts/release_integrity.sh"

cat >"$TEMP_DIR/bin/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [ "${1:-}" = "--help" ]; then exit 0; fi
printf '%s\n' "$*" >>"$TEST_ROOT/curl.log"
out=""
https_only=false
https_redirects=false
while [ "$#" -gt 0 ]; do
  case "$1" in
    --proto)
      [ "$2" = '=https' ]
      https_only=true
      shift 2 ;;
    --proto-redir)
      [ "$2" = '=https' ]
      https_redirects=true
      shift 2 ;;
    -o) out="$2"; shift 2 ;;
    *) url="$1"; shift ;;
  esac
done
if [ "$https_only" != true ] || [ "$https_redirects" != true ]; then
  echo "missing HTTPS-only initial/redirect policy" >&2; exit 96
fi
case "$url" in
  */checksums.txt) cp "$TEST_ROOT/checksums.txt" "$out" ;;
  */SHA256SUMS.txt) cp "$TEST_ROOT/SHA256SUMS.txt" "$out" ;;
  */awf-bundle.js|*/awf-linux-*|*/awf-darwin-*) cp "$TEST_ROOT/payload" "$out" ;;
  */copilot-linux-x64.tar.gz) cp "$TEST_ROOT/copilot-linux-x64.tar.gz" "$out" ;;
  */copilot-linuxmusl-x64.tar.gz) cp "$TEST_ROOT/copilot-linuxmusl-x64.tar.gz" "$out" ;;
  *) echo "unexpected download: $url" >&2; exit 97 ;;
esac
EOF
cat >"$TEMP_DIR/bin/node" <<'EOF'
#!/usr/bin/env bash
if [ "$1" = "--version" ]; then echo "${AWF_TEST_NODE_VERSION:-v24.0.0}"; else echo EXECUTED >>"$EXECUTION_LOG"; fi
EOF
cat >"$TEMP_DIR/bin/uname" <<'EOF'
#!/usr/bin/env bash
case "$1" in -s) echo "${AWF_TEST_OS:-Linux}" ;; -m) echo "${AWF_TEST_ARCH:-x86_64}" ;; esac
EOF
printf '#!/usr/bin/env bash\nexit 0\n' >"$TEMP_DIR/bin/sudo"
cat >"$TEMP_DIR/bin/find" <<'EOF'
#!/usr/bin/env bash
if [ "${1:-}" = /tmp ]; then exit 0; fi
exec /usr/bin/find "$@"
EOF
chmod +x "$TEMP_DIR/bin/"*

run_awf() {
  env TEST_ROOT="$TEMP_DIR" EXECUTION_LOG="$TEMP_DIR/executions" HOME="$TEMP_DIR/home" \
    GITHUB_PATH="$TEMP_DIR/github-path" PATH="$TEMP_DIR/bin:$PATH" \
    bash "$TEMP_DIR/scripts/install_awf_binary.sh" "$@" >"$TEMP_DIR/output" 2>&1
}
run_copilot() {
  env TEST_ROOT="$TEMP_DIR" EXECUTION_LOG="$TEMP_DIR/executions" HOME="$TEMP_DIR/home" \
    GITHUB_PATH="$TEMP_DIR/github-path" RUNNER_TOOL_CACHE="$TEMP_DIR/empty-cache" \
    GH_AW_COPILOT_GUEST_LIBC="${PIN_TEST_GUEST_LIBC:-}" RUNNER_TEMP="$TEMP_DIR/runner" \
    GH_AW_COMPILED_VERSION="" PATH="$TEMP_DIR/bin:$PATH" \
    bash "$TEMP_DIR/scripts/install_copilot_cli.sh" "$@" >"$TEMP_DIR/output" 2>&1
}

run_awf v0.28.62 --rootless
test -x "$TEMP_DIR/home/.local/bin/awf"
grep -q EXECUTED "$TEMP_DIR/executions"
echo "PASS: authenticated AWF bundle installs and executes"
run_copilot 1.0.98 --rootless
test -x "$TEMP_DIR/home/.local/bin/copilot"
rm -f "$TEMP_DIR/home/.local/bin/copilot"
echo "PASS: authenticated Copilot archive installs and executes"
for platform in Linux Darwin; do
  for arch in x86_64 arm64; do
    AWF_TEST_NODE_VERSION=v18.0.0 AWF_TEST_OS="$platform" AWF_TEST_ARCH="$arch" run_awf v0.28.62 --rootless
    test -x "$TEMP_DIR/home/.local/bin/awf"
  done
done
echo "PASS: authenticated AWF fallback binaries cover Linux/macOS x64/arm64"

run_awf v9.9.9 --checksums-sha256 "$manifest_digest" --rootless
grep -q '/v9.9.9/awf-bundle.js' "$TEMP_DIR/curl.log"
echo "PASS: frontmatter-style AWF pin enables a different version"

cache_bin="$TEMP_DIR/empty-cache/copilot-cli/9.9.9/x64/bin"
mkdir -p "$cache_bin"
printf '#!/usr/bin/env bash\necho CACHED_EXECUTED >>"$EXECUTION_LOG"\n' >"$cache_bin/copilot"
chmod +x "$cache_bin/copilot"
rm -f "$TEMP_DIR/curl.log" "$TEMP_DIR/executions"
PIN_TEST_GUEST_LIBC=musl run_copilot 9.9.9 --rootless --checksums-sha256 "$copilot_digest"
grep -q '/v9.9.9/copilot-linux-x64.tar.gz' "$TEMP_DIR/curl.log"
grep -q '/v9.9.9/copilot-linuxmusl-x64.tar.gz' "$TEMP_DIR/curl.log"
test -x "$TEMP_DIR/runner/gh-aw/nvx-engine/copilot"
if grep -q CACHED_EXECUTED "$TEMP_DIR/executions"; then
  echo "FAIL: explicit pin was bypassed by an unverified cache hit" >&2; exit 1
fi
rm -f "$cache_bin/copilot" "$TEMP_DIR/home/.local/bin/copilot"
echo "PASS: explicit Copilot pin verifies both host/musl archives and bypasses toolcache"

rm -f "$TEMP_DIR/curl.log" "$TEMP_DIR/executions"
for bad_pin in "" abc "${copilot_digest^^}" '${{ inputs.pin }}'; do
  if run_awf v9.9.9 --checksums-sha256 "$bad_pin" --rootless ||
    run_copilot 9.9.9 --checksums-sha256 "$bad_pin" --rootless; then
    echo "FAIL: malformed explicit digest accepted" >&2; exit 1
  fi
  test ! -e "$TEMP_DIR/curl.log"
done
if run_awf v9.9.9 --checksums-sha256 ||
  run_copilot 9.9.9 --checksums-sha256 ||
  run_awf v9.9.9 --checksums-sha256 "$manifest_digest" --checksums-sha256 "$manifest_digest" ||
  run_copilot 9.9.9 --checksums-sha256 "$copilot_digest" --checksums-sha256 "$copilot_digest" ||
  run_awf latest --checksums-sha256 "$manifest_digest" ||
  run_copilot latest --checksums-sha256 "$copilot_digest" ||
  run_copilot --checksums-sha256 "$copilot_digest"; then
  echo "FAIL: duplicate, missing or unbound explicit pin accepted" >&2; exit 1
fi
test ! -e "$TEMP_DIR/curl.log"
echo "PASS: malformed, duplicate and unbound explicit pins fail before network access"

wrong_pin="$(printf '0%.0s' {1..64})"
if run_awf v9.9.9 --rootless --checksums-sha256 "$wrong_pin" ||
  run_copilot 9.9.9 --rootless --checksums-sha256 "$wrong_pin"; then
  echo "FAIL: mismatched explicit manifest pin accepted" >&2; exit 1
fi
test ! -e "$TEMP_DIR/executions"
echo "PASS: mismatched explicit pins never execute downloaded code"

# Replacing the manifest and executable together must fail before any execution.
rm -f "$TEMP_DIR/executions" "$TEMP_DIR/curl.log"
printf 'attacker payload\n' >"$TEMP_DIR/payload"
attacker_digest="$(shasum -a 256 "$TEMP_DIR/payload" | awk '{print $1}')"
printf '%s  awf-bundle.js\n' "$attacker_digest" >"$TEMP_DIR/checksums.txt"
if run_awf v0.28.62 --rootless; then
  echo "FAIL: AWF accepted artifact-plus-manifest replacement" >&2; exit 1
fi
grep -q 'Trusted SHA256 verification failed' "$TEMP_DIR/output"
test ! -e "$TEMP_DIR/executions"
test "$(wc -l <"$TEMP_DIR/curl.log" | tr -d ' ')" = 1
echo "PASS: AWF rejects artifact-plus-manifest replacement before downloading executable"

# An authentic manifest must not allow a tampered bundle to trigger a binary fallback.
cp "$TEMP_DIR/valid-checksums.txt" "$TEMP_DIR/checksums.txt"
rm -f "$TEMP_DIR/curl.log"
if run_awf v0.28.62 --rootless; then
  echo "FAIL: AWF accepted a tampered bundle" >&2; exit 1
fi
grep -q 'refusing fallback' "$TEMP_DIR/output"
test ! -e "$TEMP_DIR/executions"
test "$(wc -l <"$TEMP_DIR/curl.log" | tr -d ' ')" = 2
echo "PASS: AWF bundle integrity failure cannot fall back"

printf '%s  copilot-linux-x64.tar.gz\n' "$attacker_digest" >"$TEMP_DIR/SHA256SUMS.txt"
rm -f "$TEMP_DIR/curl.log"
if run_copilot 1.0.98 --rootless; then
  echo "FAIL: Copilot accepted artifact-plus-manifest replacement" >&2; exit 1
fi
grep -q 'Trusted SHA256 verification failed' "$TEMP_DIR/output"
test "$(wc -l <"$TEMP_DIR/curl.log" | tr -d ' ')" = 1
test ! -e "$TEMP_DIR/home/.local/bin/copilot"
echo "PASS: Copilot authenticates its manifest before downloading or extracting executable"

rm -f "$TEMP_DIR/curl.log"
if run_awf latest --rootless || run_awf v9.9.9 --rootless; then
  echo "FAIL: AWF accepted an unreviewed release" >&2; exit 1
fi
test ! -e "$TEMP_DIR/curl.log"
if run_copilot latest --rootless; then
  echo "FAIL: Copilot accepted an unreviewed latest download" >&2; exit 1
fi
test ! -e "$TEMP_DIR/curl.log"
echo "PASS: unreviewed/latest release downloads fail before network access"

test -n "$(release_artifact_sha256 github/gh-aw-firewall v0.28.62 checksums.txt)"
if release_artifact_sha256 github/gh-aw-firewall v0.28.62 wrong-asset >/dev/null 2>&1; then
  echo "FAIL: digest pin was not asset-bound" >&2; exit 1
fi
if verify_release_digest "$TEMP_DIR/payload" invalid >/dev/null 2>&1; then
  echo "FAIL: malformed trusted digest accepted" >&2; exit 1
fi
echo "PASS: trusted pins are asset-bound and validated"

awk '{print} /^github\/gh-aw-firewall / {print}' "$SCRIPT_DIR/release_integrity.sh" >"$TEMP_DIR/duplicate-pins.sh"
if bash -c 'source "$1"; release_artifact_sha256 github/gh-aw-firewall v0.28.62 checksums.txt' -- "$TEMP_DIR/duplicate-pins.sh" >"$TEMP_DIR/output" 2>&1; then
  echo "FAIL: duplicate trusted pins accepted" >&2; exit 1
fi
grep -q 'No reviewed SHA256 pin' "$TEMP_DIR/output"
echo "PASS: duplicate trusted release entries fail closed"

# Real curl enforces the protocol policy, independent of the download mock.
if curl --proto '=https' --proto-redir '=https' -fsSL --max-time 2 http://127.0.0.1:1 >"$TEMP_DIR/curl-output" 2>&1; then
  echo "FAIL: HTTP URL accepted" >&2; exit 1
fi
grep -qi 'protocol.*disabled' "$TEMP_DIR/curl-output"
echo "PASS: real curl refuses HTTP before connecting"
