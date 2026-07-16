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

Call `agenticworkflows logs --start-date -1d` to download workflow run directories to `/tmp/gh-aw/aw-mcp/logs`. Each run's `aw_info.json` contains `features["gh-aw-detection"]` (boolean), `status`, `total_tokens`, and `engine_id`. Use `features.gh-aw-detection` directly — do not infer detection status from `.lock.yml` files.

### Analyze Runs

In a single pass over the run directories at `/tmp/gh-aw/aw-mcp/logs`, classify and aggregate all runs:

**Classify each run** using `features.gh-aw-detection` from `aw_info.json`:
- **Detection-enabled**: value is `true`
- **Regular**: value is `false`, absent, or unset

Collect per-run: `workflow_name`, `status`, `total_tokens`, `engine_id`, `detection_enabled`.

**Flag a workflow as misconfigured** when any of the following apply:
1. `gh-aw-detection: false` on a workflow with >3 total runs in the last 7 days
2. Workflow name contains `audit`, `analyzer`, `report`, `detector`, `monitor`, or `inspector` but lacks `gh-aw-detection: true`
3. Run has `gh-aw-detection: true` but detection-related steps failed
4. Workflow alternates between detection-enabled and detection-disabled within the 24h window

For each misconfigured workflow, record: `workflow_name`, `misconfiguration_type`, `run_count`, `example_run_id`, `recommended_fix`.

**Save** aggregated metrics to `/tmp/gh-aw/python/data/metrics.json`:

```json
{
  "regular_runs": <count>,
  "detection_runs": <count>,
  "regular_success_rate": <percent>,
  "detection_success_rate": <percent>,
  "regular_failure_count": <count>,
  "detection_failure_count": <count>,
  "regular_avg_tokens": <mean>,
  "detection_avg_tokens": <mean>,
  "misconfigured_count": <count>
}
```

### Step 5 — Generate Chart

```bash
python3 .github/scripts/detection_comparison.py \
  --metrics /tmp/gh-aw/python/data/metrics.json \
  --output /tmp/gh-aw/python/charts/detection_comparison.png
```

Upload the chart using the `upload_asset` safe-output tool. Record the returned asset URL and embed it in the discussion body.

### Step 6 — Historical Trending

Use the `update-detection-trending` agent to append today's metrics to the cache-memory trending history and optionally generate a trend chart.

If `/tmp/gh-aw/python/charts/detection_trend.png` was generated, upload it with `upload_asset` and embed in the report under "View Historical Trend".

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

---

## agent: `update-detection-trending`
---
model: small
description: Appends detection metrics to cache-memory trending history and generates a trend chart when ≥7 days of data exist
---
Append today's detection metrics to the cache-memory trending history store.

Create the directory before writing:

```bash
mkdir -p /tmp/gh-aw/cache-memory/trending/detection-metrics
```

Read `/tmp/gh-aw/python/data/metrics.json` and append one JSON line to `/tmp/gh-aw/cache-memory/trending/detection-metrics/history.jsonl`:

```json
{"timestamp": "<ISO-8601 UTC>", "regular_runs": N, "detection_runs": N, "regular_success_rate": X.X, "detection_success_rate": X.X, "regular_avg_tokens": X, "detection_avg_tokens": X, "misconfigured_count": N}
```

If the history file has ≥7 entries, generate a 30-day trend chart at `/tmp/gh-aw/python/charts/detection_trend.png`:
- Line chart: `regular_runs` vs. `detection_runs` per day for the last 30 days
- Shaded area band for the detection success rate
- Style: seaborn whitegrid, 12×7 inches, 150 DPI