---
description: Guidance for using Python data visualization shared workflows to create charts and trend analyses in agentic workflows.
---

# Python Data Visualization in Agentic Workflows

Consult this file when creating or updating a workflow that generates charts, trend graphs, or any Python-based data visualization.

## Shared Workflow Imports

Three shared workflows provide Python charting capabilities. Choose based on your needs:

| Import | Best for |
|---|---|
| `shared/trending-charts-simple.md` | Quick setup, simple trend charts, no nested imports, strict-mode compatible |
| `shared/python-dataviz.md` | Custom charts without trending; includes library install and artifact upload |
| `shared/charts-with-trending.md` | Full trending analysis with cache-memory persistence (imports both above) |

**Default recommendation**: Use `shared/trending-charts-simple.md` for most new workflows. Use `shared/charts-with-trending.md` when you need detailed patterns, advanced cache guidance, or in-prompt documentation.

## Frontmatter Template

```yaml
imports:
  - shared/trending-charts-simple.md  # or shared/charts-with-trending.md
safe-outputs:
  upload-asset:                        # enables chart image embedding in issues/discussions
```

## Installed Libraries

The import setup step installs: **NumPy, Pandas, Matplotlib, Seaborn, SciPy**

## Directory Structure

| Path | Purpose |
|---|---|
| `/tmp/gh-aw/python/data/` | Input data files (CSV, JSON) |
| `/tmp/gh-aw/python/charts/` | Generated PNG chart files |
| `/tmp/gh-aw/python/*.py` | Python scripts |
| `/tmp/gh-aw/cache-memory/trending/` | Persistent historical data across runs |

## Artifacts

Charts are automatically uploaded as GitHub Actions artifacts:

- **`trending-charts`** (from `trending-charts-simple`) or **`data-charts`** (from `python-dataviz`) — PNGs from `/tmp/gh-aw/python/charts/`
- **`trending-source-and-data`** or **`python-source-and-data`** — Python scripts and data files

Retention: 30 days. Both artifacts use `if: always()` so they upload even on failure.

## Data Visualization Best Practices

### Data Separation (Required)

Data must **never** be inlined in Python scripts. Always write data to a file first, then load it.

```python
# ❌ PROHIBITED — inline data
data = [10, 20, 30, 40, 50]

# ✅ Required — external file
df = pd.read_csv('/tmp/gh-aw/python/data/metrics.csv')
```

### Chart Quality (Required)

- DPI: 300 minimum
- Figure size: 12×7 inches (adjust for multi-panel layouts)
- White background: `facecolor='white'`
- Style: `sns.set_style("whitegrid")`

```python
import matplotlib.pyplot as plt
import seaborn as sns

sns.set_style("whitegrid")
fig, ax = plt.subplots(figsize=(12, 7), dpi=300)

# ... plotting code ...

plt.tight_layout()
plt.savefig('/tmp/gh-aw/python/charts/chart.png',
            dpi=300, bbox_inches='tight', facecolor='white')
```

## Trending / Historical Data

Use `cache-memory` to persist time-series data across workflow runs in JSON Lines format:

```python
import json
from datetime import datetime

# Append a new data point
with open('/tmp/gh-aw/cache-memory/trending/<metric>/history.jsonl', 'a') as f:
    f.write(json.dumps({"timestamp": datetime.now().isoformat(), "value": 42}) + '\n')

# Load history into a DataFrame
import pandas as pd
df = pd.read_json('/tmp/gh-aw/cache-memory/trending/<metric>/history.jsonl', lines=True)
df['date'] = pd.to_datetime(df['timestamp']).dt.date
```

Implement 90-day retention to prevent unbounded growth:

```python
from datetime import timedelta
cutoff = datetime.now() - timedelta(days=90)
df = df[pd.to_datetime(df['timestamp']) >= cutoff]
df.to_json('/tmp/gh-aw/cache-memory/trending/<metric>/history.jsonl', orient='records', lines=True)
```

## Embedding Charts in Reports

1. Generate chart to `/tmp/gh-aw/python/charts/`
2. Upload via the `upload asset` tool → returns a raw GitHub URL
3. Embed in the issue or discussion body: `![Chart](URL_FROM_UPLOAD_ASSET)`

## Example Prompt Snippet

Include steps like these in the workflow prompt to guide the agent:

```markdown
## Data Visualization

1. Collect metrics and write to `/tmp/gh-aw/python/data/metrics.csv`
2. Check `/tmp/gh-aw/cache-memory/trending/metrics/history.jsonl` for historical data
3. Append today's data point in JSON Lines format with an ISO 8601 timestamp
4. Generate a trend line chart (DPI 300, 12×7 in, seaborn whitegrid style)
5. Save to `/tmp/gh-aw/python/charts/metrics_trend.png`
6. Upload the chart using `upload asset` and embed the URL in the report
```
