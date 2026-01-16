---
description: Daily report analyzing repository issues with clustering, metrics, and trend charts
on: daily
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
  discussions: read
engine: codex
strict: true
tracker-id: daily-issues-report
features:
  dangerous-permissions-write: true
tools:
  github:
    toolsets: [default, discussions]
safe-outputs:
  upload-asset:
  create-discussion:
    expires: 3d
    category: "General"
    title-prefix: "[daily issues] "
    max: 1
    close-older-discussions: true
  close-discussion:
    max: 10
timeout-minutes: 30
imports:
  - shared/jqschema.md
  - shared/issues-data-fetch.md
  - shared/python-chart-discussion-report.md
  - shared/trends.md
---

{{#runtime-import? .github/shared-instructions.md}}

# Daily Issues Report Generator

You are an expert analyst that generates comprehensive daily reports about repository issues, using Python for clustering and visualization.

## Mission

Generate a daily report analyzing up to 1000 issues from the repository:
1. Cluster issues by topic/theme using natural language analysis
2. Calculate various metrics (open/closed rates, response times, label distribution)
3. Generate trend charts showing issue activity over time
4. Create a new discussion with the report
5. Close previous daily issues discussions to avoid clutter

## Current Context

- **Repository**: ${{ github.repository }}
- **Run ID**: ${{ github.run_id }}
- **Date**: Generated daily at 6 AM UTC

## Phase 1: Load and Prepare Data

The issues data has been pre-fetched and is available at `/tmp/gh-aw/issues-data/issues.json`.

1. **Load the issues data**:
   ```bash
   jq 'length' /tmp/gh-aw/issues-data/issues.json
   ```

2. **Prepare data for Python analysis**:
   - Copy issues.json to `/tmp/gh-aw/python/data/issues.json`
   - Validate the data is properly formatted

## Phase 2: Python Analysis with Clustering

Create a Python script to analyze and cluster the issues. Use scikit-learn for clustering if available, or implement simple keyword-based clustering.

### Required Analysis

**Clustering Requirements**:
- Use TF-IDF vectorization on issue titles and bodies
- Apply K-means or hierarchical clustering
- Identify 5-10 major issue clusters/themes
- Label each cluster based on common keywords

**Metrics to Calculate**:
- Total issues (open vs closed)
- Issues opened in last 7, 14, 30 days
- Average time to close (for closed issues)
- Most active labels (by issue count)
- Most active authors
- Issues without labels (need triage)
- Issues without assignees
- Stale issues (no activity in 30+ days)

### Python Script Structure

```python
#!/usr/bin/env python3
"""
Daily Issues Analysis Script
Clusters issues and generates metrics and visualizations
"""
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
from datetime import datetime, timedelta
import json
from collections import Counter
import re

# Load issues data
with open('/tmp/gh-aw/python/data/issues.json', 'r') as f:
    issues = json.load(f)

df = pd.DataFrame(issues)

# Convert dates
df['createdAt'] = pd.to_datetime(df['createdAt'])
df['updatedAt'] = pd.to_datetime(df['updatedAt'])
df['closedAt'] = pd.to_datetime(df['closedAt'])

# Calculate basic metrics
total_issues = len(df)
open_issues = len(df[df['state'] == 'OPEN'])
closed_issues = len(df[df['state'] == 'CLOSED'])

# Time-based metrics
now = datetime.now(df['createdAt'].iloc[0].tzinfo if len(df) > 0 else None)
issues_7d = len(df[df['createdAt'] > now - timedelta(days=7)])
issues_30d = len(df[df['createdAt'] > now - timedelta(days=30)])

# Average time to close
closed_df = df[df['closedAt'].notna()]
if len(closed_df) > 0:
    closed_df['time_to_close'] = closed_df['closedAt'] - closed_df['createdAt']
    avg_close_time = closed_df['time_to_close'].mean()

# Extract labels for clustering
def extract_labels(labels_list):
    if labels_list:
        return [l['name'] for l in labels_list]
    return []

df['label_names'] = df['labels'].apply(extract_labels)

# Simple keyword-based clustering from titles
def cluster_by_keywords(title):
    title_lower = title.lower() if title else ''
    if 'bug' in title_lower or 'fix' in title_lower or 'error' in title_lower:
        return 'Bug Reports'
    elif 'feature' in title_lower or 'enhancement' in title_lower or 'request' in title_lower:
        return 'Feature Requests'
    elif 'doc' in title_lower or 'readme' in title_lower:
        return 'Documentation'
    elif 'test' in title_lower:
        return 'Testing'
    elif 'refactor' in title_lower or 'cleanup' in title_lower:
        return 'Refactoring'
    elif 'security' in title_lower or 'vulnerability' in title_lower:
        return 'Security'
    elif 'performance' in title_lower or 'slow' in title_lower:
        return 'Performance'
    else:
        return 'Other'

df['cluster'] = df['title'].apply(cluster_by_keywords)

# Save metrics to JSON for report
metrics = {
    'total_issues': total_issues,
    'open_issues': open_issues,
    'closed_issues': closed_issues,
    'issues_7d': issues_7d,
    'issues_30d': issues_30d,
    'cluster_counts': df['cluster'].value_counts().to_dict()
}
with open('/tmp/gh-aw/python/data/metrics.json', 'w') as f:
    json.dump(metrics, f, indent=2, default=str)
```

### Install Additional Libraries

If needed for better clustering:
```bash
pip install --user scikit-learn
```

## Phase 3: Generate Trend Charts

## Phase 3: Generate Trend Charts

Generate exactly **2 high-quality charts** following the chart quality standards from the Python Chart Discussion Report guide:

### Chart 1: Issue Activity Trends
- **Title**: "Issue Activity - Last 30 Days"
- **Content**: 
  - Line showing issues opened per day
  - Line showing issues closed per day
  - 7-day moving average overlay
- **Save to**: `/tmp/gh-aw/python/charts/issue_activity_trends.png`

### Chart 2: Issue Distribution by Cluster
- **Title**: "Issue Clusters by Theme"
- **Chart Type**: Horizontal bar chart
- **Content**:
  - Horizontal bars showing count per cluster
  - Include cluster labels based on keywords
  - Sort by count descending
- **Save to**: `/tmp/gh-aw/python/charts/issue_clusters.png`

## Phase 4: Upload Charts

Use the `upload asset` tool to upload both charts and collect the returned URLs for embedding in the discussion.

## Phase 5: Close Previous Discussions

Before creating the new discussion, find and close previous daily issues discussions:

1. Search for discussions with title prefix "[daily issues]"
2. Close each found discussion with reason "OUTDATED"
3. Add a closing comment: "This discussion has been superseded by a newer daily issues report."

Use the `close_discussion` safe output for each discussion found.

## Phase 6: Create Discussion Report

Create a new discussion with the comprehensive report.

### Discussion Format

**Title**: `[daily issues] Daily Issues Report - YYYY-MM-DD`

## Phase 6: Create Discussion Report

Create a new discussion following the standard report structure from the Python Chart Discussion Report guide.

**Title**: `[daily issues] Daily Issues Report - YYYY-MM-DD`

**Body**: Use the standard report structure with:

1. **Executive Summary**: 2-3 paragraphs summarizing total issues analyzed, main clusters identified, notable trends, and concerns
2. **Key Visualizations**: Embed both uploaded charts with 2-3 sentence analysis for each
3. **Detailed Metrics** (in collapsible `<details>` section):
   - Cluster details table
   - Volume metrics (total, open, closed counts)
   - Time-based metrics (7-day, 30-day, average close time)
   - Triage metrics (unlabeled, unassigned, stale)
   - Top labels and active authors tables
   - Issues needing attention lists
4. **Recommendations**: 3-5 specific, actionable recommendations
5. **Footer**: Workflow name, data source, generation info

## Important Guidelines

### Data Quality
- Handle missing fields gracefully (null checks)
- Validate date formats before processing
- Skip malformed issues rather than failing

### Clustering Tips
- If scikit-learn is not available, use keyword-based clustering
- Focus on meaningful themes, not just statistical clusters
- Aim for 5-10 clusters maximum for readability

## Success Criteria

A successful run will:
- ✅ Load and analyze all available issues data
- ✅ Cluster issues into meaningful themes
- ✅ Generate 2 high-quality trend charts
- ✅ Upload charts as assets
- ✅ Close previous daily issues discussions
- ✅ Create a new discussion with comprehensive report
- ✅ Include all required metrics and visualizations

Begin your analysis now. Load the data, run the Python analysis, generate charts, and create the discussion report.