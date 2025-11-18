---
tools:
  cache-memory:
    key: copilot-pr-data
  bash:
    - "gh pr list *"
    - "gh api *"
    - "jq *"
    - "/tmp/gh-aw/jqschema.sh"
    - "mkdir *"
    - "date *"
    - "cp *"
    - "ln *"

steps:
  - name: Fetch Copilot PR data
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Create output directories
      mkdir -p /tmp/gh-aw/pr-data
      mkdir -p /tmp/gh-aw/cache-memory
      
      # Get today's date for cache identification
      TODAY=$(date '+%Y-%m-%d')
      CACHE_DIR="/tmp/gh-aw/cache-memory"
      
      # Check if cached data exists from today
      if [ -f "$CACHE_DIR/copilot-prs-${TODAY}.json" ] && [ -s "$CACHE_DIR/copilot-prs-${TODAY}.json" ]; then
        echo "✓ Found cached PR data from ${TODAY}"
        cp "$CACHE_DIR/copilot-prs-${TODAY}.json" /tmp/gh-aw/pr-data/copilot-prs.json
        
        # Regenerate schema if missing
        if [ ! -f "$CACHE_DIR/copilot-prs-${TODAY}-schema.json" ]; then
          /tmp/gh-aw/jqschema.sh < /tmp/gh-aw/pr-data/copilot-prs.json > "$CACHE_DIR/copilot-prs-${TODAY}-schema.json"
        fi
        cp "$CACHE_DIR/copilot-prs-${TODAY}-schema.json" /tmp/gh-aw/pr-data/copilot-prs-schema.json
        
        echo "Using cached data from ${TODAY}"
        echo "Total PRs in cache: $(jq 'length' /tmp/gh-aw/pr-data/copilot-prs.json)"
      else
        echo "⬇ Downloading fresh PR data..."
        
        # Calculate date 30 days ago
        DATE_30_DAYS_AGO=$(date -d '30 days ago' '+%Y-%m-%d' 2>/dev/null || date -v-30d '+%Y-%m-%d')

        # Search for PRs from copilot/* branches in the last 30 days using gh CLI
        # Using branch prefix search (head:copilot/) instead of author for reliability
        echo "Fetching Copilot PRs from the last 30 days..."
        gh pr list --repo ${{ github.repository }} \
          --search "head:copilot/ created:>=${DATE_30_DAYS_AGO}" \
          --state all \
          --json number,title,author,headRefName,createdAt,state,url,body,labels,updatedAt,closedAt,mergedAt \
          --limit 1000 \
          > /tmp/gh-aw/pr-data/copilot-prs.json

        # Generate schema for reference
        /tmp/gh-aw/jqschema.sh < /tmp/gh-aw/pr-data/copilot-prs.json > /tmp/gh-aw/pr-data/copilot-prs-schema.json

        # Store in cache with today's date
        cp /tmp/gh-aw/pr-data/copilot-prs.json "$CACHE_DIR/copilot-prs-${TODAY}.json"
        cp /tmp/gh-aw/pr-data/copilot-prs-schema.json "$CACHE_DIR/copilot-prs-${TODAY}-schema.json"

        echo "✓ PR data saved to cache: copilot-prs-${TODAY}.json"
        echo "Total PRs found: $(jq 'length' /tmp/gh-aw/pr-data/copilot-prs.json)"
      fi
      
      # Always ensure data is available at expected locations for backward compatibility
      echo "PR data available at: /tmp/gh-aw/pr-data/copilot-prs.json"
      echo "Schema available at: /tmp/gh-aw/pr-data/copilot-prs-schema.json"
---

<!--
## Copilot PR Data Fetch

This shared component fetches pull request data for GitHub Copilot agent-created PRs from the last 30 days, with intelligent caching to avoid redundant API calls.

### What It Does

1. Creates output directories at `/tmp/gh-aw/pr-data/` and `/tmp/gh-aw/cache-memory/`
2. Checks for cached PR data from today's date in cache-memory
3. If cache exists (from earlier workflow runs today):
   - Uses cached data instead of making API calls
   - Copies data from cache to working directory
4. If cache doesn't exist:
   - Calculates the date 30 days ago (cross-platform compatible)
   - Fetches all PRs from branches starting with `copilot/` using `gh pr list`
   - Saves data to cache with date-based filename (e.g., `copilot-prs-2024-11-18.json`)
   - Copies data to working directory for use
5. Generates a schema of the data structure

### Caching Strategy

- **Cache Key**: `copilot-pr-data` for workflow-level sharing
- **Cache Files**: Stored with today's date in the filename (e.g., `copilot-prs-2024-11-18.json`)
- **Cache Location**: `/tmp/gh-aw/cache-memory/`
- **Cache Benefits**: 
  - Multiple workflows running on the same day share the same PR data
  - Reduces GitHub API rate limit usage
  - Faster workflow execution after first fetch of the day

### Output Files

- **`/tmp/gh-aw/pr-data/copilot-prs.json`**: Full PR data including number, title, author, branch name, timestamps, state, URL, body, labels, etc.
- **`/tmp/gh-aw/pr-data/copilot-prs-schema.json`**: JSON schema showing the structure of the PR data
- **`/tmp/gh-aw/cache-memory/copilot-prs-YYYY-MM-DD.json`**: Cached PR data with date
- **`/tmp/gh-aw/cache-memory/copilot-prs-YYYY-MM-DD-schema.json`**: Cached schema with date

### Usage

Import this component in your workflow:

```yaml
imports:
  - shared/copilot-pr-data-fetch.md
  - shared/jqschema.md  # Required for schema generation
```

Then access the pre-fetched data in your workflow prompt:

```bash
# Get PRs from the last 24 hours
TODAY="$(date -d '24 hours ago' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -v-24H '+%Y-%m-%dT%H:%M:%SZ')"
jq --arg today "$TODAY" '[.[] | select(.createdAt >= $today)]' /tmp/gh-aw/pr-data/copilot-prs.json

# Count total PRs
jq 'length' /tmp/gh-aw/pr-data/copilot-prs.json

# Get PR numbers
jq '[.[].number]' /tmp/gh-aw/pr-data/copilot-prs.json
```

### Requirements

- Requires `jqschema.md` to be imported for schema generation
- Uses `gh pr list` with the `--search "head:copilot/"` pattern for reliable Copilot PR detection
- Cross-platform date calculation (works on both GNU and BSD date commands)
- Cache-memory tool is automatically configured for data persistence

### Why Branch-Based Search?

GitHub Copilot creates branches with the `copilot/` prefix, making branch-based search more reliable than author-based search which may miss PRs due to author name variations.

### Cache Behavior

The cache is date-based, meaning:
- All workflows running on the same day share cached data
- Cache refreshes automatically the next day
- First workflow of the day fetches fresh data and populates the cache
- Subsequent workflows use the cached data for faster execution
-->
