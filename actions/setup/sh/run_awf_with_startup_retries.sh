#!/usr/bin/env bash
set +o histexpand
set -uo pipefail

# Runs an AWF command and retries startup/configuration failures that occur
# before the engine harness writes its first marker line.
#
# Required environment:
#   GH_AW_AWF_ENGINE_NAME      Engine ID used in retry diagnostics.
#   GH_AW_AWF_HARNESS_MARKER  Marker emitted by the engine harness, e.g. [claude-harness].
#   GH_AW_AWF_LOG_FILE        Main agent stdio log file to append to.
#
# Optional environment:
#   GH_AW_AWF_ATTEMPT_LOG_NAME         Safe token used in the temporary attempt log file name.
#   GH_AW_HARNESS_STARTUP_RETRIES      Shared startup retry budget, 0-2.
#   GH_AW_CLAUDE_STARTUP_RETRIES       Backward-compatible fallback when the shared budget is unset.
#   GH_AW_HARNESS_INITIAL_DELAY_MS     Retry delay in milliseconds, rounded up to seconds.

usage() {
  echo "Usage: $0 -- <awf-command> [args...]" >&2
}

if [ "${1:-}" = "--" ]; then
  shift
fi

if [ "$#" -eq 0 ]; then
  usage
  exit 2
fi

if [ -z "${GH_AW_AWF_ENGINE_NAME:-}" ]; then
  echo "GH_AW_AWF_ENGINE_NAME is required" >&2
  exit 2
fi

if [ -z "${GH_AW_AWF_HARNESS_MARKER:-}" ]; then
  echo "GH_AW_AWF_HARNESS_MARKER is required" >&2
  exit 2
fi

if [ -z "${GH_AW_AWF_LOG_FILE:-}" ]; then
  echo "GH_AW_AWF_LOG_FILE is required" >&2
  exit 2
fi

gh_aw_awf_startup_retries="${GH_AW_HARNESS_STARTUP_RETRIES:-${GH_AW_CLAUDE_STARTUP_RETRIES:-1}}"
if ! [[ "$gh_aw_awf_startup_retries" =~ ^[0-9]+$ ]]; then
  gh_aw_awf_startup_retries=1
fi
if [ "$gh_aw_awf_startup_retries" -gt 2 ]; then
  gh_aw_awf_startup_retries=2
fi

gh_aw_awf_initial_delay_ms="${GH_AW_HARNESS_INITIAL_DELAY_MS:-5000}"
if ! [[ "$gh_aw_awf_initial_delay_ms" =~ ^[0-9]+$ ]]; then
  gh_aw_awf_initial_delay_ms=5000
fi

gh_aw_awf_delay_s=$(((gh_aw_awf_initial_delay_ms + 999) / 1000))
gh_aw_awf_attempt=0
gh_aw_awf_attempt_log_name="${GH_AW_AWF_ATTEMPT_LOG_NAME:-agent}"

while true; do
  gh_aw_awf_attempt_log="$(mktemp "${RUNNER_TEMP:-/tmp}/gh-aw-awf-${gh_aw_awf_attempt_log_name}.XXXXXX")"
  "$@" 2>&1 | tee -a "$GH_AW_AWF_LOG_FILE" "$gh_aw_awf_attempt_log"
  gh_aw_awf_status=${PIPESTATUS[0]}
  if [ "$gh_aw_awf_status" -eq 0 ]; then
    rm -f "$gh_aw_awf_attempt_log"
    exit 0
  fi
  if ! grep -Fq "$GH_AW_AWF_HARNESS_MARKER" "$gh_aw_awf_attempt_log" &&
    grep -Eqi '(Fatal error:|Process exiting with code:|Refusing to use symlink as bind mountpoint|mcp gateway[^[:cntrl:]]{0,80}(startup failed|failed to start|startup error))' "$gh_aw_awf_attempt_log" &&
    [ "$gh_aw_awf_attempt" -lt "$gh_aw_awf_startup_retries" ]; then
    gh_aw_awf_attempt=$((gh_aw_awf_attempt + 1))
    echo "[${GH_AW_AWF_ENGINE_NAME}-awf-retry] AWF startup failed before ${GH_AW_AWF_ENGINE_NAME} harness; retrying fresh (startup retry ${gh_aw_awf_attempt}/${gh_aw_awf_startup_retries})"
    rm -f "$gh_aw_awf_attempt_log"
    sleep "$gh_aw_awf_delay_s"
    continue
  fi
  rm -f "$gh_aw_awf_attempt_log"
  exit "$gh_aw_awf_status"
done
