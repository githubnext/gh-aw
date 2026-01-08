---
# Trending Charts (Simple) - Python environment with NumPy, Pandas, Matplotlib, Seaborn, SciPy
# Cache-memory integration for persistent trending data, automatic artifact upload

tools:
  cache-memory:
    key: trending-data-${{ github.workflow }}-${{ github.run_id }}
  bash:
    - "*"

steps:
  - name: Setup Python environment
    run: |
      mkdir -p /tmp/gh-aw/python/{data,charts,artifacts}
      pip install --user --quiet numpy pandas matplotlib seaborn scipy

  - name: Upload charts
    if: always()
    uses: actions/upload-artifact@v6
    with:
      name: trending-charts
      path: /tmp/gh-aw/python/charts/*.png
      if-no-files-found: warn
      retention-days: 30

  - name: Upload source and data
    if: always()
    uses: actions/upload-artifact@v6
    with:
      name: trending-source-and-data
      path: |
        /tmp/gh-aw/python/*.py
        /tmp/gh-aw/python/data/*
      if-no-files-found: warn
      retention-days: 30
---

# Python Environment Ready

Libraries: NumPy, Pandas, Matplotlib, Seaborn, SciPy
Directories: `/tmp/gh-aw/python/{data,charts,artifacts}`, `/tmp/gh-aw/cache-memory/`

## Store Historical Data (JSON Lines)

```python
import json
from datetime import datetime

# Append data point
with open('/tmp/gh-aw/cache-memory/trending/<metric>/history.jsonl', 'a') as f:
    f.write(json.dumps({"timestamp": datetime.now().isoformat(), "value": 42}) + '\n')
```

## Generate Charts

```python
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns

df = pd.read_json('history.jsonl', lines=True)
df['date'] = pd.to_datetime(df['timestamp']).dt.date

sns.set_style("whitegrid")
fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
df.groupby('date')['value'].mean().plot(ax=ax, marker='o')
ax.set_title('Trend', fontsize=16, fontweight='bold')
plt.xticks(rotation=45)
plt.tight_layout()
plt.savefig('/tmp/gh-aw/python/charts/trend.png', dpi=300, bbox_inches='tight')
```

## Best Practices

- Use JSON Lines (`.jsonl`) for append-only storage
- Include ISO 8601 timestamps in all data points
- Implement 90-day retention: `df[df['timestamp'] >= cutoff_date]`
- Charts: 300 DPI, 12x7 inches, clear labels, seaborn style
