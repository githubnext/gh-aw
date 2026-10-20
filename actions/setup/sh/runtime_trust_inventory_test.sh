#!/usr/bin/env bash
set +o histexpand
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_ROOT="${PWD}/.runtime-trust-inventory-test-$$"
mkdir "$TEST_ROOT"
trap 'rm -rf "$TEST_ROOT"' EXIT
mkdir "${TEST_ROOT}/bin" "${TEST_ROOT}/destination"
for tool in bash sed; do
  ln -s "$(command -v "$tool")" "${TEST_ROOT}/bin/${tool}"
done

for tool in curl node go; do
  cat >"${TEST_ROOT}/bin/${tool}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
tool="${0##*/}"
if [[ "$tool" == go && "${GOTOOLCHAIN:-}" != local ]]; then
  echo 'Go inspection must not download a toolchain' >&2
  exit 95
fi
if [[ "$FAIL_PROBE" == all || "$FAIL_PROBE" == "$tool" ]]; then
  echo "${tool}: no selected runtime" >&2
  exit 1
fi
case "$tool" in
  curl) echo 'curl test TLS-backend' ;;
  node) echo 'Node test; OpenSSL test' ;;
  go)
    if [[ "$*" == 'env GOVERSION' ]]; then
      echo go-test
    else
      echo 'binary: go-test'
    fi
    ;;
esac
EOF
done
for tool in gh gh-aw; do
  printf '#!/usr/bin/env bash\nexit 95\n' >"${TEST_ROOT}/bin/${tool}"
done
chmod +x "${TEST_ROOT}/bin/"*

for probe in none go curl node all; do
  status=0
  output=$(PATH="${TEST_ROOT}/bin" FAIL_PROBE="$probe" \
    "${BASH}" "${SCRIPT_DIR}/runtime_trust_inventory.sh" 2>&1) || status=$?
  if [ "$status" -ne 0 ]; then
    printf 'FAIL: %s probe failure aborted inventory: %s\n' "$probe" "$output" >&2
    exit 1
  fi
  grep -qF '::endgroup::' <<<"$output"
  grep -qF 'Browser updates and successful HTTPS connections do not attest CA distrust.' <<<"$output"
  case "$probe" in
    none) ! grep -qF WARNING <<<"$output" ;;
    go)
      grep -qF 'WARNING: Go version inspection failed; trust status unknown' <<<"$output"
      grep -qF 'WARNING: gh build-toolchain inspection failed; trust status unknown' <<<"$output"
      grep -qF 'WARNING: gh-aw build-toolchain inspection failed; trust status unknown' <<<"$output"
      ;;
    curl) grep -qF 'WARNING: curl version inspection failed; trust status unknown' <<<"$output" ;;
    node) grep -qF 'WARNING: Node/OpenSSL version inspection failed; trust status unknown' <<<"$output" ;;
    all)
      for warning in 'Go version' 'curl version' 'Node/OpenSSL version'; do
        grep -qF "WARNING: ${warning} inspection failed; trust status unknown" <<<"$output"
      done
      ;;
  esac
  echo "PASS: inventory completes with explicit diagnostics (failed probe: ${probe})"
done

# Exercise the actual setup call site without setup's host-directory operations.
awk '
  /^if ! bash .*runtime_trust_inventory.sh/ { copying = 1 }
  copying { print }
  copying && /^fi$/ { exit }
' "${SCRIPT_DIR}/../setup.sh" >"${TEST_ROOT}/setup-inventory-call.sh"
[ -s "${TEST_ROOT}/setup-inventory-call.sh" ]
printf '#!/usr/bin/env bash\nexit 1\n' >"${TEST_ROOT}/destination/runtime_trust_inventory.sh"
# The child shell expands its own positional argument.
# shellcheck disable=SC2016
output=$(DESTINATION="${TEST_ROOT}/destination" "${BASH}" -ec \
  'source "$1"; echo setup-continued' bash "${TEST_ROOT}/setup-inventory-call.sh" 2>&1)
grep -qF '::warning::Runtime TLS inventory inspection failed; trust status unknown.' <<<"$output"
grep -qxF setup-continued <<<"$output"
echo 'PASS: setup warns and continues even when the diagnostic script exits unsuccessfully'
