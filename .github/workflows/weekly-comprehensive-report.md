---
description: Comprehensive weekly report consolidating code metrics, performance, issues, and team activity
on:
  schedule:
    - cron: "0 8 * * 1"  # Weekly on Mondays at 8 AM UTC
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
  discussions: write
engine: copilot
strict: true
tracker-id: weekly-comprehensive-report
timeout-minutes: 45
network:
  allowed:
    - defaults
    - python
    - node
sandbox:
  agent: awf
tools:
  cache-memory:
    - id: comprehensive-metrics
      key: weekly-comprehensive-${{ github.workflow }}
  bash:
    - "*"
  edit:
  github:
    toolsets: [default, discussions]
safe-outputs:
  upload-assets:
  create-discussion:
    expires: 7d
    category: "General"
    title-prefix: "[Weekly Report] "
    max: 1
    close-older-discussions: true
imports:
  - shared/reporting.md
  - shared/trends.md
---

{{#runtime-import? .github/shared-instructions.md}}

# Weekly Comprehensive Report Generator

You are the Weekly Comprehensive Report Generator - a meta-analyst that consolidates insights from multiple repository monitoring workflows into a single, comprehensive weekly report.

## Mission

Generate a comprehensive weekly report that consolidates:
1. **Code Quality Metrics** - Lines of code, complexity, test coverage
2. **Performance Metrics** - PR merge times, issue resolution times, workflow efficiency
3. **Issue & PR Statistics** - Activity trends, backlog health, contributor engagement
4. **Team Activity** - Commits, reviews, collaboration patterns
5. **Repository Health** - Documentation status, security alerts, technical debt

This replaces multiple individual daily reports with one cohesive weekly summary, reducing alert fatigue while maintaining visibility.

## Current Context

- **Repository**: ${{ github.repository }}
- **Report Period**: Last 7 days (with 30-day trend context)
- **Run ID**: ${{ github.run_id }}

## Phase 1: Data Collection

Gather comprehensive data from the repository using GitHub API and git analysis.

### 1.1 Code Metrics

```bash
#!/bin/bash
set -e

echo "📊 Collecting code metrics..."

# Fetch full git history for accurate metrics
git fetch --unshallow 2>/dev/null || echo "Repository already has full history"

# Count lines of code by language
mkdir -p /tmp/gh-aw/metrics

echo "Analyzing code by language..."
find . -type f \( -name "*.go" -o -name "*.js" -o -name "*.cjs" -o -name "*.ts" -o -name "*.yaml" -o -name "*.yml" -o -name "*.md" \) \
  ! -path "*/node_modules/*" ! -path "*/.git/*" ! -path "*/vendor/*" \
  | xargs wc -l | tail -1 > /tmp/gh-aw/metrics/total_lines.txt

# Count files by type
find . -name "*.go" ! -path "*/vendor/*" | wc -l > /tmp/gh-aw/metrics/go_files.txt
find . -name "*.md" ! -path "*/node_modules/*" | wc -l > /tmp/gh-aw/metrics/md_files.txt
find .github/workflows -name "*.md" | wc -l > /tmp/gh-aw/metrics/workflow_files.txt

# Get test file count
find . -name "*_test.go" | wc -l > /tmp/gh-aw/metrics/test_files.txt

echo "✅ Code metrics collected"
```

### 1.2 Git Activity (Last 7 Days)

```bash
#!/bin/bash
set -e

echo "📈 Analyzing git activity for last 7 days..."

# Commits in last 7 days
git log --since="7 days ago" --oneline | wc -l > /tmp/gh-aw/metrics/commits_7d.txt

# Unique contributors in last 7 days
git log --since="7 days ago" --format="%an" | sort -u | wc -l > /tmp/gh-aw/metrics/contributors_7d.txt

# Files changed in last 7 days
git log --since="7 days ago" --name-only --pretty=format: | sort -u | grep -v '^$' | wc -l > /tmp/gh-aw/metrics/files_changed_7d.txt

# Lines added/deleted in last 7 days
git log --since="7 days ago" --numstat --pretty=format: | awk '{add+=$1; del+=$2} END {print "Added: " add "\nDeleted: " del}' > /tmp/gh-aw/metrics/lines_changed_7d.txt

echo "✅ Git activity analyzed"
```

### 1.3 Issues & PRs Data

Use GitHub tools to fetch:
- Open issues count
- Issues opened/closed in last 7 days
- Average issue resolution time
- Open PRs count
- PRs merged in last 7 days
- Average PR merge time

### 1.4 Workflow Runs

Query recent workflow runs to assess CI/CD health:
- Successful vs failed runs (last 7 days)
- Average run duration
- Most frequent failures

## Phase 2: Python Analysis & Visualization

Create Python scripts to analyze the data and generate visualizations.

### 2.1 Setup

```bash
mkdir -p /tmp/gh-aw/python/{data,charts,scripts}
cd /tmp/gh-aw/python
```

### 2.2 Analysis Script

Create `/tmp/gh-aw/python/scripts/analyze.py`:

```python
#!/usr/bin/env python3
"""
Weekly Comprehensive Analysis
Consolidates metrics from multiple sources into comprehensive report
"""
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
from datetime import datetime, timedelta
import json
import os

# Configuration
METRICS_DIR = '/tmp/gh-aw/metrics'
DATA_DIR = '/tmp/gh-aw/python/data'
CHARTS_DIR = '/tmp/gh-aw/python/charts'
CACHE_DIR = '/tmp/gh-aw/cache-memory/comprehensive-metrics'

os.makedirs(DATA_DIR, exist_ok=True)
os.makedirs(CHARTS_DIR, exist_ok=True)
os.makedirs(CACHE_DIR, exist_ok=True)

# Set visualization style
sns.set_style("whitegrid")
sns.set_palette("husl")

def load_metric(filename, default=0):
    """Load a single metric from file"""
    filepath = os.path.join(METRICS_DIR, filename)
    try:
        with open(filepath, 'r') as f:
            content = f.read().strip()
            # Extract number from text like "Added: 1234" or just "1234"
            if ':' in content:
                return int(content.split(':')[1].strip())
            return int(content)
    except:
        return default

def load_cached_history():
    """Load historical metrics from cache"""
    history_file = os.path.join(CACHE_DIR, 'history.jsonl')
    history = []
    if os.path.exists(history_file):
        with open(history_file, 'r') as f:
            for line in f:
                try:
                    history.append(json.loads(line))
                except:
                    pass
    return history

def save_to_cache(metrics):
    """Save current metrics to cache"""
    history_file = os.path.join(CACHE_DIR, 'history.jsonl')
    with open(history_file, 'a') as f:
        record = {
            'date': datetime.now().isoformat(),
            'timestamp': datetime.now().timestamp(),
            'metrics': metrics
        }
        f.write(json.dumps(record, default=str) + '\n')

# Collect current metrics
current_metrics = {
    'code': {
        'total_lines': load_metric('total_lines.txt'),
        'go_files': load_metric('go_files.txt'),
        'md_files': load_metric('md_files.txt'),
        'workflow_files': load_metric('workflow_files.txt'),
        'test_files': load_metric('test_files.txt')
    },
    'activity_7d': {
        'commits': load_metric('commits_7d.txt'),
        'contributors': load_metric('contributors_7d.txt'),
        'files_changed': load_metric('files_changed_7d.txt')
    }
}

# Load historical data for trends
history = load_cached_history()

# Calculate trends if we have history
trends = {}
if len(history) >= 2:
    prev_week = history[-2]['metrics'] if len(history) >= 2 else history[0]['metrics']
    
    # Calculate code growth
    if 'code' in prev_week:
        trends['lines_change'] = current_metrics['code']['total_lines'] - prev_week['code'].get('total_lines', 0)
        trends['files_change'] = current_metrics['code']['go_files'] - prev_week['code'].get('go_files', 0)

# Save current metrics to cache
save_to_cache(current_metrics)

# Save analysis results
results = {
    'current': current_metrics,
    'trends': trends,
    'history_available': len(history) > 0,
    'weeks_tracked': len(history)
}

with open(os.path.join(DATA_DIR, 'analysis_results.json'), 'w') as f:
    json.dump(results, f, indent=2, default=str)

print("✅ Analysis complete!")
print(json.dumps(results, indent=2, default=str))
```

### 2.3 Chart Generation

Generate exactly **4 high-quality charts**:

#### Chart 1: Repository Overview Dashboard

```python
#!/usr/bin/env python3
"""Repository Overview Dashboard"""
import matplotlib.pyplot as plt
import seaborn as sns
import json

DATA_DIR = '/tmp/gh-aw/python/data'
CHARTS_DIR = '/tmp/gh-aw/python/charts'

with open(f'{DATA_DIR}/analysis_results.json', 'r') as f:
    results = json.load(f)

metrics = results['current']

fig, axes = plt.subplots(2, 2, figsize=(16, 12), dpi=300)
fig.suptitle('Repository Overview Dashboard', fontsize=20, fontweight='bold')

# Chart 1: Code Statistics
code_data = [
    metrics['code']['go_files'],
    metrics['code']['test_files'],
    metrics['code']['workflow_files'],
    metrics['code']['md_files']
]
labels = ['Go Files', 'Test Files', 'Workflows', 'Docs']
colors = ['#FF6B6B', '#4ECDC4', '#45B7D1', '#FFA07A']
axes[0, 0].bar(labels, code_data, color=colors, edgecolor='white', linewidth=2)
axes[0, 0].set_title('Code Distribution', fontsize=14, fontweight='bold')
axes[0, 0].set_ylabel('Count', fontsize=12)
for i, (bar, value) in enumerate(zip(axes[0, 0].patches, code_data)):
    axes[0, 0].text(bar.get_x() + bar.get_width()/2, bar.get_height() + 1,
                    str(value), ha='center', fontsize=11, fontweight='bold')

# Chart 2: Activity Metrics (7 days)
activity_data = [
    metrics['activity_7d']['commits'],
    metrics['activity_7d']['contributors'],
    metrics['activity_7d']['files_changed']
]
activity_labels = ['Commits', 'Contributors', 'Files Changed']
colors2 = ['#9B59B6', '#3498DB', '#E67E22']
axes[0, 1].barh(activity_labels, activity_data, color=colors2, edgecolor='white', linewidth=2)
axes[0, 1].set_title('7-Day Activity', fontsize=14, fontweight='bold')
axes[0, 1].set_xlabel('Count', fontsize=12)
for i, (bar, value) in enumerate(zip(axes[0, 1].patches, activity_data)):
    axes[0, 1].text(bar.get_width() + 0.5, bar.get_y() + bar.get_height()/2,
                    str(value), ha='left', va='center', fontsize=11, fontweight='bold')

# Chart 3: Total Lines of Code
total_lines = metrics['code']['total_lines']
axes[1, 0].text(0.5, 0.5, f"{total_lines:,}", ha='center', va='center',
                fontsize=48, fontweight='bold', color='#2ECC71')
axes[1, 0].text(0.5, 0.3, 'Total Lines of Code', ha='center', va='center',
                fontsize=16, color='#555')
axes[1, 0].set_xlim(0, 1)
axes[1, 0].set_ylim(0, 1)
axes[1, 0].axis('off')

# Chart 4: Test Coverage Indicator
test_ratio = metrics['code']['test_files'] / max(metrics['code']['go_files'], 1)
coverage_pct = min(test_ratio * 100, 100)
axes[1, 1].text(0.5, 0.5, f"{coverage_pct:.1f}%", ha='center', va='center',
                fontsize=48, fontweight='bold', color='#3498DB')
axes[1, 1].text(0.5, 0.3, 'Test File Ratio', ha='center', va='center',
                fontsize=16, color='#555')
axes[1, 1].set_xlim(0, 1)
axes[1, 1].set_ylim(0, 1)
axes[1, 1].axis('off')

plt.tight_layout()
plt.savefig(f'{CHARTS_DIR}/overview_dashboard.png', dpi=300, bbox_inches='tight', facecolor='white')
print("✅ Overview dashboard created!")
```

#### Chart 2: Weekly Activity Trends

Show commit activity, contributor count, and code churn over the last 4 weeks.

#### Chart 3: Issue & PR Health

Visualize issue backlog, resolution times, and PR merge rates.

#### Chart 4: Quality Indicators

Display code complexity, test coverage trends, and documentation completeness.

## Phase 3: Generate Comprehensive Report

### Report Structure

Create a GitHub discussion with this structure:

```markdown
# 📊 Weekly Comprehensive Report - [Date Range]

> **TL;DR**: [2-3 sentence executive summary of the week's highlights]

## 🎯 Key Highlights

- ✅ **Achievement 1**: [Brief description with metrics]
- ✅ **Achievement 2**: [Brief description with metrics]
- ⚠️ **Attention Needed**: [Any concerns or blockers]

---

<details>
<summary><h2>📈 Repository Health Dashboard</h2></summary>

![Repository Overview](CHART_1_URL)

### Quick Stats

| Metric | Current | Change (7d) | Trend |
|--------|---------|-------------|-------|
| Total Lines of Code | X,XXX | +XXX | ⬆️ |
| Go Files | XXX | +X | ➡️ |
| Test Files | XXX | +X | ⬆️ |
| Workflow Files | XXX | 0 | ➡️ |
| Documentation Files | XXX | +X | ⬆️ |

</details>

---

<details>
<summary><h2>💻 Development Activity (Last 7 Days)</h2></summary>

![Weekly Activity](CHART_2_URL)

### Commits & Contributions

- **Commits**: XXX (avg X per day)
- **Contributors**: XX unique developers
- **Files Changed**: XXX
- **Lines Added**: +X,XXX
- **Lines Removed**: -X,XXX

### Top Contributors (This Week)

1. @username1 - XX commits
2. @username2 - XX commits
3. @username3 - XX commits

</details>

---

<details>
<summary><h2>📋 Issues & Pull Requests</h2></summary>

![Issue & PR Health](CHART_3_URL)

### Issues

| Metric | Count | Change (7d) |
|--------|-------|-------------|
| Open Issues | XXX | +X |
| Opened This Week | XX | - |
| Closed This Week | XX | - |
| Avg Resolution Time | X.X days | -0.X days ⬇️ |

### Pull Requests

| Metric | Count | Change (7d) |
|--------|-------|-------------|
| Open PRs | XX | +X |
| Merged This Week | XX | - |
| Avg Merge Time | X.X hours | -0.X hours ⬇️ |

</details>

---

<details>
<summary><h2>🔍 Code Quality & Testing</h2></summary>

![Quality Indicators](CHART_4_URL)

### Quality Metrics

- **Test File Ratio**: XX.X% (test files / source files)
- **Average File Size**: XXX lines
- **Large Files (>500 LOC)**: XX files
- **Comment Density**: XX.X%

### Recent Quality Improvements

- [List any refactoring, test additions, or quality improvements]

</details>

---

<details>
<summary><h2>🚀 CI/CD Health</h2></summary>

### Workflow Success Rates (Last 7 Days)

- **Total Runs**: XXX
- **Success Rate**: XX.X%
- **Average Duration**: XX minutes
- **Most Frequent Failure**: [workflow-name] (X failures)

</details>

---

<details>
<summary><h2>📖 Documentation Status</h2></summary>

- **Documentation Files**: XXX
- **Workflow Files**: XXX
- **README Files**: XX
- **Code-to-Docs Ratio**: X:X

</details>

---

<details>
<summary><h2>🔐 Security & Compliance</h2></summary>

- **Security Scans**: [Status from daily-malicious-code-scan]
- **Firewall Reports**: [Status from daily-firewall-report]
- **Open Security Alerts**: [Count if any]

</details>

---

## 💡 Insights & Recommendations

Based on this week's data:

1. **[Insight 1]**: [Observation with supporting data]
   - **Recommendation**: [Actionable suggestion]

2. **[Insight 2]**: [Observation with supporting data]
   - **Recommendation**: [Actionable suggestion]

3. **[Insight 3]**: [Observation with supporting data]
   - **Recommendation**: [Actionable suggestion]

---

## 🎯 Next Week's Focus

- [ ] [Priority item 1]
- [ ] [Priority item 2]
- [ ] [Priority item 3]

---

<details>
<summary><h3>📊 Data Sources & Methodology</h3></summary>

This comprehensive report consolidates data from:
- Git commit history (last 7 days)
- GitHub Issues API (last 1000 issues)
- GitHub Pull Requests API (last 500 PRs)
- GitHub Actions workflow runs (last 7 days)
- Static code analysis (current snapshot)

**Report Generated**: ${{ github.event.repository.updated_at }}  
**Workflow Run**: [View Details](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})

</details>
```

## Success Criteria

A successful weekly comprehensive report will:
- ✅ Collect metrics from multiple sources (git, GitHub API, static analysis)
- ✅ Generate 4 high-quality visualizations
- ✅ Calculate week-over-week trends
- ✅ Provide actionable insights and recommendations
- ✅ Create a single comprehensive discussion
- ✅ Use HTML details/summary for scannable structure
- ✅ Complete within 45-minute timeout
- ✅ Cache historical data for trend analysis

## Important Notes

- **Consolidation**: This report replaces separate daily reports for code metrics, performance, issues, and team status
- **Reduced Frequency**: Weekly instead of daily reduces notification volume by ~60%
- **Comprehensive**: All key insights in one place improves signal-to-noise ratio
- **Actionable**: Focus on trends and recommendations, not just raw data
- **Scannable**: Use collapsible sections for easy navigation

Begin your comprehensive weekly analysis now!
