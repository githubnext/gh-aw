#!/bin/bash
# Query GitHub pull requests with jq filtering support
#
# Usage: ./query-prs.sh [OPTIONS]
#
# Options:
#   --repo OWNER/REPO    Repository to query (default: current repo)
#   --state STATE        PR state: open, closed, merged, all (default: open)
#   --limit N            Maximum number of PRs (default: 30)
#   --jq EXPRESSION      jq filter expression to apply to output

set -e

# Default values
REPO=""
STATE="open"
LIMIT=30
JQ_FILTER=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --repo)
            REPO="$2"
            shift 2
            ;;
        --state)
            STATE="$2"
            shift 2
            ;;
        --limit)
            LIMIT="$2"
            shift 2
            ;;
        --jq)
            JQ_FILTER="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

# JSON fields to fetch
JSON_FIELDS="number,title,state,author,createdAt,updatedAt,mergedAt,closedAt,headRefName,baseRefName,isDraft,reviewDecision,additions,deletions,changedFiles,labels,assignees,reviewRequests,url"

# Build and execute gh command with proper quoting
if [[ -n "$REPO" ]]; then
    OUTPUT=$(gh pr list --state "$STATE" --limit "$LIMIT" --json "$JSON_FIELDS" --repo "$REPO")
else
    OUTPUT=$(gh pr list --state "$STATE" --limit "$LIMIT" --json "$JSON_FIELDS")
fi

# Apply jq filter if specified
if [[ -n "$JQ_FILTER" ]]; then
    echo "$OUTPUT" | jq "$JQ_FILTER"
else
    echo "$OUTPUT"
fi
