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
    - "gh api *"
    - "jq *"
    - "./.github/skills/jqschema/jqschema.sh"
    - "mkdir *"
    - "date *"
    - "cp *"
    - "unzip *"
    - "find *"
    - "rm *"
    - "cat *"
    - "grep *"
    - "wc *"
    - "head *"
    - "tee *"

steps:
  - name: Install gh CLI
    run: |
      bash "${RUNNER_TEMP}/gh-aw/actions/install_gh_cli.sh"

  - name: Fetch Copilot session data
    env:
      GITHUB_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
    run: |
      # Create output directories
      mkdir -p /tmp/gh-aw/agent/session-data
      mkdir -p /tmp/gh-aw/agent/session-data/logs
      mkdir -p /tmp/gh-aw/cache-memory
      
      # Cache entries use stable names so caches restored from previous runs are reusable
      # regardless of the date they were written on. Freshness is decided by file age.
      CACHE_DIR="/tmp/gh-aw/cache-memory"
      CACHE_FILE="$CACHE_DIR/copilot-sessions-latest.json"
      CACHE_SCHEMA_FILE="$CACHE_DIR/copilot-sessions-latest-schema.json"
      CACHE_LOGS_DIR="$CACHE_DIR/session-logs-latest"
      CACHE_MAX_AGE_SECONDS="${CACHE_MAX_AGE_SECONDS:-21600}"
      RUN_TAG="${GITHUB_RUN_ID:-local}"

      # Returns success when the file exists, is non-empty and younger than the max age.
      cache_is_fresh() {
        cache_path="$1"
        [ -s "$cache_path" ] || return 1
        cache_mtime=$(date -r "$cache_path" '+%s' 2>/dev/null) || return 1
        [ -n "$cache_mtime" ] || return 1
        now=$(date '+%s')
        [ "$((now - cache_mtime))" -lt "$CACHE_MAX_AGE_SECONDS" ]
      }

      count_agent_logs() {
        sessions_file="$1"
        logs_dir="$2"
        count=0

        if [ ! -f "$sessions_file" ] || [ ! -d "$logs_dir" ]; then
          echo 0
          return
        fi

        while read -r agent_run_id; do
          if [ -s "$logs_dir/${agent_run_id}-events.jsonl" ] || [ -s "$logs_dir/${agent_run_id}-conversation.txt" ]; then
            count=$((count + 1))
          fi
        done < <(jq -r '.[] | select(.status == "completed" and .conclusion != "action_required") | .id' "$sessions_file")

        echo "$count"
      }

      USE_CACHE=false

      # Check if restored cache data is still fresh enough to reuse
      if cache_is_fresh "$CACHE_FILE"; then
        CACHED_AGENT_COUNT=$(jq '[.[] | select(.status == "completed" and .conclusion != "action_required")] | length' "$CACHE_FILE")
        CACHED_LOG_RUN_COUNT=$(count_agent_logs "$CACHE_FILE" "$CACHE_LOGS_DIR")
        if [ "$CACHED_AGENT_COUNT" -eq 0 ] || [ "$CACHED_LOG_RUN_COUNT" -ge "$CACHED_AGENT_COUNT" ]; then
          USE_CACHE=true
        else
          echo "::warning::Cached session metadata exists but only $CACHED_LOG_RUN_COUNT of $CACHED_AGENT_COUNT agent runs have logs; refreshing session data"
        fi
      fi

      if [ "$USE_CACHE" = "true" ]; then
        CACHE_AGE_MINUTES=$(( ( $(date '+%s') - $(date -r "$CACHE_FILE" '+%s') ) / 60 ))
        echo "✓ Found cached session data (${CACHE_AGE_MINUTES} minutes old)"
        cp "$CACHE_FILE" /tmp/gh-aw/agent/session-data/sessions-list.json
        
        # Regenerate schema if missing
        if [ ! -s "$CACHE_SCHEMA_FILE" ]; then
          ./.github/skills/jqschema/jqschema.sh < /tmp/gh-aw/agent/session-data/sessions-list.json > "$CACHE_SCHEMA_FILE"
        fi
        cp "$CACHE_SCHEMA_FILE" /tmp/gh-aw/agent/session-data/sessions-schema.json
        
        # Restore cached log files if they exist
        if [ -d "$CACHE_LOGS_DIR" ]; then
          echo "✓ Found cached session logs"
          cp -r "$CACHE_LOGS_DIR"/* /tmp/gh-aw/agent/session-data/logs/ 2>/dev/null || true
          echo "Restored $(find /tmp/gh-aw/agent/session-data/logs -type f | wc -l) session log files from cache"
        fi
        
        echo "Using cached data written ${CACHE_AGE_MINUTES} minutes ago"
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

        # Download per-session logs for actual Copilot agent runs.
        # CI gate runs (e.g. "Smoke CI", "CGO", "CWI" quality gates) always end with
        # conclusion=action_required because a human must approve them to continue; they
        # contain no Copilot agent activity and have no conversation transcript.
        # Prefer structured events.jsonl artifacts; fall back to [cca-engine] turn= lines
        # in raw Actions job logs when no events.jsonl artifact is available.
        SESSION_COUNT=$(jq 'length' /tmp/gh-aw/agent/session-data/sessions-list.json)
        AGENT_COUNT=$(jq '[.[] | select(.status == "completed" and .conclusion != "action_required")] | length' /tmp/gh-aw/agent/session-data/sessions-list.json)
        echo "Downloading per-session logs for $AGENT_COUNT agent runs (skipping $((SESSION_COUNT - AGENT_COUNT)) CI gate runs)..."

        RUNS_TO_FETCH="${RUNNER_TEMP:-/tmp}/copilot-session-runs-${RUN_TAG}.txt"
        jq -r '.[] | select(.status == "completed" and .conclusion != "action_required") | "\(.id) \(.head_branch)"' /tmp/gh-aw/agent/session-data/sessions-list.json > "$RUNS_TO_FETCH"

        FETCH_STATUS_LOG="${RUNNER_TEMP:-/tmp}/copilot-session-fetch-status-${RUN_TAG}.log"
        rm -f "$FETCH_STATUS_LOG"
        API_ERROR_COUNT=0
        NO_JOBS_COUNT=0
        EMPTY_LOG_COUNT=0
        NO_MATCH_COUNT=0

        while read -r run_id branch; do
          if [ -n "$run_id" ]; then
            echo "Downloading session log for run $run_id (branch: $branch)"

            # Prefer structured Copilot session events from run artifacts. The agent
            # writes events.jsonl under the session-state logs artifact when available.
            artifact_work_dir="${RUNNER_TEMP:-/tmp}/copilot-session-artifacts-${run_id}"
            rm -rf "$artifact_work_dir"
            mkdir -p "$artifact_work_dir"
            events_tmp="/tmp/gh-aw/agent/session-data/logs/${run_id}-events.jsonl.tmp"
            events_out="/tmp/gh-aw/agent/session-data/logs/${run_id}-events.jsonl"
            rm -f "$events_tmp" "$events_out"

            for artifact_id in $(gh api "repos/$GITHUB_REPOSITORY/actions/runs/${run_id}/artifacts" \
              --jq '.artifacts[] | select(.expired == false) | .id' 2>/dev/null || true); do
              artifact_zip="${artifact_work_dir}/${artifact_id}.zip"
              artifact_dir="${artifact_work_dir}/${artifact_id}"
              mkdir -p "$artifact_dir"

              if gh api "repos/$GITHUB_REPOSITORY/actions/artifacts/${artifact_id}/zip" \
                > "$artifact_zip" 2>/dev/null && [ -s "$artifact_zip" ]; then
                unzip -q -o "$artifact_zip" -d "$artifact_dir" 2>/dev/null || true
              fi
            done

            for events_file in $(find "$artifact_work_dir" -type f -name events.jsonl 2>/dev/null || true); do
              cat "$events_file" >> "$events_tmp"
            done

            if [ -s "$events_tmp" ]; then
              cp "$events_tmp" "$events_out"
              EVENT_COUNT=$(wc -l < "$events_out")
              echo "  Saved events.jsonl: $EVENT_COUNT events for run $run_id"
            fi
            rm -f "$events_tmp"
            rm -rf "$artifact_work_dir"

            if [ ! -s "$events_out" ]; then
              # Download raw job logs as a transcript fallback. gh api follows the
              # 302 redirect automatically; inspect every job in case the agent job
              # is not first in the run.
              conversation_tmp="/tmp/gh-aw/agent/session-data/logs/${run_id}-conversation.txt.tmp"
              conversation_out="/tmp/gh-aw/agent/session-data/logs/${run_id}-conversation.txt"
              rm -f "$conversation_tmp" "$conversation_out"

              jobs_err="${RUNNER_TEMP:-/tmp}/copilot-session-jobs-${run_id}.err"
              rm -f "$jobs_err"
              jobs_rc=0
              job_ids="$(gh api "repos/$GITHUB_REPOSITORY/actions/runs/${run_id}/jobs" \
                --jq '.jobs[].id' 2>"$jobs_err")" || jobs_rc=$?

              if [ -z "$job_ids" ]; then
                if [ "$jobs_rc" -ne 0 ]; then
                  reason="$(head -c 300 "$jobs_err" 2>/dev/null | tr '\n' ' ')"
                  echo "  Failed to list jobs for run $run_id (exit=$jobs_rc): ${reason:-no error output}" | tee -a "$FETCH_STATUS_LOG"
                  API_ERROR_COUNT=$((API_ERROR_COUNT + 1))
                else
                  echo "  No jobs found for run $run_id" | tee -a "$FETCH_STATUS_LOG"
                  NO_JOBS_COUNT=$((NO_JOBS_COUNT + 1))
                fi
              fi
              rm -f "$jobs_err"

              for job_id in $job_ids; do
                raw_log="/tmp/gh-aw/agent/session-data/logs/${run_id}-${job_id}-raw.log"
                raw_log_err="${RUNNER_TEMP:-/tmp}/copilot-session-log-${run_id}-${job_id}.err"
                rm -f "$raw_log_err"
                log_rc=0
                gh api "repos/$GITHUB_REPOSITORY/actions/jobs/${job_id}/logs" \
                  > "$raw_log" 2>"$raw_log_err" || log_rc=$?

                if [ -f "$raw_log" ] && [ -s "$raw_log" ]; then
                  if grep "\[cca-engine\] turn=" "$raw_log" >> "$conversation_tmp" 2>/dev/null; then
                    :
                  else
                    echo "  Job $job_id log downloaded (no [cca-engine] turn= lines; likely a CI/gate job, not the agent job)" | tee -a "$FETCH_STATUS_LOG"
                    NO_MATCH_COUNT=$((NO_MATCH_COUNT + 1))
                  fi
                elif [ "$log_rc" -ne 0 ]; then
                  reason="$(head -c 300 "$raw_log_err" 2>/dev/null | tr '\n' ' ')"
                  echo "  Failed to download log for job $job_id (run $run_id, exit=$log_rc): ${reason:-no error output}" | tee -a "$FETCH_STATUS_LOG"
                  API_ERROR_COUNT=$((API_ERROR_COUNT + 1))
                else
                  echo "  Log for job $job_id (run $run_id) was empty" | tee -a "$FETCH_STATUS_LOG"
                  EMPTY_LOG_COUNT=$((EMPTY_LOG_COUNT + 1))
                fi
                rm -f "$raw_log" "$raw_log_err" 2>/dev/null || true
              done

              if [ -s "$conversation_tmp" ]; then
                cp "$conversation_tmp" "$conversation_out"
                LINE_COUNT=$(wc -l < "$conversation_out")
                echo "  Saved transcript fallback: $LINE_COUNT lines for run $run_id"
              fi
              rm -f "$conversation_tmp"
            fi

            if [ ! -s "/tmp/gh-aw/agent/session-data/logs/${run_id}-events.jsonl" ] && [ ! -s "/tmp/gh-aw/agent/session-data/logs/${run_id}-conversation.txt" ]; then
              echo "::warning::No events.jsonl artifact or conversation transcript could be downloaded for agent run $run_id"
            fi
          fi
        done < "$RUNS_TO_FETCH"

        LOG_RUN_COUNT=$(count_agent_logs /tmp/gh-aw/agent/session-data/sessions-list.json /tmp/gh-aw/agent/session-data/logs)
        EVENTS_COUNT=$(find /tmp/gh-aw/agent/session-data/logs/ -type f -name "*-events.jsonl" | wc -l)
        CONVERSATION_COUNT=$(find /tmp/gh-aw/agent/session-data/logs/ -type f -name "*-conversation.txt" | wc -l)
        echo "Session logs downloaded: $LOG_RUN_COUNT of $AGENT_COUNT agent runs ($EVENTS_COUNT events.jsonl, $CONVERSATION_COUNT transcript fallbacks)"

        if [ "$CONVERSATION_COUNT" -eq 0 ] && [ "$EVENTS_COUNT" -eq 0 ] && [ "$AGENT_COUNT" -gt 0 ]; then
          echo "::warning::No transcripts retrieved for any of the $AGENT_COUNT agent runs (API errors: $API_ERROR_COUNT, no jobs found: $NO_JOBS_COUNT, empty logs: $EMPTY_LOG_COUNT, no matching lines: $NO_MATCH_COUNT). If API errors dominate, this may indicate the token in use lacks access to download job logs for Copilot coding-agent runs (permission error), a transient network/rate-limit issue, or the runs were deleted (404); check the per-run reasons logged above. If permission errors are the cause, configure the GH_AW_GITHUB_TOKEN secret with a PAT that has actions:read on this repository."
        fi

        if [ "$AGENT_COUNT" -gt 0 ] && [ "$LOG_RUN_COUNT" -lt "$AGENT_COUNT" ]; then
          echo "::warning::$((AGENT_COUNT - LOG_RUN_COUNT)) of $AGENT_COUNT agent runs have no retrievable session log; proceeding with $LOG_RUN_COUNT available logs"
        fi

        # Refresh the log cache first so metadata and logs stay consistent with each other
        rm -rf "$CACHE_LOGS_DIR"
        mkdir -p "$CACHE_LOGS_DIR"
        cp -r /tmp/gh-aw/agent/session-data/logs/* "$CACHE_LOGS_DIR/" 2>/dev/null || true

        # Store in cache under stable names so future runs can reuse it on any date
        cp /tmp/gh-aw/agent/session-data/sessions-list.json "$CACHE_FILE"
        cp /tmp/gh-aw/agent/session-data/sessions-schema.json "$CACHE_SCHEMA_FILE"

        # Drop legacy date-keyed cache entries so the cache does not grow unbounded
        rm -rf "$CACHE_DIR"/session-logs-[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9] 2>/dev/null || true
        rm -f "$CACHE_DIR"/copilot-sessions-[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9].json \
              "$CACHE_DIR"/copilot-sessions-[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]-schema.json 2>/dev/null || true

        echo "✓ Session data saved to cache: $(basename "$CACHE_FILE")"
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
2. Checks the restored cache for `copilot-sessions-latest.json` and reuses it when it is younger than `CACHE_MAX_AGE_SECONDS` (default 6 hours) and its cached logs cover every agent run
3. If a fresh and complete cache exists (written by any earlier run, on any date):
   - Uses cached data instead of making API calls
   - Copies data from cache to working directory
   - Restores cached log files if available
4. If the cache is missing, stale, or incomplete:
   - Calculates the date 30 days ago (cross-platform compatible)
   - Fetches all workflow runs from branches starting with `copilot/` using GitHub API
   - **Downloads per-session logs** for actual agent runs (skips CI gate runs): structured `events.jsonl` from run artifacts first, then conversation transcripts from GitHub Actions job logs as a fallback
   - Saves data to cache under the stable filename `copilot-sessions-latest.json` (logs under `session-logs-latest/`)
   - Copies data to working directory for use
5. Generates a schema of the data structure

### Session Log Access

The fetcher first downloads non-expired GitHub Actions artifacts for each completed real agent run (status = `completed`, conclusion ≠ `action_required`) and extracts any `events.jsonl` files. When no structured events artifact is available, it falls back to raw GitHub Actions job logs and extracts `[cca-engine] turn=` lines as a transcript. CI gate runs (`action_required`) are skipped because they have no agent conversation.

If a real agent run has neither an `events.jsonl` artifact nor a transcript fallback, the fetch step emits a GitHub Actions warning and continues with the available logs. Every per-run failure (job listing error, log download error, empty log, or a downloaded log with no matching `[cca-engine]` lines) is logged individually with its specific reason, and a summary warning breaks down the failure counts by category (API errors, no jobs found, empty logs, no matching lines) so the actual cause is visible instead of a silently empty `logs/` directory. API errors are counted by non-zero `gh api` exit code and can stem from permission errors, rate limiting, or deleted runs — check the per-run log lines for the specific reason rather than assuming a permissions issue.

**Known root cause of empty transcripts**: downloading job logs for GitHub Copilot coding-agent runs (the special `dynamic/copilot-swe-agent/copilot` workflow path) via `gh api repos/{owner}/{repo}/actions/jobs/{job_id}/logs` can fail with an authorization error when using the default `GITHUB_TOKEN`, even with `actions: read` permission — mirroring the same OAuth requirement documented below for `gh agent-task view --log`. The fetch step now authenticates with `secrets.GH_AW_GITHUB_TOKEN` (falling back to `secrets.GITHUB_TOKEN` if unset) so that, once a PAT with `actions:read` is configured as `GH_AW_GITHUB_TOKEN`, job-log downloads for these runs succeed.

The `gh agent-task view --log` approach that was previously used **requires an OAuth token** that the default `GITHUB_TOKEN` does not provide, and relied on extracting a numeric session ID from the branch name — which stopped working when Copilot switched to descriptive branch slugs (e.g., `copilot/fix-mcp-gateway-docker-daemon-access`).

### Caching Strategy

- **Cache Key**: `copilot-session-data` for workflow-level sharing
- **Cache Files**: Stored under stable, date-independent names (`copilot-sessions-latest.json`, `session-logs-latest/`)
- **Cache Freshness**: File modification time is compared against `CACHE_MAX_AGE_SECONDS` (default `21600`, i.e. 6 hours); stale data is refetched
- **Cache Location**: `/tmp/gh-aw/cache-memory/`
- **Cache Benefits**: 
  - Cache restored from a previous run is reusable regardless of the date it was written on, so scheduled workflows running daily or weekly actually hit the cache instead of always refetching
  - Reduces GitHub API rate limit usage
  - Faster workflow execution after the first fetch within the freshness window
  - Includes structured event and transcript fallback cache
- **Cache Consistency**: On refresh, the log cache directory is rebuilt before the session metadata file is written, so metadata never points at a partially refreshed log set

### Output Files

- **`/tmp/gh-aw/agent/session-data/sessions-list.json`**: Full session data including run ID, name, branch, timestamps, status, conclusion, and URL
- **`/tmp/gh-aw/agent/session-data/sessions-schema.json`**: JSON schema showing the structure of the session data
- **`/tmp/gh-aw/agent/session-data/logs/`**: Directory containing session logs
  - **`{run_id}-events.jsonl`**: Structured Copilot session events extracted from run artifacts (preferred; only present for actual agent runs with events artifacts)
  - **`{run_id}-conversation.txt`**: Agent conversation transcript fallback — `[cca-engine] turn=` lines from the job log containing turn-by-turn model/token/tool data (only present when `events.jsonl` is unavailable)
- **`/tmp/gh-aw/cache-memory/copilot-sessions-latest.json`**: Cached session data (stable filename)
- **`/tmp/gh-aw/cache-memory/copilot-sessions-latest-schema.json`**: Cached schema (stable filename)
- **`/tmp/gh-aw/cache-memory/session-logs-latest/`**: Cached log files (stable directory name)

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
jq -r '.[] | select(.status == "completed" and .conclusion != "action_required") | "\(.id) \(.name)"' /tmp/gh-aw/agent/session-data/sessions-list.json

# List session log files (one events or transcript fallback file per actual agent run)
find /tmp/gh-aw/agent/session-data/logs -type f

# Read a specific events.jsonl file (by run ID)
cat /tmp/gh-aw/agent/session-data/logs/29001106791-events.jsonl

# Read a specific transcript fallback (by run ID)
cat /tmp/gh-aw/agent/session-data/logs/29001106791-conversation.txt
```

### Requirements

- Automatically imports the `jqschema` skill for schema generation (via transitive import closure)
- Uses GitHub Actions API to fetch workflow runs from `copilot/*` branches
- **Fetches structured `events.jsonl` from GitHub Actions artifacts**, then falls back to conversation transcripts from GitHub Actions job logs using `actions: read` permission. For most repository workflow runs the standard `GITHUB_TOKEN` is sufficient; downloading job logs from GitHub Copilot coding-agent runs specifically may require a PAT configured as the `GH_AW_GITHUB_TOKEN` repository secret (the fetch step falls back to `GITHUB_TOKEN` automatically if that secret is unset)
- Cross-platform date calculation (works on both GNU and BSD date commands)
- Cache-memory tool is automatically configured for data persistence

### Why Branch-Based Search?

GitHub Copilot creates branches with the `copilot/` prefix, making branch-based workflow run search a reliable way to identify Copilot coding agent sessions.

### Session Log Format

Structured logs (`{run_id}-events.jsonl`) contain one JSON event per line from the Copilot session-state artifact. Transcript fallbacks (`{run_id}-conversation.txt`) contain one `[cca-engine] turn=` line per event, for example:

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
