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
  upload-asset:
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
  - shared/charts-with-trending.md
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

### Phase 1.5: Save Today's Data to Cache-Memory

After computing per-workflow statistics, persist today's aggregated data for trending. Use the `bash` tool:

```bash
mkdir -p /tmp/gh-aw/cache-memory/trending/token-usage
TODAY=$(date -u +%Y-%m-%d)  # Always use UTC date for consistency with the Python charts

# Append daily aggregated totals (one JSON object per line)
cat >> /tmp/gh-aw/cache-memory/trending/token-usage/history.jsonl << EOF
{"date":"${TODAY}","total_tokens":TOTAL_TOKENS,"total_runs":TOTAL_RUNS,"total_cost":TOTAL_COST,"total_turns":TOTAL_TURNS}
EOF

# Append per-workflow breakdown for heatmap (one entry per workflow — repeat for each workflow):
cat >> /tmp/gh-aw/cache-memory/trending/token-usage/workflows.jsonl << EOF
{"date":"${TODAY}","workflow":"WORKFLOW_NAME","tokens":TOKENS,"runs":RUNS,"cost":COST}
EOF
```

Replace the placeholder values (TOTAL_TOKENS, TOTAL_RUNS, etc.) with the actual computed numbers. **Only append entries for workflows that actually ran today** — do not append zero-entries for missing days, as the Python charts gracefully skip charts when data is insufficient.

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

### Phase 3.5: Generate Trending Charts

Generate Python charts to embed in the report issue. Use the Python environment provided by `shared/charts-with-trending.md`.

```bash
mkdir -p /tmp/gh-aw/python/{data,charts}
```

Write the following Python script to `/tmp/gh-aw/python/token_charts.py` and execute it:

```python
#!/usr/bin/env python3
"""
Copilot token usage trending charts
Generates: top-consumers bar, daily trend line, workflow heatmap
"""
import json, os
import pandas as pd
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import seaborn as sns
from datetime import datetime

CACHE_DIR = '/tmp/gh-aw/cache-memory/trending/token-usage'
CHARTS_DIR = '/tmp/gh-aw/python/charts'
os.makedirs(CHARTS_DIR, exist_ok=True)

sns.set_style('whitegrid')

# --- Chart 1: Top-10 Consumers (always) ---
# Build from today's per-workflow data already in workflows.jsonl
wf_file = os.path.join(CACHE_DIR, 'workflows.jsonl')
today = datetime.utcnow().strftime('%Y-%m-%d')
wf_rows = []
if os.path.exists(wf_file):
    with open(wf_file) as f:
        for line in f:
            line = line.strip()
            if line:
                obj = json.loads(line)
                if obj.get('date') == today:
                    wf_rows.append(obj)

if wf_rows:
    df_wf = pd.DataFrame(wf_rows)
    df_top = df_wf.groupby('workflow')['tokens'].sum().nlargest(10).sort_values()
    fig, ax = plt.subplots(figsize=(12, 7), dpi=150)
    colors = sns.color_palette('YlOrRd', len(df_top))
    ax.barh(df_top.index, df_top.values, color=colors)
    ax.set_xlabel('Total Tokens', fontsize=12)
    ax.set_title(f'🔥 Top-10 Copilot Token Consumers — {today}', fontsize=14, fontweight='bold')
    for i, v in enumerate(df_top.values):
        ax.text(v * 1.005, i, f'{v:,.0f}', va='center', fontsize=9)
    plt.tight_layout()
    plt.savefig(f'{CHARTS_DIR}/top_consumers.png', dpi=150, bbox_inches='tight', facecolor='white')
    plt.close()
    print(f'✅ top_consumers.png saved ({len(df_wf)} workflows)')
else:
    print('⚠️  No workflow data for today — skipping top_consumers chart')

# --- Chart 2: Daily trend line (>=2 data points) ---
hist_file = os.path.join(CACHE_DIR, 'history.jsonl')
hist_rows = []
if os.path.exists(hist_file):
    with open(hist_file) as f:
        for line in f:
            line = line.strip()
            if line:
                hist_rows.append(json.loads(line))

if len(hist_rows) >= 2:
    df_hist = pd.DataFrame(hist_rows)
    df_hist['date'] = pd.to_datetime(df_hist['date'])
    df_hist = df_hist.sort_values('date').drop_duplicates('date')

    fig, ax1 = plt.subplots(figsize=(12, 6), dpi=150)
    color_tok = '#d62728'
    color_run = '#1f77b4'
    ax1.set_xlabel('Date', fontsize=11)
    ax1.set_ylabel('Total Tokens', color=color_tok, fontsize=11)
    ax1.plot(df_hist['date'], df_hist['total_tokens'], color=color_tok,
             marker='o', linewidth=2, label='Total Tokens')
    ax1.tick_params(axis='y', labelcolor=color_tok)

    ax2 = ax1.twinx()
    ax2.set_ylabel('Total Runs', color=color_run, fontsize=11)
    ax2.plot(df_hist['date'], df_hist['total_runs'], color=color_run,
             marker='s', linewidth=2, linestyle='--', label='Total Runs')
    ax2.tick_params(axis='y', labelcolor=color_run)

    lines1, labels1 = ax1.get_legend_handles_labels()
    lines2, labels2 = ax2.get_legend_handles_labels()
    ax1.legend(lines1 + lines2, labels1 + labels2, loc='upper left')
    fig.suptitle('📈 Copilot Token Usage — Daily Trend', fontsize=14, fontweight='bold')
    plt.xticks(rotation=30)
    plt.tight_layout()
    plt.savefig(f'{CHARTS_DIR}/daily_trend.png', dpi=150, bbox_inches='tight', facecolor='white')
    plt.close()
    print(f'✅ daily_trend.png saved ({len(df_hist)} data points)')
else:
    print(f'ℹ️  Only {len(hist_rows)} history point(s) — daily_trend requires ≥2')

# --- Chart 3: Workflow heatmap (>=3 data points) ---
if os.path.exists(wf_file) and len(hist_rows) >= 3:
    all_wf = []
    with open(wf_file) as f:
        for line in f:
            line = line.strip()
            if line:
                all_wf.append(json.loads(line))
    if all_wf:
        df_all = pd.DataFrame(all_wf)
        df_all['date'] = pd.to_datetime(df_all['date'])
        top8 = df_all.groupby('workflow')['tokens'].sum().nlargest(8).index.tolist()
        df_heat = df_all[df_all['workflow'].isin(top8)].copy()
        recent_dates = sorted(df_heat['date'].unique())[-14:]  # last 14 days
        df_heat = df_heat[df_heat['date'].isin(recent_dates)]
        pivot = df_heat.pivot_table(index='workflow', columns='date',
                                    values='tokens', aggfunc='sum', fill_value=0)
        pivot.columns = [d.strftime('%m/%d') for d in pivot.columns]
        fig, ax = plt.subplots(figsize=(max(10, len(pivot.columns) * 0.9), 6), dpi=150)
        sns.heatmap(pivot, cmap='YlOrRd', annot=True, fmt='.0f',
                    linewidths=0.5, ax=ax, cbar_kws={'label': 'Tokens'})
        ax.set_title('🗓️ Workflow Token Heatmap — Top-8 Workflows', fontsize=14, fontweight='bold')
        ax.set_xlabel('Date', fontsize=11)
        ax.set_ylabel('Workflow', fontsize=11)
        plt.tight_layout()
        plt.savefig(f'{CHARTS_DIR}/workflow_heatmap.png', dpi=150, bbox_inches='tight', facecolor='white')
        plt.close()
        print(f'✅ workflow_heatmap.png saved ({len(pivot)} workflows × {len(pivot.columns)} dates)')
    else:
        print('ℹ️  No multi-day workflow data yet — heatmap requires ≥3 history points')
else:
    print(f'ℹ️  Only {len(hist_rows)} history point(s) — heatmap requires ≥3')
```

Run the script:
```bash
python3 /tmp/gh-aw/python/token_charts.py
```

After the script succeeds, upload each generated chart using the `upload asset` safe-output tool. **Check file existence before uploading**:
- If `/tmp/gh-aw/python/charts/top_consumers.png` exists: upload it → save URL as `TOP_CONSUMERS_URL`
- If `/tmp/gh-aw/python/charts/daily_trend.png` exists: upload it → save URL as `DAILY_TREND_URL`
- If `/tmp/gh-aw/python/charts/workflow_heatmap.png` exists: upload it → save URL as `HEATMAP_URL`

Skip the upload call entirely for any chart that was not generated.

### Phase 4: Create Report Issue

Create an issue with the title format: `YYYY-MM-DD` (date only — the prefix `📊 Copilot Token Usage Report:` is automatically added).

#### Issue Body Structure

```markdown
### Summary

Analyzed **[N]** Copilot workflow runs from **[DATE]** covering **[M]** unique workflows.
Total: **[TOTAL_TOKENS]** tokens (~**$[TOTAL_COST]**) across **[TOTAL_TURNS]** turns.

### 📊 Token Usage Charts

#### 🔥 Top Consumers
![Top Consumers](TOP_CONSUMERS_URL)

#### 📈 Daily Trend
_(Include this section only when DAILY_TREND_URL is available — requires ≥ 2 historical data points)_
![Daily Token Trend](DAILY_TREND_URL)

#### 🗓️ Workflow Heatmap
_(Include this section only when HEATMAP_URL is available — requires ≥ 3 historical data points)_
![Workflow Heatmap](HEATMAP_URL)

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
