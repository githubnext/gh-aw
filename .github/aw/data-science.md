---
description: Guidelines for creating agentic workflows that generate charts and trend visualizations using Python scientific computing libraries with persistent historical data.
---

# Data Science & Chart Generation

Use when creating a workflow that generates charts, trend visualizations, dashboards, or any Python-based metric output.

## Frontmatter Template

```yaml
---
description: [what the workflow visualizes]
on:
  schedule:
    - cron: "0 9 * * 1"  # weekly; adjust as needed
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: copilot
tools:
  cache-memory:
    key: trending-${{ github.workflow }}-${{ github.run_id }}
  bash:
    - "*"
network:
  allowed:
    - defaults
    - python
safe-outputs:
  upload-asset:
  create-issue:          # or create-discussion
    title-prefix: "📊 [Report Name]:"
    labels: [report]
    close-older-issues: true
    expires: 30
steps:
  - name: setup
    run: |
      mkdir -p /tmp/gh-aw/python/{data,charts}
      mkdir -p /tmp/gh-aw/cache-memory/trending
      pip install --user --quiet numpy pandas matplotlib seaborn scipy
  - name: upload charts
    if: always()
    uses: actions/upload-artifact@v7.0.0
    with:
      name: charts
      path: /tmp/gh-aw/python/charts/*.png
      if-no-files-found: warn
      retention-days: 30
---
```

## Agent Prompt Structure

Write the agent prompt as five ordered steps:

1. **Load history** — read `/tmp/gh-aw/cache-memory/trending/<metric>/history.jsonl` into a DataFrame if it exists; otherwise start empty.
2. **Collect data** — fetch metrics from the GitHub API (or generate with NumPy). Save to `/tmp/gh-aw/python/data/<metric>.csv` — **never inline data in Python code**.
3. **Append & prune** — append a JSON Lines record `{"timestamp": "<iso8601>", "metric": "...", "value": ...}` to `history.jsonl`; drop records older than 90 days.
4. **Chart** — if ≥ 2 history points exist, generate a time-series line chart with 7-day moving average; otherwise use a bar/distribution chart. Save to `/tmp/gh-aw/python/charts/` at DPI 300.
5. **Report** — upload each chart with `upload asset`, then create an issue/discussion embedding the URLs. Call `noop` if there is nothing to report.

## Python Patterns

### History: load → append → prune

```python
import json, os, pandas as pd
from datetime import datetime, timedelta

HISTORY = '/tmp/gh-aw/cache-memory/trending/issues/history.jsonl'

# Load
df = pd.read_json(HISTORY, lines=True) if os.path.exists(HISTORY) else pd.DataFrame()
if not df.empty:
    df['timestamp'] = pd.to_datetime(df['timestamp'])
    df = df.sort_values('timestamp')

# Append
with open(HISTORY, 'a') as f:
    f.write(json.dumps({"timestamp": datetime.now().isoformat(), "metric": "issue_count", "value": 42}) + '\n')

# Prune to 90 days
if not df.empty:
    df = df[df['timestamp'] >= pd.Timestamp.now() - timedelta(days=90)]
    df.to_json(HISTORY, orient='records', lines=True)
```

### Chart: trend with moving average

```python
import matplotlib.pyplot as plt
import seaborn as sns

sns.set_style("whitegrid"); sns.set_palette("husl")
fig, ax = plt.subplots(figsize=(12, 7), dpi=300)

df['rolling'] = df['value'].rolling(window=7, min_periods=1).mean()
ax.plot(df['timestamp'], df['value'], label='Actual', alpha=0.5, marker='o')
ax.plot(df['timestamp'], df['rolling'], label='7-day avg', linewidth=2.5)
ax.fill_between(df['timestamp'], df['value'], df['rolling'], alpha=0.2)
ax.set_title('Metric Trend', fontsize=16, fontweight='bold')
ax.set_xlabel('Date', fontsize=12); ax.set_ylabel('Value', fontsize=12)
ax.legend(); ax.grid(True, alpha=0.3); plt.xticks(rotation=45); plt.tight_layout()
plt.savefig('/tmp/gh-aw/python/charts/trend.png', dpi=300, bbox_inches='tight', facecolor='white')
```

**Chart standards**: 300 DPI · 12×7 in · labeled axes and title · legend for multi-series · `husl` palette

### Multiple metrics

```python
for metric in metrics:
    sub = df[df['metric'] == metric]
    ax.plot(sub['timestamp'], sub['value'], marker='o', label=metric, linewidth=2)
```

## Report Template

```markdown
# 📊 [Title] — [Date]

## Summary
[2–3 sentences on trends and key findings]

### [Metric] Trend
![chart](URL_FROM_UPLOAD_ASSET)
[direction, moving average, notable events]

## Data Details
- **Source**: … | **Points**: … | **Range**: … | **Period**: …N days
- **Cache**: `/tmp/gh-aw/cache-memory/trending/`
```

Use `###` and deeper for all headers inside the report body.

## Use Cases

| Intent | Trigger | Notes |
|---|---|---|
| Weekly GitHub activity chart | `schedule` weekly | track issues, PRs, commits |
| Test coverage trends | `push`/`pull_request` | append per-run |
| Workflow run durations | `schedule` daily | GitHub Actions API |
| Stale repo aging distribution | `workflow_dispatch` | no cache needed |
| Contributor growth | `schedule` monthly | 365-day retention |
