---
name: Token Logs Fetch
description: Pre-fetches Copilot and Claude workflow run logs daily and stores them in cache-memory to avoid redundant API calls and rate-limiting in downstream token analysis and optimization workflows
on:
  schedule:
    - cron: "daily around 08:45 on weekdays"
  workflow_dispatch:

permissions:
  contents: read
  actions: read

engine: copilot
features:
  copilot-requests: true

tools:
  bash:
    - "*"
  cache-memory: true

safe-outputs:
  noop:

network: defaults

timeout-minutes: 15

steps:
  - name: Install gh-aw CLI
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      if gh extension list | grep -q "github/gh-aw"; then
        gh extension upgrade gh-aw || true
      else
        gh extension install github/gh-aw
      fi
      gh aw --version
  - name: Fetch Copilot and Claude workflow run logs
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -euo pipefail
      mkdir -p /tmp/token-logs

      echo "📥 Fetching Copilot workflow runs from last 24 hours..."
      gh aw logs \
        --engine copilot \
        --start-date -1d \
        --json \
        -c 300 \
        > /tmp/token-logs/copilot-runs-raw.json 2>/dev/null || echo '{"runs":[]}' > /tmp/token-logs/copilot-runs-raw.json

      jq '.runs // []' /tmp/token-logs/copilot-runs-raw.json > /tmp/token-logs/copilot-runs.json 2>/dev/null || echo "[]" > /tmp/token-logs/copilot-runs.json
      echo "✅ Copilot runs: $(jq 'length' /tmp/token-logs/copilot-runs.json)"

      echo "📥 Fetching Claude workflow runs from last 24 hours..."
      gh aw logs \
        --engine claude \
        --start-date -1d \
        --json \
        -c 300 \
        > /tmp/token-logs/claude-runs-raw.json 2>/dev/null || echo '{"runs":[]}' > /tmp/token-logs/claude-runs-raw.json

      jq '.runs // []' /tmp/token-logs/claude-runs-raw.json > /tmp/token-logs/claude-runs.json 2>/dev/null || echo "[]" > /tmp/token-logs/claude-runs.json
      echo "✅ Claude runs: $(jq 'length' /tmp/token-logs/claude-runs.json)"
---

# Token Logs Fetch

Pre-fetched workflow run logs are available at `/tmp/token-logs/`:

- `/tmp/token-logs/copilot-runs.json` — Copilot workflow runs from the last 24 hours
- `/tmp/token-logs/claude-runs.json` — Claude workflow runs from the last 24 hours

Your task is to cache these files for use by downstream token analysis and optimization workflows:

```bash
mkdir -p /tmp/gh-aw/cache-memory/token-logs
cp /tmp/token-logs/copilot-runs.json /tmp/gh-aw/cache-memory/token-logs/
cp /tmp/token-logs/claude-runs.json /tmp/gh-aw/cache-memory/token-logs/
date -u +%Y-%m-%d > /tmp/gh-aw/cache-memory/token-logs/fetch-date.txt
echo "Cached logs for $(cat /tmp/gh-aw/cache-memory/token-logs/fetch-date.txt)"
```

Once the files are cached, call noop.

```json
{"noop": {"message": "Token logs fetched and cached for downstream analysis workflows"}}
```
