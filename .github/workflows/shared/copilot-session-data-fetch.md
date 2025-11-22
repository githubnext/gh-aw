---
tools:
  cache-memory:
    key: copilot-session-data
  bash:
    - "gh api *"
    - "gh run download *"
    - "jq *"
    - "/tmp/gh-aw/jqschema.sh"
    - "mkdir *"
    - "date *"
    - "cp *"
    - "find *"
    - "unzip *"

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
        
        # Restore cached logs if available
        if [ -d "$CACHE_DIR/session-logs-${TODAY}" ]; then
          cp -r "$CACHE_DIR/session-logs-${TODAY}"/* /tmp/gh-aw/session-data/logs/ 2>/dev/null || true
          echo "✓ Restored cached session logs"
        fi
        
        # Regenerate schema if missing
        if [ ! -f "$CACHE_DIR/copilot-sessions-${TODAY}-schema.json" ]; then
          /tmp/gh-aw/jqschema.sh < /tmp/gh-aw/session-data/sessions-list.json > "$CACHE_DIR/copilot-sessions-${TODAY}-schema.json"
        fi
        cp "$CACHE_DIR/copilot-sessions-${TODAY}-schema.json" /tmp/gh-aw/session-data/sessions-schema.json
        
        echo "Using cached data from ${TODAY}"
        echo "Total sessions in cache: $(jq 'length' /tmp/gh-aw/session-data/sessions-list.json)"
      else
        echo "⬇ Downloading fresh session data..."
        
        # Calculate date 30 days ago (cross-platform compatible)
        DATE_30_DAYS_AGO=$(date -d '30 days ago' '+%Y-%m-%d' 2>/dev/null || date -v-30d '+%Y-%m-%d')

        # Search for workflow runs from copilot/* branches in the last 30 days
        echo "Fetching Copilot workflow runs from the last 30 days..."
        gh api "repos/${{ github.repository }}/actions/runs" \
          --paginate \
          --jq ".workflow_runs[] | select(.head_branch | startswith(\"copilot/\")) | select(.created_at >= \"${DATE_30_DAYS_AGO}\") | {id: .id, name: .name, head_branch: .head_branch, status: .status, conclusion: .conclusion, created_at: .created_at, updated_at: .updated_at, run_number: .run_number, event: .event, html_url: .html_url}" \
          | jq -s '.' \
          > /tmp/gh-aw/session-data/sessions-list.json

        # Generate schema for reference
        /tmp/gh-aw/jqschema.sh < /tmp/gh-aw/session-data/sessions-list.json > /tmp/gh-aw/session-data/sessions-schema.json

        # Download logs for each session (limit to first 50 for performance)
        echo "Downloading session logs..."
        TOTAL_SESSIONS=$(jq 'length' /tmp/gh-aw/session-data/sessions-list.json)
        echo "Total sessions found: $TOTAL_SESSIONS"
        
        if [ "$TOTAL_SESSIONS" -gt 0 ]; then
          jq -r '.[].id' /tmp/gh-aw/session-data/sessions-list.json | head -50 | while read -r run_id; do
            if [ -n "$run_id" ]; then
              echo "Downloading logs for run: $run_id"
              
              # Download logs as zip file
              if gh api "repos/${{ github.repository }}/actions/runs/${run_id}/logs" > "/tmp/gh-aw/session-data/logs/${run_id}.zip" 2>/dev/null; then
                # Extract the zip file
                unzip -q -o "/tmp/gh-aw/session-data/logs/${run_id}.zip" -d "/tmp/gh-aw/session-data/logs/${run_id}/" 2>/dev/null || true
                # Remove the zip file to save space
                rm -f "/tmp/gh-aw/session-data/logs/${run_id}.zip"
              else
                echo "Failed to download logs for run: $run_id"
              fi
            fi
          done
          
          LOG_COUNT=$(find /tmp/gh-aw/session-data/logs/ -maxdepth 1 -type d | wc -l)
          echo "Session logs downloaded to /tmp/gh-aw/session-data/logs/"
          echo "Total log directories: $((LOG_COUNT - 1))"
        fi

        # Store in cache with today's date
        cp /tmp/gh-aw/session-data/sessions-list.json "$CACHE_DIR/copilot-sessions-${TODAY}.json"
        cp /tmp/gh-aw/session-data/sessions-schema.json "$CACHE_DIR/copilot-sessions-${TODAY}-schema.json"
        
        # Cache the logs directory
        if [ -d /tmp/gh-aw/session-data/logs ]; then
          mkdir -p "$CACHE_DIR/session-logs-${TODAY}"
          cp -r /tmp/gh-aw/session-data/logs/* "$CACHE_DIR/session-logs-${TODAY}/" 2>/dev/null || true
          echo "✓ Session logs cached"
        fi

        echo "✓ Session data saved to cache: copilot-sessions-${TODAY}.json"
        echo "Total sessions found: $(jq 'length' /tmp/gh-aw/session-data/sessions-list.json)"
      fi
      
      # Always ensure data is available at expected locations
      echo "Session data available at: /tmp/gh-aw/session-data/sessions-list.json"
      echo "Schema available at: /tmp/gh-aw/session-data/sessions-schema.json"
      echo "Logs available at: /tmp/gh-aw/session-data/logs/"
---

<!--
## Copilot Session Data Fetch

This shared component fetches workflow run data for GitHub Copilot agent sessions from the last 30 days, with intelligent caching to avoid redundant API calls.

### What It Does

1. Creates output directories at `/tmp/gh-aw/session-data/` and `/tmp/gh-aw/cache-memory/`
2. Checks for cached session data from today's date in cache-memory
3. If cache exists (from earlier workflow runs today):
   - Uses cached data instead of making API calls
   - Restores session list and logs from cache
   - Copies data from cache to working directory
4. If cache doesn't exist:
   - Calculates the date 30 days ago (cross-platform compatible)
   - Fetches all workflow runs from branches starting with `copilot/` using GitHub API
   - Downloads logs for up to 50 most recent sessions
   - Extracts log zip files into organized directories
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
  - Logs are cached to avoid re-downloading

### Output Files

- **`/tmp/gh-aw/session-data/sessions-list.json`**: Full workflow run data including id, name, branch, status, conclusion, timestamps, URLs, etc.
- **`/tmp/gh-aw/session-data/sessions-schema.json`**: JSON schema showing the structure of the session data
- **`/tmp/gh-aw/session-data/logs/`**: Directory containing extracted log files organized by run ID
- **`/tmp/gh-aw/cache-memory/copilot-sessions-YYYY-MM-DD.json`**: Cached session data with date
- **`/tmp/gh-aw/cache-memory/copilot-sessions-YYYY-MM-DD-schema.json`**: Cached schema with date
- **`/tmp/gh-aw/cache-memory/session-logs-YYYY-MM-DD/`**: Cached log files with date

### Usage

Import this component in your workflow:

```yaml
imports:
  - shared/copilot-session-data-fetch.md
  - shared/jqschema.md  # Required for schema generation
```

Then access the pre-fetched data in your workflow prompt:

```bash
# Count total sessions
jq 'length' /tmp/gh-aw/session-data/sessions-list.json

# Get session IDs
jq '[.[].id]' /tmp/gh-aw/session-data/sessions-list.json

# Filter by status
jq '[.[] | select(.status == "completed")]' /tmp/gh-aw/session-data/sessions-list.json

# Access logs for a specific run
ls /tmp/gh-aw/session-data/logs/<run_id>/
```

### Requirements

- Requires `jqschema.md` to be imported for schema generation
- Uses GitHub API `repos/{owner}/{repo}/actions/runs` endpoint
- Cross-platform date calculation (works on both GNU and BSD date commands)
- Cache-memory tool is automatically configured for data persistence

### Why Branch-Based Search?

GitHub Copilot creates branches with the `copilot/` prefix for agent tasks, making branch-based workflow run search reliable for identifying Copilot sessions without requiring specialized CLI extensions.

### Cache Behavior

The cache is date-based, meaning:
- All workflows running on the same day share cached data
- Cache refreshes automatically the next day
- First workflow of the day fetches fresh data and populates the cache
- Subsequent workflows use the cached data for faster execution
- Logs are cached to avoid repeated downloads

### Advantages Over gh agent-task Extension

- **Universal Compatibility**: Works on all GitHub environments (not just Enterprise)
- **Standard Authentication**: Uses regular `GITHUB_TOKEN`, no special PAT required
- **No Installation**: No CLI extension installation needed
- **Simpler Maintenance**: Fewer failure modes and dependencies
- **Better Portability**: Can be used in any GitHub Actions workflow
-->
