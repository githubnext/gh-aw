---
# Python Data Visualization Setup
# Shared configuration for Python-based data visualization workflows
#
# Usage:
#   imports:
#     - shared/python-dataviz.md
#
# This import provides:
# - Python environment setup with directory structure
# - Scientific library installation (NumPy, Pandas, Matplotlib, Seaborn, SciPy)
# - Automatic artifact upload for charts and source files
# - Asset upload capability via safe-outputs (upload-assets)
# - Instructions on data visualization best practices including asset uploads
#
# Note: This configuration ensures data separation by enforcing external data storage.

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
      # Create working directory for Python scripts
      mkdir -p /tmp/gh-aw/python
      mkdir -p /tmp/gh-aw/python/data
      mkdir -p /tmp/gh-aw/python/charts
      mkdir -p /tmp/gh-aw/python/artifacts
      
      echo "Python environment setup complete"
      echo "Working directory: /tmp/gh-aw/python"
      echo "Data directory: /tmp/gh-aw/python/data"
      echo "Charts directory: /tmp/gh-aw/python/charts"
      echo "Artifacts directory: /tmp/gh-aw/python/artifacts"

  - name: Install Python scientific libraries
    run: |
      pip install --user --quiet numpy pandas matplotlib seaborn scipy
      
      # Verify installations
      python3 -c "import numpy; print(f'NumPy {numpy.__version__} installed')"
      python3 -c "import pandas; print(f'Pandas {pandas.__version__} installed')"
      python3 -c "import matplotlib; print(f'Matplotlib {matplotlib.__version__} installed')"
      python3 -c "import seaborn; print(f'Seaborn {seaborn.__version__} installed')"
      python3 -c "import scipy; print(f'SciPy {scipy.__version__} installed')"
      
      echo "All scientific libraries installed successfully"

  - name: Upload generated charts
    if: always()
    uses: actions/upload-artifact@v6
    with:
      name: data-charts
      path: /tmp/gh-aw/python/charts/*.png
      if-no-files-found: warn
      retention-days: 30

  - name: Upload source files and data
    if: always()
    uses: actions/upload-artifact@v6
    with:
      name: python-source-and-data
      path: |
        /tmp/gh-aw/python/*.py
        /tmp/gh-aw/python/data/*
      if-no-files-found: warn
      retention-days: 30
---

# Python Data Visualization Guide

Python scientific libraries have been installed and are ready for use. A temporary folder structure has been created at `/tmp/gh-aw/python/` for organizing scripts, data, and outputs.

## Installed Libraries

- **NumPy**: Array processing and numerical operations
- **Pandas**: Data manipulation and analysis
- **Matplotlib**: Chart generation and plotting
- **Seaborn**: Statistical data visualization
- **SciPy**: Scientific computing utilities

## Directory Structure

```
/tmp/gh-aw/python/
├── data/          # Store all data files here (CSV, JSON, etc.)
├── charts/        # Generated chart images (PNG)
├── artifacts/     # Additional output files
└── *.py           # Python scripts
```

## Data Separation Requirement

**CRITICAL**: Data must NEVER be inlined in Python code. Always store data in external files and load using pandas.

### ❌ PROHIBITED - Inline Data
```python
# DO NOT do this
data = [10, 20, 30, 40, 50]
labels = ['A', 'B', 'C', 'D', 'E']
```

### ✅ REQUIRED - External Data Files
```python
# Always load data from external files
import pandas as pd

# Load data from CSV
data = pd.read_csv('/tmp/gh-aw/python/data/data.csv')

# Or from JSON
data = pd.read_json('/tmp/gh-aw/python/data/data.json')
```

## Chart Generation Best Practices

### High-Quality Chart Settings

```python
import matplotlib.pyplot as plt
import seaborn as sns

# Set style for better aesthetics
sns.set_style("whitegrid")
sns.set_palette("husl")

# Create figure with high DPI
fig, ax = plt.subplots(figsize=(10, 6), dpi=300)

# Your plotting code here
# ...

# Save with high quality
plt.savefig('/tmp/gh-aw/python/charts/chart.png', 
            dpi=300, 
            bbox_inches='tight',
            facecolor='white',
            edgecolor='none')
```

### Chart Quality Guidelines

- **DPI**: Use 300 or higher for publication quality
- **Figure Size**: Standard is 10x6 inches (adjustable based on needs)
- **Labels**: Always include clear axis labels and titles
- **Legend**: Add legends when plotting multiple series
- **Grid**: Enable grid lines for easier reading
- **Colors**: Use colorblind-friendly palettes (seaborn defaults are good)

## Including Images in Reports

When creating reports (issues, discussions, etc.), use the `upload asset` tool to make images URL-addressable and include them in markdown:

### Step 1: Generate and Upload Chart
```python
# Generate your chart
plt.savefig('/tmp/gh-aw/python/charts/my_chart.png', dpi=300, bbox_inches='tight')
```

### Step 2: Upload as Asset
Use the `upload asset` tool to upload the chart file. The tool will return a GitHub raw content URL.

### Step 3: Include in Markdown Report
When creating your discussion or issue, include the image using markdown:

```markdown
## Visualization Results

![Chart Description](https://raw.githubusercontent.com/owner/repo/assets/workflow-name/my_chart.png)

The chart above shows...
```

**Important**: Assets are published to an orphaned git branch and become URL-addressable after workflow completion.

## Cache Memory Integration

The cache memory at `/tmp/gh-aw/cache-memory/` is available for storing reusable code:

**Helper Functions to Cache:**
- Data loading utilities: `data_loader.py`
- Chart styling functions: `chart_utils.py`
- Common data transformations: `transforms.py`

**Check Cache Before Creating:**
```bash
# Check if helper exists in cache
if [ -f /tmp/gh-aw/cache-memory/data_loader.py ]; then
  cp /tmp/gh-aw/cache-memory/data_loader.py /tmp/gh-aw/python/
  echo "Using cached data_loader.py"
fi
```

**Save to Cache for Future Runs:**
```bash
# Save useful helpers to cache
cp /tmp/gh-aw/python/data_loader.py /tmp/gh-aw/cache-memory/
echo "Saved data_loader.py to cache for future runs"
```

## Complete Example Workflow

```python
#!/usr/bin/env python3
"""
Example data visualization script
Generates a bar chart from external data
"""
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns

# Set style
sns.set_style("whitegrid")
sns.set_palette("husl")

# Load data from external file (NEVER inline)
data = pd.read_csv('/tmp/gh-aw/python/data/data.csv')

# Process data
summary = data.groupby('category')['value'].sum()

# Create chart
fig, ax = plt.subplots(figsize=(10, 6), dpi=300)
summary.plot(kind='bar', ax=ax)

# Customize
ax.set_title('Data Summary by Category', fontsize=16, fontweight='bold')
ax.set_xlabel('Category', fontsize=12)
ax.set_ylabel('Value', fontsize=12)
ax.grid(True, alpha=0.3)

# Save chart
plt.savefig('/tmp/gh-aw/python/charts/chart.png',
            dpi=300,
            bbox_inches='tight',
            facecolor='white')

print("Chart saved to /tmp/gh-aw/python/charts/chart.png")
```

## Error Handling

**Check File Existence:**
```python
import os

data_file = '/tmp/gh-aw/python/data/data.csv'
if not os.path.exists(data_file):
    raise FileNotFoundError(f"Data file not found: {data_file}")
```

**Validate Data:**
```python
# Check for required columns
required_cols = ['category', 'value']
missing = set(required_cols) - set(data.columns)
if missing:
    raise ValueError(f"Missing columns: {missing}")
```

## Artifact Upload

Charts and source files are automatically uploaded as artifacts:

**Charts Artifact:**
- Name: `data-charts`
- Contents: PNG files from `/tmp/gh-aw/python/charts/`
- Retention: 30 days

**Source and Data Artifact:**
- Name: `python-source-and-data`
- Contents: Python scripts and data files
- Retention: 30 days

Both artifacts are uploaded with `if: always()` condition, ensuring they're available even if the workflow fails.

## Tips for Success

1. **Always Separate Data**: Store data in files, never inline in code
2. **Use Cache Memory**: Store reusable helpers for faster execution
3. **High Quality Charts**: Use DPI 300+ and proper sizing
4. **Clear Documentation**: Add docstrings and comments
5. **Error Handling**: Validate data and check file existence
6. **Type Hints**: Use type annotations for better code quality
7. **Seaborn Defaults**: Leverage seaborn for better aesthetics
8. **Reproducibility**: Set random seeds when needed

## Common Data Sources

Based on common use cases:

**Repository Statistics:**
```python
# Collect via GitHub API, save to data.csv
# Then load and visualize
data = pd.read_csv('/tmp/gh-aw/python/data/repo_stats.csv')
```

**Workflow Metrics:**
```python
# Collect via GitHub Actions API, save to data.json
data = pd.read_json('/tmp/gh-aw/python/data/workflow_metrics.json')
```

**Sample Data Generation:**
```python
# Generate with NumPy, save to file first
import numpy as np
data = np.random.randn(100, 2)
df = pd.DataFrame(data, columns=['x', 'y'])
df.to_csv('/tmp/gh-aw/python/data/sample_data.csv', index=False)

# Then load it back (demonstrating the pattern)
data = pd.read_csv('/tmp/gh-aw/python/data/sample_data.csv')
```
