---
# Trending Charts (Simple) - Python environment with NumPy, Pandas, Matplotlib, Seaborn, SciPy
# Cache-memory integration for persistent trending data, automatic artifact upload

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
  upload-artifact:
    max-uploads: 5
    retention-days: 30
    skip-archive: true
    allowed-paths:
      - "**/*.png"
      - "**/*.jpg"
      - "**/*.svg"

steps:
  - name: Setup Python environment
    run: |
      mkdir -p /tmp/gh-aw/python/{data,charts,artifacts}
      # Create a virtual environment for proper package isolation (avoids --break-system-packages)
      if [ ! -d /tmp/gh-aw/venv ]; then
        python3 -m venv /tmp/gh-aw/venv
      fi
      echo "/tmp/gh-aw/venv/bin" >> "$GITHUB_PATH"
      /tmp/gh-aw/venv/bin/pip install --quiet numpy pandas matplotlib seaborn scipy

  - name: Upload source files and data
    if: always()
    uses: actions/upload-artifact@v7
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

## Upload Charts (skip-archive)

Chart images are uploaded individually via the `upload_artifact` safe-output tool with `skip-archive: true`. This stores each image as an individual file and returns a direct artifact URL, enabling inline rendering in issues, discussions, and pull requests.

### Step 1: Generate Chart

```python
plt.savefig('/tmp/gh-aw/python/charts/trend.png', dpi=300, bbox_inches='tight')
```

### Step 2: Upload as Artifact

Call the `upload_artifact` tool for each chart image:

```json
{ "type": "upload_artifact", "path": "/tmp/gh-aw/python/charts/trend.png" }
```

The tool returns `slot_N_artifact_url` with a direct link to the uploaded image.

### Step 3: Embed in Markdown

Use the returned artifact URL to render the chart inline:

```markdown
![Trend Chart](ARTIFACT_URL_FROM_SLOT_N)
```

> **Note**: Up to 5 chart images can be uploaded per run. Artifact URLs require GitHub authentication to access.

## Best Practices

- Use JSON Lines (`.jsonl`) for append-only storage
- Include ISO 8601 timestamps in all data points
- Implement 90-day retention: `df[df['timestamp'] >= cutoff_date]`
- Charts: 300 DPI, 12x7 inches, clear labels, seaborn style
