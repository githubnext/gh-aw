---
name: Copilot Token Usage Analyzer
description: Daily analysis of Copilot token consumption across all agentic workflows, creating a usage report issue with per-workflow statistics and optimization opportunities
on:
  schedule:
    - cron: "daily around 09:00 on weekdays"
  workflow_dispatch:

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read

engine: copilot
features:
  copilot-requests: true

tools:
  bash:
    - "*"
  github:
    toolsets: [default, issues, actions]

safe-outputs:
  create-issue:
    title-prefix: "📊 Copilot Token Usage Report: "
    labels: [automated-analysis, token-usage, copilot]
    expires: 2d
    max: 1
    close-older-issues: true
  noop:

network: defaults

timeout-minutes: 30

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
  - name: Download Copilot workflow runs (last 24h)
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -euo pipefail
      mkdir -p /tmp/token-analyzer

      # Use pre-fetched logs from the shared token-logs-24h pre-step
      cp /tmp/gh-aw/token-logs/copilot-runs.json /tmp/token-analyzer/copilot-runs.json 2>/dev/null || echo "[]" > /tmp/token-analyzer/copilot-runs.json

      RUN_COUNT=$(jq 'length' /tmp/token-analyzer/copilot-runs.json 2>/dev/null || echo 0)
      echo "✅ Found ${RUN_COUNT} Copilot workflow runs"

      # Download token-usage.jsonl artifacts for per-model breakdown
      # We look for the firewall-audit-logs artifact which contains token-usage.jsonl
      ARTIFACT_DIR="/tmp/token-analyzer/artifacts"
      mkdir -p "$ARTIFACT_DIR"

      echo "📥 Downloading token-usage.jsonl artifacts..."
      jq -r '.[0:50][]?.database_id' /tmp/token-analyzer/copilot-runs.json 2>/dev/null > /tmp/token-analyzer/run-ids.txt || true
      while read -r run_id; do
        run_dir="$ARTIFACT_DIR/$run_id"
        mkdir -p "$run_dir"
        gh run download "$run_id" \
          --repo "$GITHUB_REPOSITORY" \
          --name "firewall-audit-logs" \
          --dir "$run_dir" \
          2>/dev/null || true
      done < /tmp/token-analyzer/run-ids.txt

      # Count how many token-usage.jsonl files we got
      JSONL_COUNT=$(find "$ARTIFACT_DIR" -name "token-usage.jsonl" 2>/dev/null | wc -l)
      echo "✅ Downloaded ${JSONL_COUNT} token-usage.jsonl artifacts"

      # Merge all token-usage.jsonl files into a single aggregate file annotated with run_id
      MERGED_FILE="/tmp/token-analyzer/token-usage-merged.jsonl"
      > "$MERGED_FILE"
      find "$ARTIFACT_DIR" -name "token-usage.jsonl" > /tmp/token-analyzer/jsonl-files.txt 2>/dev/null || true
      while read -r f; do
        run_id=$(echo "$f" | grep -oP '(?<=/artifacts/)\d+(?=/)' || true)
        while IFS= read -r line; do
          if [ -n "$line" ]; then
            echo "${line}" | jq --arg run_id "$run_id" '. + {run_id: $run_id}' >> "$MERGED_FILE" 2>/dev/null || true
          fi
        done < "$f"
      done < /tmp/token-analyzer/jsonl-files.txt

      RECORD_COUNT=$(wc -l < "$MERGED_FILE" 2>/dev/null || echo 0)
      echo "✅ Merged ${RECORD_COUNT} token usage records"

imports:
  - shared/token-logs-24h.md
  - shared/reporting.md
---

# Copilot Token Usage Analyzer

You are the Copilot Token Usage Analyzer. Your job is to analyze Copilot token consumption across all agentic workflows that ran in the past 24 hours and create a concise, actionable report issue.

## Current Context

- **Repository**: ${{ github.repository }}
- **Analysis Date**: $(date -u +%Y-%m-%d)
- **Engine Filter**: Copilot only
- **Window**: Last 24 hours

## Data Sources

Pre-downloaded data is available in `/tmp/token-analyzer/`:

- **`/tmp/token-analyzer/copilot-runs.json`** — All Copilot workflow runs from the last 24 hours (array of run objects with `workflow_name`, `database_id`, `token_usage`, `turns`, `url`, `conclusion`, etc.)
- **`/tmp/token-analyzer/token-usage-merged.jsonl`** — Merged per-request token records from `firewall-audit-logs` artifacts, with fields: `model`, `provider`, `input_tokens`, `output_tokens`, `cache_read_tokens`, `cache_write_tokens`, `duration_ms`, `run_id`

## Analysis Process

### Phase 1: Parse Workflow Run Data

Process `/tmp/token-analyzer/copilot-runs.json` to compute per-workflow statistics:

```bash
jq -r '.[] | [.workflow_name, .token_usage, .turns, .conclusion, .url, .database_id] | @tsv' \
  /tmp/token-analyzer/copilot-runs.json
```

Compute for each workflow:
- **Total runs** and **successful runs** (conclusion == "success")
- **Total tokens** and **average tokens per run**
- **Total estimated cost** and **average cost per run**
- **Average turns per run**
- **Run IDs** for the most expensive runs (for artifact links)

### Phase 2: Parse Token-Level Data (if available)

Process `/tmp/token-analyzer/token-usage-merged.jsonl` for per-model breakdown:

```bash
# Aggregate by model
jq -r '[.model, .input_tokens, .output_tokens, .cache_read_tokens, .cache_write_tokens] | @tsv' \
  /tmp/token-analyzer/token-usage-merged.jsonl 2>/dev/null | awk '...'
```

Compute for each model:
- **Total input tokens** (billed at full rate)
- **Total output tokens** (billed at full rate)
- **Total cache read tokens** (billed at reduced rate ~10%)
- **Cache hit rate**: `cache_read / (input + cache_read)` × 100%
- **Billable token equivalent**: approximate total considering cache discounts

### Phase 3: Identify Top Workflows and Anomalies

From the per-workflow statistics, identify:
1. **Top 5 most expensive workflows** by total estimated cost
2. **Highest token-per-turn ratio** (potential for optimization)
3. **Lowest cache hit rate** (may benefit from prompt restructuring)
4. **Highest run volume** (most frequent consumers)

### Phase 4: Create Report Issue

Create an issue with the title format: `YYYY-MM-DD` (date only — the prefix `📊 Copilot Token Usage Report:` is automatically added).

#### Issue Body Structure

```markdown
### Summary

Analyzed **[N]** Copilot workflow runs from **[DATE]** covering **[M]** unique workflows.
Total: **[TOTAL_TOKENS]** tokens (~**$[TOTAL_COST]**) across **[TOTAL_TURNS]** turns.

### Top Workflows by Cost

| Workflow | Runs | Total Tokens | Avg Tokens/Run | Est. Cost | Avg Turns |
|----------|------|--------------|----------------|-----------|-----------|
| [name] | [n] | [tokens] | [avg] | $[cost] | [turns] |
| ... | | | | | |

### Token Breakdown by Model

| Model | Input Tokens | Output Tokens | Cache Read | Cache Hit % | Requests |
|-------|-------------|---------------|------------|-------------|----------|
| [model] | [n] | [n] | [n] | [pct]% | [n] |

_(Only shown when token-usage.jsonl artifacts are available)_

<details>
<summary><b>All Workflows (Full Statistics)</b></summary>

| Workflow | Runs | Success Rate | Total Tokens | Total Cost | Avg Turns | Avg Cost/Run |
|----------|------|--------------|--------------|------------|-----------|--------------|
| [name] | [n] | [pct]% | [tokens] | $[cost] | [turns] | $[avg] |
| ... | | | | | | |

</details>

### Optimization Opportunities

1. **[Workflow]** — [specific observation, e.g., "avg 45k tokens/run with 0% cache hit rate — consider restructuring prompt for better caching"]
2. **[Workflow]** — [observation]

### References

- Triggered by: [§RUN_ID](RUN_URL)
```

## Important Guidelines

- **If no runs found**: Call `noop` with message explaining no Copilot runs in the last 24 hours.
- **Be precise**: Use exact numbers from the data, not estimates.
- **Link runs**: Format run IDs as `[§ID](URL)` for easy navigation.
- **One issue only**: The `max: 1` configuration ensures only one issue is created; older issues are auto-closed.
- **Use `noop` if needed**: If you cannot create a meaningful report (no data, parse errors), call `noop` with an explanation.

**Important**: You MUST call a safe-output tool (`create-issue` or `noop`) at the end of your analysis. Failing to call any safe-output tool is the most common cause of workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation]"}}
```
