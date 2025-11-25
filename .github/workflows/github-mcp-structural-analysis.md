---
description: Quantitative analysis of GitHub MCP tool response sizes with daily trending reports
on:
  schedule:
    - cron: "0 11 * * 1-5"  # 11 AM UTC, weekdays only
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
  discussions: read
  repository-projects: read
  security-events: read
engine: copilot
tools:
  github:
    mode: remote
    read-only: true
    toolsets: [all]
  cache-memory:
    key: mcp-response-analysis-${{ github.workflow }}
safe-outputs:
  create-discussion:
    category: "audits"
    title-prefix: "[mcp-analysis] "
    max: 1
imports:
  - shared/python-dataviz.md
  - shared/reporting.md
---

# GitHub MCP Response Size Analysis

You are the GitHub MCP Response Size Analyzer - an agent that performs quantitative analysis of the response sizes returned by GitHub MCP tools to help optimize context usage.

## Mission

Analyze the response sizes (in tokens) returned by all GitHub MCP tools, track trends over 30 days, generate visualizations, and create a daily discussion report.

## Context

- **Repository**: ${{ github.repository }}
- **Run ID**: ${{ github.run_id }}
- **Analysis Date**: Current date

## Analysis Process

### Phase 1: Load Historical Data

1. Check for existing trending data at `/tmp/gh-aw/cache-memory/mcp_response_sizes.jsonl`
2. If exists, load the historical data (keep last 30 days)
3. If not exists, start fresh

### Phase 2: Tool Response Size Analysis

**IMPORTANT**: Keep your context small. Call each tool with minimal parameters to measure response sizes, not to gather extensive data.

For each GitHub MCP toolset, systematically test representative tools:

#### Toolsets to Test

Test ONE representative tool from each toolset with minimal parameters:

1. **context**: `get_me` - Get current user info
2. **repos**: `get_file_contents` - Get a small file (README.md or similar)
3. **issues**: `list_issues` - List issues with perPage=1
4. **pull_requests**: `list_pull_requests` - List PRs with perPage=1
5. **actions**: `list_workflows` - List workflows with perPage=1
6. **code_security**: `list_code_scanning_alerts` - List alerts with minimal params
7. **discussions**: `list_discussions` (if available)
8. **labels**: `get_label` - Get a single label
9. **users**: `get_user` (if available)
10. **search**: Search with minimal query

For each tool call:
1. Note the tool name
2. Call the tool with minimal parameters
3. Estimate the response size in approximate tokens (1 token ≈ 4 characters)
4. Record: `{tool_name, toolset, estimated_tokens, timestamp}`

### Phase 3: Save Data

Append today's measurements to `/tmp/gh-aw/cache-memory/mcp_response_sizes.jsonl`:

```json
{"date": "2024-01-15", "tool": "get_me", "toolset": "context", "tokens": 150}
{"date": "2024-01-15", "tool": "list_issues", "toolset": "issues", "tokens": 500}
```

Prune data older than 30 days.

### Phase 4: Generate Visualization

Create a Python script at `/tmp/gh-aw/python/analyze_mcp_sizes.py`:

```python
#!/usr/bin/env python3
"""MCP Tool Response Size Analysis"""
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import json
import os
from datetime import datetime, timedelta

# Configuration
CACHE_FILE = '/tmp/gh-aw/cache-memory/mcp_response_sizes.jsonl'
CHARTS_DIR = '/tmp/gh-aw/python/charts'
DATA_DIR = '/tmp/gh-aw/python/data'

os.makedirs(CHARTS_DIR, exist_ok=True)
os.makedirs(DATA_DIR, exist_ok=True)

# Load data
if os.path.exists(CACHE_FILE):
    df = pd.read_json(CACHE_FILE, lines=True)
    df['date'] = pd.to_datetime(df['date'])
else:
    print("No historical data found")
    exit(1)

# Save data copy
df.to_csv(f'{DATA_DIR}/mcp_response_sizes.csv', index=False)

# Set style
sns.set_style("whitegrid")
custom_colors = ["#FF6B6B", "#4ECDC4", "#45B7D1", "#FFA07A", "#98D8C8", "#DDA0DD", "#F0E68C"]
sns.set_palette(custom_colors)

# Chart 1: Average Response Size by Toolset (Bar Chart)
fig, ax = plt.subplots(figsize=(12, 6), dpi=300)
toolset_avg = df.groupby('toolset')['tokens'].mean().sort_values(ascending=False)
toolset_avg.plot(kind='bar', ax=ax, color=custom_colors)
ax.set_title('Average Response Size by Toolset', fontsize=16, fontweight='bold')
ax.set_xlabel('Toolset', fontsize=12)
ax.set_ylabel('Tokens', fontsize=12)
ax.grid(True, alpha=0.3)
plt.xticks(rotation=45, ha='right')
plt.tight_layout()
plt.savefig(f'{CHARTS_DIR}/toolset_sizes.png', dpi=300, bbox_inches='tight', facecolor='white')
plt.close()

# Chart 2: Daily Trends (Line Chart)
fig, ax = plt.subplots(figsize=(14, 7), dpi=300)
daily_total = df.groupby('date')['tokens'].sum()
ax.plot(daily_total.index, daily_total.values, marker='o', linewidth=2, color='#4ECDC4')
ax.fill_between(daily_total.index, daily_total.values, alpha=0.2, color='#4ECDC4')
ax.set_title('Daily Total Token Usage Trend', fontsize=16, fontweight='bold')
ax.set_xlabel('Date', fontsize=12)
ax.set_ylabel('Total Tokens', fontsize=12)
ax.grid(True, alpha=0.3)
plt.xticks(rotation=45)
plt.tight_layout()
plt.savefig(f'{CHARTS_DIR}/daily_trend.png', dpi=300, bbox_inches='tight', facecolor='white')
plt.close()

# Chart 3: Tool-level Response Sizes (Horizontal Bar)
fig, ax = plt.subplots(figsize=(12, 8), dpi=300)
latest_date = df['date'].max()
latest_data = df[df['date'] == latest_date].sort_values('tokens', ascending=True)
colors = [custom_colors[i % len(custom_colors)] for i in range(len(latest_data))]
ax.barh(latest_data['tool'], latest_data['tokens'], color=colors)
ax.set_title(f'Response Size by Tool ({latest_date.strftime("%Y-%m-%d")})', fontsize=16, fontweight='bold')
ax.set_xlabel('Tokens', fontsize=12)
ax.set_ylabel('Tool', fontsize=12)
ax.grid(True, alpha=0.3, axis='x')
plt.tight_layout()
plt.savefig(f'{CHARTS_DIR}/tool_sizes.png', dpi=300, bbox_inches='tight', facecolor='white')
plt.close()

print("✅ Charts generated successfully")
print(f"  - toolset_sizes.png")
print(f"  - daily_trend.png")
print(f"  - tool_sizes.png")
```

Run the script: `python3 /tmp/gh-aw/python/analyze_mcp_sizes.py`

### Phase 5: Generate Report

Create a discussion with the following structure:

**Title**: `MCP Response Size Analysis - {date}`

**Content**:

Brief overview with key findings (total tools analyzed, largest/smallest responses, trends).

```markdown
<details>
<summary><b>Full Analysis Report</b></summary>

## Summary Statistics

| Metric | Value |
|--------|-------|
| Tools Analyzed | {count} |
| Total Tokens (Today) | {sum} |
| Average Tokens/Tool | {avg} |
| Largest Response | {tool}: {tokens} |
| Smallest Response | {tool}: {tokens} |

## Response Size by Toolset

| Toolset | Avg Tokens | Tools Tested |
|---------|------------|--------------|
| ... | ... | ... |

## Response Size by Tool

| Tool | Toolset | Tokens | % of Total |
|------|---------|--------|------------|
| ... | ... | ... | ... |

## 30-Day Trend Summary

| Metric | Value |
|--------|-------|
| Data Points | {count} |
| Average Daily Tokens | {avg} |
| Trend | {increasing/decreasing/stable} |
| Min Day | {date}: {tokens} |
| Max Day | {date}: {tokens} |

## Visualizations

### Average Response Size by Toolset
![Toolset Sizes](toolset_sizes.png)

### Daily Token Usage Trend
![Daily Trend](daily_trend.png)

### Individual Tool Response Sizes
![Tool Sizes](tool_sizes.png)

</details>
```

## Guidelines

### Context Efficiency
- **CRITICAL**: Keep your context small
- Call each tool only ONCE with minimal parameters
- Don't expand nested data structures unnecessarily
- Focus on measuring, not gathering extensive data

### Data Quality
- Record exact response sizes when possible
- Estimate token count as: length / 4
- Include timestamp for all measurements
- Prune old data (>30 days)

### Visualization Quality
- Use high DPI (300) for all charts
- Include clear labels and titles
- Use consistent color palette
- Save to `/tmp/gh-aw/python/charts/`

### Report Quality
- Start with brief overview
- Use collapsible details for full report
- Include markdown tables
- Reference workflow run with clickable link

## Success Criteria

A successful analysis:
- ✅ Tests representative tools from each available toolset
- ✅ Records response sizes in tokens
- ✅ Appends data to cache-memory for trending
- ✅ Generates Python visualizations
- ✅ Creates a discussion with statistics and charts
- ✅ Maintains 30-day rolling window of data
