---
description: Daily unified security observability report combining firewall traffic analysis and DIFC integrity-filtered event analysis
on:
  schedule:
    # Every day at 10am UTC
    - cron: daily
  workflow_dispatch:

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
  discussions: read
  security-events: read

tracker-id: daily-security-observability
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
  cli-proxy: true
  agentic-workflows:
  github:
    toolsets:
      - all
  bash:
    - "*"
  edit:

safe-outputs:
  upload-asset:
    max: 5
    allowed-exts: [.png, .jpg, .jpeg, .svg]

timeout-minutes: 60

imports:
  - uses: shared/daily-audit-charts.md
    with:
      title-prefix: "[security-observability] "
  - shared/python-dataviz.md

---
{{#runtime-import? .github/shared-instructions.md}}

# Daily Security Observability Report

You are a security observability analyst. Your job is to produce a unified daily security intelligence report that combines two signals:

1. **Firewall traffic analysis** — which domains and requests were allowed or blocked across all agentic workflow runs
2. **DIFC integrity-filtered event analysis** — which tool calls were blocked by the Data Integrity and Flow Control system, with statistical charts and actionable tuning recommendations

Both datasets cover the **last 7 days** and share the cache-memory path `/tmp/gh-aw/cache-memory/security-observability/`.

## Context

- **Repository**: ${{ github.repository }}
- **Run ID**: ${{ github.run_id }}
- **Analysis window**: Last 7 days

---

## Phase 1: Collect Firewall-Enabled Workflow Runs

### Step 1.1: Collect Recent Firewall-Enabled Workflow Runs

**ALWAYS PERFORM FRESH ANALYSIS**: This report must always use fresh data from the audit tool. Do NOT skip analysis based on cached results or reuse aggregated statistics from previous runs.

Use the `logs` tool from the agentic-workflows MCP server to collect workflow runs that have firewall enabled:

**Tool call:**
```json
{
  "firewall": true,
  "start_date": "-7d",
  "count": 100
}
```

### Step 1.2: Early Exit if No Firewall Data

If Step 1.1 returns zero workflow runs, note this in the final report as "No firewall-enabled workflow runs found in the past 7 days." and proceed directly to Phase 3 (DIFC analysis). Do not skip the final report.

---

## Phase 2: Analyze Firewall Logs

### Step 2.1: Fetch Firewall Audit Data

For each run collected in Phase 1, call the `audit` tool with the run_id to get detailed firewall information:

```json
{
  "run_id": 12345678
}
```

The audit tool returns:
- `firewall_analysis.blocked_domains` — blocked domain names
- `firewall_analysis.allowed_domains` — allowed domain names
- `firewall_analysis.total_requests` / `blocked_requests` / `allowed_requests`
- `firewall_analysis.requests_by_domain` — per-domain statistics
- `policy_analysis` (when present) — rule-level attribution with `rule_hits`, `denied_requests`, `policy_summary`

**Important:** Do NOT manually download and parse firewall log files. Always use the `audit` tool.

### Step 2.2: Aggregate Firewall Results

Combine data from all runs:
1. Build a master list of all blocked domains with frequency counts and which workflows blocked them
2. Calculate overall statistics:
   - Total workflows analyzed (`workflow_runs_analyzed`)
   - Total blocked domains (`firewall_domains_blocked`) — unique count
   - Total blocked requests (`firewall_requests_blocked`)
   - Total allowed requests (`firewall_requests_allowed`)
3. If `policy_analysis` is present, aggregate rule hit counts and denied requests with rule attribution across all runs

### Step 2.3: Generate Firewall Trend Charts

Create CSV files in `/tmp/gh-aw/python/data/` and generate exactly **2 firewall charts**:

**Chart 1: Firewall Request Trends**
- Stacked area or multi-line chart: allowed requests (green) vs blocked requests (red) over the last 30 days
- Save as: `/tmp/gh-aw/python/charts/firewall_requests_trends.png`

**Chart 2: Top Blocked Domains Frequency**
- Horizontal bar chart: top 10–15 most frequently blocked domains with block counts
- Save as: `/tmp/gh-aw/python/charts/blocked_domains_frequency.png`

**Chart quality**: DPI 300 minimum, 12×7 inches, seaborn styling, clear labels.

Upload both charts using `upload_asset` and record the returned URLs.

---

## Phase 3: Collect DIFC Integrity-Filtered Events

### Step 3.1: Check for DIFC Data

Read `/tmp/gh-aw/integrity/filtered-logs.json`. If the array is empty (no runs found in the last 7 days), note "No DIFC integrity-filtered events found in the last 7 days." and proceed directly to Phase 5 (combined report).

### Step 3.2: Fetch Detailed DIFC Gateway Data

1. Read `/tmp/gh-aw/integrity/filtered-logs.json` and extract all run IDs from each entry's `databaseId` field.
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

3. Annotate each event with `workflow_name` (from `workflowName`) and `run_id` (from `databaseId`).
4. Save all annotated events to `/tmp/gh-aw/integrity/all-events.json`.

### Step 3.3: Bucketize DIFC Events

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

with open(f"{DATA_DIR}/all-events.json") as f:
    events = json.load(f)

if not events:
    print("No events to analyze.")
    summary = {"total": 0, "by_tool": {}, "by_server": {}, "by_reason": {}, "by_hour": {}, "by_day": {}, "by_workflow": {}, "by_user": {}}
    with open(f"{DATA_DIR}/summary.json", "w") as f:
        json.dump(summary, f, indent=2)
    exit(0)

for e in events:
    try:
        e["_dt"] = datetime.fromisoformat(e["timestamp"].replace("Z", "+00:00"))
    except Exception:
        e["_dt"] = None

by_tool      = Counter(e["tool_name"]   for e in events if e.get("tool_name"))
by_server    = Counter(e["server_id"]   for e in events if e.get("server_id"))
by_reason    = Counter(e["reason"]      for e in events if e.get("reason"))
by_workflow  = Counter(e.get("workflow_name", "unknown") for e in events)
by_user      = Counter(e.get("author_login", "unknown") for e in events)

by_hour = Counter()
by_day  = Counter()
for e in events:
    if e["_dt"]:
        by_hour[e["_dt"].strftime("%Y-%m-%dT%H:00")] += 1
        by_day[e["_dt"].strftime("%Y-%m-%d")] += 1

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

---

## Phase 4: Generate DIFC Statistical Charts

Create and run chart scripts using matplotlib/seaborn. Save all charts to `/tmp/gh-aw/integrity/charts/`.

```bash
mkdir -p /tmp/gh-aw/integrity/charts
```

### Chart 3: DIFC Events Over Time (Daily)

Create `/tmp/gh-aw/integrity/chart_timeline.py`:

```python
#!/usr/bin/env python3
"""Chart 3: DIFC filtered events per day."""
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
    print("No daily data; skipping chart 3.")
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
print("Chart 3 saved.")
```

Run: `python3 /tmp/gh-aw/integrity/chart_timeline.py`

### Chart 4: Top Filtered Tools (Horizontal Bar)

Create `/tmp/gh-aw/integrity/chart_tools.py`:

```python
#!/usr/bin/env python3
"""Chart 4: Top tools that trigger DIFC filtering."""
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
    print("No tool data; skipping chart 4.")
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
print("Chart 4 saved.")
```

Run: `python3 /tmp/gh-aw/integrity/chart_tools.py`

### Chart 5: Filter Reason Breakdown (Pie / Donut)

Create `/tmp/gh-aw/integrity/chart_reasons.py`:

```python
#!/usr/bin/env python3
"""Chart 5: Breakdown of filter reasons and integrity/secrecy tags."""
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
print("Chart 5 saved.")
```

Run: `python3 /tmp/gh-aw/integrity/chart_reasons.py`

### Upload DIFC Charts

Upload each generated DIFC chart using the `upload asset` tool and collect the returned URLs:
1. Upload `/tmp/gh-aw/integrity/charts/events_timeline.png`
2. Upload `/tmp/gh-aw/integrity/charts/top_tools.png`
3. Upload `/tmp/gh-aw/integrity/charts/reasons_tags.png`

---

## Phase 5: Generate Combined Security Observability Report

Create a single GitHub discussion combining both signals.

**Title**: `[security-observability] Daily Security Observability Report — YYYY-MM-DD`

**Body** (use h3 and lower for all headers per reporting guidelines):

```markdown
### Executive Summary

[2–3 paragraph overview combining both signals: firewall traffic patterns and DIFC integrity filtering activity for the last 7 days. Highlight the most significant findings from each domain and any cross-cutting themes (e.g., the same workflow appearing in both firewall blocks and DIFC filtering).]

---

## 🔥 Firewall Analysis

### Key Firewall Metrics

| Metric | Value |
|--------|-------|
| Workflows analyzed (firewall-enabled) | [N] |
| Total network requests monitored | [N] |
| ✅ Allowed requests | [N] |
| 🚫 Blocked requests | [N] |
| Block rate | [N]% |
| Total unique blocked domains | [N] |

### 📈 Firewall Request Trends

![Firewall Request Trends](URL_CHART_1)

[2–3 sentence analysis of firewall activity trends, noting increases in blocked traffic or changes in patterns]

### Top Blocked Domains

![Blocked Domains Frequency](URL_CHART_2)

[Brief 2–3 sentence analysis of frequently blocked domains, identifying potential security concerns or overly restrictive rules]

#### Most Frequently Blocked Domains

| Domain | Times Blocked | Workflows | Category |
|--------|--------------|-----------|----------|
[Top 20 domains sorted by block count descending]

[When policy_analysis is available:]
#### Policy Rule Attribution

📋 Policy: [policy_summary from most recent run]

| Rule | Action | Description | Total Hits |
|------|--------|-------------|------------|
[Rules sorted by hits descending, 🟢 for allow / 🔴 for deny]

<details>
<summary>View Detailed Request Patterns by Workflow</summary>

[Per-workflow firewall breakdown]

</details>

<details>
<summary>View Complete Blocked Domains List</summary>

[Alphabetical full list of unique blocked domains]

</details>

### 🔒 Firewall Security Recommendations

[Actionable recommendations: domains to allowlist, suspicious domains, policy rule improvements, workflows needing network permission updates]

---

## 🔒 DIFC Integrity Analysis

### Key DIFC Metrics

| Metric | Value |
|--------|-------|
| Total filtered events | [N] |
| Unique tools filtered | [N] |
| Unique workflows affected | [N] |
| Most common filter reason | [reason] |
| Busiest day | [YYYY-MM-DD] ([N] events) |

### 📈 DIFC Events Over Time

![DIFC Events Timeline](URL_CHART_3)

[2–3 sentence analysis: is there a trend? any spikes?]

### 🔧 Top Filtered Tools

![Top Filtered Tools](URL_CHART_4)

[Brief analysis: which tools trigger the most filtering and why]

### 🏷️ Filter Reasons and Tags

![Filter Reasons and Tags](URL_CHART_5)

[Analysis of integrity vs. secrecy filtering and top tags]

<details>
<summary>📋 Per-Workflow DIFC Breakdown</summary>

| Workflow | Filtered Events |
|----------|----------------|
[one row per workflow sorted by count descending]

</details>

<details>
<summary>📋 Per-Server DIFC Breakdown</summary>

| MCP Server | Filtered Events |
|------------|----------------|
[one row per server sorted by count descending]

</details>

<details>
<summary>👤 Per-User DIFC Breakdown</summary>

| Author Login | Filtered Events |
|--------------|----------------|
[one row per user sorted by count descending]

</details>

### 💡 DIFC Tuning Recommendations

[Numbered list of specific, actionable recommendations derived from DIFC analysis.]

---

*Generated by the Daily Security Observability workflow (consolidated from Daily Firewall Reporter + Daily DIFC Analyzer)*
*Analysis window: Last 7 days | Repository: ${{ github.repository }}*
*Run: https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }}*
```

## Important

**Always** call a safe-output tool at the end of your run. If both datasets are empty, call `noop`:

```json
{"noop": {"message": "No firewall-enabled runs and no DIFC integrity-filtered events found in the last 7 days; no report generated."}}
```

{{#runtime-import shared/noop-reminder.md}}
