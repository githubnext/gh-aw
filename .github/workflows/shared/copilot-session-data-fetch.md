---
# This shared component depends on jqschema.md being imported first.
#
# NOTE: Due to BFS import ordering, transitive imports are not guaranteed to have their
# steps executed before the parent import's steps. To ensure correct execution order,
# import jqschema.md directly in your workflow BEFORE importing this file:
#
#   imports:
#     - shared/jqschema.md  # Must come first
#     - shared/copilot-session-data-fetch.md
#
imports:
  - shared/jqschema.md

tools:
  cache-memory:
    key: copilot-session-data
  bash:
    - "gh api *"
    - "jq *"
    - "/tmp/gh-aw/jqschema.sh"
    - "mkdir *"
    - "date *"
    - "cp *"
    - "unzip *"
    - "find *"
    - "rm *"

steps:
  - name: Fetch Copilot session data
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Create output directories
      mkdir -p /tmp/gh-aw/session-data
      mkdir -p /tmp/gh-aw/session-data/logs
      mkdir -p /tmp/gh-aw/cache-memory
      
      # Get today's date for cache identification
      TODAY=$(date '+%Y-%m-%d')
      CACHE_DIR="/tmp/gh-aw/cache-memory"
      
      # Check if cached data exists from today
      if [ -f "$CACHE_DIR/copilot-sessions-${TODAY}.json" ] && [ -s "$CACHE_DIR/copilot-sessions-${TODAY}.json" ]; then
        echo "✓ Found cached session data from ${TODAY}"
        cp "$CACHE_DIR/copilot-sessions-${TODAY}.json" /tmp/gh-aw/session-data/sessions-list.json
        
        # Regenerate schema if missing
        if [ ! -f "$CACHE_DIR/copilot-sessions-${TODAY}-schema.json" ]; then
          /tmp/gh-aw/jqschema.sh < /tmp/gh-aw/session-data/sessions-list.json > "$CACHE_DIR/copilot-sessions-${TODAY}-schema.json"
        fi
        cp "$CACHE_DIR/copilot-sessions-${TODAY}-schema.json" /tmp/gh-aw/session-data/sessions-schema.json
        
        # Restore cached log files if they exist
        if [ -d "$CACHE_DIR/session-logs-${TODAY}" ]; then
          echo "✓ Found cached session logs from ${TODAY}"
          cp -r "$CACHE_DIR/session-logs-${TODAY}"/* /tmp/gh-aw/session-data/logs/ 2>/dev/null || true
          echo "Restored $(find /tmp/gh-aw/session-data/logs -type f | wc -l) session log files from cache"
        fi
        
        echo "Using cached data from ${TODAY}"
        echo "Total sessions in cache: $(jq 'length' /tmp/gh-aw/session-data/sessions-list.json)"
      else
        echo "⬇ Downloading fresh session data..."
        
        # Calculate date 30 days ago
        DATE_30_DAYS_AGO=$(date -d '30 days ago' '+%Y-%m-%d' 2>/dev/null || date -v-30d '+%Y-%m-%d')

        # Search for workflow runs from copilot/* branches
        # This fetches GitHub Copilot agent task runs by searching for workflow runs on copilot/* branches
        echo "Fetching Copilot agent workflow runs from the last 30 days..."
        
        # Get workflow runs from copilot/* branches
        gh api "repos/${{ github.repository }}/actions/runs" \
          --paginate \
          --jq ".workflow_runs[] | select(.head_branch | startswith(\"copilot/\")) | select(.created_at >= \"${DATE_30_DAYS_AGO}\") | {id, name, head_branch, created_at, updated_at, status, conclusion, html_url}" \
          | jq -s '.[0:50]' \
          > /tmp/gh-aw/session-data/sessions-list.json

        # Generate schema for reference
        /tmp/gh-aw/jqschema.sh < /tmp/gh-aw/session-data/sessions-list.json > /tmp/gh-aw/session-data/sessions-schema.json

        # Download logs for each workflow run (limit to first 50)
        SESSION_COUNT=$(jq 'length' /tmp/gh-aw/session-data/sessions-list.json)
        echo "Downloading logs for $SESSION_COUNT sessions..."
        
        jq -r '.[].id' /tmp/gh-aw/session-data/sessions-list.json | while read -r run_id; do
          if [ -n "$run_id" ]; then
            echo "Downloading logs for run: $run_id"
            # Download workflow run logs using GitHub API
            gh api "repos/${{ github.repository }}/actions/runs/${run_id}/logs" \
              > "/tmp/gh-aw/session-data/logs/${run_id}.zip" 2>&1 || true
            
            # Extract the logs if download succeeded
            if [ -f "/tmp/gh-aw/session-data/logs/${run_id}.zip" ] && [ -s "/tmp/gh-aw/session-data/logs/${run_id}.zip" ]; then
              unzip -q "/tmp/gh-aw/session-data/logs/${run_id}.zip" -d "/tmp/gh-aw/session-data/logs/${run_id}/" 2>/dev/null || true
              rm "/tmp/gh-aw/session-data/logs/${run_id}.zip"
            fi
          fi
        done
        
        LOG_COUNT=$(find /tmp/gh-aw/session-data/logs/ -type d -mindepth 1 | wc -l)
        echo "Session logs downloaded: $LOG_COUNT log directories"

        # Store in cache with today's date
        cp /tmp/gh-aw/session-data/sessions-list.json "$CACHE_DIR/copilot-sessions-${TODAY}.json"
        cp /tmp/gh-aw/session-data/sessions-schema.json "$CACHE_DIR/copilot-sessions-${TODAY}-schema.json"
        
        # Cache the log files
        mkdir -p "$CACHE_DIR/session-logs-${TODAY}"
        cp -r /tmp/gh-aw/session-data/logs/* "$CACHE_DIR/session-logs-${TODAY}/" 2>/dev/null || true

        echo "✓ Session data saved to cache: copilot-sessions-${TODAY}.json"
        echo "Total sessions found: $(jq 'length' /tmp/gh-aw/session-data/sessions-list.json)"
      fi
      
      # Always ensure data is available at expected locations for backward compatibility
      echo "Session data available at: /tmp/gh-aw/session-data/sessions-list.json"
      echo "Schema available at: /tmp/gh-aw/session-data/sessions-schema.json"
      echo "Logs available at: /tmp/gh-aw/session-data/logs/"
      
      # Set outputs for downstream use
      echo "sessions_count=$(jq 'length' /tmp/gh-aw/session-data/sessions-list.json)" >> "$GITHUB_OUTPUT"
---

<!--
## Copilot Session Data Fetch

This shared component fetches GitHub Copilot agent session data by analyzing workflow runs from `copilot/*` branches, with intelligent caching to avoid redundant API calls.

### What It Does

1. Creates output directories at `/tmp/gh-aw/session-data/` and `/tmp/gh-aw/cache-memory/`
2. Checks for cached session data from today's date in cache-memory
3. If cache exists (from earlier workflow runs today):
   - Uses cached data instead of making API calls
   - Copies data from cache to working directory
   - Restores cached log files if available
4. If cache doesn't exist:
   - Calculates the date 30 days ago (cross-platform compatible)
   - Fetches all workflow runs from branches starting with `copilot/` using GitHub API
   - Downloads logs for up to 50 most recent runs
   - Extracts and organizes log files
   - Saves data to cache with date-based filename (e.g., `copilot-sessions-2024-11-22.json`)
   - Copies data to working directory for use
5. Generates a schema of the data structure

### Caching Strategy

- **Cache Key**: `copilot-session-data` for workflow-level sharing
- **Cache Files**: Stored with today's date in the filename (e.g., `copilot-sessions-2024-11-22.json`)
- **Cache Location**: `/tmp/gh-aw/cache-memory/`
- **Cache Benefits**: 
  - Multiple workflows running on the same day share the same session data
  - Reduces GitHub API rate limit usage
  - Faster workflow execution after first fetch of the day
  - Avoids need for `gh agent-task` extension

### Output Files

- **`/tmp/gh-aw/session-data/sessions-list.json`**: Full session data including run ID, name, branch, timestamps, status, conclusion, and URL
- **`/tmp/gh-aw/session-data/sessions-schema.json`**: JSON schema showing the structure of the session data
- **`/tmp/gh-aw/session-data/logs/`**: Directory containing extracted workflow run logs
- **`/tmp/gh-aw/cache-memory/copilot-sessions-YYYY-MM-DD.json`**: Cached session data with date
- **`/tmp/gh-aw/cache-memory/copilot-sessions-YYYY-MM-DD-schema.json`**: Cached schema with date
- **`/tmp/gh-aw/cache-memory/session-logs-YYYY-MM-DD/`**: Cached log files with date

### Usage

Import this component in your workflow:

```yaml
imports:
  - shared/copilot-session-data-fetch.md
```

**Note**: This component automatically imports `jqschema.md` as a dependency. The compiler handles the transitive closure of imports, ensuring all required utilities are set up in the correct order.

Then access the pre-fetched data in your workflow prompt:

```bash
# Get sessions from the last 24 hours
TODAY="$(date -d '24 hours ago' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -v-24H '+%Y-%m-%dT%H:%M:%SZ')"
jq --arg today "$TODAY" '[.[] | select(.created_at >= $today)]' /tmp/gh-aw/session-data/sessions-list.json

# Count total sessions
jq 'length' /tmp/gh-aw/session-data/sessions-list.json

# Get run IDs
jq '[.[].id]' /tmp/gh-aw/session-data/sessions-list.json

# List log directories
find /tmp/gh-aw/session-data/logs -type d -mindepth 1
```

### Requirements

- Automatically imports `jqschema.md` for schema generation (via transitive import closure)
- Uses GitHub Actions API to fetch workflow runs from `copilot/*` branches
- Cross-platform date calculation (works on both GNU and BSD date commands)
- Cache-memory tool is automatically configured for data persistence

### Why Branch-Based Search?

GitHub Copilot creates branches with the `copilot/` prefix, making branch-based workflow run search a reliable way to identify Copilot agent sessions without requiring the `gh agent-task` extension.

### Advantages Over gh agent-task Extension

- **No Extension Required**: Works without installing `gh agent-task` CLI extension
- **Better Caching**: Leverages cache-memory for efficient data reuse
- **API-Based**: Uses standard GitHub API endpoints accessible to all users
- **Broader Access**: Works in all GitHub environments, not just Enterprise with Copilot

### Cache Behavior

The cache is date-based, meaning:
- All workflows running on the same day share cached data
- Cache refreshes automatically the next day
- First workflow of the day fetches fresh data and populates the cache
- Subsequent workflows use the cached data for faster execution
-->
