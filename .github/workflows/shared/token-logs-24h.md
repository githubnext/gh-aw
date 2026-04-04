---
# Shared pre-step: restore 24h Copilot and Claude token logs from cache or download fresh.
#
# After this step, JSON log arrays are available at:
#   /tmp/gh-aw/token-logs/copilot-runs.json  — Copilot workflow runs (last 24h)
#   /tmp/gh-aw/token-logs/claude-runs.json   — Claude workflow runs (last 24h)
#
# Data is sourced from the Token Logs Fetch workflow cache-memory artifact when available
# (matching today's date), ensuring logs are downloaded at most once per day across all
# workflows that import this shared step.

steps:
  - name: Restore 24h token logs from cache
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -euo pipefail
      TOKEN_LOGS_DIR="/tmp/gh-aw/token-logs"
      mkdir -p "$TOKEN_LOGS_DIR"
      TODAY=$(date -u +%Y-%m-%d)

      # Look for today's pre-fetched data from the Token Logs Fetch workflow
      FETCH_RUN_ID=$(gh run list \
        --workflow "token-logs-fetch.lock.yml" \
        --status success \
        --limit 1 \
        --json databaseId \
        --jq '.[0].databaseId' 2>/dev/null || echo "")

      USED_CACHE=false
      if [ -n "$FETCH_RUN_ID" ]; then
        CACHE_TMP="/tmp/gh-aw/token-logs-fetch-cache"
        mkdir -p "$CACHE_TMP"
        gh run download "$FETCH_RUN_ID" \
          --repo "$GITHUB_REPOSITORY" \
          --name "cache-memory" \
          --dir "$CACHE_TMP" \
          2>/dev/null || true
        CACHE_DATE=$(cat "$CACHE_TMP/token-logs/fetch-date.txt" 2>/dev/null || echo "")
        if [ "$CACHE_DATE" = "$TODAY" ] && \
           [ -s "$CACHE_TMP/token-logs/copilot-runs.json" ] && \
           [ -s "$CACHE_TMP/token-logs/claude-runs.json" ]; then
          echo "✅ Using pre-fetched logs from Token Logs Fetch run $FETCH_RUN_ID (date: $CACHE_DATE)"
          cp "$CACHE_TMP/token-logs/copilot-runs.json" "$TOKEN_LOGS_DIR/copilot-runs.json"
          cp "$CACHE_TMP/token-logs/claude-runs.json" "$TOKEN_LOGS_DIR/claude-runs.json"
          USED_CACHE=true
        else
          echo "ℹ️ No valid cached logs found (cache date: ${CACHE_DATE:-none}, today: $TODAY)"
        fi
      fi

      if [ "$USED_CACHE" != "true" ]; then
        echo "📥 Downloading Copilot and Claude workflow runs from last 24 hours..."

        # Ensure gh-aw CLI is installed — this shared step runs before user-defined steps.
        # Install failure is non-fatal to match the fallback-safe behavior of gh aw logs below.
        GH_AW_AVAILABLE=false
        if gh extension list 2>/dev/null | grep -q "github/gh-aw"; then
          GH_AW_AVAILABLE=true
        else
          echo "📦 Installing gh-aw CLI extension..."
          if gh extension install github/gh-aw 2>/dev/null; then
            GH_AW_AVAILABLE=true
          else
            echo "⚠️ Failed to install gh-aw CLI extension; continuing with empty token logs."
          fi
        fi

        # Check GitHub API rate limit before downloading logs
        RATE_INFO=$(gh api rate_limit --jq '.rate | "\(.remaining)/\(.limit) (resets \(.reset | todate))"' 2>/dev/null || echo "unknown")
        echo "📊 GitHub API rate limit before download: $RATE_INFO"

        if [ "$GH_AW_AVAILABLE" = "true" ]; then
          LOGS_STDERR="/tmp/token-logs-copilot-stderr.log"
          # Use || to prevent set -e from killing the script on non-zero exit
          COPILOT_EXIT=0
          gh aw logs \
            --engine copilot \
            --start-date -1d \
            --json \
            -c 300 \
            > /tmp/token-logs-copilot-raw.json 2>"$LOGS_STDERR" || COPILOT_EXIT=$?
          if [ "$COPILOT_EXIT" -ne 0 ]; then
            echo "⚠️ gh aw logs --engine copilot exited with code $COPILOT_EXIT"
            cat "$LOGS_STDERR" | tail -5
            echo '{"runs":[]}' > /tmp/token-logs-copilot-raw.json
          fi
        else
          echo '{"runs":[]}' > /tmp/token-logs-copilot-raw.json
        fi
        jq '.runs // []' /tmp/token-logs-copilot-raw.json > "$TOKEN_LOGS_DIR/copilot-runs.json" 2>/dev/null || echo "[]" > "$TOKEN_LOGS_DIR/copilot-runs.json"

        if [ "$GH_AW_AVAILABLE" = "true" ]; then
          LOGS_STDERR="/tmp/token-logs-claude-stderr.log"
          # Use || to prevent set -e from killing the script on non-zero exit
          CLAUDE_EXIT=0
          gh aw logs \
            --engine claude \
            --start-date -1d \
            --json \
            -c 300 \
            > /tmp/token-logs-claude-raw.json 2>"$LOGS_STDERR" || CLAUDE_EXIT=$?
          if [ "$CLAUDE_EXIT" -ne 0 ]; then
            echo "⚠️ gh aw logs --engine claude exited with code $CLAUDE_EXIT"
            cat "$LOGS_STDERR" | tail -5
            echo '{"runs":[]}' > /tmp/token-logs-claude-raw.json
          fi
        else
          echo '{"runs":[]}' > /tmp/token-logs-claude-raw.json
        fi
        jq '.runs // []' /tmp/token-logs-claude-raw.json > "$TOKEN_LOGS_DIR/claude-runs.json" 2>/dev/null || echo "[]" > "$TOKEN_LOGS_DIR/claude-runs.json"

        # Check rate limit after downloads to see consumption
        RATE_INFO=$(gh api rate_limit --jq '.rate | "\(.remaining)/\(.limit) (resets \(.reset | todate))"' 2>/dev/null || echo "unknown")
        echo "📊 GitHub API rate limit after download: $RATE_INFO"
      fi

      echo "✅ Copilot runs: $(jq 'length' "$TOKEN_LOGS_DIR/copilot-runs.json")"
      echo "✅ Claude runs: $(jq 'length' "$TOKEN_LOGS_DIR/claude-runs.json")"
---

## 24h Token Logs

Pre-fetched workflow run logs are available at `/tmp/gh-aw/token-logs/`:

- `/tmp/gh-aw/token-logs/copilot-runs.json` — Copilot workflow runs from the last 24 hours
- `/tmp/gh-aw/token-logs/claude-runs.json` — Claude workflow runs from the last 24 hours

Each file is a JSON array of run objects with fields:
- `.workflow_name` — workflow name string
- `.token_usage` — total tokens (int, may be null/0)
- `.estimated_cost` — estimated cost in USD (float, may be null/0)
- `.database_id` — run ID (int64)
- `.created_at` — run creation timestamp
- `.url` — run URL

Data is sourced from the [Token Logs Fetch](../token-logs-fetch.md) workflow cache when available
(matching today's UTC date), or downloaded fresh otherwise — ensuring logs are fetched at most
once per day across all workflows that import this shared step.
