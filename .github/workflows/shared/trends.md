---
# Trends Visualization Shared Workflow
# Provides guidance for creating trending data charts
#
# Usage:
#   imports:
#     - shared/trends.md
#
# This import provides:
# - Python data visualization environment (via python-dataviz import)
# - Prompts for generating awesome trending charts
# - Best practices for visualizing trends over time
# - Guidelines for creating engaging and informative trend visualizations

imports:
  - shared/python-dataviz.md
---

# Trends Visualization Guide

Use trending charts to reveal patterns from time-series data stored in repo-memory.

## Data Source Convention

Historical metrics are stored as JSON Lines at `/tmp/gh-aw/repo-memory/default/history.jsonl` (one record per day). Load with:
```python
import json
from pathlib import Path
history = [json.loads(l) for l in Path('/tmp/gh-aw/repo-memory/default/history.jsonl').read_text().splitlines() if l.strip()]
```

## Styling

Use `sns.set_style("whitegrid")`, `sns.set_context("notebook", font_scale=1.2)`, `figsize=(14, 8)`, `dpi=300`, `bbox_inches='tight'`. Annotate key peaks/troughs with `ax.annotate()` using `arrowprops`.

## Repo-Specific Chart Types

- **Historical trends**: multi-line chart of LOC, test coverage %, and quality score over 30 days
- **Churn over time**: 7-day rolling sum of files modified / lines added+deleted
- **Workflow growth**: total `.md` and `.lock.yml` file counts over time
- **Quality score components**: stacked area showing how each component (test coverage, churn stability, etc.) evolves
- **Moving averages**: 7-day rolling mean overlaid on raw data to reduce noise

## Tips

- Match time granularity to the pattern (daily for code metrics, weekly for macro trends)
- Use 7-day moving averages for volatile daily data
- Annotate changes >10% with emoji indicators (⬆️/⬇️)
- Include summary stats (min, max, latest, 7d change) in chart titles or subtitles
- Color palettes: `"husl"` for multiple series, `"RdYlGn"` for diverging/quality-score charts
