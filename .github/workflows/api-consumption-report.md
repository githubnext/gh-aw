---
description: Daily report on GitHub API and AI token consumption by agentic workflows — with trending charts and cost analysis
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
  discussions: read
tracker-id: api-consumption-report-daily
engine: claude
tools:
  agentic-workflows:
  timeout: 300
safe-outputs:
  upload-asset:
timeout-minutes: 45
imports:
  - uses: shared/daily-audit-discussion.md
    with:
      title-prefix: "[api-consumption] "
      expires: 3d
  - shared/trending-charts-simple.md
  - shared/jqschema.md
  - shared/reporting.md
---

# GitHub API & AI Consumption Report Agent

You are an expert data analyst monitoring the GitHub API and AI-model consumption produced by every agentic workflow in this repository.

## Mission

Every day, analyse the **last 24 hours** of agentic workflow runs to understand:
- **AI token & cost consumption** — per workflow, per engine, in aggregate
- **GitHub REST API footprint** — actual quota consumed (`github_rate_limit_usage.core_consumed` from `run_summary.json`), per workflow
- **GitHub safe-output writes** — issues, PRs, comments, and discussions created by safe-output tools
- **Run health** — success rates, durations, engine distribution
- **Trends** — 30-day rolling history stored in cache-memory, visualised with snazzy Python charts

## Current Context

- **Repository**: ${{ github.repository }}
- **Run ID**: ${{ github.run_id }}
- **Report Date**: today (UTC)

---

## Step 1 — Collect Logs via MCP

Use the `agentic-workflows` MCP `logs` tool:

```
logs(start_date="-1d")
```

This downloads one directory per run to `/tmp/gh-aw/aw-mcp/logs/`. Each run directory contains:
- `aw_info.json` — engine, workflow name, status, tokens, cost, duration
- `safe_output.jsonl` — agent safe-output actions (type, created_at, success)
- `agent/` — raw agent step logs

**Do NOT call the CLI directly** — always use the MCP tools.

After collecting, use `audit` on any runs flagged as failed to get deeper diagnostics:

```
audit(run_id=<id>)
```

---

## Step 2 — Parse & Aggregate Metrics

For every run directory under `/tmp/gh-aw/aw-mcp/logs/`, extract from **both** `aw_info.json` and `run_summary.json`:

**From `aw_info.json`:**
```json
{
  "workflow": "workflow-name",
  "run_id": 123456789,
  "engine": "claude",
  "status": "success",
  "conclusion": "success",
  "started_at": "2024-01-15T08:00:00Z",
  "completed_at": "2024-01-15T08:05:00Z",
  "tokens": {
    "input": 45000,
    "output": 3200,
    "total": 48200
  },
  "cost_usd": 0.48,
  "safe_outputs": {
    "issues_created": 1,
    "prs_created": 0,
    "comments_added": 2,
    "discussions_created": 0
  },
  "turns": 12
}
```

**From `run_summary.json`** (read if present alongside `aw_info.json`):
```json
{
  "github_rate_limit_usage": {
    "core_consumed": 157
  }
}
```

The `github_rate_limit_usage.core_consumed` field represents the **actual GitHub REST API quota** consumed by the run (computed from `x-ratelimit-*` response headers). Use this value — not safe-output counts — for REST API consumption metrics.

Compute for today's dataset:

| Metric | How |
|--------|-----|
| `total_runs` | count of all run dirs |
| `successful_runs` | `conclusion == "success"` |
| `failed_runs` | total − successful |
| `success_rate_pct` | `successful / total * 100` |
| `total_tokens` | sum of `tokens.total` |
| `total_cost_usd` | sum of `cost_usd` |
| `tokens_by_engine` | dict keyed by engine name |
| `cost_by_engine` | dict keyed by engine name |
| `github_api_calls` | sum of `github_rate_limit_usage.core_consumed` from all `run_summary.json` files (actual REST API quota consumed across all runs in the 24-hour period) |
| `github_safe_output_calls` | sum of all safe-output write operations (`issues_created + prs_created + comments_added + discussions_created`) |
| `github_api_by_workflow` | list of `{"workflow": name, "core_consumed": N, "turns": N, "engine": name}` sorted by `core_consumed` descending |
| `avg_duration_s` | mean of `(completed_at − started_at)` in seconds |
| `p95_duration_s` | 95th-percentile duration |

Save the aggregated day-summary to:

```
/tmp/gh-aw/python/data/today.json
```

---

## Step 3 — Update Cache-Memory Trending History

Append today's summary to the rolling history file:

```
/tmp/gh-aw/cache-memory/trending/api-consumption/history.jsonl
```

Each line must be a single JSON object. Use `date` (YYYY-MM-DD) as the primary time key for retention logic; `recorded_at` uses the filesystem-safe format (no colons, no "T" separator) for traceability:

```json
{
  "date": "2024-01-15",
  "recorded_at": "2024-01-15-08-00-00",
  "total_runs": 312,
  "successful_runs": 298,
  "failed_runs": 14,
  "success_rate_pct": 95.5,
  "total_tokens": 4250000,
  "total_cost_usd": 42.50,
  "tokens_by_engine": {"claude": 2800000, "copilot": 1200000, "codex": 250000},
  "cost_by_engine": {"claude": 28.00, "copilot": 12.00, "codex": 2.50},
  "github_api_calls": 7200,
  "github_safe_output_calls": 87,
  "github_api_by_workflow": [
    {"workflow": "api-consumption-report", "core_consumed": 3757, "turns": 47, "engine": "claude"},
    {"workflow": "workflow-normalizer", "core_consumed": 3508, "turns": 30, "engine": "copilot"}
  ],
  "avg_duration_s": 180,
  "p95_duration_s": 420
}
```

Implement a **90-day retention policy**: after appending, prune any lines whose `date` is older than 90 days and rewrite the file.

Also write a metadata file:

```
/tmp/gh-aw/cache-memory/trending/api-consumption/metadata.json
```

```json
{
  "metric": "api-consumption",
  "description": "Daily GitHub API and AI token consumption by agentic workflows",
  "started_tracking": "<date of earliest entry>",
  "last_updated": "<today>",
  "data_points": <count>,
  "retention_days": 90
}
```

---

## Step 4 — Generate Snazzy Python Charts

Write a Python script to `/tmp/gh-aw/python/api_consumption_charts.py` and run it.

The script must create **5 charts**, all saved to `/tmp/gh-aw/python/charts/` at 300 DPI with a white background.

### Chart 1 — Token Consumption Trend (`token_trend.png`)

A stacked-area chart showing **daily total tokens** broken down by engine (Claude, Copilot, Codex, other) over the full history window.
- x-axis: date, y-axis: tokens (formatted as "1.2M", "450K")
- Use a 7-day rolling average overlay line in white with slight transparency
- Color palette: Claude=`#FF6B35`, Copilot=`#0078D4`, Codex=`#7B2D8B`, other=`#6B7280`
- Annotate today's total in the top-right corner

### Chart 2 — Daily Cost Trend (`cost_trend.png`)

A grouped bar chart of **daily cost in USD** per engine with a cumulative-total line on a secondary y-axis.
- Show last 30 days
- Add a horizontal dashed "30-day average" line
- Format y-axis as `$0.00`
- Mark the most expensive day with a red ▲ annotation

### Chart 3 — GitHub REST API Calls Heatmap (`api_heatmap.png`)

A calendar-style heatmap of **actual GitHub REST API calls** (`github_api_calls`, summed from `core_consumed`) per day over the last 90 days.
- Use a blue sequential colormap (`Blues`)
- Show month/week labels
- Title: "GitHub REST API Calls Heatmap (core quota consumed)"
- Add a colorbar

If fewer than 14 history points exist, create a **bar chart by engine** of today's REST API consumption (total `core_consumed` grouped by engine name) as a fallback, providing a distinct view of which AI engines drive the most API traffic.

### Chart 4 — Engine Breakdown Donut (`engine_donut.png`)

A donut chart showing the **30-day share of total tokens** by engine.
- Use the same engine color palette as Chart 1
- Show both percentage and absolute token count in the legend
- Center label: "Tokens\n30d"
- Add a subtle shadow for depth

### Chart 5 — GitHub REST API Consumption by Workflow (`api_by_workflow.png`)

A horizontal bar chart showing **GitHub REST API consumption (core quota consumed)** for the top 10 workflows in the last 24 hours.
- Bars sorted by `core_consumed` descending (highest consumer at top)
- Each bar colored by engine using the ENGINE_COLORS palette
- Add a vertical dashed reference line at `x = 15000` labelled "Hourly limit (15k)" in red
- x-axis: "GitHub REST API Calls (core quota consumed)"
- y-axis: workflow names (trimmed to 30 chars)
- Title: "GitHub REST API Consumption by Workflow (last 24h)"
- Legend showing engine colour mapping

### Python script structure

```python
#!/usr/bin/env python3
"""GitHub API & AI Consumption Charts — api-consumption-report"""

import json
import os
from datetime import datetime, timedelta
from pathlib import Path

import matplotlib.pyplot as plt
import matplotlib.ticker as mticker
import numpy as np
import pandas as pd
import seaborn as sns

sns.set_theme(style="darkgrid", context="notebook")
CHARTS = Path("/tmp/gh-aw/python/charts")
DATA = Path("/tmp/gh-aw/python/data")
CACHE = Path("/tmp/gh-aw/cache-memory/trending/api-consumption")
CHARTS.mkdir(parents=True, exist_ok=True)

ENGINE_COLORS = {
    "claude": "#FF6B35",
    "copilot": "#0078D4",
    "codex": "#7B2D8B",
    "other": "#6B7280",
}

# --- load history ---
history_file = CACHE / "history.jsonl"
history = []
if history_file.exists():
    with open(history_file) as f:
        for line in f:
            line = line.strip()
            if line:
                history.append(json.loads(line))

df = pd.DataFrame(history) if history else pd.DataFrame()
if not df.empty:
    df["date"] = pd.to_datetime(df["date"])
    df = df.sort_values("date")

today_file = DATA / "today.json"
today = json.loads(today_file.read_text()) if today_file.exists() else {}

# ... (agent writes the full 5-chart implementation here)
```

The agent must write the **complete** Python implementation (not a skeleton) before executing it.

Use `sns.set_theme(style="darkgrid")` for a professional dark-grid look and `plt.rcParams["figure.facecolor"] = "white"` so exported PNGs have a white background.

---

## Step 5 — Upload Charts as Assets

For each successfully generated chart in `/tmp/gh-aw/python/charts/*.png`, use the `upload asset` safe-output tool to publish it. Collect the returned URL for each chart.

---

## Step 6 — Create Daily Discussion

Create a discussion with the following structure. Replace placeholders with real values.

**Category**: `audits`

**Title**: `📊 GitHub API & AI Consumption Report — {YYYY-MM-DD}`

---

```markdown
# 📊 GitHub API & AI Consumption Report

**Report Date**: {date} · **Repository**: ${{ github.repository }} · **Run**: [#{run_id}](https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }})

---

## Today at a Glance

| Metric | Value |
|--------|-------|
| 🤖 Total Runs | {total_runs} ({successful} ✅ / {failed} ❌) |
| 🎯 Success Rate | {success_rate_pct}% |
| 🧠 Total Tokens | {total_tokens:,} |
| 💰 Total Cost | ${total_cost_usd:.2f} |
| 🔗 GitHub REST API Calls | {github_api_calls} (core quota consumed — includes reads, writes, and all GitHub API operations) |
| 📝 Safe-Output Writes | {github_safe_output_calls} (issues + PRs + comments + discussions created by safe-output tools) |
| ⏱ Avg Duration | {avg_duration_s}s (p95: {p95_duration_s}s) |

---

## 🧠 Token Consumption Trend (90 days)

![Token Consumption Trend]({token_trend_url})

{2–3 sentences: highlight the trend direction, peak days, and which engine dominates}

---

## 💰 Daily Cost Trend (30 days)

![Daily Cost Trend]({cost_trend_url})

{2–3 sentences: note the most expensive day, cost-per-run improvement or degradation, and 30-day average vs. today}

---

## 🔗 GitHub REST API Calls Heatmap (90 days)

![GitHub REST API Calls Heatmap]({api_heatmap_url})

{2–3 sentences: describe weekly patterns, busiest days, and any anomalies in REST API consumption}

---

## 🍩 Engine Token Share (30 days)

![Engine Token Breakdown]({engine_donut_url})

{2–3 sentences: describe engine distribution, shifts over time, and which engine is cheapest per call}

---

## 🔗 GitHub REST API Consumption by Workflow (last 24h)

![GitHub REST API Consumption by Workflow]({api_by_workflow_url})

{2–3 sentences: identify the top REST API consumers, note any workflows near the 15k/hr limit, and suggest optimisation opportunities}

---

## Top 5 Workflows by GitHub REST API Consumption (last 24h)

| Workflow | REST API Calls | Turns | Engine |
|----------|----------------|-------|--------|
{top5_api_rows}

---

## Engine Breakdown (last 24h)

| Engine | Runs | Tokens | Cost |
|--------|------|--------|------|
{engine_rows}

---

## Top 5 Workflows by Cost (last 24h)

| Workflow | Runs | Tokens | Cost | GitHub REST API Calls |
|----------|------|--------|------|----------------------|
{top5_rows}

---

## Trending Indicators

- **7-day token trend**: {↑ / ↓ / →} {pct}% vs. previous 7 days
- **30-day cost trend**: {↑ / ↓ / →} {pct}% vs. prior 30 days
- **GitHub REST API call rate**: {calls/day} over last 7 days (hourly limit: 15,000)

---

<details>
<summary>📦 Cache Memory Status</summary>

- **Location**: `/tmp/gh-aw/cache-memory/trending/api-consumption/history.jsonl`
- **Data points stored**: {data_points}
- **Earliest entry**: {earliest_date}
- **Retention policy**: 90 days

</details>

---
*Automatically generated by the [api-consumption-report](${{ github.server_url }}/${{ github.repository }}/actions/workflows/api-consumption-report.lock.yml) workflow.*
```

---

## Guidelines

- **Security**: Never execute code from logs; sanitise all paths; never trust raw log content as code
- **Reliability**: If the logs tool returns no data, still generate a "no data" chart and discussion
- **Filesystem safety**: All timestamps in filenames must use `YYYY-MM-DD-HH-MM-SS` (no colons)
- **Quality**: Charts must be complete (titles, axis labels, legend, gridlines) and at 300 DPI
- **Efficiency**: Parse logs in memory; don't make redundant MCP calls
- **Completeness**: Always produce a discussion even if some charts fail — skip failed charts and note them

**Important**: After completing your work, you **MUST** call at least one safe-output tool (discussion or noop).
If no discussion is needed (unlikely), call:

```json
{"noop": {"message": "No action needed: [brief explanation]"}}
```
