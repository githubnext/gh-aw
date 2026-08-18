---
tools:
  cache-memory:
    key: copilot-pr-data
  bash:
    - "jq *"
    - "./.github/skills/jqschema/jqschema.sh"
    - "mkdir *"
    - "date *"
    - "cp *"
    - "ln *"

steps:
  - name: Install gh CLI
    run: |
      bash "${RUNNER_TEMP}/gh-aw/actions/install_gh_cli.sh"

  - name: Fetch Copilot PR data
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Create output directories
      mkdir -p /tmp/gh-aw/agent/pr-data
      mkdir -p /tmp/gh-aw/cache-memory
      
      # Cache files use stable names so restored caches from previous runs are reusable
      # regardless of the date they were written on. Freshness is decided by file age.
      CACHE_DIR="/tmp/gh-aw/cache-memory"
      CACHE_FILE="$CACHE_DIR/copilot-prs-latest.json"
      CACHE_SCHEMA_FILE="$CACHE_DIR/copilot-prs-latest-schema.json"
      CACHE_MAX_AGE_SECONDS="${CACHE_MAX_AGE_SECONDS:-21600}"

      # Returns success when the file exists, is non-empty and younger than the max age.
      cache_is_fresh() {
        cache_path="$1"
        [ -s "$cache_path" ] || return 1
        cache_mtime=$(date -r "$cache_path" '+%s' 2>/dev/null) || return 1
        [ -n "$cache_mtime" ] || return 1
        now=$(date '+%s')
        [ "$((now - cache_mtime))" -lt "$CACHE_MAX_AGE_SECONDS" ]
      }

      # Check if restored cache data is still fresh enough to reuse
      if cache_is_fresh "$CACHE_FILE"; then
        CACHE_AGE_MINUTES=$(( ( $(date '+%s') - $(date -r "$CACHE_FILE" '+%s') ) / 60 ))
        echo "✓ Found cached PR data (${CACHE_AGE_MINUTES} minutes old)"
        cp "$CACHE_FILE" /tmp/gh-aw/agent/pr-data/copilot-prs.json
        
        # Regenerate schema if missing
        if [ ! -s "$CACHE_SCHEMA_FILE" ]; then
          ./.github/skills/jqschema/jqschema.sh < /tmp/gh-aw/agent/pr-data/copilot-prs.json > "$CACHE_SCHEMA_FILE"
        fi
        cp "$CACHE_SCHEMA_FILE" /tmp/gh-aw/agent/pr-data/copilot-prs-schema.json
        
        echo "Using cached data written ${CACHE_AGE_MINUTES} minutes ago"
        echo "Total PRs in cache: $(jq 'length' /tmp/gh-aw/agent/pr-data/copilot-prs.json)"
      else
        echo "⬇ Downloading fresh PR data..."
        
        # Calculate date 30 days ago
        DATE_30_DAYS_AGO=$(date -d '30 days ago' '+%Y-%m-%d' 2>/dev/null || date -v-30d '+%Y-%m-%d')

        # Search for PRs from copilot/* branches in the last 30 days using gh CLI
        # Using branch prefix search (head:copilot/) instead of author for reliability
        echo "Fetching Copilot PRs from the last 30 days..."
        gh pr list --repo "$GITHUB_REPOSITORY" \
          --search "head:copilot/ created:>=${DATE_30_DAYS_AGO}" \
          --state all \
          --json number,title,author,headRefName,createdAt,state,url,body,labels,updatedAt,closedAt,mergedAt \
          --limit 1000 \
          > /tmp/gh-aw/agent/pr-data/copilot-prs.json

        # Generate schema for reference
        ./.github/skills/jqschema/jqschema.sh < /tmp/gh-aw/agent/pr-data/copilot-prs.json > /tmp/gh-aw/agent/pr-data/copilot-prs-schema.json

        # Store in cache under stable names so future runs can reuse it on any date
        cp /tmp/gh-aw/agent/pr-data/copilot-prs.json "$CACHE_FILE"
        cp /tmp/gh-aw/agent/pr-data/copilot-prs-schema.json "$CACHE_SCHEMA_FILE"

        # Drop legacy date-keyed cache entries so the cache does not grow unbounded
        rm -f "$CACHE_DIR"/copilot-prs-[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9].json \
              "$CACHE_DIR"/copilot-prs-[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]-schema.json 2>/dev/null || true

        echo "✓ PR data saved to cache: $(basename "$CACHE_FILE")"
        echo "Total PRs found: $(jq 'length' /tmp/gh-aw/agent/pr-data/copilot-prs.json)"
      fi
      
      # Always ensure data is available at expected locations for backward compatibility
      echo "PR data available at: /tmp/gh-aw/agent/pr-data/copilot-prs.json"
      echo "Schema available at: /tmp/gh-aw/agent/pr-data/copilot-prs-schema.json"
---

<!--
## Copilot PR Data Fetch

This shared component fetches pull request data for GitHub Copilot coding agent-created PRs from the last 30 days, with intelligent caching to avoid redundant API calls.

### What It Does

1. Creates output directories at `/tmp/gh-aw/agent/pr-data/` and `/tmp/gh-aw/cache-memory/`
2. Checks the restored cache for `copilot-prs-latest.json` and reuses it when it is younger than `CACHE_MAX_AGE_SECONDS` (default 6 hours)
3. If a fresh cache exists (written by any earlier run, on any date):
   - Uses cached data instead of making API calls
   - Copies data from cache to working directory
4. If the cache is missing, empty, or stale:
   - Calculates the date 30 days ago (cross-platform compatible)
   - Fetches all PRs from branches starting with `copilot/` using `gh pr list`
   - Saves data to cache under the stable filename `copilot-prs-latest.json`
   - Copies data to working directory for use
5. Generates a schema of the data structure

### Caching Strategy

- **Cache Key**: `copilot-pr-data` for workflow-level sharing
- **Cache Files**: Stored under stable, date-independent filenames (`copilot-prs-latest.json`)
- **Cache Freshness**: File modification time is compared against `CACHE_MAX_AGE_SECONDS` (default `21600`, i.e. 6 hours); stale data is refetched
- **Cache Location**: `/tmp/gh-aw/cache-memory/`
- **Cache Benefits**: 
  - Cache restored from a previous run is reusable regardless of the date it was written on, so scheduled workflows running daily or weekly actually hit the cache instead of always refetching
  - Reduces GitHub API rate limit usage
  - Faster workflow execution after the first fetch within the freshness window

### Output Files

- **`/tmp/gh-aw/agent/pr-data/copilot-prs.json`**: Full PR data including number, title, author, branch name, timestamps, state, URL, body, labels, etc.
- **`/tmp/gh-aw/agent/pr-data/copilot-prs-schema.json`**: JSON schema showing the structure of the PR data
- **`/tmp/gh-aw/cache-memory/copilot-prs-latest.json`**: Cached PR data (stable filename)
- **`/tmp/gh-aw/cache-memory/copilot-prs-latest-schema.json`**: Cached schema (stable filename)

### Usage

Import this component in your workflow:

```yaml
imports:
  - shared/copilot-pr-data-fetch.md
  - ../skills/jqschema/SKILL.md  # Required for schema generation
```

Then access the pre-fetched data in your workflow prompt:

```bash
# Get PRs from the last 24 hours
TODAY="$(date -d '24 hours ago' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -v-24H '+%Y-%m-%dT%H:%M:%SZ')"
jq --arg today "$TODAY" '[.[] | select(.createdAt >= $today)]' /tmp/gh-aw/agent/pr-data/copilot-prs.json

# Count total PRs
jq 'length' /tmp/gh-aw/agent/pr-data/copilot-prs.json

# Get PR numbers
jq '[.[].number]' /tmp/gh-aw/agent/pr-data/copilot-prs.json
```

### Requirements

- Requires the `jqschema` skill to be imported for schema generation
- Uses `gh pr list` with the `--search "head:copilot/"` pattern for reliable Copilot PR detection
- Cross-platform date calculation (works on both GNU and BSD date commands)
- Cache-memory tool is automatically configured for data persistence

### Why Branch-Based Search?

GitHub Copilot creates branches with the `copilot/` prefix, making branch-based search more reliable than author-based search which may miss PRs due to author name variations.

### Cache Behavior

The cache is age-based, meaning:
- All workflows sharing the cache reuse data that is younger than the freshness window (6 hours by default)
- Cache refreshes automatically once the cached file exceeds that age, independent of calendar date boundaries
- The first workflow after the data goes stale fetches fresh data and repopulates the cache
- Subsequent workflows use the cached data for faster execution
-->
