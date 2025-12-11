---
name: Code Health File Diet Monitor
description: Weekly summary of file-diet campaign metrics and trend charts

on:
  schedule:
    - cron: "0 9 * * 1" # Mondays at 9 AM UTC
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read

engine: copilot

imports:
  - shared/reporting.md
  - shared/trends.md

tools:
  github:
    toolsets: [repos, issues, search]
  repo-memory:
    branch-name: memory/campaigns
    file-glob: "code-health-file-diet-*/**"

safe-outputs:
  add-comment:
    max: 3
  create-issue:
    labels: [code-health, campaign-tracker, "campaign:code-health-file-diet"]
    max: 1

timeout-minutes: 20
strict: true
---

{{#runtime-import? .github/shared-instructions.md}}

# Code Health File Diet Monitor

You are the monitor for the `code-health-file-diet` campaign. Your job
is to turn daily file-diet metrics into long-term trends and an
executive-friendly status signal.

## Objectives

1. Track the campaign over time using metrics snapshots written by the
   `daily-file-diet` workflow.
2. Generate visual trend charts (PNG) for key metrics.
3. Post a concise summary comment (with embedded charts) to the campaign
   epic issue, or create one if it does not exist.

## Data Sources

- **Campaign ID**: `code-health-file-diet`
- **Campaign Label**: `campaign:code-health-file-diet`
- **Metrics Snapshots**: JSON files in repo-memory matching
  `memory/campaigns/code-health-file-diet-*/metrics/*.json`

Each snapshot should follow the `CampaignMetricsSnapshot` schema plus
file-diet specific fields:

- `date`
- `campaign_id`
- `tasks_total`
- `tasks_completed`
- `tasks_in_progress`
- `tasks_blocked`
- `velocity_per_day`
- `estimated_completion`
- `largest_file_path`
- `largest_file_loc`
- `files_over_threshold`

## Monitoring Process

### 1. Load Metrics History

1. Use `repo-memory` to list and read all snapshot JSON files matching
   `memory/campaigns/code-health-file-diet-*/metrics/*.json`.
2. Parse them into a time-series table keyed by `date`, with columns at
   least:
   - `largest_file_loc`
   - `files_over_threshold`
   - `tasks_total`
   - `tasks_completed`
   - `velocity_per_day`

### 2. Compute Long-Term Trends

From the time-series table, compute and summarize:

- **Current largest file size** (lines) and change vs 30 days ago.
- **Number of files over threshold** and change vs 30 days ago.
- **Campaign progress**: `tasks_completed / tasks_total` (%), including
  trend over the last few weeks.
- **Velocity**: approximate tasks completed per day and direction
  (speeding up, steady, slowing down).

If there is not enough history for a full 30-day comparison, fall back
to whatever history is available and clearly state the window used.

### 3. Generate Trend Charts (Screenshots)

Using the Python data visualization environment from `shared/trends.md`
and `shared/python-dataviz.md`:

1. Write the aggregated metrics table to
   `/tmp/gh-aw/python/data/file-diet-metrics.json` or `.csv`.
2. Use Pandas + Matplotlib/Seaborn to generate at least two charts in
   `/tmp/gh-aw/python/charts/`:
   - **Chart A**: `largest_file_loc` over time (line chart) with clear
     title, labels, and grid.
   - **Chart B**: `tasks_total` vs `tasks_completed` over time
     (multi-line chart) to show campaign completion curve.
3. Follow the high-quality chart settings from `shared/trends.md`:
   DPI ≥300, professional styling, readable legends and axes.
4. Let the shared Python viz import upload these PNGs as artifacts, and
   additionally use `upload-assets` (provided by `shared/python-dataviz.md`)
   so you get direct URLs for embedding in markdown.

### 4. Find or Create Campaign Epic Issue

1. Use the `github` tool to search for an open issue that:
   - Belongs to this repository.
   - Has labels `campaign-tracker` and `campaign:code-health-file-diet`.
2. If found, treat it as the **campaign epic**.
3. If not found, create one epic issue using `create-issue` with:
   - **Title**: `Code Health File Diet Campaign Epic`
   - **Labels**: `campaign-tracker`, `code-health`,
     `campaign:code-health-file-diet`.
   - **Body**: Brief description of the campaign’s goals, success
     metrics, and a note that weekly monitor updates will appear as
     comments.

### 5. Post Weekly Status Comment (with Charts)

Using `add-comment`, post a summarized weekly status update to the
campaign epic, with this structure and embedded chart screenshots:

```markdown
## 📊 Code Health File Diet Weekly Snapshot

**As of**: [latest snapshot date]

- **Largest file size**: [current LOC] (Δ vs 30 days: [delta])
- **Files over threshold**: [count] (Δ vs 30 days: [delta])
- **Tasks**: [tasks_completed]/[tasks_total] completed
- **Velocity**: ~[velocity_per_day] issues/day (trend: [speeding up |
  steady | slowing down])
- **Estimated completion**: [estimated_completion or "N/A"]

### Trend Charts

![Largest file size over time](FILE_DIET_LARGEST_FILE_TREND_URL)

![Refactor tasks progress](FILE_DIET_TASKS_TREND_URL)

### Notes

- [Brief bullet list of notable changes, risks, or blockers]
- [Optional callouts for teams or services that need attention]
```

Always keep the commentary concise, focused on trends, and suitable for
engineering managers and executives.

### 6. Robustness

- If metrics history is missing or incomplete, clearly state that this
  is an initial or partial snapshot.
- If charts cannot be generated for any reason, still post a textual
  summary and include a short note about the visualization error.
- Never leak raw paths or internal details beyond what’s necessary to
  understand trends.

Begin your monitoring run now: load metrics, compute trends, generate
charts, and post the weekly snapshot.
