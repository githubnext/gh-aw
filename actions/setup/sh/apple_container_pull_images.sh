#!/usr/bin/env bash
set +o histexpand

# apple_container_pull_images.sh - Populate Apple Container's image store with
# the digest-pinned images AWF needs for the apple-container runtime.
#
# Apple Container keeps its own content store, completely separate from Docker's.
# The Docker pre-download step earlier in the job populates the daemon that runs
# Squid, the API proxy and the CLI proxy; it does nothing for the agent VM. AWF
# honours --skip-pull for this runtime by *verifying* the image is present in the
# Apple store, so a missing image here is a hard AWF failure later, not a silent
# re-pull.
#
# Two images are required:
#   - the agent image the workload runs
#   - the AWF apple-init image carrying the guest capability relay
#
# Every reference must be digest-pinned. There is no daemon-side content trust to
# fall back on here, so a floating tag would mean the VM boots whatever the
# registry happens to serve. References are passed as arguments, already
# substituted and pin-resolved by the compiler, and are re-checked here so a
# floating reference cannot reach `container image pull` even if some future
# caller forgets.
#
# Usage: apple_container_pull_images.sh REF [REF ...]
#
# Inputs:
#   GH_AW_APPLE_CONTAINER_BIN       optional explicit CLI path
#   CONTAINER_APP_ROOT              application root (exported by the start step)

set -euo pipefail

fail() {
  echo "::error::$1"
  exit 1
}

if [[ $# -eq 0 ]]; then
  fail "at least one image reference is required"
fi

CONTAINER_BIN="${GH_AW_APPLE_CONTAINER_BIN:-}"
if [[ -z "${CONTAINER_BIN}" ]]; then
  CONTAINER_BIN="$(command -v container 2>/dev/null || true)"
fi
[[ -n "${CONTAINER_BIN}" && -x "${CONTAINER_BIN}" ]] || fail "the Apple container CLI was not found; the setup step must run before images are pulled."

# The store is selected by CONTAINER_APP_ROOT. If the start step exported one and
# it has been lost, pulling here would populate a different store than the one
# AWF reads, and AWF's verification would fail on an image that was just pulled.
if [[ -z "${CONTAINER_APP_ROOT:-}" ]]; then
  echo "::warning::CONTAINER_APP_ROOT is not set; using the Apple Container default application root. The AWF invocation must use the same root."
else
  echo "using Apple Container application root: ${CONTAINER_APP_ROOT}"
fi

"${CONTAINER_BIN}" system status >/dev/null 2>&1 ||
  fail "Apple container services are not running; 'container image pull' requires them. The service start step must run first."

pull_with_retry() {
  local reference="$1"
  local max_attempts=3
  local wait_time=5
  local attempt

  for attempt in 1 2 3; do
    echo "Attempt ${attempt} of ${max_attempts}: pulling ${reference} into the Apple Container store..."
    # --platform is explicit: the agent and init images must be the native arm64
    # variants. Rosetta translation is refused by AWF, and an accidental amd64
    # manifest selection would produce a guest that cannot execute its own init.
    if "${CONTAINER_BIN}" image pull --platform linux/arm64 --progress plain "${reference}"; then
      echo "pulled ${reference}"
      return 0
    fi
    if (( attempt < max_attempts )); then
      echo "pull failed; retrying in ${wait_time}s"
      sleep "${wait_time}"
      wait_time=$(( wait_time * 2 ))
    fi
  done

  return 1
}

echo "::group::Pull Apple Container images"
echo "Apple Container uses an image store separate from Docker's; these pulls do not duplicate the Docker pre-download."

for reference in "$@"; do
  [[ -n "${reference}" ]] || fail "empty image reference"

  # Fail closed on anything that is not digest-pinned. This is the guarantee that
  # what boots inside the VM is exactly what the workflow declared.
  case "${reference}" in
    *@sha256:*) ;;
    *) fail "image reference '${reference}' is not digest-pinned. The apple-container runtime requires digest-pinned images: Apple Container's store has no daemon-side content trust, so a floating tag would boot whatever the registry currently serves." ;;
  esac

  # Reject shell metacharacters outright. References reach this script as argv
  # and are never re-parsed by a shell, but a reference that could not be a valid
  # OCI reference indicates the compiler substituted something unexpected, and
  # continuing would pull an attacker-chosen image.
  case "${reference}" in
    *[\ \'\"\$\`\;\&\|\<\>\(\)\{\}\*\?\!$'\n'$'\t']*)
      fail "image reference '${reference}' contains characters that are not valid in an OCI reference." ;;
  esac

  pull_with_retry "${reference}" ||
    fail "failed to pull ${reference} into the Apple Container image store after 3 attempts."
done

echo "::endgroup::"

echo "::group::Verify Apple Container images"
# Verification is separate from the pull so a store that reports a successful
# pull but cannot resolve the reference afterwards is caught here rather than by
# AWF, whose failure would come from inside the VM launch path.
for reference in "$@"; do
  if ! "${CONTAINER_BIN}" image inspect "${reference}" >/dev/null 2>&1; then
    echo "--- images in the Apple Container store ---"
    "${CONTAINER_BIN}" image list --format json 2>/dev/null || "${CONTAINER_BIN}" image list || true
    echo "-------------------------------------------"
    fail "${reference} is not resolvable in the Apple Container image store after a successful pull. AWF verifies this reference under --skip-pull and would fail during VM launch."
  fi
  echo "verified ${reference}"
done
echo "::endgroup::"

echo "Apple Container images ready"
