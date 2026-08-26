#!/usr/bin/env bash
set +o histexpand

# apple_container_host_preflight.sh - Validate runner eligibility for AWF's
# apple-container runtime (Apple Virtualization.framework workloads).
#
# This runs before anything is downloaded or installed, so an ineligible runner
# fails in seconds with a named cause instead of after a multi-gigabyte pull.
#
# The supported scope is narrow and deliberate:
# - self-hosted runners only (GitHub-hosted macOS images are themselves VMs and
#   report kern.hv_support=0, so they can never host a nested guest)
# - macOS >= GH_AW_APPLE_CONTAINER_MIN_MACOS
# - arm64 (Rosetta translation is refused by AWF)
# - Virtualization.framework available (kern.hv_support=1)
# - a usable launchd GUI/user domain, because `container system start` registers
#   its API server as a per-user LaunchAgent
# - a working Docker daemon, because AWF keeps Squid, the API proxy and the CLI
#   proxy in Docker Compose regardless of where the agent runs
#
# Inputs:
#   GH_AW_APPLE_CONTAINER_MIN_MACOS - minimum macOS major version (required)

set -euo pipefail

MIN_MACOS="${GH_AW_APPLE_CONTAINER_MIN_MACOS:-}"
if [[ -z "${MIN_MACOS}" ]]; then
  echo "::error::GH_AW_APPLE_CONTAINER_MIN_MACOS is required"
  exit 1
fi

fail() {
  echo "::error::$1"
  exit 1
}

echo "::group::apple-container host preflight"

# --- Runner provenance ------------------------------------------------------
# RUNNER_ENVIRONMENT is "github-hosted" or "self-hosted". Anything else (unset,
# or a value a future runner introduces) is treated as ineligible rather than
# assumed safe.
if [[ "${RUNNER_ENVIRONMENT:-}" != "self-hosted" ]]; then
  fail "apple-container requires a self-hosted runner (RUNNER_ENVIRONMENT='${RUNNER_ENVIRONMENT:-unset}'). GitHub-hosted macOS runners are virtual machines without nested virtualisation and report kern.hv_support=0."
fi

if [[ "${RUNNER_OS:-}" != "macOS" ]]; then
  fail "apple-container requires a macOS runner (RUNNER_OS='${RUNNER_OS:-unset}')."
fi

if [[ "${RUNNER_ARCH:-}" != "ARM64" ]]; then
  fail "apple-container requires an Apple Silicon runner (RUNNER_ARCH='${RUNNER_ARCH:-unset}'). Rosetta translation is refused."
fi

# uname is checked as well as RUNNER_ARCH: the runner variables describe how the
# runner was registered, uname describes the process actually executing. A
# runner registered as ARM64 but executing under Rosetta would pass the first
# check and fail here.
uname_s="$(uname -s)"
uname_m="$(uname -m)"
[[ "${uname_s}" == "Darwin" ]] || fail "apple-container requires Darwin (uname -s reported '${uname_s}')."
[[ "${uname_m}" == "arm64" ]] || fail "apple-container requires native arm64 (uname -m reported '${uname_m}'). This step must not run under Rosetta."

# --- macOS version ----------------------------------------------------------
product_version="$(sw_vers -productVersion)"
macos_major="${product_version%%.*}"
case "${macos_major}" in
  ''|*[!0-9]*) fail "could not parse macOS major version from sw_vers -productVersion='${product_version}'." ;;
esac
if (( macos_major < MIN_MACOS )); then
  fail "apple-container requires macOS ${MIN_MACOS} or newer; this runner reports ${product_version}."
fi

# --- Hypervisor -------------------------------------------------------------
# kern.hv_support is the authoritative signal that Virtualization.framework can
# create a VM here. It is 0 inside every nested/virtualised macOS environment.
hv_support="$(sysctl -n kern.hv_support 2>/dev/null || echo "")"
if [[ "${hv_support}" != "1" ]]; then
  fail "Virtualization.framework is unavailable: kern.hv_support='${hv_support:-unreadable}'. apple-container requires bare-metal Apple Silicon; this is 0 on GitHub-hosted macOS and inside any nested VM."
fi

# --- launchd user domain ----------------------------------------------------
# `container system start` bootstraps com.apple.container.apiserver into the
# calling user's launchd domain. When the Actions runner is installed as a
# LaunchDaemon (or is otherwise running without a user session) that domain does
# not exist, and the service registration fails with an opaque launchd error
# well after the CLI has been installed. Detect it up front and say exactly what
# has to change.
runner_uid="$(id -u)"
if [[ "${runner_uid}" == "0" ]]; then
  fail "apple-container must not run as root: 'container system start' registers a per-user LaunchAgent and the agent's content store is per-user. Run the Actions runner as an unprivileged user."
fi
if ! launchctl print "gui/${runner_uid}" >/dev/null 2>&1; then
  fail "launchd GUI domain gui/${runner_uid} is unavailable, so 'container system start' cannot register com.apple.container.apiserver. Install the Actions runner as a per-user LaunchAgent under an auto-logged-in account (./svc.sh install), not as a LaunchDaemon or over a bare SSH session."
fi

# --- Toolchain ---------------------------------------------------------------
# gh-aw's generated setup scripts use bash 4 features (associative arrays,
# mapfile, ${var@Q}, ${var,,}). macOS ships bash 3.2 as /bin/bash and always
# will, so a runner without a newer bash on PATH would fail deep inside an
# unrelated script with a confusing syntax error. Check it here instead.
bash_major="${BASH_VERSINFO[0]:-0}"
if (( bash_major < 4 )); then
  fail "bash ${BASH_VERSION:-unknown} is too old. gh-aw's generated setup scripts require bash 4 or newer, and macOS only ships bash 3.2. Install a newer bash on the runner (for example 'brew install bash') and make sure it precedes /bin/bash on PATH."
fi

# --- Docker infrastructure --------------------------------------------------
# Apple Container runs only the agent. Squid, the API proxy and the CLI proxy
# stay in Docker Compose, so a missing Docker daemon is fatal here even though
# the agent never sees it.
command -v docker >/dev/null 2>&1 || fail "docker is not installed. apple-container moves only the agent into a VM; AWF still runs Squid, the API proxy and the CLI proxy under Docker Compose on the host."
docker info >/dev/null 2>&1 || fail "the Docker daemon is not reachable. AWF requires it for the Squid/API-proxy/CLI-proxy infrastructure containers."
docker compose version >/dev/null 2>&1 || fail "the Docker Compose plugin is unavailable ('docker compose version' failed). AWF orchestrates its infrastructure containers with Compose."

echo "runner is eligible for apple-container:"
echo "  macOS ${product_version} (arm64, kern.hv_support=1)"
echo "  launchd user domain gui/${runner_uid} available"
echo "  bash ${BASH_VERSION}"
echo "  docker: $(docker version --format '{{.Server.Version}}' 2>/dev/null || echo 'unknown')"
echo "::endgroup::"
