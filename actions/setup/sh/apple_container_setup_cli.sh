#!/usr/bin/env bash
set +o histexpand

# apple_container_setup_cli.sh - Ensure a supported Apple `container` CLI is
# present for AWF's apple-container runtime.
#
# Two paths, in order:
#
#   1. Verify a preinstalled CLI. If `container --version` reports a version
#      inside the supported window, nothing is downloaded or installed. This is
#      the expected steady state on a maintained self-hosted Mac.
#   2. Install the pinned release. Only when --allow-install is passed (that is,
#      when the workflow opted in with sandbox.agent.runtime-install). The
#      package is pinned by version AND by SHA-256, and its Developer ID
#      installer signature is checked before `installer` is ever invoked, so a
#      swapped release asset fails on the digest and a re-signed package fails on
#      the identity.
#
# There is deliberately no "latest" path and no unsigned-package path.
#
# Usage: apple_container_setup_cli.sh [--allow-install]
#
# Inputs (all required):
#   GH_AW_APPLE_CONTAINER_VERSION      pinned apple/container release, e.g. 0.12.3
#   GH_AW_APPLE_CONTAINER_PKG_SHA256   SHA-256 of the signed installer package
#   GH_AW_APPLE_CONTAINER_PKG_IDENTITY expected Developer ID Installer identity
#   GH_AW_APPLE_CONTAINER_MIN_CLI      inclusive minimum supported CLI version
#   GH_AW_APPLE_CONTAINER_MAX_CLI      exclusive maximum supported CLI version
#
# Outputs (GITHUB_ENV):
#   GH_AW_APPLE_CONTAINER_BIN          absolute path to the verified CLI

set -euo pipefail

ALLOW_INSTALL=false
for arg in "$@"; do
  case "$arg" in
    --allow-install) ALLOW_INSTALL=true ;;
    *) echo "::error::unknown argument: ${arg}"; exit 1 ;;
  esac
done

require_env() {
  local name="$1" value="$2"
  if [[ -z "${value}" ]]; then
    echo "::error::${name} is required"
    exit 1
  fi
}

require_env GH_AW_APPLE_CONTAINER_VERSION "${GH_AW_APPLE_CONTAINER_VERSION:-}"
require_env GH_AW_APPLE_CONTAINER_PKG_SHA256 "${GH_AW_APPLE_CONTAINER_PKG_SHA256:-}"
require_env GH_AW_APPLE_CONTAINER_PKG_IDENTITY "${GH_AW_APPLE_CONTAINER_PKG_IDENTITY:-}"
require_env GH_AW_APPLE_CONTAINER_MIN_CLI "${GH_AW_APPLE_CONTAINER_MIN_CLI:-}"
require_env GH_AW_APPLE_CONTAINER_MAX_CLI "${GH_AW_APPLE_CONTAINER_MAX_CLI:-}"

PINNED_VERSION="${GH_AW_APPLE_CONTAINER_VERSION}"
PINNED_SHA256="${GH_AW_APPLE_CONTAINER_PKG_SHA256}"
EXPECTED_IDENTITY="${GH_AW_APPLE_CONTAINER_PKG_IDENTITY}"
MIN_CLI="${GH_AW_APPLE_CONTAINER_MIN_CLI}"
MAX_CLI="${GH_AW_APPLE_CONTAINER_MAX_CLI}"

fail() {
  echo "::error::$1"
  exit 1
}

# version_key normalises a dotted version to a zero-padded sortable key so
# comparisons work in bash 3.2 (which is what macOS ships as /bin/bash) without
# sort -V, which BSD sort lacks. Pre-release suffixes are dropped: the window
# check must not accept 1.0.0-beta as "less than 1.0.0".
version_key() {
  local raw="$1"
  raw="${raw%%-*}"
  raw="${raw%%+*}"
  local major minor patch
  IFS='.' read -r major minor patch <<<"${raw}"
  major="${major:-0}"; minor="${minor:-0}"; patch="${patch:-0}"
  case "${major}${minor}${patch}" in
    ''|*[!0-9]*) return 1 ;;
  esac
  printf '%05d%05d%05d\n' "${major}" "${minor}" "${patch}"
}

# version_in_window reports whether $1 satisfies MIN_CLI <= v < MAX_CLI.
version_in_window() {
  local candidate min max
  candidate="$(version_key "$1")" || return 1
  min="$(version_key "${MIN_CLI}")" || return 1
  max="$(version_key "${MAX_CLI}")" || return 1
  [[ "${candidate}" > "${min}" || "${candidate}" == "${min}" ]] || return 1
  [[ "${candidate}" < "${max}" ]] || return 1
  return 0
}

# installed_version prints the bare version of a `container` binary, or nothing.
# `container --version` prints "container CLI version 0.12.3 (build: ...)"-style
# output across releases, so the first dotted token is extracted rather than
# assuming a fixed field position.
installed_version() {
  local bin="$1" out
  out="$("${bin}" --version 2>/dev/null || true)"
  printf '%s\n' "${out}" | tr ' ' '\n' | grep -Eo '^[0-9]+\.[0-9]+(\.[0-9]+)?$' | head -n1
}

echo "::group::Resolve Apple container CLI"

CONTAINER_BIN="$(command -v container 2>/dev/null || true)"
CURRENT_VERSION=""
if [[ -n "${CONTAINER_BIN}" ]]; then
  CURRENT_VERSION="$(installed_version "${CONTAINER_BIN}")"
  echo "found preinstalled CLI at ${CONTAINER_BIN} reporting version '${CURRENT_VERSION:-unparseable}'"
else
  echo "no 'container' CLI on PATH"
fi

if [[ -n "${CURRENT_VERSION}" ]] && version_in_window "${CURRENT_VERSION}"; then
  echo "preinstalled container ${CURRENT_VERSION} is within the supported window [${MIN_CLI}, ${MAX_CLI}); skipping installation"
  echo "::endgroup::"
  if [[ -n "${GITHUB_ENV:-}" ]]; then
    echo "GH_AW_APPLE_CONTAINER_BIN=${CONTAINER_BIN}" >> "${GITHUB_ENV}"
  fi
  exit 0
fi

if [[ "${ALLOW_INSTALL}" != "true" ]]; then
  if [[ -n "${CURRENT_VERSION}" ]]; then
    fail "installed Apple container CLI ${CURRENT_VERSION} is outside the supported window [${MIN_CLI}, ${MAX_CLI}). AWF's init image contract is only validated against that range: a release outside it may relocate the real vminitd and boot a guest with no capability relay. Install ${PINNED_VERSION} on the runner, or set sandbox.agent.runtime-install: true to let the workflow install the pinned release."
  fi
  fail "the Apple container CLI is not installed on this runner. Install apple/container ${PINNED_VERSION} (https://github.com/apple/container/releases/tag/${PINNED_VERSION}), or set sandbox.agent.runtime-install: true to let the workflow install the pinned, checksum-verified and signature-verified release."
fi

echo "::endgroup::"

echo "::group::Install pinned Apple container ${PINNED_VERSION}"

# `installer` writes to /usr/local and requires root. Non-interactive sudo is
# mandatory: a password prompt on a headless runner would hang the job until the
# step timeout rather than failing.
if ! sudo -n true 2>/dev/null; then
  fail "sandbox.agent.runtime-install requires passwordless sudo to run 'installer -pkg', which this runner does not grant. Either configure NOPASSWD sudo for the runner user, or preinstall apple/container ${PINNED_VERSION} and drop runtime-install."
fi

pkg_name="container-${PINNED_VERSION}-installer-signed.pkg"
pkg_url="https://github.com/apple/container/releases/download/${PINNED_VERSION}/${pkg_name}"

work_dir="$(mktemp -d)"
trap 'rm -rf "${work_dir}"' EXIT
pkg_path="${work_dir}/${pkg_name}"

echo "downloading ${pkg_url}"
curl -fsSL --retry 5 --retry-delay 10 --retry-max-time 300 --retry-all-errors -o "${pkg_path}" "${pkg_url}"

actual_sha256="$(shasum -a 256 "${pkg_path}" | awk '{print $1}' | tr 'A-F' 'a-f')"
expected_sha256="$(printf '%s' "${PINNED_SHA256}" | tr 'A-F' 'a-f')"
if [[ "${actual_sha256}" != "${expected_sha256}" ]]; then
  fail "checksum verification failed for ${pkg_name}: expected ${expected_sha256}, got ${actual_sha256}. The release asset does not match the digest pinned in this repository; do not install it."
fi
echo "checksum verified: ${actual_sha256}"

# The digest proves the bytes are the reviewed ones. The signature additionally
# proves who produced them, so a checksum bump alone cannot launder a package
# from another signer.
signature_output="$(pkgutil --check-signature "${pkg_path}" 2>&1)" || fail "pkgutil --check-signature failed for ${pkg_name}"
if ! grep -Fq "${EXPECTED_IDENTITY}" <<<"${signature_output}"; then
  echo "${signature_output}"
  fail "${pkg_name} is not signed by the expected identity '${EXPECTED_IDENTITY}'."
fi
if ! grep -Fq "signed by a developer certificate issued by Apple for distribution" <<<"${signature_output}"; then
  echo "${signature_output}"
  fail "${pkg_name} does not carry a valid Apple distribution signature."
fi
echo "signature verified: ${EXPECTED_IDENTITY}"

sudo -n installer -pkg "${pkg_path}" -target / >/dev/null

hash -r 2>/dev/null || true
CONTAINER_BIN="$(command -v container 2>/dev/null || true)"
if [[ -z "${CONTAINER_BIN}" ]]; then
  # The package installs into /usr/local/bin, which is on the default PATH, but
  # a runner with a trimmed PATH would not see it. Fall back to the receipt's
  # install location rather than failing on a PATH detail.
  if [[ -x /usr/local/bin/container ]]; then
    CONTAINER_BIN=/usr/local/bin/container
  else
    fail "apple/container ${PINNED_VERSION} installed but no 'container' binary was found on PATH or at /usr/local/bin/container."
  fi
fi

CURRENT_VERSION="$(installed_version "${CONTAINER_BIN}")"
if [[ -z "${CURRENT_VERSION}" ]]; then
  fail "could not read a version from '${CONTAINER_BIN} --version' after installation."
fi
if ! version_in_window "${CURRENT_VERSION}"; then
  fail "installed apple/container ${CURRENT_VERSION} is outside the supported window [${MIN_CLI}, ${MAX_CLI}); refusing to continue."
fi
if [[ "${CURRENT_VERSION}" != "${PINNED_VERSION}" ]]; then
  echo "::warning::installed container reports ${CURRENT_VERSION} but ${PINNED_VERSION} was requested"
fi

echo "installed apple/container ${CURRENT_VERSION} at ${CONTAINER_BIN}"
echo "::endgroup::"

if [[ -n "${GITHUB_ENV:-}" ]]; then
  echo "GH_AW_APPLE_CONTAINER_BIN=${CONTAINER_BIN}" >> "${GITHUB_ENV}"
fi
