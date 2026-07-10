#!/usr/bin/env bash
set +o histexpand
set -euo pipefail

# Reclaim stale root-owned /tmp/gh-aw/sandbox/firewall before creating it fresh.
# A previous AWF run can leave this directory (or its subdirectories) owned by root
# (Docker containers run as root, e.g. Squid runs as UID 13 inside a container, writing
# files to the host-mounted volume).
# setup.sh removes the entire /tmp/gh-aw tree using sudo, but if that cleanup partially
# failed (e.g. sudo was unavailable), the stale directory persists here.
# This targeted cleanup is defense-in-depth: it removes the specific directory tree that
# AWF's writeConfigs step will try to populate, so AWF can always create it fresh as the
# runner user.
# Note: the writability check covers both the firewall root AND its subdirectories (logs/,
# audit/) because Docker/Squid may own only the subdirectories when the parent was
# pre-created by the runner user on a prior clean run.
if [ -d /tmp/gh-aw/sandbox/firewall ] && \
   { [ ! -w /tmp/gh-aw/sandbox/firewall ] || \
     { [ -d /tmp/gh-aw/sandbox/firewall/logs ] && [ ! -w /tmp/gh-aw/sandbox/firewall/logs ]; } || \
     { [ -d /tmp/gh-aw/sandbox/firewall/audit ] && [ ! -w /tmp/gh-aw/sandbox/firewall/audit ]; }; }; then
  echo "Pre-flight: /tmp/gh-aw/sandbox/firewall is not writable (likely root-owned from prior AWF run); reclaiming with sudo"
  if command -v sudo >/dev/null 2>&1; then
    sudo -n rm -rf /tmp/gh-aw/sandbox/firewall 2>/dev/null || echo "::warning::sudo rm failed for /tmp/gh-aw/sandbox/firewall; AWF may fail with EACCES"
  else
    echo "::warning::sudo unavailable; cannot reclaim /tmp/gh-aw/sandbox/firewall; AWF may fail with EACCES"
  fi
fi

mkdir -p /tmp/gh-aw/agent
mkdir -p /tmp/gh-aw/sandbox/agent/logs
# Pre-create the firewall sandbox dirs as the runner user (uid=1001) before AWF starts.
# If the stale directory was successfully removed above, these create fresh dirs.
# If removal failed and the dirs still exist as root-owned, mkdir -p will emit EACCES here
# rather than deep inside AWF, surfacing the problem at an earlier step.
mkdir -p /tmp/gh-aw/sandbox/firewall/logs
mkdir -p /tmp/gh-aw/sandbox/firewall/audit

# Rootless AWF writes per-run chroot files (including generated /etc/hosts) under
# os.tmpdir(). Pin TMPDIR to a runner-owned subtree so writeConfigs never lands in a
# stale root-owned /tmp/awf-* path on persistent runners.
if [ -d /tmp/gh-aw/awf-tmp ] && [ ! -w /tmp/gh-aw/awf-tmp ]; then
  echo "Pre-flight: /tmp/gh-aw/awf-tmp is not writable; reclaiming with sudo"
  if command -v sudo >/dev/null 2>&1; then
    sudo -n rm -rf /tmp/gh-aw/awf-tmp 2>/dev/null || echo "::warning::sudo rm failed for /tmp/gh-aw/awf-tmp; GitHub Agentic Workflows may fail with EACCES"
  else
    echo "::warning::sudo unavailable; cannot reclaim /tmp/gh-aw/awf-tmp; GitHub Agentic Workflows may fail with EACCES"
  fi
fi
mkdir -p /tmp/gh-aw/awf-tmp

# Fast pre-flight probe: fail early with a clear message if AWF's temp root can't
# create/write the chroot hosts file it needs during config generation.
AWF_CHROOT_PROBE_DIR="/tmp/gh-aw/awf-tmp/chroot-preflight-$$"
mkdir -p "${AWF_CHROOT_PROBE_DIR}"
if ! printf '127.0.0.1 localhost\n' > "${AWF_CHROOT_PROBE_DIR}/hosts"; then
  echo "::error::gh-aw pre-flight failed: cannot write probe hosts file in /tmp/gh-aw/awf-tmp. This usually means rootless temp directory permissions are broken; check ownership/permissions under /tmp/gh-aw/awf-tmp."
  exit 1
fi
rm -rf "${AWF_CHROOT_PROBE_DIR}"

# Persist TMPDIR for subsequent workflow steps (including Execute GitHub Copilot CLI).
if [ -n "${GITHUB_ENV:-}" ]; then
  {
    echo "TMPDIR=/tmp/gh-aw/awf-tmp"
    echo "TMP=/tmp/gh-aw/awf-tmp"
    echo "TEMP=/tmp/gh-aw/awf-tmp"
  } >> "${GITHUB_ENV}"
fi

echo "Created /tmp/gh-aw/agent directory for agentic workflow temporary files"
