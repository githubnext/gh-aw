---
tools:
  cache-memory:
    key: weekly-issues-data
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
  - name: Fetch weekly issues data
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Create output directories
      mkdir -p /tmp/gh-aw/weekly-issues-data
      mkdir -p /tmp/gh-aw/cache-memory
      
      # Get today's date for cache identification
      TODAY=$(date '+%Y-%m-%d')
      CACHE_DIR="/tmp/gh-aw/cache-memory"
      
      # Check if cached data exists from today
      if [ -f "$CACHE_DIR/weekly-issues-${TODAY}.json" ] && [ -s "$CACHE_DIR/weekly-issues-${TODAY}.json" ]; then
        echo "✓ Found cached weekly issues data from ${TODAY}"
        cp "$CACHE_DIR/weekly-issues-${TODAY}.json" /tmp/gh-aw/weekly-issues-data/issues.json
        
        # Regenerate schema if missing
        if [ ! -f "$CACHE_DIR/weekly-issues-${TODAY}-schema.json" ]; then
          /tmp/gh-aw/jqschema.sh < /tmp/gh-aw/weekly-issues-data/issues.json > "$CACHE_DIR/weekly-issues-${TODAY}-schema.json"
        fi
        cp "$CACHE_DIR/weekly-issues-${TODAY}-schema.json" /tmp/gh-aw/weekly-issues-data/issues-schema.json
        
        echo "Using cached data from ${TODAY}"
        echo "Total issues in cache: $(jq 'length' /tmp/gh-aw/weekly-issues-data/issues.json)"
      else
        echo "⬇ Downloading fresh weekly issues data..."
        
        # Calculate date 7 days ago (cross-platform: GNU date first, BSD fallback)
        DATE_7_DAYS_AGO=$(date -d '7 days ago' '+%Y-%m-%d' 2>/dev/null || date -v-7d '+%Y-%m-%d')
        
        echo "Fetching issues created or updated since ${DATE_7_DAYS_AGO}..."
        
        # Fetch issues from the last 7 days using gh CLI
        # Using --search with updated filter to get recent activity
        gh issue list --repo ${{ github.repository }} \
          --search "updated:>=${DATE_7_DAYS_AGO}" \
          --state all \
          --json number,title,author,createdAt,state,url,body,labels,updatedAt,closedAt,milestone,assignees,comments \
          --limit 500 \
          > /tmp/gh-aw/weekly-issues-data/issues.json

        # Generate schema for reference
        /tmp/gh-aw/jqschema.sh < /tmp/gh-aw/weekly-issues-data/issues.json > /tmp/gh-aw/weekly-issues-data/issues-schema.json

        # Store in cache with today's date
        cp /tmp/gh-aw/weekly-issues-data/issues.json "$CACHE_DIR/weekly-issues-${TODAY}.json"
        cp /tmp/gh-aw/weekly-issues-data/issues-schema.json "$CACHE_DIR/weekly-issues-${TODAY}-schema.json"

        echo "✓ Weekly issues data saved to cache: weekly-issues-${TODAY}.json"
        echo "Total issues found: $(jq 'length' /tmp/gh-aw/weekly-issues-data/issues.json)"
      fi
      
      # Always ensure data is available at expected locations for backward compatibility
      echo "Weekly issues data available at: /tmp/gh-aw/weekly-issues-data/issues.json"
      echo "Schema available at: /tmp/gh-aw/weekly-issues-data/issues-schema.json"
---

<!--
## Weekly Issues Data Fetch

This shared component fetches issues from the last 7 days, with intelligent caching to avoid redundant API calls.

### What It Does

1. Creates output directories at `/tmp/gh-aw/weekly-issues-data/` and `/tmp/gh-aw/cache-memory/`
2. Checks for cached issues data from today's date in cache-memory
3. If cache exists (from earlier workflow runs today):
   - Uses cached data instead of making API calls
   - Copies data from cache to working directory
4. If cache doesn't exist:
   - Calculates the date 7 days ago (cross-platform compatible)
   - Fetches issues updated in the last 7 days using `gh issue list`
   - Saves data to cache with date-based filename (e.g., `weekly-issues-2024-11-26.json`)
   - Copies data to working directory for use
5. Generates a schema of the data structure

### Caching Strategy

- **Cache Key Pattern**: Uses `weekly-issues-data-${{ github.run_id }}` for saving, with `restore-keys: weekly-issues-data-` for restoring from previous runs
- **Cache Files**: Stored with today's date in the filename (e.g., `weekly-issues-2024-11-26.json`)
- **Cache Location**: `/tmp/gh-aw/cache-memory/`
- **Cache Benefits**: 
  - Multiple workflows running on the same day share the same issues data via restore-keys
  - Reduces GitHub API rate limit usage
  - Faster workflow execution after first fetch of the day

### Output Files

- **`/tmp/gh-aw/weekly-issues-data/issues.json`**: Issues data from the last 7 days
- **`/tmp/gh-aw/weekly-issues-data/issues-schema.json`**: JSON schema showing the data structure

### Requirements

- Requires `jqschema.md` to be imported for schema generation
- Uses `gh issue list` with `--search "updated:>=[DATE]"` to get recent activity
- Cross-platform date calculation (works on both GNU and BSD date commands)
- Cache-memory tool is automatically configured for data persistence
-->

## Weekly Issues Data

Pre-fetched issues data from the last 7 days is available at `/tmp/gh-aw/weekly-issues-data/issues.json`.

This includes issues that were created or updated within the past week, providing a focused dataset for recent activity analysis.

### Schema

The weekly issues data structure is:

```json
[
  {
    "number": "number",
    "title": "string",
    "state": "string (OPEN or CLOSED)",
    "url": "string",
    "body": "string",
    "createdAt": "string (ISO 8601 timestamp)",
    "updatedAt": "string (ISO 8601 timestamp)",
    "closedAt": "string (ISO 8601 timestamp, null if open)",
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
# Get total number of issues from the last week
jq 'length' /tmp/gh-aw/weekly-issues-data/issues.json

# Get only open issues
jq '[.[] | select(.state == "OPEN")]' /tmp/gh-aw/weekly-issues-data/issues.json

# Get only closed issues
jq '[.[] | select(.state == "CLOSED")]' /tmp/gh-aw/weekly-issues-data/issues.json

# Get issue numbers
jq '[.[].number]' /tmp/gh-aw/weekly-issues-data/issues.json

# Get issues with specific label
jq '[.[] | select(.labels | any(.name == "bug"))]' /tmp/gh-aw/weekly-issues-data/issues.json

# Get issues created in the last 3 days
DATE_3_DAYS_AGO=$(date -d '3 days ago' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -v-3d '+%Y-%m-%dT%H:%M:%SZ')
jq --arg date "$DATE_3_DAYS_AGO" '[.[] | select(.createdAt >= $date)]' /tmp/gh-aw/weekly-issues-data/issues.json

# Count issues by state
jq 'group_by(.state) | map({state: .[0].state, count: length})' /tmp/gh-aw/weekly-issues-data/issues.json

# Get unique authors
jq '[.[].author.login] | unique' /tmp/gh-aw/weekly-issues-data/issues.json

# Get issues sorted by update time (most recent first)
jq 'sort_by(.updatedAt) | reverse' /tmp/gh-aw/weekly-issues-data/issues.json
```
