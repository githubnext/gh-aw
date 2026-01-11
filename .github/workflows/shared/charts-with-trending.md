---
# Charts with Trending - Shared Agentic Workflow
# Provides complete setup for generating charts with trending analysis and cache-memory
#
# For quick setup with minimal configuration, use: shared/trending-charts-simple.md
# For comprehensive trending patterns and advanced features, use this file
#
# Usage:
#   imports:
#     - shared/charts-with-trending.md
#
# This import provides:
# - Python data visualization environment with scientific libraries
# - Trending analysis capabilities and best practices
# - Cache-memory integration for persistent trending data
# - Asset upload configuration for embedding charts in discussions/issues
#
# Key Features:
# - Automatic cache-memory for storing historical trending data
# - Python environment with NumPy, Pandas, Matplotlib, Seaborn, SciPy
# - Helper functions for loading/saving trending data
# - Best practices for creating impactful trend visualizations

imports:
  - shared/python-dataviz.md
  - shared/trends.md

tools:
  cache-memory:
    key: charts-trending-${{ github.workflow }}-${{ github.run_id }}
---

# Charts with Trending - Complete Guide

This shared workflow provides everything you need to create compelling trend visualizations with persistent data storage.

> [!TIP]
> **Quick Start Alternative**
>
> Looking for a simpler setup? Use `shared/trending-charts-simple.md` for:
> - No nested imports (standalone configuration)
> - No network restrictions (strict mode compatible)
> - Quick start examples for common trending patterns
> - Minimal configuration overhead
>
> The simplified version is perfect for basic trending needs while this comprehensive version offers advanced patterns and best practices.

## Cache-Memory for Trending Data

You have access to persistent cache-memory at `/tmp/gh-aw/cache-memory/` that survives across workflow runs. Use it to store historical trending data.

### Trending Data Organization

Organize your trending data in cache-memory:

```
/tmp/gh-aw/cache-memory/
├── trending/
│   ├── <metric-name>/
│   │   ├── history.jsonl      # Time-series data (JSON Lines format)
│   │   ├── metadata.json      # Data schema and descriptions
│   │   └── last_updated.txt   # Timestamp of last update
│   └── index.json             # Index of all tracked metrics
```

### Helper Functions for Trending Data

**Load Historical Data:**
```bash
# Check if historical data exists
if [ -f /tmp/gh-aw/cache-memory/trending/issues/history.jsonl ]; then
  echo "Loading historical issue trending data..."
  cp /tmp/gh-aw/cache-memory/trending/issues/history.jsonl /tmp/gh-aw/python/data/
else
  echo "No historical data found. Starting fresh."
  mkdir -p /tmp/gh-aw/cache-memory/trending/issues
fi
```

**Append New Data:**
```python
import json
from datetime import datetime

# New data point
data_point = {
    "timestamp": datetime.now().isoformat(),
    "metric": "issue_count",
    "value": 42,
    "metadata": {"source": "github_api"}
}

# Append to history (JSON Lines format)
with open('/tmp/gh-aw/cache-memory/trending/issues/history.jsonl', 'a') as f:
    f.write(json.dumps(data_point) + '\n')
```

**Load All Historical Data for Analysis:**
```python
import pandas as pd
import json

# Load all historical data
data_points = []
history_file = '/tmp/gh-aw/cache-memory/trending/issues/history.jsonl'

if os.path.exists(history_file):
    with open(history_file, 'r') as f:
        for line in f:
            data_points.append(json.loads(line))
    
    # Convert to DataFrame for analysis
    df = pd.DataFrame(data_points)
    df['timestamp'] = pd.to_datetime(df['timestamp'])
    df = df.sort_values('timestamp')
else:
    df = pd.DataFrame()  # Empty if no history
```

## Trending Analysis Patterns

### Pattern 1: Daily Metrics Tracking

Track daily metrics and visualize trends over time:

```python
#!/usr/bin/env python3
"""
Daily metrics trending example
"""
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import json
import os
from datetime import datetime

# Set style
sns.set_style("whitegrid")
sns.set_palette("husl")

# Load historical data
history_file = '/tmp/gh-aw/cache-memory/trending/daily_metrics/history.jsonl'
if os.path.exists(history_file):
    data = pd.read_json(history_file, lines=True)
    data['date'] = pd.to_datetime(data['timestamp']).dt.date
else:
    data = pd.DataFrame()

# Add today's data
today_data = {
    "timestamp": datetime.now().isoformat(),
    "issues_opened": 5,
    "issues_closed": 3,
    "prs_merged": 2
}

# Append to history
os.makedirs(os.path.dirname(history_file), exist_ok=True)
with open(history_file, 'a') as f:
    f.write(json.dumps(today_data) + '\n')

# Reload with today's data
data = pd.read_json(history_file, lines=True)
data['date'] = pd.to_datetime(data['timestamp']).dt.date
daily_stats = data.groupby('date').sum()

# Create trend chart
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

print(f"Chart saved. Total data points: {len(data)}")
```

### Pattern 2: Moving Averages and Smoothing

```python
# Calculate 7-day moving average
df['rolling_avg'] = df['value'].rolling(window=7, min_periods=1).mean()

# Plot with trend line
fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
ax.plot(df['date'], df['value'], label='Actual', alpha=0.5, marker='o')
ax.plot(df['date'], df['rolling_avg'], label='7-day Average', linewidth=2.5)
ax.fill_between(df['date'], df['value'], df['rolling_avg'], alpha=0.2)
```

### Pattern 3: Comparative Trends

```python
# Compare multiple metrics over time
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

## Best Practices for Cache-Memory Trending

### 1. Use JSON Lines Format

JSON Lines (`.jsonl`) is ideal for append-only trending data:
- One JSON object per line
- Easy to append new data
- Efficient for time-series data
- Simple to load with pandas: `pd.read_json(file, lines=True)`

### 2. Include Metadata

Store metadata alongside data:
```json
{
  "metric_name": "issue_resolution_time",
  "unit": "hours",
  "description": "Average time to close issues",
  "started_tracking": "2024-01-01",
  "updated": "2024-03-15"
}
```

### 3. Maintain Index

Keep an index of all tracked metrics:
```json
{
  "metrics": [
    "issue_count",
    "pr_count",
    "commit_count",
    "test_coverage"
  ],
  "last_updated": "2024-03-15T10:30:00Z"
}
```

### 4. Data Retention Strategy

Implement retention policies to prevent unbounded growth:
```python
# Keep only last 90 days
cutoff_date = datetime.now() - timedelta(days=90)
df = df[df['timestamp'] >= cutoff_date]

# Save pruned data
df.to_json('/tmp/gh-aw/cache-memory/trending/history.jsonl', 
           orient='records', lines=True)
```

## Complete Trending Workflow Example

```python
#!/usr/bin/env python3
"""
Complete trending analysis workflow
Collects data, updates history, generates trend charts
"""
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import json
import os
from datetime import datetime, timedelta

# Configuration
CACHE_DIR = '/tmp/gh-aw/cache-memory/trending'
METRIC_NAME = 'github_activity'
HISTORY_FILE = f'{CACHE_DIR}/{METRIC_NAME}/history.jsonl'
CHARTS_DIR = '/tmp/gh-aw/python/charts'

# Ensure directories exist
os.makedirs(f'{CACHE_DIR}/{METRIC_NAME}', exist_ok=True)
os.makedirs(CHARTS_DIR, exist_ok=True)

# Collect today's data (example)
today_data = {
    "timestamp": datetime.now().isoformat(),
    "issues_opened": 8,
    "prs_merged": 12,
    "commits": 45,
    "contributors": 6
}

# Append to history
with open(HISTORY_FILE, 'a') as f:
    f.write(json.dumps(today_data) + '\n')

# Load all historical data
df = pd.read_json(HISTORY_FILE, lines=True)
df['date'] = pd.to_datetime(df['timestamp']).dt.date
df = df.sort_values('timestamp')

# Aggregate by date
daily_stats = df.groupby('date').sum()

# Generate trend chart
sns.set_style("whitegrid")
sns.set_palette("husl")

fig, axes = plt.subplots(2, 2, figsize=(16, 12), dpi=300)
fig.suptitle('GitHub Activity Trends', fontsize=18, fontweight='bold')

# Chart 1: Issues Opened
axes[0, 0].plot(daily_stats.index, daily_stats['issues_opened'], 
                marker='o', linewidth=2, color='#FF6B6B')
axes[0, 0].set_title('Issues Opened', fontsize=14)
axes[0, 0].grid(True, alpha=0.3)

# Chart 2: PRs Merged
axes[0, 1].plot(daily_stats.index, daily_stats['prs_merged'], 
                marker='s', linewidth=2, color='#4ECDC4')
axes[0, 1].set_title('PRs Merged', fontsize=14)
axes[0, 1].grid(True, alpha=0.3)

# Chart 3: Commits
axes[1, 0].plot(daily_stats.index, daily_stats['commits'], 
                marker='^', linewidth=2, color='#45B7D1')
axes[1, 0].set_title('Commits', fontsize=14)
axes[1, 0].grid(True, alpha=0.3)

# Chart 4: Contributors
axes[1, 1].plot(daily_stats.index, daily_stats['contributors'], 
                marker='D', linewidth=2, color='#FFA07A')
axes[1, 1].set_title('Active Contributors', fontsize=14)
axes[1, 1].grid(True, alpha=0.3)

plt.tight_layout()
plt.savefig(f'{CHARTS_DIR}/activity_trends.png',
            dpi=300, bbox_inches='tight', facecolor='white')

print(f"✅ Trend chart generated with {len(df)} data points")
print(f"📊 Chart saved to: {CHARTS_DIR}/activity_trends.png")
print(f"💾 Historical data: {HISTORY_FILE}")
```

## Integration with Asset Upload and Discussions

After generating charts, use the safe-outputs tools to share them:

```markdown
## Example Discussion with Trending Charts

Upload each chart using the `upload asset` tool, then create a discussion:

**Title**: "📈 Weekly Trending Analysis - [Date]"

**Content**:
# 📈 Trending Analysis Report

Generated on: {date}

## Activity Trends

![Activity Trends](URL_FROM_UPLOAD_ASSET)

Analysis shows:
- Issues opened: Up 15% from last week
- PR velocity: Stable at 12 PRs/day
- Commit activity: Peak on Tuesdays and Wednesdays
- Active contributors: Growing trend (+20% this month)

## Data Summary

- **Total data points**: {count}
- **Date range**: {start} to {end}
- **Tracking period**: {days} days

---

*Generated using Charts with Trending shared workflow*
*Historical data stored in cache-memory for continuous tracking*
```

## Tips for Success

1. **Consistency**: Use same metric names across runs
2. **Timestamps**: Always include ISO 8601 timestamps
3. **Validation**: Check data quality before appending
4. **Backup**: Keep metadata for data recovery
5. **Documentation**: Comment your data schemas
6. **Testing**: Validate charts before uploading
7. **Cleanup**: Implement retention policies
8. **Indexing**: Maintain metric index for discovery

## Common Use Cases

### Repository Activity Trends
```python
# Track: commits, PRs, issues, contributors
# Frequency: Daily
# Retention: 90 days
```

### Performance Metrics Trends
```python
# Track: build time, test coverage, bundle size
# Frequency: Per commit/PR
# Retention: 180 days
```

### Quality Metrics Trends
```python
# Track: code complexity, test failures, security alerts
# Frequency: Weekly
# Retention: 1 year
```

### Workflow Efficiency Trends
```python
# Track: workflow duration, token usage, success rate
# Frequency: Per run
# Retention: 30 days
```

---

Remember: The power of trending comes from consistent data collection over time. Use cache-memory to build a rich historical dataset that reveals insights and patterns!
