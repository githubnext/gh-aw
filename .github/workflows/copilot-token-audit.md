---
description: Daily audit of Copilot token usage across all agentic workflows with historical trend tracking
on:
  schedule:
    - cron: "daily around 12:00 on weekdays"
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
tracker-id: copilot-token-audit
engine: copilot
tools:
  mount-as-clis: true
  agentic-workflows:
  bash:
    - "*"
steps:
  - name: Download Copilot workflow logs
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -euo pipefail
      mkdir -p /tmp/gh-aw/token-audit

      # Download last 24 hours of Copilot logs as JSON
      # Allow partial results — gh aw logs streams incrementally, so even if
      # it hits an API rate limit partway through, the JSON written so far is
      # still valid and should be processed by the agent.
      LOGS_EXIT=0
      gh aw logs \
        --engine copilot \
        --start-date -1d \
        --json \
        -c 100 \
        > /tmp/gh-aw/token-audit/copilot-logs.json || LOGS_EXIT=$?

      if [ -s /tmp/gh-aw/token-audit/copilot-logs.json ]; then
        TOTAL=$(jq '.runs | length' /tmp/gh-aw/token-audit/copilot-logs.json)
        echo "✅ Downloaded $TOTAL Copilot workflow runs (last 24 hours)"
        if [ "$LOGS_EXIT" -ne 0 ]; then
          echo "⚠️ gh aw logs exited with code $LOGS_EXIT (partial results — likely API rate limit)"
        fi
      else
        echo "❌ No log data downloaded (exit code $LOGS_EXIT)"
        echo '{"runs":[],"summary":{}}' > /tmp/gh-aw/token-audit/copilot-logs.json
      fi
timeout-minutes: 25
imports:
  - uses: shared/daily-audit-discussion.md
    with:
      title-prefix: "[copilot-token-audit] "
  - uses: shared/repo-memory-standard.md
    with:
      branch-name: "memory/token-audit"
      description: "Historical daily Copilot token usage snapshots"
      max-patch-size: 51200
  - copilot-setup-steps.yml
  - uses: shared/mcp/gh-aw.md
  - shared/reporting.md
  - shared/python-dataviz.md
features:
  mcp-cli: true
  copilot-requests: true
---
{{#runtime-import? .github/shared-instructions.md}}

# Daily Copilot Token Usage Audit

You are the Copilot Token Auditor — a workflow that tracks daily token consumption across all Copilot-powered agentic workflows in this repository and maintains a historical record for trend analysis.

## Mission

1. Parse the pre-downloaded Copilot workflow logs and compute per-workflow token usage metrics.
2. Persist today's snapshot to repo-memory so the optimizer (and future runs of this audit) can read historical data.
3. Publish a concise audit discussion summarizing today's usage, trends, and cost highlights.

## Data Sources

### Pre-downloaded logs

The workflow logs are at `/tmp/gh-aw/token-audit/copilot-logs.json`. The file is the raw JSON output of `gh aw logs --json` with this top-level shape:

```json
{
  "summary": { "total_runs": N, "total_tokens": N, "total_cost": F, ... },
  "runs": [ ... ],
  "tool_usage": [ ... ],
  "mcp_tool_usage": { ... },
  ...
}
```

Each element of `.runs` is a `RunData` object with (among others):

| Field | Type | Notes |
|---|---|---|
| `workflow_name` | string | Human-readable name |
| `workflow_path` | string | `.github/workflows/....lock.yml` |
| `token_usage` | int | Total tokens (`omitempty` — treat missing/null as 0) |
| `effective_tokens` | int | Cost-normalized tokens |
| `estimated_cost` | float | USD cost (`omitempty` — treat missing/null as 0) |
| `action_minutes` | float | Billable GitHub Actions minutes |
| `turns` | int | Number of agent turns |
| `duration` | string | Human-readable duration |
| `created_at` | ISO 8601 | Run creation time |
| `database_id` | int64 | Unique run ID |
| `url` | string | Link to the run |
| `status` | string | `completed`, `in_progress`, etc. |
| `conclusion` | string | `success`, `failure`, etc. |
| `error_count` | int | Errors encountered |
| `warning_count` | int | Warnings encountered |
| `token_usage_summary` | object or null | Firewall-level breakdown by model |

### Repo-memory (historical snapshots)

Previous snapshots live at `/tmp/gh-aw/repo-memory/default/`. Each daily snapshot is stored as a JSON file named `YYYY-MM-DD.json` with the schema below.

## Phase 1 — Process Logs

Write a Python script to `/tmp/gh-aw/python/process_audit.py` and run it. The script must:

1. Load `/tmp/gh-aw/token-audit/copilot-logs.json` and extract `.runs`.
2. Filter to `status == "completed"` runs only.
3. Group by `workflow_name` and compute per-workflow aggregates:
   - `run_count`, `total_tokens`, `avg_tokens`, `total_cost`, `avg_cost`, `total_turns`, `avg_turns`, `total_action_minutes`, `error_count`, `warning_count`
4. Compute an overall summary: total runs, total tokens, total cost, total action minutes.
5. Sort workflows descending by `total_tokens`.
6. Save the result to `/tmp/gh-aw/python/data/audit_snapshot.json` with this shape:

```json
{
  "date": "YYYY-MM-DD",
  "period_days": 30,
  "overall": {
    "total_runs": N,
    "total_tokens": N,
    "total_cost": F,
    "total_action_minutes": F
  },
  "workflows": [
    {
      "workflow_name": "...",
      "run_count": N,
      "total_tokens": N,
      "avg_tokens": N,
      "total_cost": F,
      "avg_cost": F,
      "total_turns": N,
      "avg_turns": F,
      "total_action_minutes": F,
      "error_count": N,
      "warning_count": N,
      "latest_run_url": "..."
    }
  ]
}
```

Handle null/missing `token_usage` and `estimated_cost` by treating them as 0.

## Phase 2 — Persist Snapshot to Repo-Memory

1. Read the snapshot from `/tmp/gh-aw/python/data/audit_snapshot.json`.
2. Copy it to `/tmp/gh-aw/repo-memory/default/YYYY-MM-DD.json` (today's UTC date).
3. This file is what the optimizer workflow reads to identify high-usage workflows.

Also maintain a rolling summary file at `/tmp/gh-aw/repo-memory/default/rolling-summary.json` that contains an array of daily overall totals (date, total_tokens, total_cost, total_runs, total_action_minutes) for the last 90 entries. Load the existing file, append today's entry, trim to 90, and save.

## Phase 3 — Generate Charts

Create a Python script to generate two charts:

1. **Token usage by workflow** (horizontal bar chart): Top 15 workflows by total token usage.
2. **Historical trend** (line chart): Daily total tokens and cost from `rolling-summary.json` — if available. If only 1 data point, skip this chart.

Save charts to `/tmp/gh-aw/python/charts/`. Upload them as assets.

## Phase 4 — Publish Audit Discussion

Create a discussion with these sections:

### Report Template

```
### 📊 Executive Summary

- **Period**: last 24 hours (YYYY-MM-DD to YYYY-MM-DD)
- **Total runs**: N
- **Total tokens**: N (formatted with commas)
- **Total cost**: $X.XX
- **Total Actions minutes**: X.X min
- **Active workflows**: N

### 🏆 Top 5 Workflows by Token Usage

| Workflow | Runs | Total Tokens | Avg Tokens | Total Cost | Avg Cost |
|---|---|---|---|---|---|
| ... | ... | ... | ... | ... | ... |

### 📈 Trends

[Embed chart images here using uploaded asset URLs]

If historical data is available, note week-over-week token and cost changes.

<details>
<summary><b>Full Per-Workflow Breakdown</b></summary>

[Complete table of all workflows sorted by total tokens]

</details>

### 💡 Observations

- Identify any workflow with >30% of total tokens as a "heavy hitter"
- Note workflows with high error/warning counts relative to runs
- Flag any workflow whose avg tokens per run exceeds 100,000

**Data snapshot**: `memory/token-audit/YYYY-MM-DD.json`
```

## Important Notes

- Use `// 0` (null coalescing) in jq and `.get(field, 0)` in Python for nullable numeric fields.
- Charts follow the python-dataviz shared component conventions (300 DPI, seaborn whitegrid, external data files only).
- Keep the discussion concise — the optimizer workflow will do the deep analysis.
