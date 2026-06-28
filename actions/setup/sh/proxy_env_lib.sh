#!/usr/bin/env bash
# Shared environment utilities sourced by start_difc_proxy.sh and start_cli_proxy.sh.
# Do not invoke this file directly — source it from the proxy startup scripts.

# normalize_github_host strips a URL down to its bare hostname, removing the
# protocol prefix, any trailing slashes, any path components, and any port
# number.  The port is stripped because GH_HOST is a hostname-only value that
# the gh CLI does not expect to include a port; upstream API URLs that need the
# port are constructed from GITHUB_SERVER_URL (which preserves the port).
#
# Examples:
#   https://github.com/         → github.com
#   https://myorg.ghe.com       → myorg.ghe.com
#   https://myghes.corp:8443    → myghes.corp
#   https://myghes.corp:8443/   → myghes.corp
normalize_github_host() {
  local host="$1"

  host="${host%/}"
  if [[ "$host" =~ ^https?:// ]]; then
    host="${host#http://}"
    host="${host#https://}"
    host="${host%%/*}"
  fi

  # Strip port number (e.g. myghes.corp:8443 → myghes.corp).
  # The regex matches "one or more non-[ chars, then :digits at end-of-string",
  # which catches host:port notation while skipping IPv6 bracket notation ([::1]).
  if [[ "$host" =~ ^[^\[]+:[0-9]+$ ]]; then
    host="${host%:*}"
  fi

  echo "$host"
}

# derive_proxy_upstream_env normalises the upstream GitHub host and exports the
# environment variables that the proxy container needs for correct routing.
#
# Design notes:
#
#  GH_HOST is always set unconditionally to the value derived from
#  GITHUB_SERVER_URL.  On GitHub-hosted runners the workflow-level environment
#  can have GH_HOST=github.com even when the server URL points at a *.ghe.com
#  tenant, which would cause the proxy to route to the wrong upstream.
#  Unconditional assignment (no :-) is intentional so that any stale
#  github.com default is always corrected.
#
#  GITHUB_HOST and GITHUB_ENTERPRISE_HOST use ${:-} fallback because they are
#  supplementary aliases; if the caller has already set them to the correct
#  tenant hostname, preserving that value is safe.
#
#  GITHUB_COPILOT_BASE_URL is derived automatically for *.ghe.com tenants only.
#  On GHES (on-premises) installations the Copilot API endpoint is not
#  predictable from the server URL alone, so no automatic derivation is
#  attempted.  Callers that need a non-default Copilot URL must set
#  GITHUB_COPILOT_BASE_URL explicitly before invoking this function.
derive_proxy_upstream_env() {
  local server_url="${GITHUB_SERVER_URL:-https://github.com}"
  local server_host
  local github_host="${GH_HOST:-${GITHUB_HOST:-${GITHUB_ENTERPRISE_HOST:-}}}"

  server_url="${server_url%/}"
  server_host="$(normalize_github_host "$server_url")"
  # Unconditionally normalise to the server host when the current value is
  # absent or is a stale github.com default on a non-github.com server.
  if [ -z "$github_host" ] || { [ "$server_host" != "github.com" ] && [ "$github_host" = "github.com" ]; }; then
    github_host="$server_host"
  fi
  if [ -z "$github_host" ]; then
    github_host="github.com"
  fi

  # Always export the normalised host so any stale default is overridden.
  export GH_HOST="$github_host"

  if [ "$github_host" != "github.com" ]; then
    export GITHUB_HOST="${GITHUB_HOST:-$github_host}"
    export GITHUB_ENTERPRISE_HOST="${GITHUB_ENTERPRISE_HOST:-$github_host}"
  fi

  if [ -z "${GITHUB_API_URL:-}" ] || { [ "$github_host" != "github.com" ] && [ "${GITHUB_API_URL}" = "https://api.github.com" ]; }; then
    if [ "$github_host" = "github.com" ]; then
      export GITHUB_API_URL="https://api.github.com"
    elif [[ "$github_host" == *.ghe.com ]]; then
      export GITHUB_API_URL="https://api.${github_host}"
    else
      export GITHUB_API_URL="${server_url}/api/v3"
    fi
  fi

  if [ -z "${GITHUB_GRAPHQL_URL:-}" ] || { [ "$github_host" != "github.com" ] && [ "${GITHUB_GRAPHQL_URL}" = "https://api.github.com/graphql" ]; }; then
    if [ "$github_host" = "github.com" ]; then
      export GITHUB_GRAPHQL_URL="https://api.github.com/graphql"
    elif [[ "$github_host" == *.ghe.com ]]; then
      export GITHUB_GRAPHQL_URL="https://api.${github_host}/graphql"
    else
      export GITHUB_GRAPHQL_URL="${server_url}/api/graphql"
    fi
  fi

  # Auto-derive the Copilot API URL for *.ghe.com data-residency tenants only.
  # For GHES (on-premises), the endpoint is not predictable; callers must set
  # GITHUB_COPILOT_BASE_URL explicitly if they need it.
  if [ -z "${GITHUB_COPILOT_BASE_URL:-}" ] && [[ "$github_host" == *.ghe.com ]]; then
    export GITHUB_COPILOT_BASE_URL="https://copilot-api.${github_host}"
  fi
}
