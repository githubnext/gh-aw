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

        gh aw logs \
          --engine copilot \
          --start-date -1d \
          --json \
          -c 300 \
          > /tmp/token-logs-copilot-raw.json 2>/dev/null || echo '{"runs":[]}' > /tmp/token-logs-copilot-raw.json
        jq '.runs // []' /tmp/token-logs-copilot-raw.json > "$TOKEN_LOGS_DIR/copilot-runs.json" 2>/dev/null || echo "[]" > "$TOKEN_LOGS_DIR/copilot-runs.json"

        gh aw logs \
          --engine claude \
          --start-date -1d \
          --json \
          -c 300 \
          > /tmp/token-logs-claude-raw.json 2>/dev/null || echo '{"runs":[]}' > /tmp/token-logs-claude-raw.json
        jq '.runs // []' /tmp/token-logs-claude-raw.json > "$TOKEN_LOGS_DIR/claude-runs.json" 2>/dev/null || echo "[]" > "$TOKEN_LOGS_DIR/claude-runs.json"
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
