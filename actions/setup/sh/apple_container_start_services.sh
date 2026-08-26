#!/usr/bin/env bash
set +o histexpand

# apple_container_start_services.sh - Start and verify Apple `container` system
# services for AWF's apple-container runtime.
#
# Three things have to be true before AWF may launch an agent VM:
#
#   1. The API server is running and `container system status` reports it
#      healthy. AWF's own preflight repeats this, but doing it here means a
#      broken service surfaces as a named setup failure rather than as an opaque
#      agent-launch error.
#   2. A default kernel is installed. `container system start` prompts for this
#      interactively by default, which would hang a headless runner forever, so
#      --enable-kernel-install is always passed.
#   3. The service and every later `container` invocation — including the ones
#      AWF makes — agree on one application root. The root holds the image
#      content store, so a service started under one root and an image pulled
#      under another produce a VM that cannot find its own agent image. The root
#      is therefore exported through CONTAINER_APP_ROOT in $GITHUB_ENV, which is
#      the same variable the CLI reads for --app-root.
#
# State is run-scoped by default: a fresh root per run means no image, kernel or
# container left by an earlier job on this persistent runner can influence this
# one. Operators who would rather keep a warm content store can point
# GH_AW_APPLE_CONTAINER_APP_ROOT at a stable directory; the isolation is then
# theirs to manage.
#
# Inputs:
#   GH_AW_APPLE_CONTAINER_APP_ROOT   optional override for the application root
#   GH_AW_APPLE_CONTAINER_BIN        optional explicit CLI path
#   GH_AW_APPLE_CONTAINER_TIMEOUT    optional API readiness timeout in seconds
#
# Outputs (GITHUB_ENV):
#   CONTAINER_APP_ROOT, CONTAINER_LOG_ROOT, GH_AW_APPLE_CONTAINER_APP_ROOT

set -euo pipefail

fail() {
  echo "::error::$1"
  exit 1
}

CONTAINER_BIN="${GH_AW_APPLE_CONTAINER_BIN:-}"
if [[ -z "${CONTAINER_BIN}" ]]; then
  CONTAINER_BIN="$(command -v container 2>/dev/null || true)"
fi
[[ -n "${CONTAINER_BIN}" && -x "${CONTAINER_BIN}" ]] || fail "the Apple container CLI was not found; the setup step must run before services are started."

START_TIMEOUT="${GH_AW_APPLE_CONTAINER_TIMEOUT:-120}"
case "${START_TIMEOUT}" in
  ''|*[!0-9]*) fail "GH_AW_APPLE_CONTAINER_TIMEOUT must be a positive integer number of seconds, got '${START_TIMEOUT}'." ;;
esac

: "${RUNNER_TEMP:?RUNNER_TEMP is required}"

# Run-scoped default. GITHUB_RUN_ID/ATTEMPT are always set in Actions; the job
# name is included so two jobs of the same run on the same runner cannot collide.
run_scope="${GITHUB_RUN_ID:-local}-${GITHUB_RUN_ATTEMPT:-1}-${GITHUB_JOB:-agent}"
run_scope="$(printf '%s' "${run_scope}" | tr -c 'A-Za-z0-9._-' '-')"

APP_ROOT="${GH_AW_APPLE_CONTAINER_APP_ROOT:-}"
APP_ROOT_IS_SHARED=true
if [[ -z "${APP_ROOT}" ]]; then
  APP_ROOT="${RUNNER_TEMP}/gh-aw/apple-container/${run_scope}/app-root"
  APP_ROOT_IS_SHARED=false
fi
LOG_ROOT="${APP_ROOT}/logs"

mkdir -p "${APP_ROOT}" "${LOG_ROOT}"
# The content store and every VM bundle live here. 0700 keeps another local
# account on a shared Mac from reading or planting image content.
chmod 700 "${APP_ROOT}"

# realpath is not on stock macOS; resolve through the shell instead so the
# exported root is absolute and symlink-free. This matters because /tmp and
# /var are symlinks into /private on macOS, and a root recorded as /tmp/... but
# reported by the service as /private/tmp/... would fail the consistency check
# below for no real reason.
APP_ROOT="$(cd "${APP_ROOT}" && pwd -P)"
LOG_ROOT="$(cd "${LOG_ROOT}" && pwd -P)"

echo "::group::Start Apple container services"
echo "app root: ${APP_ROOT}"
echo "log root: ${LOG_ROOT}"
if [[ "${APP_ROOT_IS_SHARED}" == "true" ]]; then
  echo "::warning::GH_AW_APPLE_CONTAINER_APP_ROOT is set, so Apple Container state is shared across runs on this runner. Run isolation of the image store and container state is the operator's responsibility."
fi

export CONTAINER_APP_ROOT="${APP_ROOT}"
export CONTAINER_LOG_ROOT="${LOG_ROOT}"

status_json() {
  "${CONTAINER_BIN}" system status --format json 2>/dev/null || true
}

service_running() {
  # `container system status` exits non-zero when the API server is not
  # reachable, so the exit status is the signal. The JSON body is only used for
  # diagnostics.
  "${CONTAINER_BIN}" system status >/dev/null 2>&1
}

if service_running; then
  echo "container system services are already running; reusing them"
else
  echo "starting container system services (kernel install enabled, timeout ${START_TIMEOUT}s)"
  # --enable-kernel-install is not optional here. Without it the CLI prompts on
  # stdin for permission to fetch the default kernel; a headless runner has no
  # stdin, and the process would block until the step is cancelled.
  #
  # Output is captured so a launchd registration failure can be reported with
  # its own message plus the actionable cause, instead of being buried in the
  # step log.
  start_log="${RUNNER_TEMP}/gh-aw/apple-container-start.log"
  mkdir -p "$(dirname "${start_log}")"
  if ! "${CONTAINER_BIN}" system start \
      --app-root "${APP_ROOT}" \
      --log-root "${LOG_ROOT}" \
      --enable-kernel-install \
      --timeout "${START_TIMEOUT}" >"${start_log}" 2>&1; then
    echo "--- container system start output ---"
    cat "${start_log}" || true
    echo "-------------------------------------"
    if grep -qiE 'bootstrap|launchd|domain|Load failed|5:  Input/output error' "${start_log}"; then
      fail "'container system start' could not register com.apple.container.apiserver with launchd. This is the known headless failure: the service is a per-user LaunchAgent and needs a real user (Aqua) session. Install the Actions runner with ./svc.sh install under an auto-logged-in account rather than as a LaunchDaemon or over SSH."
    fi
    fail "'container system start' failed; see the captured output above."
  fi
  cat "${start_log}" || true
fi

# Readiness gate. `container system start --timeout` already waits, but a
# service that reports started and then dies still has to be caught before AWF
# is invoked, so status is polled independently.
deadline=$(( $(date +%s) + START_TIMEOUT ))
until service_running; do
  if (( $(date +%s) >= deadline )); then
    echo "--- container system status ---"
    "${CONTAINER_BIN}" system status || true
    echo "-------------------------------"
    fail "Apple container services did not become healthy within ${START_TIMEOUT}s ('container system status' never succeeded)."
  fi
  sleep 2
done

echo "container system status:"
status_json | head -40 || true
"${CONTAINER_BIN}" system status || true

# A default kernel must exist or every `container create` fails late with
# "No default kernel configured." There is no read-only query for it in this CLI
# range (`container system kernel` exposes only `set`), so the start output is
# the signal: --enable-kernel-install installs it silently when missing, and
# anything else is repaired here rather than surfacing during agent launch.
if [[ -f "${RUNNER_TEMP}/gh-aw/apple-container-start.log" ]] &&
   grep -qi 'no default kernel configured' "${RUNNER_TEMP}/gh-aw/apple-container-start.log"; then
  echo "no default kernel was configured; installing the recommended kernel"
  "${CONTAINER_BIN}" system kernel set --recommended --arch arm64 ||
    fail "'container system kernel set --recommended' failed. Every 'container create' needs a default kernel; check the runner's network access to Apple's kernel download endpoint."
fi

echo "::endgroup::"

# Export the roots so the AWF invocation, the image pull step and the teardown
# step all address the same store. Without this the pull step would populate a
# run-scoped store while AWF read the user default store, and --skip-pull
# verification would fail on an image that had in fact just been pulled.
if [[ -n "${GITHUB_ENV:-}" ]]; then
  {
    echo "CONTAINER_APP_ROOT=${APP_ROOT}"
    echo "CONTAINER_LOG_ROOT=${LOG_ROOT}"
    echo "GH_AW_APPLE_CONTAINER_APP_ROOT=${APP_ROOT}"
    echo "GH_AW_APPLE_CONTAINER_BIN=${CONTAINER_BIN}"
  } >> "${GITHUB_ENV}"
fi

echo "Apple container services ready"
