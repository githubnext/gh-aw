#!/bin/bash
# Query GitHub discussions with jq filtering support
#
# Usage: ./query-discussions.sh [OPTIONS]
#
# Options:
#   --repo OWNER/REPO    Repository to query (default: current repo)
#   --limit N            Maximum number of discussions (default: 30)
#   --jq EXPRESSION      jq filter expression to apply to output (REQUIRED for data)
#
# Note: When --jq is not provided, returns schema and data size instead of full data.
#       This prevents overwhelming responses. Use --jq '.' to get all data.

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
    "description": "Array of discussion objects",
    "item_fields": {
      "number": "integer - Discussion number",
      "title": "string - Discussion title",
      "author": "object - Author info with login field",
      "createdAt": "string - ISO timestamp of creation",
      "updatedAt": "string - ISO timestamp of last update",
      "body": "string - Discussion body content",
      "category": "object - Category info with name field",
      "labels": "array - Array of label objects with name field",
      "comments": "object - Comments info with totalCount field",
      "answer": "object|null - Accepted answer if exists",
      "url": "string - Discussion URL"
    }
  },
  "suggested_queries": [
    {"description": "Get all data", "query": "."},
    {"description": "Get discussion numbers and titles", "query": ".[] | {number, title}"},
    {"description": "Get discussions by author", "query": ".[] | select(.author.login == \"USERNAME\")"},
    {"description": "Get discussions in category", "query": ".[] | select(.category.name == \"Ideas\")"},
    {"description": "Get answered discussions", "query": ".[] | select(.answer != null)"},
    {"description": "Get unanswered discussions", "query": ".[] | select(.answer == null) | {number, title, category: .category.name}"},
    {"description": "Get discussions with labels", "query": ".[] | {number, title, labels: [.labels[].name]}"},
    {"description": "Count by category", "query": "group_by(.category.name) | map({category: .[0].category.name, count: length})"}
  ]
}
EOF
fi
