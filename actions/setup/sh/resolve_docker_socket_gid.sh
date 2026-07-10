#!/usr/bin/env bash
set +o histexpand

# Resolve Docker Socket Path and Group ID
# This script resolves the Docker socket path and group ID for use with --group-add
# in containerized MCP gateway setups. It supports both auto-detection from DOCKER_HOST
# and explicit override via GH_AW_DOCKER_SOCK_PATH and GH_AW_DOCKER_SOCK_GID.
#
# Usage:
#   source resolve_docker_socket_gid.sh
#   # Sets: DOCKER_SOCK_PATH and DOCKER_SOCK_GID
#
# Environment Variables (Inputs):
#   GH_AW_DOCKER_SOCK_PATH - Override socket path (optional)
#   GH_AW_DOCKER_SOCK_GID  - Override socket group ID (optional)
#   DOCKER_HOST            - Docker daemon connection URL (used for auto-detection if no override)
#
# Environment Variables (Outputs):
#   DOCKER_SOCK_PATH - Resolved Docker socket path
#   DOCKER_SOCK_GID  - Resolved Docker socket group ID (numeric)
#
# Exit Codes:
#   0 - Success
#   1 - Failed to resolve socket path or GID, or GID is not numeric

set -e

# Resolve the Docker socket path. GH_AW_DOCKER_SOCK_PATH takes precedence over the
# DOCKER_HOST-derived guess, allowing operators on split-daemon / ARC-DinD setups to
# point the gateway at the real socket without needing host-side symlink hacks.
# When the override is absent, only unix:// and bare absolute paths are treated as socket
# paths; all other schemes (tcp://, ssh://, npipe://, etc.) fall back to /var/run/docker.sock.
DOCKER_SOCK_PATH="${GH_AW_DOCKER_SOCK_PATH:-}"
if [ -z "$DOCKER_SOCK_PATH" ]; then
  case "${DOCKER_HOST:-}" in
    unix://* )
      DOCKER_SOCK_PATH="${DOCKER_HOST#unix://}"
      # Strip optional authority component (e.g., unix://hostname/path -> /path)
      case "$DOCKER_SOCK_PATH" in
        /* ) ;; # already absolute
        * ) DOCKER_SOCK_PATH="/${DOCKER_SOCK_PATH#*/}" ;;
      esac ;;
    /* ) DOCKER_SOCK_PATH="$DOCKER_HOST" ;;
    * ) DOCKER_SOCK_PATH=/var/run/docker.sock ;;
  esac
fi

# Resolve the Docker socket group. GH_AW_DOCKER_SOCK_GID takes precedence, letting
# operators supply the group directly. stat -Lc follows symlinks so a symlinked socket
# resolves to the real socket's group without requiring a matching chown -h on the link.
# Note: stat -Lc is GNU coreutils (Linux only). macOS self-hosted runners must set
# GH_AW_DOCKER_SOCK_GID explicitly to bypass stat.
# Fail loudly instead of silently falling back to group 0 (root): passing --group-add 0
# to a non-root container gives no Docker-socket access and produces a confusing
# downstream "Docker daemon is not accessible" error.
if [ -n "${GH_AW_DOCKER_SOCK_GID:-}" ]; then
  DOCKER_SOCK_GID="$GH_AW_DOCKER_SOCK_GID"
else
  DOCKER_SOCK_GID=$(stat -Lc '%g' "$DOCKER_SOCK_PATH" 2>/dev/null || true)
  if [ -z "$DOCKER_SOCK_GID" ]; then
    echo "::error::Cannot determine Docker socket group for '$DOCKER_SOCK_PATH'. Set GH_AW_DOCKER_SOCK_PATH and GH_AW_DOCKER_SOCK_GID to configure the socket path and group explicitly." >&2
    [ -e "$DOCKER_SOCK_PATH" ] || echo "::warning::'$DOCKER_SOCK_PATH' does not exist on this runner." >&2
    exit 1
  fi
fi

# Validate that DOCKER_SOCK_GID is numeric before passing it to docker --group-add.
# Non-numeric values will cause docker run to fail with a confusing error.
case "$DOCKER_SOCK_GID" in
  ''|*[!0-9]*)
    echo "::error::DOCKER_SOCK_GID='$DOCKER_SOCK_GID' is not a valid numeric group ID. Set GH_AW_DOCKER_SOCK_GID to a numeric value." >&2
    exit 1
    ;;
esac

# Export the variables for use by the caller
export DOCKER_SOCK_PATH
export DOCKER_SOCK_GID
