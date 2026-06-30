#!/usr/bin/env bash
set +o histexpand

# Reclaim /tmp/gh-aw/sandbox if it is not writable by the current user (e.g. root-owned, left by
# a prior rootless container run on the same runner).  A root-owned sandbox causes AWF's
# writeConfigs() to fail with EACCES when it tries to mkdir /tmp/gh-aw/sandbox/firewall/logs —
# killing the run before the agent is ever invoked.  The chmod-based fallback in AWF also fails
# Permission denied for the same reason, so the only reliable fix is to remove and recreate the
# tree here, before AWF starts.
sandbox_dir="/tmp/gh-aw/sandbox"
if [ -d "${sandbox_dir}" ] && ! [ -w "${sandbox_dir}" ]; then
  echo "[WARN] ${sandbox_dir} is not writable by the current user (uid $(id -u)); reclaiming before AWF starts..."
  if sudo rm -rf "${sandbox_dir}" 2>/dev/null; then
    echo "Removed stale non-writable ${sandbox_dir} via sudo"
  elif rm -rf "${sandbox_dir}" 2>/dev/null; then
    echo "Removed stale ${sandbox_dir}"
  else
    echo "[WARN] Failed to remove ${sandbox_dir}; AWF writeConfigs() may fail with EACCES" >&2
  fi
fi

mkdir -p /tmp/gh-aw/agent
mkdir -p /tmp/gh-aw/sandbox/agent/logs
echo "Created /tmp/gh-aw/agent directory for agentic workflow temporary files"
