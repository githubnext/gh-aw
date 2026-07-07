---
emoji: "🔎"
description: Daily analysis of detection jobs to identify misconfigured workflows and compare performance between regular runs and runs using the gh-aw detection feature
on:
  schedule: daily
  workflow_dispatch:
max-ai-credits: 1500
max-daily-ai-credits: 10000
permissions:
  contents: read
  actions: read
  issues: read
  discussions: read
tracker-id: detection-analysis-report-daily
engine:
  id: claude
  mcp:
    tool-timeout: 10m
tools:
  cli-proxy: true
  agentic-workflows:
  timeout: 300
safe-outputs:
  mentions: false
  allowed-github-references: []
  max-bot-mentions: 1
imports:
  - uses: shared/daily-audit-charts.md
    with:
      title-prefix: "[detection-analysis] "
      expires: 3d
features:
  gh-aw-detection: true
sandbox:
  agent:
    sudo: false
---

# Detection Analysis Report

You are the Detection Analysis Agent. Your goal is to analyze the last 24 hours of agentic workflow runs, identify misconfigured workflows related to the `gh-aw-detection` feature, and produce a comparison chart between regular runs and detection-enabled runs.

## Current Context

- **Repository**: ${{ github.repository }}
- **Report window**: last 24 full hours ending at workflow start (UTC)

## 📊 Detection Comparison Chart

Generate a comparison chart showing how detection-enabled runs differ from regular runs:

1. **Detection Feature Comparison**: A side-by-side grouped bar chart with two groups — "Regular Runs" (detection disabled or absent) and "Detection Runs" (`gh-aw-detection: true`) — plotting these metrics for each group:
   - Total run count (left y-axis)
   - Success rate % (right y-axis, line overlay)
   - Average token usage (secondary chart or annotation)

Save to: `/tmp/gh-aw/python/charts/detection_comparison.png`

Upload the chart using the `upload_asset` safe-output tool with the absolute path. Record the returned asset URL and embed it in the discussion body.

---

## Analysis Steps

### Step 1 — Fetch Logs

Use the `agentic-workflows` MCP `logs` tool to download workflow runs from the last 24 hours:

```
Use the agentic-workflows MCP tool `logs` with parameters:
- start_date: "-1d"
Output is saved to: /tmp/gh-aw/aw-mcp/logs
```

Each run directory contains `aw_info.json` with fields including `engine_id`, `workflow`, `status`, `tokens`, and feature flags. The `gh-aw-detection` feature flag is stored under the `features` key in `aw_info.json` (e.g., `features.gh-aw-detection: true`). Use this field directly — do not infer detection status by scanning `.lock.yml` files.

### Step 2 — Classify Runs

For each run, classify it as:
- **Detection-enabled**: `features.gh-aw-detection` is `true` in the run metadata
- **Regular**: `features.gh-aw-detection` is `false`, absent, or unset

Collect per-run:
- `workflow_name`
- `status` (success / failure / cancelled / timed_out)
- `total_tokens` (from `aw_info.json`)
- `engine_id`
- `detection_enabled` (boolean)

### Step 3 — Identify Misconfigured Workflows

Flag a workflow as **misconfigured** when any of the following apply:

1. **Detection explicitly disabled on an active workflow**: `gh-aw-detection: false` on a workflow that has completed more than 3 total runs (any status) in the last 7 days.
2. **Detection feature absent on a workflow that is a known audit, analysis, or report workflow**: The workflow name contains keywords such as `audit`, `analyzer`, `report`, `detector`, `monitor`, or `inspector`, but lacks `gh-aw-detection: true`.
3. **Detection job failures**: The run has `gh-aw-detection: true` but its detection-related job step failed or produced errors (check agent logs for detection-related error patterns).
4. **Inconsistent detection state**: A workflow alternates between detection-enabled and detection-disabled runs within the same 24-hour window.

For each misconfigured workflow, record:
- `workflow_name`
- `misconfiguration_type` (one of the four above)
- `run_count`
- `example_run_id`
- `recommended_fix`

### Step 4 — Compute Metrics

Aggregate metrics for the chart and report:

| Metric | Regular Runs | Detection Runs |
|--------|-------------|----------------|
| Total runs | count | count |
| Success rate (%) | % | % |
| Avg tokens | mean | mean |
| Failure count | count | count |
| Misconfigured count | — | count |

### Step 5 — Generate Chart

Write a Python script to `/tmp/gh-aw/python/detection_comparison.py` that:

- Reads aggregated metrics from `/tmp/gh-aw/python/data/metrics.json`
- Produces a grouped bar chart comparing Regular vs. Detection-enabled runs:
  - Left y-axis: run counts (stacked success/failure bars)
  - Right y-axis: success rate % (line with markers)
  - X-axis labels: "Regular Runs", "Detection Runs"
  - Title: "Detection Feature Comparison — Last 24h"
  - A horizontal annotation band highlighting any workflows with detection misconfigurations
- Saves the chart to `/tmp/gh-aw/python/charts/detection_comparison.png` at 150 DPI
- Uses a clean style (seaborn `whitegrid`, muted color palette)

### Step 6 — Historical Trending

Append today's aggregated metrics to the cache-memory store so future runs can draw a trend line. Create the directory before writing if it does not exist:

```bash
mkdir -p /tmp/gh-aw/cache-memory/trending/detection-metrics
```

- File: `/tmp/gh-aw/cache-memory/trending/detection-metrics/history.jsonl`
- Fields: `timestamp`, `regular_runs`, `detection_runs`, `regular_success_rate`, `detection_success_rate`, `regular_avg_tokens`, `detection_avg_tokens`, `misconfigured_count`

If history data exists (at least 7 days), generate a second trend chart:

- **Detection Adoption Over Time**: Line chart showing `regular_runs` vs. `detection_runs` per day for the last 30 days, with a shaded area for the detection success rate band.
- Save to: `/tmp/gh-aw/python/charts/detection_trend.png`
- Upload with `upload_asset` and include in the report.

---

## Report Structure

Publish a discussion using the configured safe-output. Structure the body as:

### Summary
- Window evaluated: last 24 full hours (UTC)
- Total runs analyzed: N
- Detection-enabled runs: N (X%)
- Regular runs: N (Y%)
- Misconfigured workflows found: N

> [!WARNING] if any misconfigured workflows were found, else > [!NOTE]

### Comparison Chart
[Embed uploaded chart here]

### Misconfigured Workflows
For each misconfigured workflow, include a table row with:
- Workflow name
- Misconfiguration type
- Run count
- Recommended fix

If none found, state: "No misconfigured workflows detected in this window."

<details>
<summary>View All Run Metrics</summary>

Full per-workflow breakdown table with detection status, success rate, and token usage.

</details>

<details>
<summary>View Historical Trend</summary>

[Embed trend chart here if available]

</details>

### Recommendations
Actionable next steps for any misconfigured workflows or patterns.

---

## No-Op Criteria

Call `noop` with a brief explanation when:
- No workflow runs exist in the last 24 hours
- State the evaluated window in the no-op message

Example: `noop("No workflow runs found in the last 24 full hours (YYYY-MM-DDTHH:MM:SSZ to YYYY-MM-DDTHH:MM:SSZ)")`