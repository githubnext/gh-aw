---
description: Guidelines for creating agentic workflows that generate charts and trend visualizations using Python scientific computing libraries with persistent historical data.
imports:
  - shared/python-dataviz.md
  - shared/charts-with-trending.md
  - shared/trending-charts-simple.md
---

# Data Science & Chart Generation

Consult this file when creating an agentic workflow that generates charts, visualizations, or trend analysis — including data dashboards, metric reports, time-series plots, or any Python-based visualization output.

## Choosing the Right Shared Import

| Goal | Import |
|---|---|
| Generate charts + persistent trend tracking | `shared/charts-with-trending.md` |
| Quick trending setup, no nested imports | `shared/trending-charts-simple.md` |
| Python environment only, no cache-memory | `shared/python-dataviz.md` |

Use `shared/charts-with-trending.md` by default when the workflow needs to track metrics across runs. Use `shared/trending-charts-simple.md` when strict-mode compatibility or a minimal configuration is preferred.

## Minimal Frontmatter

```yaml
---
description: [what the workflow visualizes]
on:
  schedule:
    - cron: "0 9 * * 1"   # example: every Monday at 09:00 UTC
  workflow_dispatch:
permissions:
  contents: read
  actions: read          # add issue/discussion scopes when needed
engine: copilot
imports:
  - shared/charts-with-trending.md
safe-outputs:
  upload-asset:
  create-issue:          # or create-discussion for gallery-style reports
    title-prefix: "📊 [Report Name]:"
    labels: [report]
    close-older-issues: true
    expires: 30
---
```

## Environment Reference

The import sets up everything automatically:

| Location | Purpose |
|---|---|
| `/tmp/gh-aw/python/` | Working directory for scripts |
| `/tmp/gh-aw/python/data/` | Input data files (CSV, JSON) |
| `/tmp/gh-aw/python/charts/` | Generated chart images (PNG) |
| `/tmp/gh-aw/cache-memory/trending/` | Persistent time-series history |

**Libraries available**: NumPy, Pandas, Matplotlib, Seaborn, SciPy

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

**CRITICAL**: Data must never be inlined in Python code. Always write data to a file first, then load it with pandas:

```python
# ❌ PROHIBITED
data = [10, 20, 30, 40, 50]

# ✅ REQUIRED
import pandas as pd
data = pd.read_csv('/tmp/gh-aw/python/data/metrics.csv')
```

## Trending Patterns

### Append a daily data point

```python
import json
from datetime import datetime

point = {"timestamp": datetime.now().isoformat(), "value": 42, "metric": "issue_count"}
with open('/tmp/gh-aw/cache-memory/trending/issues/history.jsonl', 'a') as f:
    f.write(json.dumps(point) + '\n')
```

### Load history and compute a 7-day moving average

```python
import pandas as pd

df = pd.read_json('/tmp/gh-aw/cache-memory/trending/issues/history.jsonl', lines=True)
df['date'] = pd.to_datetime(df['timestamp']).dt.date
df = df.sort_values('timestamp')
df['rolling_avg'] = df['value'].rolling(window=7, min_periods=1).mean()
```

### Enforce 90-day retention

```python
from datetime import timedelta

cutoff = pd.Timestamp.now() - timedelta(days=90)
df = df[pd.to_datetime(df['timestamp']) >= cutoff]
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
plt.tight_layout()
plt.savefig('/tmp/gh-aw/python/charts/trend.png',
            dpi=300, bbox_inches='tight', facecolor='white')
```

**Standards**: 300 DPI minimum · 12×7 inch figure · clear axis labels and title · legend for multi-series · grid lines enabled

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

- ✅ **Import the right shared workflow** — `charts-with-trending.md` or `trending-charts-simple.md`
- ✅ **Check cache first** — load historical data before collecting new data
- ✅ **Append, never overwrite** — use JSON Lines for time-series history
- ✅ **External data files only** — never inline data in Python
- ✅ **Upload charts before reporting** — collect all asset URLs, then create the issue/discussion
- ✅ **Call `noop` if nothing to report** — required when no safe-output action is taken
- ✅ **Use DPI 300** and seaborn styling for publication-quality charts
