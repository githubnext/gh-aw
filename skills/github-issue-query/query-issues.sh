#!/bin/bash
# Query GitHub issues with jq filtering support
#
# Usage: ./query-issues.sh [OPTIONS]
#
# Options:
#   --repo OWNER/REPO    Repository to query (default: current repo)
#   --state STATE        Issue state: open, closed, all (default: open)
#   --limit N            Maximum number of issues (default: 30)
#   --jq EXPRESSION      jq filter expression to apply to output (REQUIRED for data)
#
# Note: When --jq is not provided, returns schema and data size instead of full data.
#       This prevents overwhelming responses. Use --jq '.' to get all data.

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
JSON_FIELDS="number,title,state,author,createdAt,updatedAt,closedAt,body,labels,assignees,comments,milestone,url,projectItems"

# Build and execute gh command with proper quoting
if [[ -n "$REPO" ]]; then
    OUTPUT=$(gh issue list --state "$STATE" --limit "$LIMIT" --json "$JSON_FIELDS" --repo "$REPO")
else
    OUTPUT=$(gh issue list --state "$STATE" --limit "$LIMIT" --json "$JSON_FIELDS")
fi

# Apply jq filter if specified
if [[ -n "$JQ_FILTER" ]]; then
    echo "$OUTPUT" | jq "$JQ_FILTER"
else
    # Return schema and size instead of full data
    ITEM_COUNT=$(jq 'length' <<< "$OUTPUT")
    DATA_SIZE=${#OUTPUT}
    
    # Validate values are numeric
    if ! [[ "$ITEM_COUNT" =~ ^[0-9]+$ ]]; then
        ITEM_COUNT=0
    fi
    if ! [[ "$DATA_SIZE" =~ ^[0-9]+$ ]]; then
        DATA_SIZE=0
    fi
    
    cat << EOF
{
  "message": "No --jq filter provided. Use --jq to filter and retrieve data.",
  "item_count": $ITEM_COUNT,
  "data_size_bytes": $DATA_SIZE,
  "schema": {
    "type": "array",
    "description": "Array of issue objects",
    "item_fields": {
      "number": "integer - Issue number",
      "title": "string - Issue title",
      "state": "string - Issue state (OPEN, CLOSED)",
      "author": "object - Author info with login field",
      "createdAt": "string - ISO timestamp of creation",
      "updatedAt": "string - ISO timestamp of last update",
      "closedAt": "string|null - ISO timestamp of close",
      "body": "string - Issue body content",
      "labels": "array - Array of label objects with name field",
      "assignees": "array - Array of assignee objects with login field",
      "comments": "object - Comments info with totalCount field",
      "milestone": "object|null - Milestone info with title field",
      "url": "string - Issue URL",
      "projectItems": "object - Project assignments with totalCount and nodes array containing project info"
    }
  },
  "suggested_queries": [
    {"description": "Get all data", "query": "."},
    {"description": "Get issue numbers and titles", "query": ".[] | {number, title}"},
    {"description": "Get open issues only", "query": ".[] | select(.state == \"OPEN\")"},
    {"description": "Get issues by author", "query": ".[] | select(.author.login == \"USERNAME\")"},
    {"description": "Get issues with label", "query": ".[] | select(.labels | map(.name) | index(\"bug\"))"},
    {"description": "Get issues with many comments", "query": ".[] | select(.comments.totalCount > 5) | {number, title, comments: .comments.totalCount}"},
    {"description": "Get issues with labels", "query": ".[] | {number, title, labels: [.labels[].name]}"},
    {"description": "Get issues with project assignments", "query": ".[] | {number, title, projects: [.projectItems.nodes[]? | .project?.url]}"},
    {"description": "Count by state", "query": "group_by(.state) | map({state: .[0].state, count: length})"}
  ]
}
EOF
fi
