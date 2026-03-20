---
description: Guidelines for creating agentic workflows that generate charts and trend visualizations using Python scientific computing libraries with persistent historical data.
---

# Data Science & Chart Generation

Consult this file when creating an agentic workflow that generates charts, visualizations, or trend analysis — including data dashboards, metric reports, time-series plots, or any Python-based visualization output.

## Frontmatter Template

```yaml
---
description: [what the workflow visualizes]
on:
  schedule:
    - cron: "0 9 * * 1"   # example: every Monday at 09:00 UTC
  workflow_dispatch:
permissions:
  contents: read
  actions: read          # add issues/discussions scopes when needed
engine: copilot
tools:
  cache-memory:
    key: trending-data-${{ github.workflow }}-${{ github.run_id }}
  bash:
    - "*"
network:
  allowed:
    - defaults
    - python
safe-outputs:
  upload-asset:
  create-issue:          # or create-discussion for gallery-style reports
    title-prefix: "📊 [Report Name]:"
    labels: [report]
    close-older-issues: true
    expires: 30
steps:
  - name: Setup Python environment
    run: |
      mkdir -p /tmp/gh-aw/python/{data,charts,artifacts}
      mkdir -p /tmp/gh-aw/cache-memory/trending
      pip install --user --quiet numpy pandas matplotlib seaborn scipy
  - name: Upload charts
    if: always()
    uses: actions/upload-artifact@v7.0.0
    with:
      name: data-charts
      path: /tmp/gh-aw/python/charts/*.png
      if-no-files-found: warn
      retention-days: 30
  - name: Upload source files and data
    if: always()
    uses: actions/upload-artifact@v7.0.0
    with:
      name: python-source-and-data
      path: |
        /tmp/gh-aw/python/*.py
        /tmp/gh-aw/python/data/*
      if-no-files-found: warn
      retention-days: 30
---
```

## Environment

| Location | Purpose |
|---|---|
| `/tmp/gh-aw/python/` | Working directory for scripts |
| `/tmp/gh-aw/python/data/` | Input data files (CSV, JSON) |
| `/tmp/gh-aw/python/charts/` | Generated chart images (PNG) |
| `/tmp/gh-aw/python/artifacts/` | Additional output files |
| `/tmp/gh-aw/cache-memory/trending/` | Persistent time-series history across runs |

**Libraries**: NumPy · Pandas · Matplotlib · Seaborn · SciPy

Charts and Python source files are automatically uploaded as GitHub Actions artifacts (`data-charts`, `python-source-and-data`, retention 30 days) so they are available even if the workflow fails.

## Writing the Agent Prompt

A well-structured prompt for a data-visualization workflow has these phases:

### Phase 1 – Load historical data
```markdown
1. Check `/tmp/gh-aw/cache-memory/trending/<metric>/history.jsonl` for existing data.
2. Load it into a Pandas DataFrame if it exists; otherwise start fresh.
```

### Phase 2 – Collect or generate data
```markdown
1. Collect today's metrics from the GitHub API (or generate sample data with NumPy).
2. Save raw data to `/tmp/gh-aw/python/data/<metric>.csv` or `.json` — never inline data in Python code.
```

### Phase 3 – Append to history (JSON Lines)
```markdown
Append a new record to history.jsonl with ISO 8601 timestamp, metric name, value, and metadata.
Implement a 90-day retention policy to prevent unbounded growth.
```

### Phase 4 – Generate charts
```markdown
1. Create trend charts if ≥ 2 historical data points exist:
   - Time-series line chart with 7-day moving average
   - Comparative trend chart for multiple metrics
2. Fall back to bar/distribution charts when history is empty.
3. Save all charts to `/tmp/gh-aw/python/charts/` at DPI 300, seaborn style.
```

### Phase 5 – Upload and report
```markdown
1. Upload each chart using the `upload asset` tool.
2. Create an issue (or discussion) embedding the uploaded chart URLs in markdown.
3. If no meaningful data was found, call `noop` with a brief explanation.
```

## Data Rules

**CRITICAL**: Data must never be inlined in Python code. Always write data to an external file first, then load it with pandas:

```python
# ❌ PROHIBITED
data = [10, 20, 30, 40, 50]

# ✅ REQUIRED
import pandas as pd
data = pd.read_csv('/tmp/gh-aw/python/data/metrics.csv')
```

## Trending Patterns

Cache-memory at `/tmp/gh-aw/cache-memory/trending/` persists across runs. Organize it as:

```
/tmp/gh-aw/cache-memory/trending/
├── <metric-name>/
│   ├── history.jsonl      # Time-series data (one JSON object per line)
│   ├── metadata.json      # Data schema and description
│   └── last_updated.txt   # Timestamp of last update
└── index.json             # Index of all tracked metrics
```

### Append a daily data point

```python
import json
from datetime import datetime

point = {
    "timestamp": datetime.now().isoformat(),
    "metric": "issue_count",
    "value": 42,
    "metadata": {"source": "github_api"}
}
with open('/tmp/gh-aw/cache-memory/trending/issues/history.jsonl', 'a') as f:
    f.write(json.dumps(point) + '\n')
```

### Load history into a DataFrame

```python
import pandas as pd
import os

history_file = '/tmp/gh-aw/cache-memory/trending/issues/history.jsonl'
if os.path.exists(history_file):
    df = pd.read_json(history_file, lines=True)
    df['timestamp'] = pd.to_datetime(df['timestamp'])
    df = df.sort_values('timestamp')
else:
    df = pd.DataFrame()  # Start fresh if no history
```

### Compute a 7-day moving average

```python
df['rolling_avg'] = df['value'].rolling(window=7, min_periods=1).mean()

fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
ax.plot(df['timestamp'], df['value'], label='Actual', alpha=0.5, marker='o')
ax.plot(df['timestamp'], df['rolling_avg'], label='7-day Average', linewidth=2.5)
ax.fill_between(df['timestamp'], df['value'], df['rolling_avg'], alpha=0.2)
ax.legend(loc='best')
```

### Compare multiple metrics over time

```python
fig, ax = plt.subplots(figsize=(14, 8), dpi=300)
for metric in ['metric_a', 'metric_b', 'metric_c']:
    metric_data = df[df['metric'] == metric]
    ax.plot(metric_data['timestamp'], metric_data['value'],
            marker='o', label=metric, linewidth=2)
ax.set_title('Comparative Metrics Trends', fontsize=16, fontweight='bold')
ax.legend(loc='best', fontsize=12)
ax.grid(True, alpha=0.3)
```

### Enforce 90-day retention

```python
from datetime import timedelta

cutoff = pd.Timestamp.now() - timedelta(days=90)
df = df[df['timestamp'] >= cutoff]
df.to_json('/tmp/gh-aw/cache-memory/trending/issues/history.jsonl',
           orient='records', lines=True)
```

## Chart Quality Settings

```python
import matplotlib.pyplot as plt
import seaborn as sns

sns.set_style("whitegrid")
sns.set_palette("husl")

fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
# ... plotting code ...
ax.set_title('Title', fontsize=16, fontweight='bold')
ax.set_xlabel('Date', fontsize=12)
ax.set_ylabel('Value', fontsize=12)
ax.grid(True, alpha=0.3)
plt.xticks(rotation=45)
plt.tight_layout()
plt.savefig('/tmp/gh-aw/python/charts/trend.png',
            dpi=300, bbox_inches='tight', facecolor='white')
```

**Standards**: 300 DPI minimum · 12×7 inch figure · clear axis labels and title · legend for multi-series · grid lines enabled · colorblind-friendly palette (seaborn `husl` default)

## Including Charts in Reports

Assets uploaded with the `upload asset` tool are published to an orphaned git branch and become URL-addressable after the workflow completes.

```markdown
## Visualization Results

![Chart description](https://raw.githubusercontent.com/owner/repo/assets/workflow-name/trend.png)

The chart above shows...
```

## Report Structure

When creating the issue or discussion, use this template:

```markdown
# 📊 [Report Title] — [Date]

## Summary
[2–3 sentences describing trends and key findings]

## [Metric 1] Trend
![Metric 1 trend chart](URL_FROM_UPLOAD_ASSET)
[Brief analysis: direction, moving average, notable events]

## [Metric 2] Distribution
![Metric 2 chart](URL_FROM_UPLOAD_ASSET)
[Brief analysis]

## Data Details
- **Source**: [GitHub API / generated sample / external]
- **Data points**: [count]
- **Date range**: [start] to [end]
- **Tracking period**: [N] days

## Cache Status
- **Metrics tracked**: [list]
- **History location**: `/tmp/gh-aw/cache-memory/trending/`
```

Use `###` and lower for all headers inside the report body — `#` and `##` are reserved for issue/discussion titles.

## Common Use Cases

| Intent | Notes |
|---|---|
| "Create a weekly GitHub activity chart" | Schedule weekly; track issues, PRs, commits |
| "Visualize test coverage trends over time" | Trigger on push/PR; append per-run metrics |
| "Generate a dashboard of workflow run durations" | Schedule daily; use GitHub Actions API |
| "Plot stale repo aging distribution" | On-demand; no trending needed, skip cache |
| "Track contributor growth month-over-month" | Schedule monthly; long retention (365 days) |

## Key Reminders

- ✅ **Check cache first** — load historical data before collecting new data
- ✅ **Append, never overwrite** — use JSON Lines for time-series history
- ✅ **External data files only** — never inline data in Python
- ✅ **Upload charts before reporting** — collect all asset URLs, then create the issue/discussion
- ✅ **Call `noop` if nothing to report** — required when no safe-output action is taken
- ✅ **Use DPI 300** and seaborn styling for publication-quality charts
- ✅ **90-day retention** — prune history to prevent unbounded growth
