#!/usr/bin/env bash
set +o histexpand

# Tests for proxy_env_lib.sh — covers normalize_github_host and
# derive_proxy_upstream_env across github.com, *.ghe.com, and GHES cases.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB="${SCRIPT_DIR}/proxy_env_lib.sh"

# Source the library so we can call its functions directly.
# shellcheck source=proxy_env_lib.sh
source "$LIB"

PASS=0
FAIL=0

assert_eq() {
  local label="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    echo "PASS: $label"
    PASS=$((PASS + 1))
  else
    echo "FAIL: $label"
    echo "  expected: $expected"
    echo "  actual:   $actual"
    FAIL=$((FAIL + 1))
  fi
}

echo "Testing proxy_env_lib.sh"
echo "========================"

echo ""
echo "normalize_github_host"
echo "---------------------"

assert_eq "plain github.com" \
  "github.com" "$(normalize_github_host "github.com")"

assert_eq "https://github.com" \
  "github.com" "$(normalize_github_host "https://github.com")"

assert_eq "https://github.com/" \
  "github.com" "$(normalize_github_host "https://github.com/")"

assert_eq "*.ghe.com tenant" \
  "myorg.ghe.com" "$(normalize_github_host "https://myorg.ghe.com")"

assert_eq "GHES no port" \
  "myghes.corp" "$(normalize_github_host "https://myghes.corp")"

assert_eq "GHES with port" \
  "myghes.corp" "$(normalize_github_host "https://myghes.corp:8443")"

assert_eq "GHES with port and trailing slash" \
  "myghes.corp" "$(normalize_github_host "https://myghes.corp:8443/")"

assert_eq "GHES with port and path" \
  "myghes.corp" "$(normalize_github_host "https://myghes.corp:8443/some/path")"

assert_eq "http scheme" \
  "myghes.corp" "$(normalize_github_host "http://myghes.corp:8080")"

echo ""
echo "derive_proxy_upstream_env — public github.com"
echo "---------------------------------------------"

unset GH_HOST GITHUB_HOST GITHUB_ENTERPRISE_HOST GITHUB_API_URL GITHUB_GRAPHQL_URL GITHUB_COPILOT_BASE_URL
export GITHUB_SERVER_URL="https://github.com"
derive_proxy_upstream_env
assert_eq "github.com: GH_HOST" "github.com" "$GH_HOST"
assert_eq "github.com: GITHUB_API_URL" "https://api.github.com" "$GITHUB_API_URL"
assert_eq "github.com: GITHUB_GRAPHQL_URL" "https://api.github.com/graphql" "$GITHUB_GRAPHQL_URL"
assert_eq "github.com: GITHUB_COPILOT_BASE_URL empty" "" "${GITHUB_COPILOT_BASE_URL:-}"

echo ""
echo "derive_proxy_upstream_env — *.ghe.com tenant"
echo "---------------------------------------------"

unset GH_HOST GITHUB_HOST GITHUB_ENTERPRISE_HOST GITHUB_API_URL GITHUB_GRAPHQL_URL GITHUB_COPILOT_BASE_URL
export GITHUB_SERVER_URL="https://myorg.ghe.com"
derive_proxy_upstream_env
assert_eq "ghe.com: GH_HOST" "myorg.ghe.com" "$GH_HOST"
assert_eq "ghe.com: GITHUB_API_URL" "https://api.myorg.ghe.com" "$GITHUB_API_URL"
assert_eq "ghe.com: GITHUB_GRAPHQL_URL" "https://api.myorg.ghe.com/graphql" "$GITHUB_GRAPHQL_URL"
assert_eq "ghe.com: GITHUB_COPILOT_BASE_URL" "https://copilot-api.myorg.ghe.com" "$GITHUB_COPILOT_BASE_URL"

echo ""
echo "derive_proxy_upstream_env — stale GH_HOST=github.com on ghe.com tenant"
echo "------------------------------------------------------------------------"

unset GITHUB_HOST GITHUB_ENTERPRISE_HOST GITHUB_API_URL GITHUB_GRAPHQL_URL GITHUB_COPILOT_BASE_URL
GH_HOST="github.com"
export GITHUB_SERVER_URL="https://myorg.ghe.com"
derive_proxy_upstream_env
assert_eq "stale GH_HOST overridden" "myorg.ghe.com" "$GH_HOST"
assert_eq "stale: GITHUB_API_URL derived from tenant" "https://api.myorg.ghe.com" "$GITHUB_API_URL"

echo ""
echo "derive_proxy_upstream_env — explicit correct GH_HOST preserved"
echo "---------------------------------------------------------------"

unset GITHUB_HOST GITHUB_ENTERPRISE_HOST GITHUB_API_URL GITHUB_GRAPHQL_URL GITHUB_COPILOT_BASE_URL
GH_HOST="myorg.ghe.com"
export GITHUB_SERVER_URL="https://myorg.ghe.com"
derive_proxy_upstream_env
assert_eq "correct GH_HOST kept" "myorg.ghe.com" "$GH_HOST"

echo ""
echo "derive_proxy_upstream_env — GHES with non-standard port"
echo "--------------------------------------------------------"

unset GH_HOST GITHUB_HOST GITHUB_ENTERPRISE_HOST GITHUB_API_URL GITHUB_GRAPHQL_URL GITHUB_COPILOT_BASE_URL
export GITHUB_SERVER_URL="https://myghes.corp:8443"
derive_proxy_upstream_env
assert_eq "GHES port: GH_HOST no port" "myghes.corp" "$GH_HOST"
assert_eq "GHES port: GITHUB_API_URL with port" "https://myghes.corp:8443/api/v3" "$GITHUB_API_URL"
assert_eq "GHES port: GITHUB_GRAPHQL_URL with port" "https://myghes.corp:8443/api/graphql" "$GITHUB_GRAPHQL_URL"
assert_eq "GHES port: GITHUB_COPILOT_BASE_URL empty" "" "${GITHUB_COPILOT_BASE_URL:-}"

echo ""
echo "derive_proxy_upstream_env — explicit GITHUB_COPILOT_BASE_URL preserved"
echo "------------------------------------------------------------------------"

unset GH_HOST GITHUB_HOST GITHUB_ENTERPRISE_HOST GITHUB_API_URL GITHUB_GRAPHQL_URL
export GITHUB_SERVER_URL="https://myorg.ghe.com"
export GITHUB_COPILOT_BASE_URL="https://custom-copilot.example.com"
derive_proxy_upstream_env
assert_eq "explicit GITHUB_COPILOT_BASE_URL not overridden" \
  "https://custom-copilot.example.com" "$GITHUB_COPILOT_BASE_URL"

echo ""
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
