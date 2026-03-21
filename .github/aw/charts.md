---
description: Guidance for adding Python data visualization to agentic workflows — includes inlined setup, best practices, trending patterns, and chart embedding from the shared chart workflows.
---

# Python Data Visualization in Agentic Workflows

Consult this file when creating or updating a workflow that generates charts, trend graphs, or any Python-based data visualization.

## Choosing a Shared Workflow

Three shared workflows provide Python charting capabilities. Choose based on your needs:

| Import | Best for |
|---|---|
| `shared/trending-charts-simple.md` | Quick setup, simple trend charts, strict-mode compatible |
| `shared/python-dataviz.md` | Custom charts without trending history |
| `shared/charts-with-trending.md` | Full trending analysis with detailed in-prompt guidance |

**Default recommendation**: Use `shared/trending-charts-simple.md` for most new workflows.

---

## Option A: Trending Charts (Simple)

**Use when**: You need trend charts with cache-memory persistence and minimal configuration.

### Frontmatter

```yaml
tools:
  cache-memory:
    key: trending-data-${{ github.workflow }}-${{ github.run_id }}
  bash:
    - "*"

network:
  allowed:
    - defaults
    - python

steps:
  - name: Setup Python environment
    run: |
      mkdir -p /tmp/gh-aw/python/{data,charts,artifacts}
      pip install --user --quiet numpy pandas matplotlib seaborn scipy

  - name: Upload charts
    if: always()
    uses: actions/upload-artifact@v7.0.0
    with:
      name: trending-charts
      path: /tmp/gh-aw/python/charts/*.png
      if-no-files-found: warn
      retention-days: 30

  - name: Upload source files and data
    if: always()
    uses: actions/upload-artifact@v7.0.0
    with:
      name: trending-source-and-data
      path: |
        /tmp/gh-aw/python/*.py
        /tmp/gh-aw/python/data/*
      if-no-files-found: warn
      retention-days: 30

safe-outputs:
  upload-asset:
```

### Agent Instructions

Libraries: NumPy, Pandas, Matplotlib, Seaborn, SciPy
Directories: `/tmp/gh-aw/python/{data,charts,artifacts}`, `/tmp/gh-aw/cache-memory/`

**Store Historical Data (JSON Lines)**:

```python
import json
from datetime import datetime

# Append data point
with open('/tmp/gh-aw/cache-memory/trending/<metric>/history.jsonl', 'a') as f:
    f.write(json.dumps({"timestamp": datetime.now().isoformat(), "value": 42}) + '\n')
```

**Generate Chart**:

```python
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns

df = pd.read_json('/tmp/gh-aw/cache-memory/trending/<metric>/history.jsonl', lines=True)
df['date'] = pd.to_datetime(df['timestamp']).dt.date

sns.set_style("whitegrid")
fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
df.groupby('date')['value'].mean().plot(ax=ax, marker='o')
ax.set_title('Trend', fontsize=16, fontweight='bold')
plt.xticks(rotation=45)
plt.tight_layout()
plt.savefig('/tmp/gh-aw/python/charts/trend.png', dpi=300, bbox_inches='tight', facecolor='white')
```

**Best Practices**:
- Use JSON Lines (`.jsonl`) for append-only storage
- Include ISO 8601 timestamps in all data points
- Implement 90-day retention: `df[df['timestamp'] >= cutoff_date]`
- Charts: 300 DPI, 12×7 inches, clear labels, seaborn whitegrid style

---

## Option B: Python Data Visualization (No Trending)

**Use when**: You need charts from current data without historical tracking.

### Frontmatter

```yaml
tools:
  cache-memory: true
  bash:
    - "*"

network:
  allowed:
    - defaults
    - python

safe-outputs:
  upload-asset:

steps:
  - name: Setup Python environment
    run: |
      mkdir -p /tmp/gh-aw/python
      mkdir -p /tmp/gh-aw/python/data
      mkdir -p /tmp/gh-aw/python/charts
      mkdir -p /tmp/gh-aw/python/artifacts
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
```

### Agent Instructions

Python scientific libraries have been installed. A temporary folder structure has been created at `/tmp/gh-aw/python/` for organizing scripts, data, and outputs.

**Installed Libraries**:
- **NumPy**: Array processing and numerical operations
- **Pandas**: Data manipulation and analysis
- **Matplotlib**: Chart generation and plotting
- **Seaborn**: Statistical data visualization
- **SciPy**: Scientific computing utilities

**Directory Structure**:

```
/tmp/gh-aw/python/
├── data/          # Store all data files here (CSV, JSON, etc.)
├── charts/        # Generated chart images (PNG)
├── artifacts/     # Additional output files
└── *.py           # Python scripts
```

**Data Separation Requirement**

**CRITICAL**: Data must NEVER be inlined in Python code. Always store data in external files and load using pandas.

```python
# ❌ PROHIBITED — inline data
data = [10, 20, 30, 40, 50]
labels = ['A', 'B', 'C', 'D', 'E']

# ✅ REQUIRED — external data files
import pandas as pd
data = pd.read_csv('/tmp/gh-aw/python/data/data.csv')
# Or from JSON
data = pd.read_json('/tmp/gh-aw/python/data/data.json')
```

**High-Quality Chart Settings**:

```python
import matplotlib.pyplot as plt
import seaborn as sns

sns.set_style("whitegrid")
sns.set_palette("husl")

fig, ax = plt.subplots(figsize=(10, 6), dpi=300)

# Your plotting code here

plt.savefig('/tmp/gh-aw/python/charts/chart.png',
            dpi=300,
            bbox_inches='tight',
            facecolor='white',
            edgecolor='none')
```

**Chart Quality Guidelines**:
- **DPI**: Use 300 or higher for publication quality
- **Figure Size**: Standard is 10×6 inches (adjustable based on needs)
- **Labels**: Always include clear axis labels and titles
- **Legend**: Add legends when plotting multiple series
- **Grid**: Enable grid lines for easier reading
- **Colors**: Use colorblind-friendly palettes (seaborn defaults are good)

**Cache Memory for Reusable Helpers**:

```bash
# Check if helper exists in cache
if [ -f /tmp/gh-aw/cache-memory/data_loader.py ]; then
  cp /tmp/gh-aw/cache-memory/data_loader.py /tmp/gh-aw/python/
fi

# Save useful helpers to cache for future runs
cp /tmp/gh-aw/python/data_loader.py /tmp/gh-aw/cache-memory/
```

**Complete Example**:

```python
#!/usr/bin/env python3
"""Example data visualization script — generates a bar chart from external data"""
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns

sns.set_style("whitegrid")
sns.set_palette("husl")

# Load data from external file (NEVER inline)
data = pd.read_csv('/tmp/gh-aw/python/data/data.csv')
summary = data.groupby('category')['value'].sum()

fig, ax = plt.subplots(figsize=(10, 6), dpi=300)
summary.plot(kind='bar', ax=ax)
ax.set_title('Data Summary by Category', fontsize=16, fontweight='bold')
ax.set_xlabel('Category', fontsize=12)
ax.set_ylabel('Value', fontsize=12)
ax.grid(True, alpha=0.3)

plt.savefig('/tmp/gh-aw/python/charts/chart.png',
            dpi=300, bbox_inches='tight', facecolor='white')
print("Chart saved to /tmp/gh-aw/python/charts/chart.png")
```

**Error Handling**:

```python
import os

data_file = '/tmp/gh-aw/python/data/data.csv'
if not os.path.exists(data_file):
    raise FileNotFoundError(f"Data file not found: {data_file}")

required_cols = ['category', 'value']
missing = set(required_cols) - set(data.columns)
if missing:
    raise ValueError(f"Missing columns: {missing}")
```

---

## Option C: Charts with Trending (Full Guide)

**Use when**: You need full trending analysis with cache-memory persistence and comprehensive guidance.

### Frontmatter

```yaml
imports:
  - shared/python-dataviz.md
  - shared/trends.md

tools:
  cache-memory:
    key: charts-trending-${{ github.workflow }}-${{ github.run_id }}

safe-outputs:
  upload-asset:
```

### Agent Instructions

You are an expert at creating compelling trend visualizations with persistent data storage across workflow runs.

**Cache-Memory Organization**:

```
/tmp/gh-aw/cache-memory/
├── trending/
│   ├── <metric-name>/
│   │   ├── history.jsonl      # Time-series data (JSON Lines format)
│   │   ├── metadata.json      # Data schema and descriptions
│   │   └── last_updated.txt   # Timestamp of last update
│   └── index.json             # Index of all tracked metrics
```

**Load Historical Data**:

```bash
if [ -f /tmp/gh-aw/cache-memory/trending/issues/history.jsonl ]; then
  echo "Loading historical data..."
  cp /tmp/gh-aw/cache-memory/trending/issues/history.jsonl /tmp/gh-aw/python/data/
else
  echo "No historical data found. Starting fresh."
  mkdir -p /tmp/gh-aw/cache-memory/trending/issues
fi
```

**Append New Data**:

```python
import json
from datetime import datetime

data_point = {
    "timestamp": datetime.now().isoformat(),
    "metric": "issue_count",
    "value": 42,
    "metadata": {"source": "github_api"}
}

with open('/tmp/gh-aw/cache-memory/trending/issues/history.jsonl', 'a') as f:
    f.write(json.dumps(data_point) + '\n')
```

**Load History into DataFrame**:

```python
import pandas as pd, json, os

history_file = '/tmp/gh-aw/cache-memory/trending/issues/history.jsonl'
if os.path.exists(history_file):
    df = pd.read_json(history_file, lines=True)
    df['timestamp'] = pd.to_datetime(df['timestamp'])
    df = df.sort_values('timestamp')
else:
    df = pd.DataFrame()
```

### Trending Analysis Patterns

**Pattern 1: Daily Metrics Tracking**

```python
#!/usr/bin/env python3
import pandas as pd, matplotlib.pyplot as plt, seaborn as sns, json, os
from datetime import datetime

sns.set_style("whitegrid")
sns.set_palette("husl")

history_file = '/tmp/gh-aw/cache-memory/trending/daily_metrics/history.jsonl'
today_data = {
    "timestamp": datetime.now().isoformat(),
    "issues_opened": 5,
    "issues_closed": 3,
    "prs_merged": 2
}

os.makedirs(os.path.dirname(history_file), exist_ok=True)
with open(history_file, 'a') as f:
    f.write(json.dumps(today_data) + '\n')

data = pd.read_json(history_file, lines=True)
data['date'] = pd.to_datetime(data['timestamp']).dt.date
daily_stats = data.groupby('date').sum()

fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
daily_stats.plot(ax=ax, marker='o', linewidth=2)
ax.set_title('Daily Metrics Trends', fontsize=16, fontweight='bold')
ax.set_xlabel('Date', fontsize=12)
ax.set_ylabel('Count', fontsize=12)
ax.legend(loc='best')
ax.grid(True, alpha=0.3)
plt.xticks(rotation=45)
plt.tight_layout()
plt.savefig('/tmp/gh-aw/python/charts/daily_metrics_trend.png',
            dpi=300, bbox_inches='tight', facecolor='white')
```

**Pattern 2: Moving Averages and Smoothing**

```python
df['rolling_avg'] = df['value'].rolling(window=7, min_periods=1).mean()

fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
ax.plot(df['date'], df['value'], label='Actual', alpha=0.5, marker='o')
ax.plot(df['date'], df['rolling_avg'], label='7-day Average', linewidth=2.5)
ax.fill_between(df['date'], df['value'], df['rolling_avg'], alpha=0.2)
```

**Pattern 3: Comparative Trends**

```python
fig, ax = plt.subplots(figsize=(14, 8), dpi=300)
for metric in ['metric_a', 'metric_b', 'metric_c']:
    metric_data = df[df['metric'] == metric]
    ax.plot(metric_data['timestamp'], metric_data['value'],
            marker='o', label=metric, linewidth=2)
ax.set_title('Comparative Metrics Trends', fontsize=16, fontweight='bold')
ax.legend(loc='best', fontsize=12)
ax.grid(True, alpha=0.3)
plt.xticks(rotation=45)
```

**Data Retention (90 days)**:

```python
from datetime import timedelta
cutoff_date = datetime.now() - timedelta(days=90)
df = df[df['timestamp'] >= cutoff_date]
df.to_json('/tmp/gh-aw/cache-memory/trending/history.jsonl', orient='records', lines=True)
```

**Complete Trending Example**:

```python
#!/usr/bin/env python3
import pandas as pd, matplotlib.pyplot as plt, seaborn as sns, json, os
from datetime import datetime, timedelta

CACHE_DIR = '/tmp/gh-aw/cache-memory/trending'
METRIC_NAME = 'github_activity'
HISTORY_FILE = f'{CACHE_DIR}/{METRIC_NAME}/history.jsonl'
CHARTS_DIR = '/tmp/gh-aw/python/charts'

os.makedirs(f'{CACHE_DIR}/{METRIC_NAME}', exist_ok=True)
os.makedirs(CHARTS_DIR, exist_ok=True)

today_data = {
    "timestamp": datetime.now().isoformat(),
    "issues_opened": 8, "prs_merged": 12, "commits": 45, "contributors": 6
}
with open(HISTORY_FILE, 'a') as f:
    f.write(json.dumps(today_data) + '\n')

df = pd.read_json(HISTORY_FILE, lines=True)
df['date'] = pd.to_datetime(df['timestamp']).dt.date
df = df.sort_values('timestamp')
daily_stats = df.groupby('date').sum()

sns.set_style("whitegrid")
sns.set_palette("husl")

fig, axes = plt.subplots(2, 2, figsize=(16, 12), dpi=300)
fig.suptitle('GitHub Activity Trends', fontsize=18, fontweight='bold')

axes[0, 0].plot(daily_stats.index, daily_stats['issues_opened'], marker='o', linewidth=2, color='#FF6B6B')
axes[0, 0].set_title('Issues Opened', fontsize=14)
axes[0, 0].grid(True, alpha=0.3)

axes[0, 1].plot(daily_stats.index, daily_stats['prs_merged'], marker='s', linewidth=2, color='#4ECDC4')
axes[0, 1].set_title('PRs Merged', fontsize=14)
axes[0, 1].grid(True, alpha=0.3)

axes[1, 0].plot(daily_stats.index, daily_stats['commits'], marker='^', linewidth=2, color='#45B7D1')
axes[1, 0].set_title('Commits', fontsize=14)
axes[1, 0].grid(True, alpha=0.3)

axes[1, 1].plot(daily_stats.index, daily_stats['contributors'], marker='D', linewidth=2, color='#FFA07A')
axes[1, 1].set_title('Active Contributors', fontsize=14)
axes[1, 1].grid(True, alpha=0.3)

plt.tight_layout()
plt.savefig(f'{CHARTS_DIR}/activity_trends.png', dpi=300, bbox_inches='tight', facecolor='white')
print(f"✅ Trend chart generated with {len(df)} data points")
```

---

## Trends Visualization Best Practices

When generating trending charts, focus on:

### Time Series Excellence
- Use line charts for continuous trends over time
- Add trend lines or moving averages to highlight patterns
- Include clear date/time labels on the x-axis
- Show confidence intervals or error bands when relevant

### Comparative Trends
- Use multi-line charts to compare multiple trends
- Apply distinct colors for each series with a clear legend
- Consider using area charts for stacked trends
- Highlight key inflection points or anomalies

### Contextual Information
- Show percentage changes or growth rates
- Include baseline comparisons (year-over-year, month-over-month)
- Add summary statistics (min, max, average, median)

### Example Chart Types

**Temporal Trends**:
```python
fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
for column in data.columns:
    ax.plot(data.index, data[column], marker='o', label=column, linewidth=2)
ax.set_title('Trends Over Time', fontsize=16, fontweight='bold')
ax.set_xlabel('Date', fontsize=12)
ax.set_ylabel('Value', fontsize=12)
ax.legend(loc='best')
ax.grid(True, alpha=0.3)
plt.xticks(rotation=45)
```

**Growth Rates**:
```python
fig, ax = plt.subplots(figsize=(10, 6), dpi=300)
growth_data.plot(kind='bar', ax=ax, color=sns.color_palette("husl"))
ax.set_title('Growth Rates by Period', fontsize=16, fontweight='bold')
ax.axhline(y=0, color='black', linestyle='-', linewidth=0.8)
ax.set_ylabel('Growth %', fontsize=12)
```

**Moving Averages**:
```python
fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
ax.plot(dates, values, label='Actual', alpha=0.5, linewidth=1)
ax.plot(dates, moving_avg, label='7-day Moving Average', linewidth=2.5)
ax.fill_between(dates, values, moving_avg, alpha=0.2)
```

### Data Preparation

```python
# Time-based indexing
data['date'] = pd.to_datetime(data['date'])
data.set_index('date', inplace=True)
data = data.sort_index()

# Resampling
weekly_data = data.resample('W').mean()
data['rolling_mean'] = data['value'].rolling(window=7).mean()

# Growth calculations
data['pct_change'] = data['value'].pct_change() * 100
data['yoy_growth'] = data['value'].pct_change(periods=365) * 100
```

### Color Palettes

- **Sequential**: `sns.color_palette("viridis", n_colors=5)`
- **Diverging**: `sns.color_palette("RdYlGn", n_colors=7)`
- **Multiple series**: `sns.color_palette("husl", n_colors=8)`
- **Categorical**: `sns.color_palette("Set2", n_colors=6)`

### Annotation

```python
max_idx = data['value'].idxmax()
max_val = data['value'].max()
ax.annotate(f'Peak: {max_val:.2f}',
            xy=(max_idx, max_val),
            xytext=(10, 20),
            textcoords='offset points',
            arrowprops=dict(arrowstyle='->', color='red'),
            fontsize=10, fontweight='bold')
```

---

## Embedding Charts in Reports

1. Save chart to `/tmp/gh-aw/python/charts/`
2. Upload via the `upload asset` tool → returns a raw GitHub URL
3. Embed in issue or discussion body: `![Chart description](URL_FROM_UPLOAD_ASSET)`

**Assets are published to an orphaned git branch and become URL-addressable after workflow completion.**

Example report structure:

```markdown
## 📈 Trending Analysis

![Activity Trends](URL_FROM_UPLOAD_ASSET)

Analysis shows:
- Issues opened: Up 15% from last week
- PR velocity: Stable at 12 PRs/day
- Active contributors: Growing trend (+20% this month)

**Data**: {count} points | **Range**: {start} to {end}
```

---

## Session Analysis Chart Pattern

For workflows tracking Copilot coding agent session data:

**Two required charts**:

**Chart 1: Session Completion Trends**
- Multi-line chart: successful completions (green), failed/abandoned (red), completion rate % (secondary y-axis)
- X-axis: Date (last 30 days)
- Save as: `/tmp/gh-aw/python/charts/session_completion_trends.png`

**Chart 2: Session Duration & Efficiency**
- Average duration (line), median duration (line), sessions with loops (bar overlay)
- X-axis: Date (last 30 days), Y-axis: Duration in minutes
- Save as: `/tmp/gh-aw/python/charts/session_duration_trends.png`

**Data files**:
- `session_completion.csv` — date, successful, failed, completion_rate
- `session_duration.csv` — date, avg_duration_min, median_duration_min, loop_count

**Error handling**: If fewer than 7 days of data, use bar charts instead of line charts and note the limited range.

---

## Tips for Success

1. **Consistency**: Use the same metric names and file paths across runs
2. **Timestamps**: Always use ISO 8601 format in data points
3. **Data first**: Write data to a CSV/JSON file before plotting — never inline
4. **Quality**: DPI 300+, clear axis labels, seaborn whitegrid, white background
5. **Retention**: Prune cache-memory data to 90 days to prevent unbounded growth
6. **Upload**: Use `upload asset` for every chart to get embeddable URLs
7. **Story**: Annotate significant events; add context with moving averages
