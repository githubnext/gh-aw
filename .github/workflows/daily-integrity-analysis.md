---
description: Daily analysis of DIFC integrity-filtered events with statistical charts and actionable tuning recommendations
on:
  schedule:
    - cron: daily
  workflow_dispatch:

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
  discussions: read

tracker-id: daily-integrity-analysis
engine: copilot

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
  - name: Download integrity-filtered logs
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      mkdir -p /tmp/gh-aw/integrity
      # Download logs filtered to only runs with DIFC integrity-filtered events
      gh aw logs --filtered-integrity --start-date -7d --json -c 200 \
        > /tmp/gh-aw/integrity/filtered-logs.json

      if [ -f /tmp/gh-aw/integrity/filtered-logs.json ]; then
        count=$(jq '. | length' /tmp/gh-aw/integrity/filtered-logs.json 2>/dev/null || echo 0)
        echo "✅ Downloaded $count runs with integrity-filtered events"
      else
        echo "⚠️ No logs file produced; continuing with empty dataset"
        echo "[]" > /tmp/gh-aw/integrity/filtered-logs.json
      fi

tools:
  agentic-workflows:
  bash:
    - "*"

timeout-minutes: 30

imports:
  - uses: shared/daily-audit-discussion.md
    with:
      title-prefix: "[integrity] "
  - shared/reporting.md
  - shared/python-dataviz.md
---
{{#runtime-import? .github/shared-instructions.md}}

# Daily DIFC Integrity-Filtered Events Analyzer

You are an integrity-system analyst. Your job is to analyze DIFC (Data Integrity and Flow Control) filtered events collected from agentic workflow runs, produce statistical charts that reveal patterns, and provide actionable tuning recommendations.

## Context

- **Repository**: ${{ github.repository }}
- **Run ID**: ${{ github.run_id }}
- **Data file**: `/tmp/gh-aw/integrity/filtered-logs.json` (pre-downloaded runs with DIFC integrity-filtered events)
- **Analysis window**: Last 7 days

## Step 0: Check for Data

Read `/tmp/gh-aw/integrity/filtered-logs.json`. If the array is empty (no runs found in the last 7 days), call `noop` with the message "No DIFC integrity-filtered events found in the last 7 days." and stop.

## Step 1: Parse and Bucketize Events

The JSON file is an array of workflow run objects. Each run object contains `databaseId` (the numeric run ID) and `workflowName`. Use the `audit` tool from the agentic-workflows MCP server to get the detailed gateway data for each run. Then write a Python script to bucketize events.

### 1.1 Fetch Detailed Gateway Data

1. Read `/tmp/gh-aw/integrity/filtered-logs.json` and extract all run IDs by iterating over the array and collecting each entry's `databaseId` field.
2. For each run ID, call the `audit` tool to get its detailed DIFC filtered events:

```json
{
  "run_id": 12345678
}
```

The audit result contains `gateway_analysis.filtered_events[]` with fields:
- `timestamp` — ISO 8601 timestamp
- `server_id` — MCP server that was filtered
- `tool_name` — tool call that was blocked
- `reason` — reason for filtering (e.g., `integrity`, `secrecy`)
- `integrity_tags` — integrity labels applied
- `secrecy_tags` — secrecy labels applied
- `author_association` — contributor association of the triggering actor
- `author_login` — login of the triggering actor

3. For each event returned, annotate it with two additional fields from the corresponding run entry in `filtered-logs.json`: `workflow_name` (from `workflowName`) and `run_id` (from `databaseId`). This allows the Python analysis to group events by workflow.
4. Collect all annotated filtered events across all runs and save them to `/tmp/gh-aw/integrity/all-events.json`.

### 1.2 Python Bucketization Script

Create and run `/tmp/gh-aw/integrity/bucketize.py`:

```python
#!/usr/bin/env python3
"""Bucketize and centralize DIFC integrity-filtered events for statistical analysis."""
import json
import os
from collections import defaultdict, Counter
from datetime import datetime, timedelta

DATA_DIR = "/tmp/gh-aw/integrity"
os.makedirs(DATA_DIR, exist_ok=True)

# Load all events
with open(f"{DATA_DIR}/all-events.json") as f:
    events = json.load(f)

if not events:
    print("No events to analyze.")
    summary = {"total": 0, "by_tool": {}, "by_server": {}, "by_reason": {}, "by_hour": {}, "by_day": {}, "by_workflow": {}, "by_user": {}}
    with open(f"{DATA_DIR}/summary.json", "w") as f:
        json.dump(summary, f, indent=2)
    exit(0)

# Parse timestamps
for e in events:
    try:
        e["_dt"] = datetime.fromisoformat(e["timestamp"].replace("Z", "+00:00"))
    except Exception:
        e["_dt"] = None

# Buckets
by_tool      = Counter(e["tool_name"]   for e in events if e.get("tool_name"))
by_server    = Counter(e["server_id"]   for e in events if e.get("server_id"))
by_reason    = Counter(e["reason"]      for e in events if e.get("reason"))
by_workflow  = Counter(e.get("workflow_name", "unknown") for e in events)
by_user      = Counter(e.get("author_login", "unknown") for e in events)

# Time-based buckets
by_hour = Counter()
by_day  = Counter()
for e in events:
    if e["_dt"]:
        by_hour[e["_dt"].strftime("%Y-%m-%dT%H:00")] += 1
        by_day[e["_dt"].strftime("%Y-%m-%d")] += 1

# Integrity tag breakdown
all_integrity_tags = Counter()
all_secrecy_tags   = Counter()
for e in events:
    for tag in (e.get("integrity_tags") or []):
        all_integrity_tags[tag] += 1
    for tag in (e.get("secrecy_tags") or []):
        all_secrecy_tags[tag] += 1

summary = {
    "total": len(events),
    "by_tool":           dict(by_tool.most_common()),
    "by_server":         dict(by_server.most_common()),
    "by_reason":         dict(by_reason.most_common()),
    "by_workflow":       dict(by_workflow.most_common()),
    "by_user":           dict(by_user.most_common()),
    "by_hour":           dict(sorted(by_hour.items())),
    "by_day":            dict(sorted(by_day.items())),
    "integrity_tags":    dict(all_integrity_tags.most_common()),
    "secrecy_tags":      dict(all_secrecy_tags.most_common()),
}

with open(f"{DATA_DIR}/summary.json", "w") as f:
    json.dump(summary, f, indent=2)

print(f"Bucketized {len(events)} events.")
print(json.dumps(summary, indent=2))
```

Run the script: `python3 /tmp/gh-aw/integrity/bucketize.py`

## Step 2: Generate Statistical Charts

Create and run chart scripts using matplotlib/seaborn. Save all charts to `/tmp/gh-aw/integrity/charts/`.

```bash
mkdir -p /tmp/gh-aw/integrity/charts
```

### Chart 1: Events Over Time (Daily)

Create `/tmp/gh-aw/integrity/chart_timeline.py`:

```python
#!/usr/bin/env python3
"""Chart 1: DIFC filtered events per day."""
import json, os
import matplotlib.pyplot as plt
import matplotlib.dates as mdates
import seaborn as sns
from datetime import datetime

DATA_DIR   = "/tmp/gh-aw/integrity"
CHARTS_DIR = f"{DATA_DIR}/charts"
os.makedirs(CHARTS_DIR, exist_ok=True)

with open(f"{DATA_DIR}/summary.json") as f:
    summary = json.load(f)

by_day = summary.get("by_day", {})
if not by_day:
    print("No daily data; skipping chart 1.")
    exit(0)

dates  = [datetime.strptime(d, "%Y-%m-%d") for d in sorted(by_day)]
counts = [by_day[d.strftime("%Y-%m-%d")] for d in dates]

sns.set_style("whitegrid")
fig, ax = plt.subplots(figsize=(12, 5), dpi=300)
ax.bar(dates, counts, color="#4A90D9", edgecolor="white", linewidth=0.8)
ax.xaxis.set_major_formatter(mdates.DateFormatter("%b %d"))
ax.xaxis.set_major_locator(mdates.DayLocator())
plt.xticks(rotation=45, ha="right")
ax.set_title("DIFC Integrity-Filtered Events — Last 7 Days", fontsize=16, fontweight="bold", pad=14)
ax.set_xlabel("Date", fontsize=13)
ax.set_ylabel("Event Count", fontsize=13)
ax.grid(True, axis="y", alpha=0.4)
plt.tight_layout()
plt.savefig(f"{CHARTS_DIR}/events_timeline.png", dpi=300, bbox_inches="tight", facecolor="white")
print("Chart 1 saved.")
```

Run: `python3 /tmp/gh-aw/integrity/chart_timeline.py`

### Chart 2: Top Filtered Tools (Horizontal Bar)

Create `/tmp/gh-aw/integrity/chart_tools.py`:

```python
#!/usr/bin/env python3
"""Chart 2: Top tools that trigger DIFC filtering."""
import json, os
import matplotlib.pyplot as plt
import seaborn as sns

DATA_DIR   = "/tmp/gh-aw/integrity"
CHARTS_DIR = f"{DATA_DIR}/charts"
os.makedirs(CHARTS_DIR, exist_ok=True)

with open(f"{DATA_DIR}/summary.json") as f:
    summary = json.load(f)

by_tool = summary.get("by_tool", {})
if not by_tool:
    print("No tool data; skipping chart 2.")
    exit(0)

items   = sorted(by_tool.items(), key=lambda x: x[1], reverse=True)[:15]
tools   = [i[0] for i in items]
counts  = [i[1] for i in items]

sns.set_style("whitegrid")
fig, ax = plt.subplots(figsize=(12, max(5, len(tools) * 0.55)), dpi=300)
bars = ax.barh(tools[::-1], counts[::-1], color="#E8714A", edgecolor="white", linewidth=0.8)
for bar, val in zip(bars, counts[::-1]):
    ax.text(bar.get_width() + 0.1, bar.get_y() + bar.get_height() / 2,
            str(val), va="center", fontsize=11, fontweight="bold")
ax.set_title("Top Filtered Tool Calls (DIFC)", fontsize=16, fontweight="bold", pad=14)
ax.set_xlabel("Event Count", fontsize=13)
ax.set_ylabel("Tool Name", fontsize=13)
ax.grid(True, axis="x", alpha=0.4)
plt.tight_layout()
plt.savefig(f"{CHARTS_DIR}/top_tools.png", dpi=300, bbox_inches="tight", facecolor="white")
print("Chart 2 saved.")
```

Run: `python3 /tmp/gh-aw/integrity/chart_tools.py`

### Chart 3: Filter Reason Breakdown (Pie / Donut)

Create `/tmp/gh-aw/integrity/chart_reasons.py`:

```python
#!/usr/bin/env python3
"""Chart 3: Breakdown of filter reasons and integrity/secrecy tags."""
import json, os
import matplotlib.pyplot as plt
import seaborn as sns

DATA_DIR   = "/tmp/gh-aw/integrity"
CHARTS_DIR = f"{DATA_DIR}/charts"
os.makedirs(CHARTS_DIR, exist_ok=True)

with open(f"{DATA_DIR}/summary.json") as f:
    summary = json.load(f)

by_reason      = summary.get("by_reason", {})
integrity_tags = summary.get("integrity_tags", {})
secrecy_tags   = summary.get("secrecy_tags", {})

sns.set_style("whitegrid")
fig, axes = plt.subplots(1, 2, figsize=(14, 6), dpi=300)

# Left: filter reasons pie
if by_reason:
    labels = list(by_reason.keys())
    values = list(by_reason.values())
    colors = sns.color_palette("husl", len(labels))
    axes[0].pie(values, labels=labels, colors=colors, autopct="%1.1f%%",
                startangle=140, pctdistance=0.82,
                wedgeprops=dict(width=0.6))
    axes[0].set_title("Filter Reason Distribution", fontsize=14, fontweight="bold")
else:
    axes[0].text(0.5, 0.5, "No reason data", ha="center", va="center")
    axes[0].set_title("Filter Reason Distribution", fontsize=14, fontweight="bold")

# Right: top integrity/secrecy tags bar
all_tags = {**{f"[I] {k}": v for k, v in integrity_tags.items()},
            **{f"[S] {k}": v for k, v in secrecy_tags.items()}}
if all_tags:
    tag_items = sorted(all_tags.items(), key=lambda x: x[1], reverse=True)[:10]
    tag_names  = [i[0] for i in tag_items]
    tag_counts = [i[1] for i in tag_items]
    colors2    = ["#4A90D9" if t.startswith("[I]") else "#E8714A" for t in tag_names]
    axes[1].barh(tag_names[::-1], tag_counts[::-1], color=colors2[::-1], edgecolor="white")
    axes[1].set_title("Top Integrity [I] & Secrecy [S] Tags", fontsize=14, fontweight="bold")
    axes[1].set_xlabel("Count", fontsize=12)
    axes[1].grid(True, axis="x", alpha=0.4)
else:
    axes[1].text(0.5, 0.5, "No tag data", ha="center", va="center")
    axes[1].set_title("Top Integrity & Secrecy Tags", fontsize=14, fontweight="bold")

fig.suptitle("DIFC Filter Analysis — Reason & Tag Breakdown", fontsize=16, fontweight="bold", y=1.01)
plt.tight_layout()
plt.savefig(f"{CHARTS_DIR}/reasons_tags.png", dpi=300, bbox_inches="tight", facecolor="white")
print("Chart 3 saved.")
```

Run: `python3 /tmp/gh-aw/integrity/chart_reasons.py`

### Chart 4: Per-User Filtered Events (Horizontal Bar)

Create `/tmp/gh-aw/integrity/chart_users.py`:

```python
#!/usr/bin/env python3
"""Chart 4: Top users that trigger DIFC filtering."""
import json, os
import matplotlib.pyplot as plt
import seaborn as sns

DATA_DIR   = "/tmp/gh-aw/integrity"
CHARTS_DIR = f"{DATA_DIR}/charts"
BAR_COLOR  = "#4CAF50"
os.makedirs(CHARTS_DIR, exist_ok=True)

with open(f"{DATA_DIR}/summary.json") as f:
    summary = json.load(f)

by_user = summary.get("by_user", {})
if not by_user:
    print("No user data; skipping chart 4.")
    exit(0)

items   = sorted(by_user.items(), key=lambda x: x[1], reverse=True)[:20]
users   = [i[0] for i in items]
counts  = [i[1] for i in items]

sns.set_style("whitegrid")
fig, ax = plt.subplots(figsize=(12, max(5, len(users) * 0.55)), dpi=300)
bars = ax.barh(users[::-1], counts[::-1], color=BAR_COLOR, edgecolor="white", linewidth=0.8)
for bar, val in zip(bars, counts[::-1]):
    ax.text(bar.get_width() + 0.1, bar.get_y() + bar.get_height() / 2,
            str(val), va="center", fontsize=11, fontweight="bold")
ax.set_title("Top Users Triggering DIFC Filtering (Top 20)", fontsize=16, fontweight="bold", pad=14)
ax.set_xlabel("Event Count", fontsize=13)
ax.set_ylabel("Author Login", fontsize=13)
ax.grid(True, axis="x", alpha=0.4)
plt.tight_layout()
plt.savefig(f"{CHARTS_DIR}/top_users.png", dpi=300, bbox_inches="tight", facecolor="white")
print("Chart 4 saved.")
```

Run: `python3 /tmp/gh-aw/integrity/chart_users.py`

## Step 3: Upload Charts

Upload each generated chart using the `upload asset` tool and collect the returned URLs:
1. Upload `/tmp/gh-aw/integrity/charts/events_timeline.png`
2. Upload `/tmp/gh-aw/integrity/charts/top_tools.png`
3. Upload `/tmp/gh-aw/integrity/charts/reasons_tags.png`
4. Upload `/tmp/gh-aw/integrity/charts/top_users.png`

## Step 4: Generate Tuning Recommendations

Based on the summary data, derive actionable recommendations. For each top filtered tool or server:
- Is the tool legitimately called on untrusted data? If yes, recommend applying integrity tags.
- Are secrecy-tagged artifacts being passed to tools that don't need them? Recommend narrowing tool access.
- Is the filter rate increasing? Recommend reviewing recent prompt changes.
- Are there bursts (many events in one hour)? Investigate the workflow(s) involved.

## Step 5: Create Discussion Report

Create a GitHub discussion with the full analysis.

**Title**: `[integrity] DIFC Integrity-Filtered Events Report — YYYY-MM-DD`

**Body** (use h3 and lower for all headers per reporting guidelines):

```markdown
### Executive Summary

In the last 7 days, **[N]** DIFC integrity-filtered events were detected across **[W]** workflow runs. The most frequently filtered tool was **[tool_name]** ([X] events), and the dominant filter reason was **[reason]**. [Describe the overall trend: stable/increasing/decreasing. Highlight any notable spike or pattern that warrants attention.]

[Describe which workflows or MCP servers contributed the most events and what that suggests about how the integrity system is being exercised. If the filtering rate is high for a particular server, explain whether this is expected or a sign of misconfiguration.]

### Key Metrics

| Metric | Value |
|--------|-------|
| Total filtered events | [N] |
| Unique tools filtered | [N] |
| Unique workflows affected | [N] |
| Most common filter reason | [reason] |
| Busiest day | [YYYY-MM-DD] ([N] events) |

### 📈 Events Over Time

![DIFC Events Timeline](URL_CHART_1)

[2–3 sentence analysis: is there a trend? any spikes?]

### 🔧 Top Filtered Tools

![Top Filtered Tools](URL_CHART_2)

[Brief analysis: which tools trigger the most filtering and why]

### 🏷️ Filter Reasons and Tags

![Filter Reasons and Tags](URL_CHART_3)

[Analysis of integrity vs. secrecy filtering and top tags]

<details>
<summary>📋 Per-Workflow Breakdown</summary>

| Workflow | Filtered Events |
|----------|----------------|
[one row per workflow sorted by count descending]

</details>

<details>
<summary>📋 Per-Server Breakdown</summary>

| MCP Server | Filtered Events |
|------------|----------------|
[one row per server sorted by count descending]

</details>

<details>
<summary>👤 Per-User Breakdown</summary>

| Author Login | Filtered Events |
|--------------|----------------|
[one row per user sorted by count descending]

</details>

### 🔍 Per-User Analysis

![Per-User Filtered Events](URL_CHART_4)

[Analysis of per-user filtering: identify whether spikes are driven by automated actors (e.g., `github-actions[bot]`, Copilot agents) or human contributors. Highlight any single user or bot account responsible for a disproportionate share of filtered events and suggest whether this indicates expected automation behaviour or warrants investigation.]

### 💡 Tuning Recommendations

[Numbered list of specific, actionable recommendations derived from the analysis. Examples:
- Tools appearing in top filtered: consider whether they need access to integrity-tagged data
- High secrecy filter rate: review which workflows pass secrets to tools
- Increasing trend: monitor and review recent prompt or tool permission changes
- Workflow-specific spikes: examine the triggering events for that workflow]

---
*Generated by the Daily Integrity Analysis workflow*
*Analysis window: Last 7 days | Repository: ${{ github.repository }}*
*Run: https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }}*
```

## Important

**Always** call a safe-output tool at the end of your run. If no events were found, call `noop`:

```json
{"noop": {"message": "No DIFC integrity-filtered events found in the last 7 days; no report generated."}}
```
