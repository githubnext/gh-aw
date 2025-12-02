#!/bin/bash
# Query GitHub pull requests with jq filtering support
#
# Usage: ./query-prs.sh [OPTIONS]
#
# Options:
#   --repo OWNER/REPO    Repository to query (default: current repo)
#   --state STATE        PR state: open, closed, merged, all (default: open)
#   --limit N            Maximum number of PRs (default: 30)
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
    "description": "Array of pull request objects",
    "item_fields": {
      "number": "integer - PR number",
      "title": "string - PR title",
      "state": "string - PR state (OPEN, CLOSED, MERGED)",
      "author": "object - Author info with login field",
      "createdAt": "string - ISO timestamp of creation",
      "updatedAt": "string - ISO timestamp of last update",
      "mergedAt": "string|null - ISO timestamp of merge",
      "closedAt": "string|null - ISO timestamp of close",
      "headRefName": "string - Source branch name",
      "baseRefName": "string - Target branch name",
      "isDraft": "boolean - Whether PR is a draft",
      "reviewDecision": "string|null - Review decision (APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED)",
      "additions": "integer - Lines added",
      "deletions": "integer - Lines deleted",
      "changedFiles": "integer - Number of files changed",
      "labels": "array - Array of label objects with name field",
      "assignees": "array - Array of assignee objects with login field",
      "reviewRequests": "array - Array of review request objects",
      "url": "string - PR URL"
    }
  },
  "suggested_queries": [
    {"description": "Get all data", "query": "."},
    {"description": "Get PR numbers and titles", "query": ".[] | {number, title}"},
    {"description": "Get open PRs only", "query": ".[] | select(.state == \"OPEN\")"},
    {"description": "Get merged PRs", "query": ".[] | select(.mergedAt != null)"},
    {"description": "Get PRs by author", "query": ".[] | select(.author.login == \"USERNAME\")"},
    {"description": "Get large PRs", "query": ".[] | select(.changedFiles > 10) | {number, title, changedFiles}"},
    {"description": "Get PRs with labels", "query": ".[] | {number, title, labels: [.labels[].name]}"},
    {"description": "Count by state", "query": "group_by(.state) | map({state: .[0].state, count: length})"}
  ]
}
EOF
fi
