#!/usr/bin/env bash
set -uo pipefail

if (( $# < 3 )); then
  echo "[copilot-harness-runner] usage: <node> <harness> <command> [args...]" >&2
  exit 2
fi

node_executable=$1
harness=$2
shift 2

max_retries=${GH_AW_HARNESS_CRASH_RETRIES:-3}
delay_ms=${GH_AW_HARNESS_CRASH_DELAY_MS:-5000}
if ! [[ $max_retries =~ ^[0-9]+$ ]] || (( max_retries > 3 )); then
  max_retries=3
fi
if ! [[ $delay_ms =~ ^[0-9]+$ ]] || (( delay_ms > 60000 )); then
  delay_ms=5000
fi

attempt=0
while true; do
  "$node_executable" "$harness" "$@"
  status=$?

  case $status in
    134 | 135 | 136 | 137 | 138 | 139)
      ;;
    *)
      exit "$status"
      ;;
  esac

  if (( attempt >= max_retries )); then
    echo "[copilot-harness-runner] harness terminated by signal (exit=${status}); crash retry budget exhausted" >&2
    exit "$status"
  fi

  attempt=$((attempt + 1))
  echo "[copilot-harness-runner] harness terminated by signal (exit=${status}); retrying fresh harness ${attempt}/${max_retries}" >&2
  if (( delay_ms > 0 )); then
    printf -v delay_seconds '%d.%03d' "$((delay_ms / 1000))" "$((delay_ms % 1000))"
    sleep "$delay_seconds"
  fi
done
