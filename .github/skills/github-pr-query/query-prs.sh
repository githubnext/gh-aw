#!/bin/bash
set +o histexpand

# Query GitHub pull requests with jq filtering support
#
# Usage: ./query-prs.sh [OPTIONS]
#
# Options:
#   --repo OWNER/REPO    Repository to query (default: current repo)
#   --state STATE        PR state: open, closed, merged, all (default: open)
#   --limit N            Maximum number of PRs (default: 30)
#   --author LOGIN       Filter by PR author login
#   --app SLUG           Filter by GitHub App author
#   --search QUERY       Apply GitHub search query
#   --jq EXPRESSION      jq filter expression to apply to output (REQUIRED for data)
#
# Note: When --jq is not provided, returns schema and data size instead of full data.
#       This prevents overwhelming responses. Use --jq '.' to get all data.

set -e

# Default values
REPO=""
STATE="open"
LIMIT=30
AUTHOR=""
APP=""
SEARCH=""
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
        --author)
            AUTHOR="$2"
            shift 2
            ;;
        --app)
            APP="$2"
            shift 2
            ;;
        --search)
            SEARCH="$2"
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
JSON_FIELDS="number,title,state,author,createdAt,updatedAt,mergedAt,closedAt,headRefName,baseRefName,isDraft,reviewDecision,mergeable,mergeStateStatus,statusCheckRollup,additions,deletions,changedFiles,labels,assignees,reviewRequests,latestReviews,reviews,comments,url"

# Build and execute gh command with proper quoting
GH_ARGS=(--state "$STATE" --limit "$LIMIT" --json "$JSON_FIELDS")

if [[ -n "$AUTHOR" ]]; then
    GH_ARGS+=(--author "$AUTHOR")
fi
if [[ -n "$APP" ]]; then
    GH_ARGS+=(--app "$APP")
fi
if [[ -n "$SEARCH" ]]; then
    GH_ARGS+=(--search "$SEARCH")
fi
if [[ -n "$REPO" ]]; then
    GH_ARGS+=(--repo "$REPO")
fi
OUTPUT=$(gh pr list "${GH_ARGS[@]}")

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
      "latestReviews": "array - Latest reviews for each PR",
      "reviews": "array - Review objects for each PR",
      "comments": "object - PR conversation comments summary",
      "mergeable": "string - Mergeability status",
      "mergeStateStatus": "string - Merge state status",
      "statusCheckRollup": "array|null - Check run rollup data",
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
    {"description": "Count by state", "query": "group_by(.state) | map({state: .[0].state, count: length})"},
    {"description": "Get PRs with actions-bot reviews", "query": ".[] | {number, title, botReviewCount: ([.reviews[]? | select(.author.login == \"github-actions[bot]\")] | length)}"}
  ]
}
EOF
fi
