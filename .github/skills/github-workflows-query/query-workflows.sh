#!/bin/bash
set +o histexpand

# Query GitHub Actions workflows with per_page pagination support.
#
# Usage: ./query-workflows.sh [OPTIONS]
#
# Options:
#   --owner OWNER      Repository owner (required)
#   --repo REPO        Repository name (required)
#   --per-page N       Results per page: 1-100 (default: 10)
#   --page N           Page number (default: 1)
#
# Alternatively, inputs can be provided as environment variables using the
# mcp-scripts INPUT_* convention (INPUT_OWNER, INPUT_REPO, INPUT_PER_PAGE,
# INPUT_PAGE). CLI arguments take precedence over environment variables.
#
# Calls the GitHub REST API:
#   GET /repos/{owner}/{repo}/actions/workflows?per_page={n}&page={n}
#
# Returns JSON:
#   { "total_count": N, "per_page": N, "page": N, "workflows": [...] }

set -e

# Defaults: pick up INPUT_* env vars (mcp-scripts convention) or fall back to
# hardcoded defaults; CLI flags below will override.
OWNER="${INPUT_OWNER:-}"
REPO="${INPUT_REPO:-}"
PER_PAGE="${INPUT_PER_PAGE:-10}"
PAGE="${INPUT_PAGE:-1}"

while [[ $# -gt 0 ]]; do
    case $1 in
        --owner)
            OWNER="$2"
            shift 2
            ;;
        --repo)
            REPO="$2"
            shift 2
            ;;
        --per-page)
            PER_PAGE="$2"
            shift 2
            ;;
        --page)
            PAGE="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

if [[ -z "$OWNER" ]]; then
    echo '{"error": "owner is required"}' >&2
    exit 1
fi

if [[ -z "$REPO" ]]; then
    echo '{"error": "repo is required"}' >&2
    exit 1
fi

if ! [[ "$PER_PAGE" =~ ^[0-9]+$ ]] || [[ "$PER_PAGE" -lt 1 ]] || [[ "$PER_PAGE" -gt 100 ]]; then
    echo '{"error": "per_page must be between 1 and 100"}' >&2
    exit 1
fi

RESPONSE=$(gh api "repos/${OWNER}/${REPO}/actions/workflows?per_page=${PER_PAGE}&page=${PAGE}")

echo "$RESPONSE" | jq \
    --argjson per_page "$PER_PAGE" \
    --argjson page "$PAGE" \
    '{
        total_count: .total_count,
        per_page: $per_page,
        page: $page,
        workflows: [.workflows[] | {id, node_id, name, path, state, created_at, updated_at, url, html_url, badge_url}]
    }'
