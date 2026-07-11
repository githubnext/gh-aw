---
# This shared component depends on the jqschema skill being imported first.
#
# NOTE: Due to BFS import ordering, transitive imports are not guaranteed to have their
# steps executed before the parent import's steps. To ensure correct execution order,
# import the jqschema skill directly in your workflow BEFORE importing this file:
#
#   imports:
#     - ../skills/jqschema/SKILL.md  # Must come first
#     - shared/copilot-session-data-fetch.md
#
imports:
  - ../skills/jqschema/SKILL.md

tools:
  cache-memory:
    key: copilot-session-data
  bash:
    - "jq *"
    - "./.github/skills/jqschema/jqschema.sh"
    - "mkdir *"
    - "date *"
    - "cp *"
    - "unzip *"
    - "find *"
    - "rm *"
    - "cat *"

steps:
  - name: Install gh CLI
    run: |
      bash "${RUNNER_TEMP}/gh-aw/actions/install_gh_cli.sh"

  - name: Fetch Copilot session data
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Create output directories
      mkdir -p /tmp/gh-aw/agent/session-data
      mkdir -p /tmp/gh-aw/agent/session-data/logs
      mkdir -p /tmp/gh-aw/cache-memory
      
      # Get today's date for cache identification
      TODAY=$(date '+%Y-%m-%d')
      CACHE_DIR="/tmp/gh-aw/cache-memory"
      
      # Check if cached data exists from today
      if [ -f "$CACHE_DIR/copilot-sessions-${TODAY}.json" ] && [ -s "$CACHE_DIR/copilot-sessions-${TODAY}.json" ]; then
        echo "✓ Found cached session data from ${TODAY}"
        cp "$CACHE_DIR/copilot-sessions-${TODAY}.json" /tmp/gh-aw/agent/session-data/sessions-list.json
        
        # Regenerate schema if missing
        if [ ! -f "$CACHE_DIR/copilot-sessions-${TODAY}-schema.json" ]; then
          ./.github/skills/jqschema/jqschema.sh < /tmp/gh-aw/agent/session-data/sessions-list.json > "$CACHE_DIR/copilot-sessions-${TODAY}-schema.json"
        fi
        cp "$CACHE_DIR/copilot-sessions-${TODAY}-schema.json" /tmp/gh-aw/agent/session-data/sessions-schema.json
        
        # Restore cached log files if they exist
        if [ -d "$CACHE_DIR/session-logs-${TODAY}" ]; then
          echo "✓ Found cached session logs from ${TODAY}"
          cp -r "$CACHE_DIR/session-logs-${TODAY}"/* /tmp/gh-aw/agent/session-data/logs/ 2>/dev/null || true
          echo "Restored $(find /tmp/gh-aw/agent/session-data/logs -type f | wc -l) session log files from cache"
        fi
        
        echo "Using cached data from ${TODAY}"
        echo "Total sessions in cache: $(jq 'length' /tmp/gh-aw/agent/session-data/sessions-list.json)"
      else
        echo "⬇ Downloading fresh session data..."
        
        # Calculate date 30 days ago
        DATE_30_DAYS_AGO=$(date -d '30 days ago' '+%Y-%m-%d' 2>/dev/null || date -v-30d '+%Y-%m-%d')

        # Search for workflow runs from copilot/* branches
        # This fetches GitHub Copilot coding agent task runs by searching for workflow runs on copilot/* branches
        echo "Fetching Copilot coding agent workflow runs from the last 30 days..."
        
        # Get workflow runs from copilot/* branches
        gh api "repos/$GITHUB_REPOSITORY/actions/runs" \
          --paginate \
          --jq ".workflow_runs[] | select(.head_branch | startswith(\"copilot/\")) | select(.created_at >= \"${DATE_30_DAYS_AGO}\") | {id, name, head_branch, created_at, updated_at, status, conclusion, html_url}" \
          | jq -s '.[0:50]' \
          > /tmp/gh-aw/agent/session-data/sessions-list.json

        # Generate schema for reference
        ./.github/skills/jqschema/jqschema.sh < /tmp/gh-aw/agent/session-data/sessions-list.json > /tmp/gh-aw/agent/session-data/sessions-schema.json

        # Download conversation logs for actual Copilot agent runs.
        # CI gate runs (e.g. "Smoke CI", "CGO", "CWI" quality gates) always end with
        # conclusion=action_required because a human must approve them to continue; they
        # contain no Copilot agent activity and have no conversation transcript.
        # Each real agent run (conclusion=success/failure) emits [cca-engine] turn= lines
        # in its job log which is the per-turn conversation transcript.
        SESSION_COUNT=$(jq 'length' /tmp/gh-aw/agent/session-data/sessions-list.json)
        AGENT_COUNT=$(jq '[.[] | select(.conclusion != "action_required")] | length' /tmp/gh-aw/agent/session-data/sessions-list.json)
        echo "Downloading conversation logs for $AGENT_COUNT agent runs (skipping $((SESSION_COUNT - AGENT_COUNT)) CI gate runs)..."

        jq -r '.[] | select(.conclusion != "action_required") | "\(.id) \(.head_branch)"' /tmp/gh-aw/agent/session-data/sessions-list.json | while read -r run_id branch; do
          if [ -n "$run_id" ]; then
            echo "Downloading conversation log for run $run_id (branch: $branch)"

            # Get the first job ID for this run via the GitHub API (actions:read suffices).
            # Copilot coding agent runs have exactly one job ("Copilot Coding Agent"), so
            # .jobs[0] is always the correct and only job containing the transcript log.
            job_id=$(gh api "repos/$GITHUB_REPOSITORY/actions/runs/${run_id}/jobs" \
              --jq '.jobs[0].id' 2>/dev/null || true)

            if [ -n "$job_id" ] && [ "$job_id" != "null" ]; then
              # Download the raw job log; gh api follows the 302 redirect automatically
              gh api "repos/$GITHUB_REPOSITORY/actions/jobs/${job_id}/logs" \
                > "/tmp/gh-aw/agent/session-data/logs/${run_id}-raw.log" 2>/dev/null || true

              if [ -f "/tmp/gh-aw/agent/session-data/logs/${run_id}-raw.log" ] && [ -s "/tmp/gh-aw/agent/session-data/logs/${run_id}-raw.log" ]; then
                # Extract conversation transcript: [cca-engine] turn= lines carry turn-by-turn data
                grep "\[cca-engine\] turn=" "/tmp/gh-aw/agent/session-data/logs/${run_id}-raw.log" \
                  > "/tmp/gh-aw/agent/session-data/logs/${run_id}-conversation.txt" 2>/dev/null || true
                rm -f "/tmp/gh-aw/agent/session-data/logs/${run_id}-raw.log"

                if [ -s "/tmp/gh-aw/agent/session-data/logs/${run_id}-conversation.txt" ]; then
                  LINE_COUNT=$(wc -l < "/tmp/gh-aw/agent/session-data/logs/${run_id}-conversation.txt")
                  echo "  Saved transcript: $LINE_COUNT lines for run $run_id"
                else
                  echo "  Warning: No [cca-engine] conversation lines found for run $run_id"
                  rm -f "/tmp/gh-aw/agent/session-data/logs/${run_id}-conversation.txt"
                fi
              else
                echo "  Warning: Could not download job logs for run $run_id (may be expired)"
                rm -f "/tmp/gh-aw/agent/session-data/logs/${run_id}-raw.log" 2>/dev/null || true
              fi
            else
              echo "  Warning: Could not determine job ID for run $run_id"
            fi
          fi
        done

        LOG_COUNT=$(find /tmp/gh-aw/agent/session-data/logs/ -type f -name "*-conversation.txt" | wc -l)
        echo "Conversation logs downloaded: $LOG_COUNT session logs"

        # Store in cache with today's date
        cp /tmp/gh-aw/agent/session-data/sessions-list.json "$CACHE_DIR/copilot-sessions-${TODAY}.json"
        cp /tmp/gh-aw/agent/session-data/sessions-schema.json "$CACHE_DIR/copilot-sessions-${TODAY}-schema.json"
        
        # Cache the log files
        mkdir -p "$CACHE_DIR/session-logs-${TODAY}"
        cp -r /tmp/gh-aw/agent/session-data/logs/* "$CACHE_DIR/session-logs-${TODAY}/" 2>/dev/null || true

        echo "✓ Session data saved to cache: copilot-sessions-${TODAY}.json"
        echo "Total sessions found: $(jq 'length' /tmp/gh-aw/agent/session-data/sessions-list.json)"
      fi
      
      # Always ensure data is available at expected locations for backward compatibility
      echo "Session data available at: /tmp/gh-aw/agent/session-data/sessions-list.json"
      echo "Schema available at: /tmp/gh-aw/agent/session-data/sessions-schema.json"
      echo "Logs available at: /tmp/gh-aw/agent/session-data/logs/"
      
      # Set outputs for downstream use
      echo "sessions_count=$(jq 'length' /tmp/gh-aw/agent/session-data/sessions-list.json)" >> "$GITHUB_OUTPUT"
---

<!--
## Copilot Session Data Fetch

This shared component fetches GitHub Copilot coding agent session data by analyzing workflow runs from `copilot/*` branches, with intelligent caching to avoid redundant API calls.

### What It Does

1. Creates output directories at `/tmp/gh-aw/agent/session-data/` and `/tmp/gh-aw/cache-memory/`
2. Checks for cached session data from today's date in cache-memory
3. If cache exists (from earlier workflow runs today):
   - Uses cached data instead of making API calls
   - Copies data from cache to working directory
   - Restores cached log files if available
4. If cache doesn't exist:
   - Calculates the date 30 days ago (cross-platform compatible)
   - Fetches all workflow runs from branches starting with `copilot/` using GitHub API
   - **Downloads conversation logs** from GitHub Actions job logs for actual agent runs (skips CI gate runs)
   - Saves data to cache with date-based filename (e.g., `copilot-sessions-2024-11-22.json`)
   - Copies data to working directory for use
5. Generates a schema of the data structure

### Conversation Transcript Access

Transcripts are fetched from GitHub Actions job logs using the standard GitHub API (`actions:read` permission). Each real agent run (conclusion ≠ `action_required`) produces `[cca-engine] turn=` log lines that contain the full turn-by-turn conversation — model used, token counts, tool calls and results. CI gate runs (`action_required`) are skipped because they have no agent conversation.

The `gh agent-task view --log` approach that was previously used **requires an OAuth token** that the default `GITHUB_TOKEN` does not provide, and relied on extracting a numeric session ID from the branch name — which stopped working when Copilot switched to descriptive branch slugs (e.g., `copilot/fix-mcp-gateway-docker-daemon-access`).

### Caching Strategy

- **Cache Key**: `copilot-session-data` for workflow-level sharing
- **Cache Files**: Stored with today's date in the filename (e.g., `copilot-sessions-2024-11-22.json`)
- **Cache Location**: `/tmp/gh-aw/cache-memory/`
- **Cache Benefits**: 
  - Multiple workflows running on the same day share the same session data
  - Reduces GitHub API rate limit usage
  - Faster workflow execution after first fetch of the day
  - Includes conversation transcript cache

### Output Files

- **`/tmp/gh-aw/agent/session-data/sessions-list.json`**: Full session data including run ID, name, branch, timestamps, status, conclusion, and URL
- **`/tmp/gh-aw/agent/session-data/sessions-schema.json`**: JSON schema showing the structure of the session data
- **`/tmp/gh-aw/agent/session-data/logs/`**: Directory containing session conversation logs
  - **`{run_id}-conversation.txt`**: Agent conversation transcript — `[cca-engine] turn=` lines from the job log containing turn-by-turn model/token/tool data (only present for actual agent runs)
- **`/tmp/gh-aw/cache-memory/copilot-sessions-YYYY-MM-DD.json`**: Cached session data with date
- **`/tmp/gh-aw/cache-memory/copilot-sessions-YYYY-MM-DD-schema.json`**: Cached schema with date
- **`/tmp/gh-aw/cache-memory/session-logs-YYYY-MM-DD/`**: Cached log files with date

### Usage

Import this component in your workflow:

```yaml
imports:
  - shared/copilot-session-data-fetch.md
```

**Note**: This component automatically imports the `jqschema` skill as a dependency. The compiler handles the transitive closure of imports, ensuring all required utilities are set up in the correct order.

Then access the pre-fetched data in your workflow prompt:

```bash
# Get sessions from the last 24 hours
TODAY="$(date -d '24 hours ago' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -v-24H '+%Y-%m-%dT%H:%M:%SZ')"
jq --arg today "$TODAY" '[.[] | select(.created_at >= $today)]' /tmp/gh-aw/agent/session-data/sessions-list.json

# Count total sessions
jq 'length' /tmp/gh-aw/agent/session-data/sessions-list.json

# Find actual agent runs (not CI gate runs)
jq -r '.[] | select(.conclusion != "action_required") | "\(.id) \(.name)"' /tmp/gh-aw/agent/session-data/sessions-list.json

# List conversation log files (one per actual agent run)
find /tmp/gh-aw/agent/session-data/logs -type f -name "*-conversation.txt"

# Read a specific conversation log (by run ID)
cat /tmp/gh-aw/agent/session-data/logs/29001106791-conversation.txt
```

### Requirements

- Automatically imports the `jqschema` skill for schema generation (via transitive import closure)
- Uses GitHub Actions API to fetch workflow runs from `copilot/*` branches
- **Fetches conversation transcripts from GitHub Actions job logs** using `actions: read` permission (standard `GITHUB_TOKEN` is sufficient)
- Cross-platform date calculation (works on both GNU and BSD date commands)
- Cache-memory tool is automatically configured for data persistence

### Why Branch-Based Search?

GitHub Copilot creates branches with the `copilot/` prefix, making branch-based workflow run search a reliable way to identify Copilot coding agent sessions.

### Conversation Log Format

Transcripts (`{run_id}-conversation.txt`) contain one `[cca-engine] turn=` line per event, for example:

```
2026-07-09T07:24:53.0Z [cca-engine] turn=1 user.message: 4123 chars
2026-07-09T07:25:10.0Z [cca-engine] turn=2 assistant.usage: model=claude-sonnet-4.5 input=12345 output=678
2026-07-09T07:25:10.1Z [cca-engine] turn=2 assistant.message: 312 chars, 1 tool call(s)
2026-07-09T07:25:10.2Z [cca-engine] turn=2 tool.execution_start: edit — /path/to/file.go
2026-07-09T07:25:10.3Z [cca-engine] turn=2 tool.execution_complete: edit success=true
```

Each line carries: timestamp, turn number, event type, and event-specific payload (model, token counts, tool name, file path, success status).

**Benefits for analysis:**
- True behavioral pattern analysis (turn counts, tool sequences, error recovery)
- Token efficiency measurement per turn
- Tool usage effectiveness analysis
- Loop and context-confusion detection from turn patterns

### Cache Behavior

The cache is date-based, meaning:
- All workflows running on the same day share cached data
- Cache refreshes automatically the next day
- First workflow of the day fetches fresh data and populates the cache
- Subsequent workflows use the cached data for faster execution
-->
