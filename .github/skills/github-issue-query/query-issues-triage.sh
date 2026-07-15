#!/bin/bash
set +o histexpand

# Query GitHub issues in triage mode (no body) with per_page pagination support.
#
# Returns only metadata fields useful for triage (number, title, state, labels,
# author, created_at, html_url) — body is intentionally omitted to reduce token
# cost. Use `issue_read` to fetch full details for a specific issue when needed.
#
# Usage: ./query-issues-triage.sh [OPTIONS]
#
# Options:
#   --owner OWNER      Repository owner (required)
#   --repo REPO        Repository name (required)
#   --state STATE      Issue state: open, closed, all (default: open)
#   --labels LABELS    Comma-separated label names to filter by (default: none)
#   --assignee LOGIN   Filter by assignee login (default: none)
#   --per-page N       Results per page: 1-100 (default: 10)
#   --page N           Page number (default: 1)
#
# Alternatively, inputs can be provided as environment variables using the
# mcp-scripts INPUT_* convention (INPUT_OWNER, INPUT_REPO, INPUT_STATE,
# INPUT_LABELS, INPUT_ASSIGNEE, INPUT_PER_PAGE, INPUT_PAGE). CLI arguments
# take precedence over environment variables.
#
# Calls the GitHub REST API:
#   GET /repos/{owner}/{repo}/issues?state={state}&per_page={n}&page={n}
#
# Returns JSON:
#   { "issues": [...], "item_count": N, "per_page": N, "page": N }

set -e

# Defaults: pick up INPUT_* env vars (mcp-scripts convention) or fall back to
# hardcoded defaults; CLI flags below will override.
OWNER="${INPUT_OWNER:-}"
REPO="${INPUT_REPO:-}"
STATE="${INPUT_STATE:-open}"
LABELS="${INPUT_LABELS:-}"
ASSIGNEE="${INPUT_ASSIGNEE:-}"
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
        --state)
            STATE="$2"
            shift 2
            ;;
        --labels)
            LABELS="$2"
            shift 2
            ;;
        --assignee)
            ASSIGNEE="$2"
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

QUERY="repos/${OWNER}/${REPO}/issues?state=${STATE}&per_page=${PER_PAGE}&page=${PAGE}"
if [[ -n "$LABELS" ]]; then
    ENCODED_LABELS=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$LABELS")
    QUERY="${QUERY}&labels=${ENCODED_LABELS}"
fi
if [[ -n "$ASSIGNEE" ]]; then
    ENCODED_ASSIGNEE=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$ASSIGNEE")
    QUERY="${QUERY}&assignee=${ENCODED_ASSIGNEE}"
fi

RESPONSE=$(gh api "$QUERY")

echo "$RESPONSE" | jq \
    --argjson per_page "$PER_PAGE" \
    --argjson page "$PAGE" \
    '{
        issues: [.[] | {
            number,
            title,
            state,
            html_url,
            created_at,
            updated_at,
            author: .user.login,
            labels: [.labels[].name],
            assignees: [.assignees[].login],
            milestone: (.milestone.title // null)
        }],
        item_count: length,
        per_page: $per_page,
        page: $page
    }'
