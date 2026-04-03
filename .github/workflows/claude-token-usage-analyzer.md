---
name: Claude Token Usage Analyzer
description: Daily analysis of Claude/Anthropic token consumption across all agentic workflows, creating a usage report issue with per-workflow statistics including cache write amortization
on:
  schedule:
    - cron: "daily around 09:30 on weekdays"
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
    title-prefix: "📊 Claude Token Usage Report: "
    labels: [automated-analysis, token-usage, claude]
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
  - name: Download Claude workflow runs (last 24h)
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -euo pipefail
      mkdir -p /tmp/token-analyzer-claude

      echo "📥 Downloading Claude workflow runs from last 24 hours..."
      gh aw logs \
        --engine claude \
        --start-date -1d \
        --json \
        -c 300 \
        > /tmp/token-analyzer-claude/claude-runs.json 2>/dev/null || echo "[]" > /tmp/token-analyzer-claude/claude-runs.json

      RUN_COUNT=$(jq '. | length' /tmp/token-analyzer-claude/claude-runs.json 2>/dev/null || echo 0)
      echo "✅ Found ${RUN_COUNT} Claude workflow runs"

      # Download token-usage.jsonl artifacts for per-model breakdown
      ARTIFACT_DIR="/tmp/token-analyzer-claude/artifacts"
      mkdir -p "$ARTIFACT_DIR"

      echo "📥 Downloading token-usage.jsonl artifacts..."
      jq -r '.[0:50] | .[].databaseId' /tmp/token-analyzer-claude/claude-runs.json > /tmp/token-analyzer-claude/run-ids.txt
      while read -r run_id; do
        run_dir="$ARTIFACT_DIR/$run_id"
        mkdir -p "$run_dir"
        gh run download "$run_id" \
          --repo "$GITHUB_REPOSITORY" \
          --name "firewall-audit-logs" \
          --dir "$run_dir" \
          2>/dev/null || true
      done < /tmp/token-analyzer-claude/run-ids.txt

      # Count how many token-usage.jsonl files we got
      JSONL_COUNT=$(find "$ARTIFACT_DIR" -name "token-usage.jsonl" 2>/dev/null | wc -l)
      echo "✅ Downloaded ${JSONL_COUNT} token-usage.jsonl artifacts"

      # Merge all token-usage.jsonl files annotated with run_id
      MERGED_FILE="/tmp/token-analyzer-claude/token-usage-merged.jsonl"
      > "$MERGED_FILE"
      find "$ARTIFACT_DIR" -name "token-usage.jsonl" > /tmp/token-analyzer-claude/jsonl-files.txt 2>/dev/null || true
      while read -r f; do
        run_id=$(echo "$f" | grep -oP '(?<=/artifacts/)\d+(?=/)' || true)
        while IFS= read -r line; do
          if [ -n "$line" ]; then
            echo "${line}" | jq --arg run_id "$run_id" '. + {run_id: $run_id}' >> "$MERGED_FILE" 2>/dev/null || true
          fi
        done < "$f"
      done < /tmp/token-analyzer-claude/jsonl-files.txt

      RECORD_COUNT=$(wc -l < "$MERGED_FILE" 2>/dev/null || echo 0)
      echo "✅ Merged ${RECORD_COUNT} token usage records"

imports:
  - shared/reporting.md
---

# Claude Token Usage Analyzer

You are the Claude Token Usage Analyzer. Your job is to analyze Claude/Anthropic token consumption across all agentic workflows that ran in the past 24 hours and create a concise, actionable report issue with Anthropic-specific insights.

## Current Context

- **Repository**: ${{ github.repository }}
- **Analysis Date**: $(date -u +%Y-%m-%d)
- **Engine Filter**: Claude only
- **Window**: Last 24 hours

## Data Sources

Pre-downloaded data is available in `/tmp/token-analyzer-claude/`:

- **`/tmp/token-analyzer-claude/claude-runs.json`** — All Claude workflow runs from the last 24 hours (array of run objects with `workflowName`, `databaseId`, `tokenUsage`, `estimatedCost`, `turns`, `url`, `conclusion`, etc.)
- **`/tmp/token-analyzer-claude/token-usage-merged.jsonl`** — Merged per-request token records from `firewall-audit-logs` artifacts, with fields: `model`, `provider`, `input_tokens`, `output_tokens`, `cache_read_tokens`, `cache_write_tokens`, `duration_ms`, `run_id`

## Analysis Process

### Phase 1: Parse Workflow Run Data

Process `/tmp/token-analyzer-claude/claude-runs.json` to compute per-workflow statistics:

```bash
jq -r '.[] | [.workflowName, .tokenUsage, .estimatedCost, .turns, .conclusion, .url, .databaseId] | @tsv' \
  /tmp/token-analyzer-claude/claude-runs.json
```

Compute for each workflow:
- **Total runs** and **successful runs** (conclusion == "success")
- **Total tokens** and **average tokens per run**
- **Total estimated cost** and **average cost per run**
- **Average turns per run**

### Phase 2: Anthropic-Specific Token Analysis

Process `/tmp/token-analyzer-claude/token-usage-merged.jsonl` for Claude-specific metrics. Anthropic pricing has unique characteristics:

1. **Cache Read Tokens**: Billed at ~10% of input token rate — high cache read rate is very beneficial
2. **Cache Write Tokens**: Billed at ~125% of input token rate — one-time cost amortized over subsequent reads (5-minute TTL for short, longer for extended)
3. **Output Tokens**: Most expensive — reducing verbosity has highest ROI

Compute for each model (e.g., `claude-sonnet-4-5`, `claude-opus-4-5`):

```bash
jq -r '[.model, .input_tokens, .output_tokens, .cache_read_tokens, .cache_write_tokens] | @tsv' \
  /tmp/token-analyzer-claude/token-usage-merged.jsonl 2>/dev/null
```

Calculate:
- **Cache write rate**: `cache_write / (input + cache_write)` — how much is being cached
- **Cache read rate**: `cache_read / (input + cache_read)` — how much is served from cache
- **Cache efficiency**: Are cache writes being amortized across enough reads?
  - If `cache_read / cache_write < 2`, caching may not be saving money
- **Effective cost multiplier**: Estimate cost relative to no-caching baseline
- **Cache TTL guidance**: Short prompts (< 1024 tokens) are not cached; ensure system prompts are long enough

### Phase 3: Identify Top Workflows and Anthropic-Specific Optimization Opportunities

From the per-workflow statistics, identify:
1. **Top 5 most expensive workflows** by total estimated cost
2. **High output token ratio** (`output / (input + output) > 40%`) — verbose agents
3. **Low cache efficiency** — high cache writes but low cache reads (cache not being reused)
4. **No caching** — workflows with 0 cache reads despite repeated execution

### Phase 4: Create Report Issue

Create an issue with the title format: `YYYY-MM-DD` (date only — the prefix `📊 Claude Token Usage Report:` is automatically added).

#### Issue Body Structure

```markdown
### Summary

Analyzed **[N]** Claude workflow runs from **[DATE]** covering **[M]** unique workflows.
Total: **[TOTAL_TOKENS]** tokens (~**$[TOTAL_COST]**) across **[TOTAL_TURNS]** turns.

### Top Workflows by Cost

| Workflow | Runs | Total Tokens | Avg Tokens/Run | Est. Cost | Avg Turns |
|----------|------|--------------|----------------|-----------|-----------|
| [name] | [n] | [tokens] | [avg] | $[cost] | [turns] |
| ... | | | | | |

### Anthropic Token Breakdown

| Model | Input | Output | Cache Read | Cache Write | Cache Hit % | Cache Write Rate |
|-------|-------|--------|------------|-------------|-------------|------------------|
| [model] | [n] | [n] | [n] | [n] | [pct]% | [pct]% |

> 💡 Cache reads save ~90% vs input tokens. Cache writes cost ~125% but pay off after ~1.4x reads.

_(Only shown when token-usage.jsonl artifacts are available)_

<details>
<summary><b>Cache Efficiency Analysis</b></summary>

For each model with cache activity:
- **Cache read/write ratio**: [ratio] — [assessment: excellent/good/poor]
- **Estimated savings vs no-cache**: ~[N] tokens (~$[cost])
- **Recommendation**: [specific action if cache efficiency is poor]

</details>

<details>
<summary><b>All Workflows (Full Statistics)</b></summary>

| Workflow | Runs | Success Rate | Total Tokens | Total Cost | Avg Turns | Avg Cost/Run |
|----------|------|--------------|--------------|------------|-----------|--------------|
| [name] | [n] | [pct]% | [tokens] | $[cost] | [turns] | $[avg] |
| ... | | | | | | |

</details>

### Optimization Opportunities

1. **[Workflow]** — [e.g., "cache write rate 0% — static system prompt not meeting 1024-token minimum for caching"]
2. **[Workflow]** — [e.g., "output tokens are 55% of total — consider limiting response length in prompt"]
3. **[Workflow]** — [e.g., "cache read/write ratio is 0.3 — context changes too frequently for cache to be effective"]

### References

- Triggered by: [§RUN_ID](RUN_URL)
```

## Important Guidelines

- **If no runs found**: Call `noop` with message explaining no Claude runs in the last 24 hours.
- **Anthropic pricing nuances**: Cache writes are expensive short-term but pay off quickly. Don't flag cache writes as waste unless read/write ratio is very low.
- **Cache TTL**: Claude's cache expires after 5 minutes by default. Workflows with > 5 min between turns may not benefit from caching.
- **Be precise**: Use exact numbers from the data.
- **Link runs**: Format run IDs as `[§ID](URL)` for easy navigation.
- **One issue only**: The `max: 1` configuration ensures only one issue is created; older issues are auto-closed.

**Important**: You MUST call a safe-output tool (`create-issue` or `noop`) at the end of your analysis. Failing to call any safe-output tool is the most common cause of workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation]"}}
```
