---
tools:
  cache-memory:
    key: issues-data
  bash:
    - "gh issue list *"
    - "gh api *"
    - "jq *"
    - "/tmp/gh-aw/jqschema.sh"
    - "mkdir *"
    - "date *"
    - "cp *"
    - "ln *"

steps:
  - name: Fetch issues data
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Create output directories
      mkdir -p /tmp/gh-aw/issues-data
      mkdir -p /tmp/gh-aw/cache-memory
      
      # Get today's date for cache identification
      TODAY=$(date '+%Y-%m-%d')
      CACHE_DIR="/tmp/gh-aw/cache-memory"
      
      # Check if cached data exists from today
      if [ -f "$CACHE_DIR/issues-${TODAY}.json" ] && [ -s "$CACHE_DIR/issues-${TODAY}.json" ]; then
        echo "✓ Found cached issues data from ${TODAY}"
        cp "$CACHE_DIR/issues-${TODAY}.json" /tmp/gh-aw/issues-data/issues.json
        
        # Regenerate schema if missing
        if [ ! -f "$CACHE_DIR/issues-${TODAY}-schema.json" ]; then
          /tmp/gh-aw/jqschema.sh < /tmp/gh-aw/issues-data/issues.json > "$CACHE_DIR/issues-${TODAY}-schema.json"
        fi
        cp "$CACHE_DIR/issues-${TODAY}-schema.json" /tmp/gh-aw/issues-data/issues-schema.json
        
        echo "Using cached data from ${TODAY}"
        echo "Total issues in cache: $(jq 'length' /tmp/gh-aw/issues-data/issues.json)"
      else
        echo "⬇ Downloading fresh issues data..."
        
        # Fetch all issues (open and closed) using gh CLI
        # Using --limit 1000 to get the last 1000 issues, unfiltered
        echo "Fetching the last 1000 issues..."
        gh issue list --repo ${{ github.repository }} \
          --state all \
          --json number,title,author,createdAt,state,url,body,labels,updatedAt,closedAt,milestone,assignees,comments \
          --limit 1000 \
          > /tmp/gh-aw/issues-data/issues.json

        # Generate schema for reference
        /tmp/gh-aw/jqschema.sh < /tmp/gh-aw/issues-data/issues.json > /tmp/gh-aw/issues-data/issues-schema.json

        # Store in cache with today's date
        cp /tmp/gh-aw/issues-data/issues.json "$CACHE_DIR/issues-${TODAY}.json"
        cp /tmp/gh-aw/issues-data/issues-schema.json "$CACHE_DIR/issues-${TODAY}-schema.json"

        echo "✓ Issues data saved to cache: issues-${TODAY}.json"
        echo "Total issues found: $(jq 'length' /tmp/gh-aw/issues-data/issues.json)"
      fi
      
      # Always ensure data is available at expected locations for backward compatibility
      echo "Issues data available at: /tmp/gh-aw/issues-data/issues.json"
      echo "Schema available at: /tmp/gh-aw/issues-data/issues-schema.json"
---

<!--
## Issues Data Fetch

This shared component fetches up to 1000 issues from the repository, with intelligent caching to avoid redundant API calls.

### What It Does

1. Creates output directories at `/tmp/gh-aw/issues-data/` and `/tmp/gh-aw/cache-memory/`
2. Checks for cached issues data from today's date in cache-memory
3. If cache exists (from earlier workflow runs today):
   - Uses cached data instead of making API calls
   - Copies data from cache to working directory
4. If cache doesn't exist:
   - Fetches up to 1000 issues (both open and closed) using `gh issue list`
   - Saves data to cache with date-based filename (e.g., `issues-2024-11-18.json`)
   - Copies data to working directory for use
5. Generates a schema of the data structure

### Caching Strategy

- **Cache Key**: `issues-data` for workflow-level sharing
- **Cache Files**: Stored with today's date in the filename (e.g., `issues-2024-11-18.json`)
- **Cache Location**: `/tmp/gh-aw/cache-memory/`
- **Cache Benefits**: 
  - Multiple workflows running on the same day share the same issues data
  - Reduces GitHub API rate limit usage
  - Faster workflow execution after first fetch of the day

### Output Files

- **`/tmp/gh-aw/issues-data/issues.json`**: Full issues data including number, title, author, timestamps, state, URL, body, labels, milestone, assignees, comments, etc.
- **`/tmp/gh-aw/issues-data/issues-schema.json`**: JSON schema showing the structure of the issues data

### Requirements

- Requires `jqschema.md` to be imported for schema generation
- Uses `gh issue list` with `--state all` to get both open and closed issues
- Cache-memory tool is automatically configured for data persistence
-->

## Issues Data

Pre-fetched issues data is available at `/tmp/gh-aw/issues-data/issues.json` containing up to 1000 issues (open and closed).

### Schema

The issues data structure is:

```json
[
  {
    "number": "number",
    "title": "string",
    "state": "string",
    "url": "string",
    "body": "string",
    "createdAt": "string",
    "updatedAt": "string",
    "closedAt": "string",
    "author": {
      "id": "string",
      "login": "string",
      "name": "string"
    },
    "assignees": [
      {
        "id": "string",
        "login": "string",
        "name": "string"
      }
    ],
    "labels": [
      {
        "id": "string",
        "name": "string",
        "color": "string",
        "description": "string"
      }
    ],
    "milestone": {
      "id": "string",
      "number": "number",
      "title": "string",
      "description": "string",
      "dueOn": "string"
    },
    "comments": [
      {
        "id": "string",
        "url": "string",
        "body": "string",
        "createdAt": "string",
        "author": {
          "id": "string",
          "login": "string",
          "name": "string"
        }
      }
    ]
  }
]
```

### Usage Examples

```bash
# Get total number of issues
jq 'length' /tmp/gh-aw/issues-data/issues.json

# Get only open issues
jq '[.[] | select(.state == "OPEN")]' /tmp/gh-aw/issues-data/issues.json

# Get issues from the last 7 days (cross-platform: GNU date first, BSD fallback)
DATE_7_DAYS_AGO=$(date -d '7 days ago' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -v-7d '+%Y-%m-%dT%H:%M:%SZ')
jq --arg date "$DATE_7_DAYS_AGO" '[.[] | select(.createdAt >= $date)]' /tmp/gh-aw/issues-data/issues.json

# Get issue numbers
jq '[.[].number]' /tmp/gh-aw/issues-data/issues.json

# Get issues with specific label
jq '[.[] | select(.labels | any(.name == "bug"))]' /tmp/gh-aw/issues-data/issues.json
```
