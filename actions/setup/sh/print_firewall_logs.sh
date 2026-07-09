#!/usr/bin/env bash
set +o histexpand

# print_firewall_logs.sh - Reclaim firewall sandbox dir ownership and print AWF firewall log summary.
#
# Usage: bash print_firewall_logs.sh [--rootless]
#
# Options:
#   --rootless   Use non-interactive sudo (sudo -n) with a non-sudo chmod fallback.
#                Default: use plain sudo (AWF ran with full sudo access).
#
# Environment:
#   AWF_LOGS_DIR         Path to the AWF logs directory (must be set).
#                        The Go compiler (engine_firewall_support.go) sets this to
#                        "${RUNNER_TEMP}/gh-aw/sandbox/firewall/logs"; the parent
#                        directory (firewall sandbox root) is used for chown/chmod.
#   GITHUB_STEP_SUMMARY  Path to the GitHub Actions step summary file.
#                        If unset or empty, log output is printed to stdout only.

ROOTLESS=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --rootless) ROOTLESS=true; shift ;;
    *) echo "Unknown argument: $1" >&2; exit 1 ;;
  esac
done

# FIREWALL_DIR is the firewall sandbox root, derived as the parent of AWF_LOGS_DIR.
# The Go compiler sets AWF_LOGS_DIR="${RUNNER_TEMP}/gh-aw/sandbox/firewall/logs",
# so FIREWALL_DIR resolves to "${RUNNER_TEMP}/gh-aw/sandbox/firewall".
FIREWALL_DIR="$(dirname "${AWF_LOGS_DIR}")"

if [[ "${ROOTLESS}" == "true" ]]; then
  # Best-effort: reclaim ownership and fix permissions (AWF cleanup may not have run).
  # Chown transfers ownership back to the runner user so the next run can rm -rf /tmp/gh-aw
  # without requiring sudo, preventing EACCES failures on reused runners.
  sudo -n chown -R "$(id -u):$(id -g)" "${FIREWALL_DIR}" 2>/dev/null || true
  sudo -n chmod -R a+rX "${FIREWALL_DIR}" 2>/dev/null || chmod -R a+rX "${FIREWALL_DIR}" 2>/dev/null || true
else
  # Reclaim ownership and fix permissions on firewall dirs for artifact upload and next-run cleanup.
  # AWF runs with sudo, creating files owned by root; chown transfers ownership back to the runner
  # user so the next run's setup.sh can rm -rf /tmp/gh-aw without requiring sudo.
  sudo chown -R "$(id -u):$(id -g)" "${FIREWALL_DIR}" 2>/dev/null || true
  sudo chmod -R a+rX "${FIREWALL_DIR}" 2>/dev/null || true
fi

# Only run awf logs summary if awf command exists (it may not be installed if workflow failed before install step)
if command -v awf &> /dev/null; then
  awf logs summary | tee -a "${GITHUB_STEP_SUMMARY:-/dev/null}"
else
  echo 'AWF binary not installed, skipping firewall log summary'
fi
