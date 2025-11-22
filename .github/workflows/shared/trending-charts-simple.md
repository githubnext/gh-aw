---
# Trending Charts (Simple) - Shared Agentic Workflow
# A simplified version of charts-with-trending.md for quick trending chart integration
#
# For comprehensive trending patterns, use: shared/charts-with-trending.md
# For simple, quick trending setup, use this file
#
# Usage:
#   imports:
#     - shared/trending-charts-simple.md
#
# This import provides:
# - Python environment with NumPy, Pandas, Matplotlib, Seaborn, SciPy
# - Cache-memory integration for persistent trending data
# - Automatic artifact upload for charts
# - Quick start examples for common trending patterns
#
# Key Features:
# - No network restrictions (strict mode compatible)
# - Standalone configuration (no nested imports)
# - Minimal setup overhead
# - Compatible with all engines (Copilot, Claude, Codex)

tools:
  cache-memory:
    key: trending-data-${{ github.workflow }}-${{ github.run_id }}
  bash:
    - "*"

steps:
  - name: Setup Python environment for trending
    run: |
      # Create working directory structure
      mkdir -p /tmp/gh-aw/python
      mkdir -p /tmp/gh-aw/python/data
      mkdir -p /tmp/gh-aw/python/charts
      mkdir -p /tmp/gh-aw/python/artifacts
      
      echo "Python environment setup complete"
      echo "Working directory: /tmp/gh-aw/python"
      echo "Data directory: /tmp/gh-aw/python/data"
      echo "Charts directory: /tmp/gh-aw/python/charts"
      echo "Cache memory: /tmp/gh-aw/cache-memory/"

  - name: Install Python scientific libraries
    run: |
      pip install --user numpy pandas matplotlib seaborn scipy
      
      # Verify installations
      python3 -c "import numpy; print(f'NumPy {numpy.__version__} installed')"
      python3 -c "import pandas; print(f'Pandas {pandas.__version__} installed')"
      python3 -c "import matplotlib; print(f'Matplotlib {matplotlib.__version__} installed')"
      python3 -c "import seaborn; print(f'Seaborn {seaborn.__version__} installed')"
      python3 -c "import scipy; print(f'SciPy {scipy.__version__} installed')"
      
      echo "All scientific libraries installed successfully"

  - name: Upload generated charts
    if: always()
    uses: actions/upload-artifact@v5
    with:
      name: trending-charts
      path: /tmp/gh-aw/python/charts/*.png
      if-no-files-found: warn
      retention-days: 30

  - name: Upload source files and data
    if: always()
    uses: actions/upload-artifact@v5
    with:
      name: trending-source-and-data
      path: |
        /tmp/gh-aw/python/*.py
        /tmp/gh-aw/python/data/*
      if-no-files-found: warn
      retention-days: 30
---

# Trending Charts - Quick Start Guide

You have a complete Python environment with scientific libraries ready for generating trend charts with persistent data storage.

## Cache-Memory for Trending Data

Persistent cache-memory is available at `/tmp/gh-aw/cache-memory/` that survives across workflow runs. Use it to store historical trending data.

**Recommended Structure:**
```
/tmp/gh-aw/cache-memory/
├── trending/
│   ├── <metric-name>/
│   │   └── history.jsonl      # Time-series data (JSON Lines format)
│   └── index.json              # Index of all tracked metrics
```

## Quick Start Pattern 1: Daily Metrics Tracking

Track daily metrics and visualize trends over time:

```python
#!/usr/bin/env python3
"""Daily metrics trending"""
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import json
import os
from datetime import datetime

# Configuration
CACHE_DIR = '/tmp/gh-aw/cache-memory/trending'
METRIC_NAME = 'daily_metrics'
HISTORY_FILE = f'{CACHE_DIR}/{METRIC_NAME}/history.jsonl'
CHARTS_DIR = '/tmp/gh-aw/python/charts'

# Ensure directories exist
os.makedirs(f'{CACHE_DIR}/{METRIC_NAME}', exist_ok=True)
os.makedirs(CHARTS_DIR, exist_ok=True)

# Collect today's data (customize this section)
today_data = {
    "timestamp": datetime.now().isoformat(),
    "metric_a": 42,
    "metric_b": 85,
    "metric_c": 23
}

# Append to history
with open(HISTORY_FILE, 'a') as f:
    f.write(json.dumps(today_data) + '\n')

# Load all historical data
if os.path.exists(HISTORY_FILE):
    df = pd.read_json(HISTORY_FILE, lines=True)
    df['date'] = pd.to_datetime(df['timestamp']).dt.date
    df = df.sort_values('timestamp')
    daily_stats = df.groupby('date').sum()
    
    # Generate trend chart
    sns.set_style("whitegrid")
    sns.set_palette("husl")
    
    fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
    daily_stats.plot(ax=ax, marker='o', linewidth=2)
    ax.set_title('Daily Metrics Trends', fontsize=16, fontweight='bold')
    ax.set_xlabel('Date', fontsize=12)
    ax.set_ylabel('Count', fontsize=12)
    ax.legend(loc='best')
    ax.grid(True, alpha=0.3)
    plt.xticks(rotation=45)
    plt.tight_layout()
    
    plt.savefig(f'{CHARTS_DIR}/daily_metrics_trend.png',
                dpi=300, bbox_inches='tight', facecolor='white')
    
    print(f"✅ Chart generated with {len(df)} data points")
else:
    print("No historical data yet. Run again tomorrow to see trends.")
```

## Quick Start Pattern 2: Moving Averages

Smooth volatile data with moving averages:

```python
#!/usr/bin/env python3
"""Moving average trending"""
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import os

# Load historical data
history_file = '/tmp/gh-aw/cache-memory/trending/metrics/history.jsonl'
if os.path.exists(history_file):
    df = pd.read_json(history_file, lines=True)
    df['date'] = pd.to_datetime(df['timestamp']).dt.date
    df = df.sort_values('timestamp')
    
    # Calculate 7-day moving average
    df['rolling_avg'] = df['value'].rolling(window=7, min_periods=1).mean()
    
    # Plot with trend line
    sns.set_style("whitegrid")
    fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
    ax.plot(df['date'], df['value'], label='Actual', alpha=0.5, marker='o')
    ax.plot(df['date'], df['rolling_avg'], label='7-day Average', linewidth=2.5)
    ax.fill_between(df['date'], df['value'], df['rolling_avg'], alpha=0.2)
    ax.set_title('Trend with Moving Average', fontsize=16, fontweight='bold')
    ax.set_xlabel('Date', fontsize=12)
    ax.set_ylabel('Value', fontsize=12)
    ax.legend(loc='best')
    ax.grid(True, alpha=0.3)
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig('/tmp/gh-aw/python/charts/moving_average_trend.png',
                dpi=300, bbox_inches='tight', facecolor='white')
    print("✅ Moving average chart generated")
```

## Quick Start Pattern 3: Comparative Trends

Compare multiple metrics over time:

```python
#!/usr/bin/env python3
"""Comparative trending"""
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import os

history_file = '/tmp/gh-aw/cache-memory/trending/multi_metrics/history.jsonl'
if os.path.exists(history_file):
    df = pd.read_json(history_file, lines=True)
    df['timestamp'] = pd.to_datetime(df['timestamp'])
    
    # Plot multiple metrics
    sns.set_style("whitegrid")
    sns.set_palette("husl")
    fig, ax = plt.subplots(figsize=(14, 8), dpi=300)
    
    for metric in df['metric'].unique():
        metric_data = df[df['metric'] == metric]
        ax.plot(metric_data['timestamp'], metric_data['value'], 
                marker='o', label=metric, linewidth=2)
    
    ax.set_title('Comparative Metrics Trends', fontsize=16, fontweight='bold')
    ax.set_xlabel('Date', fontsize=12)
    ax.set_ylabel('Value', fontsize=12)
    ax.legend(loc='best', fontsize=12)
    ax.grid(True, alpha=0.3)
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig('/tmp/gh-aw/python/charts/comparative_trends.png',
                dpi=300, bbox_inches='tight', facecolor='white')
    print("✅ Comparative trends chart generated")
```

## Best Practices

### 1. Use JSON Lines Format

Store trending data as JSON Lines (`.jsonl`) for efficient append-only storage:
```python
# Append new data point
with open(history_file, 'a') as f:
    f.write(json.dumps(data_point) + '\n')

# Load all data
df = pd.read_json(history_file, lines=True)
```

### 2. Include Timestamps

Always include ISO 8601 timestamps:
```python
data_point = {
    "timestamp": datetime.now().isoformat(),
    "metric": "issue_count",
    "value": 42
}
```

### 3. Data Retention

Implement retention policies to prevent unbounded growth:
```python
from datetime import datetime, timedelta

# Keep only last 90 days
cutoff_date = datetime.now() - timedelta(days=90)
df = df[df['timestamp'] >= cutoff_date]

# Save pruned data
df.to_json(history_file, orient='records', lines=True)
```

## Directory Structure

```
/tmp/gh-aw/
├── python/
│   ├── data/          # Current run data files
│   ├── charts/        # Generated charts (auto-uploaded as artifacts)
│   ├── artifacts/     # Additional output files
│   └── *.py           # Python scripts
└── cache-memory/
    └── trending/      # Persistent historical data (survives runs)
        └── <metric>/
            └── history.jsonl
```

## Chart Quality Guidelines

- **DPI**: Use 300 or higher for publication quality
- **Figure Size**: Standard is 12x7 inches for trend charts
- **Labels**: Always include clear axis labels and titles
- **Legend**: Add legends when plotting multiple series
- **Grid**: Enable grid lines for easier reading
- **Colors**: Use colorblind-friendly palettes (seaborn defaults)

## Tips for Success

1. **Consistency**: Use same metric names across runs
2. **Validation**: Check data quality before appending
3. **Documentation**: Comment your data schemas
4. **Testing**: Validate charts before uploading
5. **Cleanup**: Implement retention policies for cache-memory

---

Remember: The power of trending comes from consistent data collection over time. Use cache-memory to build a rich historical dataset that reveals insights and patterns!
