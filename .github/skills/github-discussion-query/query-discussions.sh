#!/bin/bash
# Query GitHub discussions with jq filtering support
#
# Usage: ./query-discussions.sh [OPTIONS]
#
# Options:
#   --repo OWNER/REPO    Repository to query (default: current repo)
#   --limit N            Maximum number of discussions (default: 30)
#   --jq EXPRESSION      jq filter expression to apply to output

set -e

# Default values
REPO=""
LIMIT=30
JQ_FILTER=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --repo)
            REPO="$2"
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
JSON_FIELDS="number,title,author,createdAt,updatedAt,body,category,labels,comments,answer,url"

# Build and execute gh command with proper quoting
if [[ -n "$REPO" ]]; then
    OUTPUT=$(gh discussion list --limit "$LIMIT" --json "$JSON_FIELDS" --repo "$REPO")
else
    OUTPUT=$(gh discussion list --limit "$LIMIT" --json "$JSON_FIELDS")
fi

# Apply jq filter if specified
if [[ -n "$JQ_FILTER" ]]; then
    echo "$OUTPUT" | jq "$JQ_FILTER"
else
    echo "$OUTPUT"
fi
