---
description: Full trending-analysis patterns, best practices, and reporting guidance for chart workflows.
---
# Charts with Trending

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
    max: 3
    allowed-exts: [.png, .jpg, .jpeg, .svg]
```

### Agent Instructions

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
