#!/usr/bin/env bash
set +o histexpand

# apple_container_teardown.sh - Collect diagnostics and release Apple Container
# state after the agent has run.
#
# This runs with if: always() so a cancelled or failed job does not leave a VM
# running, a launchd service bound to a directory that is about to be deleted, or
# a run-scoped application root sitting on a persistent runner.
#
# Ordering matters: containers are stopped before the service, and the service is
# stopped before the application root is removed. Stopping the service last would
# leave the API server holding a deleted content store, which surfaces on the
# *next* job as an unexplained failure.
#
# Nothing here is fatal. A teardown failure must not mask the agent's own result,
# so every command is tolerated and reported rather than propagated.
#
# Inputs:
#   GH_AW_APPLE_CONTAINER_BIN        CLI path (exported by the setup step)
#   GH_AW_APPLE_CONTAINER_APP_ROOT   application root (exported by the start step)
#   GH_AW_APPLE_CONTAINER_PRESERVE   when "true", keep state for inspection

set -uo pipefail

CONTAINER_BIN="${GH_AW_APPLE_CONTAINER_BIN:-}"
if [[ -z "${CONTAINER_BIN}" ]]; then
  CONTAINER_BIN="$(command -v container 2>/dev/null || true)"
fi

if [[ -z "${CONTAINER_BIN}" || ! -x "${CONTAINER_BIN}" ]]; then
  echo "Apple container CLI not present; nothing to tear down"
  exit 0
fi

APP_ROOT="${GH_AW_APPLE_CONTAINER_APP_ROOT:-${CONTAINER_APP_ROOT:-}}"
PRESERVE="${GH_AW_APPLE_CONTAINER_PRESERVE:-false}"

echo "::group::Apple Container diagnostics"
echo "--- container system status ---"
"${CONTAINER_BIN}" system status 2>&1 | head -40 || true
echo "--- containers ---"
# `container list --all` output is metadata only. Deliberately no `inspect`:
# inspect output carries initProcess.environment, and this step's output is
# captured into the job log and can be uploaded as an artifact.
"${CONTAINER_BIN}" list --all 2>&1 | head -40 || true
echo "--- system logs (last 5m) ---"
"${CONTAINER_BIN}" system logs --last 5m 2>&1 | tail -100 || true
echo "::endgroup::"

if [[ "${PRESERVE}" == "true" ]]; then
  echo "::warning::GH_AW_APPLE_CONTAINER_PRESERVE=true; leaving Apple Container services and state in place for inspection"
  exit 0
fi

echo "::group::Apple Container teardown"

# Stop then delete every container this run left behind, so the service has no
# live guest when it is stopped.
container_ids="$("${CONTAINER_BIN}" list --all --quiet 2>/dev/null || true)"
if [[ -n "${container_ids}" ]]; then
  while IFS= read -r cid; do
    [[ -n "${cid}" ]] || continue
    echo "stopping container ${cid}"
    "${CONTAINER_BIN}" stop "${cid}" >/dev/null 2>&1 || true
    "${CONTAINER_BIN}" delete --force "${cid}" >/dev/null 2>&1 || true
  done <<<"${container_ids}"
else
  echo "no containers to remove"
fi

echo "stopping container system services"
"${CONTAINER_BIN}" system stop >/dev/null 2>&1 || echo "::warning::'container system stop' failed; the launchd service may still be running"

# Only a run-scoped root is removed. A root the operator pointed at a stable
# directory via GH_AW_APPLE_CONTAINER_APP_ROOT is theirs, and deleting it would
# destroy a warm content store they deliberately chose to keep.
#
# The start step resolves the root with `pwd -P`, so the prefix is compared
# against the physical RUNNER_TEMP as well. On macOS /tmp and /var are symlinks
# into /private, and comparing the logical paths would silently classify a
# run-scoped root as operator-managed and leak it onto the persistent runner.
run_scoped=false
if [[ -n "${APP_ROOT}" && -n "${RUNNER_TEMP:-}" ]]; then
  # Reject any path containing a traversal component before it is prefix-matched.
  # The start step normalises the root with `pwd -P` before exporting it, but if
  # that step died before writing $GITHUB_ENV this falls back to the raw
  # operator-supplied GH_AW_APPLE_CONTAINER_APP_ROOT, which has not been
  # normalised — and a value like "${RUNNER_TEMP}/gh-aw/apple-container/x/../../.."
  # would satisfy the prefix test below and then be handed to rm -rf.
  case "${APP_ROOT}" in
    *..*) echo "::warning::refusing to remove application root containing '..': ${APP_ROOT}" ;;
    *)
      runner_temp_physical="${RUNNER_TEMP}"
      if [[ -d "${RUNNER_TEMP}" ]]; then
        runner_temp_physical="$(cd "${RUNNER_TEMP}" && pwd -P)"
      fi
      case "${APP_ROOT}" in
        "${RUNNER_TEMP}"/gh-aw/apple-container/*) run_scoped=true ;;
        "${runner_temp_physical}"/gh-aw/apple-container/*) run_scoped=true ;;
      esac
      ;;
  esac
fi

if [[ "${run_scoped}" == "true" ]]; then
  echo "removing run-scoped application root ${APP_ROOT}"
  rm -rf "${APP_ROOT}" || echo "::warning::could not remove ${APP_ROOT}"
elif [[ -n "${APP_ROOT}" ]]; then
  echo "application root ${APP_ROOT} is operator-managed; leaving it in place"
fi

echo "::endgroup::"
