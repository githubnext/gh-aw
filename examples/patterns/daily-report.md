---
# Pattern: Daily Report
# Complexity: Intermediate
# Use Case: Generate comprehensive scheduled reports with data analysis and visualizations
name: Daily Report
description: Generates daily reports analyzing repository activity with metrics and visualizations
on:
  schedule:
    # TODO: Customize schedule (daily at 6 AM UTC by default)
    - cron: "0 6 * * *"
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
  discussions: write
  actions: read
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [default, discussions]
  bash:
    - "python3 *"
    - "pip install *"
safe-outputs:
  create-discussion:
    category: "General"
    # TODO: Customize title prefix for your report type
    title-prefix: "[Daily Report] "
    max: 1
    expires: 7d
    close-older-discussions: true
  upload-asset:
timeout-minutes: 30
strict: true
---

# Daily Report Generator

Generate comprehensive daily reports analyzing repository activity, including metrics, trends, and visualizations.

## Report Types

# TODO: Choose ONE report type to implement, or combine multiple

### Type 1: Issues Report

Analyze repository issues with clustering and metrics:

**Metrics to Include**:
- Total issues (open vs closed)
- Issues opened/closed in last 7, 14, 30 days
- Average time to first response
- Average time to close
- Most active labels
- Issues needing attention (no labels, no assignee, stale)
- Issue clustering by topic/theme

**Visualizations**:
- Issue activity trend chart (7-30 days)
- Label distribution pie chart
- Time-to-close histogram

### Type 2: Pull Request Report

Analyze PR activity and merge statistics:

**Metrics to Include**:
- PRs opened/merged/closed
- Average time to merge
- Review response time
- PR size distribution
- Most active contributors
- PRs needing review

**Visualizations**:
- PR throughput over time
- Review time trends
- Contributor activity chart

### Type 3: Repository Health Report

Overall repository health and activity:

**Metrics to Include**:
- Overall activity score
- Issue/PR velocity
- Contributor growth
- Code churn
- CI/CD success rates
- Test coverage trends

**Visualizations**:
- Health score dashboard
- Multi-metric trend chart
- Contributor growth graph

### Type 4: Code Quality Report

Analyze code quality metrics and technical debt:

**Metrics to Include**:
- Lines of code (by language)
- Code complexity trends
- Test coverage
- Documentation coverage
- Dependency health
- Security alerts

**Visualizations**:
- Code complexity heatmap
- Coverage trends
- Dependency age chart

## Implementation Steps

### Phase 1: Data Collection

```bash
# TODO: Customize data collection for your report type

# Example: Fetch issues data
gh issue list --repo ${{ github.repository }} \
  --limit 1000 \
  --json number,title,body,state,createdAt,closedAt,labels,assignees,comments \
  > /tmp/gh-aw/data/issues.json

# Example: Fetch PR data
gh pr list --repo ${{ github.repository }} \
  --limit 500 \
  --state all \
  --json number,title,createdAt,mergedAt,additions,deletions,reviews \
  > /tmp/gh-aw/data/prs.json

# Verify data was collected
jq 'length' /tmp/gh-aw/data/issues.json
```

### Phase 2: Data Analysis with Python

```python
#!/usr/bin/env python3
"""
Daily Report Analysis Script
TODO: Customize this script for your specific report type
"""
import json
import pandas as pd
import numpy as np
from datetime import datetime, timedelta
from collections import Counter

# Load data
with open('/tmp/gh-aw/data/issues.json') as f:
    issues = json.load(f)

# Convert to DataFrame for easier analysis
df = pd.DataFrame(issues)
df['createdAt'] = pd.to_datetime(df['createdAt'])
df['closedAt'] = pd.to_datetime(df['closedAt'])

# Calculate metrics
now = datetime.now()
last_7d = now - timedelta(days=7)
last_30d = now - timedelta(days=30)

metrics = {
    'total_open': len(df[df['state'] == 'open']),
    'total_closed': len(df[df['state'] == 'closed']),
    'opened_last_7d': len(df[df['createdAt'] > last_7d]),
    'closed_last_7d': len(df[(df['closedAt'] > last_7d) & (df['state'] == 'closed')]),
    'opened_last_30d': len(df[df['createdAt'] > last_30d]),
}

# Calculate average time to close (for closed issues)
closed_df = df[df['state'] == 'closed'].copy()
closed_df['time_to_close'] = (closed_df['closedAt'] - closed_df['createdAt']).dt.total_seconds() / 86400
metrics['avg_time_to_close_days'] = closed_df['time_to_close'].mean()

# Most active labels
all_labels = []
for labels in df['labels']:
    all_labels.extend([label['name'] for label in labels])
label_counts = Counter(all_labels)
metrics['top_labels'] = label_counts.most_common(10)

# Issues needing attention
metrics['no_labels'] = len(df[(df['state'] == 'open') & (df['labels'].apply(len) == 0)])
metrics['no_assignees'] = len(df[(df['state'] == 'open') & (df['assignees'].apply(len) == 0)])

# Save metrics
with open('/tmp/gh-aw/data/metrics.json', 'w') as f:
    json.dump(metrics, f, indent=2, default=str)

print(f"Analysis complete. Analyzed {len(df)} issues.")
```

### Phase 3: Generate Visualizations

```python
#!/usr/bin/env python3
"""
Visualization Generation Script
TODO: Customize charts for your report
"""
import json
import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.dates as mdates

# Load data
with open('/tmp/gh-aw/data/issues.json') as f:
    issues = json.load(f)

df = pd.DataFrame(issues)
df['createdAt'] = pd.to_datetime(df['createdAt'])

# Create assets directory
import os
os.makedirs('/tmp/gh-aw/assets', exist_ok=True)

# Chart 1: Issue activity over time (last 30 days)
plt.figure(figsize=(12, 6))
recent_df = df[df['createdAt'] > (datetime.now() - timedelta(days=30))]
daily_counts = recent_df.groupby(recent_df['createdAt'].dt.date).size()

plt.plot(daily_counts.index, daily_counts.values, marker='o', linewidth=2)
plt.title('Daily Issue Activity (Last 30 Days)', fontsize=14, fontweight='bold')
plt.xlabel('Date')
plt.ylabel('Issues Opened')
plt.grid(True, alpha=0.3)
plt.xticks(rotation=45)
plt.tight_layout()
plt.savefig('/tmp/gh-aw/assets/issue-activity.png', dpi=150)
plt.close()

# Chart 2: Label distribution (top 10)
with open('/tmp/gh-aw/data/metrics.json') as f:
    metrics = json.load(f)

top_labels = metrics['top_labels'][:10]
labels, counts = zip(*top_labels)

plt.figure(figsize=(10, 6))
plt.barh(labels, counts, color='steelblue')
plt.xlabel('Number of Issues')
plt.title('Top Issue Labels', fontsize=14, fontweight='bold')
plt.tight_layout()
plt.savefig('/tmp/gh-aw/assets/label-distribution.png', dpi=150)
plt.close()

print("Visualizations generated successfully.")
```

### Phase 4: Format Report

```markdown
Create a well-formatted markdown report:

# TODO: Customize report structure and content

## 📊 Daily Report - [Date]

### Summary

**Activity Snapshot**:
- 📈 Total Issues: [open_count] open, [closed_count] closed
- 🆕 New Issues (7d): [new_7d]
- ✅ Closed Issues (7d): [closed_7d]
- ⏱️ Avg Time to Close: [avg_days] days

### Trends

![Issue Activity](issue-activity.png)

**Key Observations**:
- [Trend 1: increasing/decreasing activity]
- [Trend 2: patterns or anomalies]
- [Trend 3: comparison to previous periods]

### Top Labels

![Label Distribution](label-distribution.png)

**Most Active Areas**:
1. [label1] - [count] issues
2. [label2] - [count] issues
3. [label3] - [count] issues

### Action Items

**Needs Attention**:
- 🏷️ [count] issues without labels
- 👤 [count] issues without assignees
- 💤 [count] stale issues (no activity in 30+ days)

**Links**:
- [Issues needing labels](filter-url)
- [Unassigned issues](filter-url)
- [Stale issues](filter-url)

---
*Report generated by [Daily Report]({run_url}) at {timestamp}*
```

### Phase 5: Create Discussion

```markdown
Use the create-discussion safe-output:
- Title includes date: "[Daily Report] December 29, 2025"
- Category: "General" (or customize)
- Body: Your formatted markdown with embedded charts
- Auto-close older reports (set expires and close-older-discussions)
```

## Customization Guide

### Adjust Report Schedule

# TODO: Choose your reporting frequency
```yaml
# Daily at 6 AM UTC
schedule:
  - cron: "0 6 * * *"

# Weekly on Monday at 9 AM UTC
schedule:
  - cron: "0 9 * * 1"

# Every 12 hours
schedule:
  - cron: "0 */12 * * *"
```

### Configure Python Dependencies

```yaml
tools:
  bash:
    - "python3 *"
    - "pip install pandas matplotlib numpy scikit-learn"
```

### Add Custom Metrics

```python
# TODO: Add your custom metrics

# Example: Calculate PR review response time
def calculate_review_time(prs):
    times = []
    for pr in prs:
        if pr['reviews']:
            created = datetime.fromisoformat(pr['createdAt'])
            first_review = datetime.fromisoformat(pr['reviews'][0]['submittedAt'])
            times.append((first_review - created).total_seconds() / 3600)
    return np.mean(times) if times else None

# Example: Detect issue clusters with NLP
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.cluster import KMeans

def cluster_issues(issues, n_clusters=5):
    texts = [f"{issue['title']} {issue['body']}" for issue in issues]
    vectorizer = TfidfVectorizer(max_features=100, stop_words='english')
    X = vectorizer.fit_transform(texts)
    kmeans = KMeans(n_clusters=n_clusters, random_state=42)
    clusters = kmeans.fit_predict(X)
    return clusters
```

### Add More Visualizations

```python
# TODO: Create additional charts

# Example: Heatmap of issue activity by day of week and hour
import seaborn as sns

df['day_of_week'] = df['createdAt'].dt.day_name()
df['hour'] = df['createdAt'].dt.hour
pivot = df.pivot_table(index='day_of_week', columns='hour', aggfunc='size', fill_value=0)

plt.figure(figsize=(14, 6))
sns.heatmap(pivot, cmap='YlOrRd', annot=True, fmt='d')
plt.title('Issue Creation Heatmap')
plt.savefig('/tmp/gh-aw/assets/activity-heatmap.png', dpi=150)
```

## Example Output

```markdown
## 📊 Daily Repository Report - December 29, 2025

### 🎯 Summary

**Issue Activity**:
- 📈 Total: 234 open, 1,456 closed
- 🆕 New (7d): 23 issues
- ✅ Closed (7d): 18 issues
- ⏱️ Avg Time to Close: 4.2 days

**Pull Requests**:
- 🔄 Open: 12 PRs
- ✅ Merged (7d): 31 PRs
- ⏱️ Avg Time to Merge: 1.8 days

### 📈 Trends

![Issue Activity](issue-activity.png)

**Observations**:
- Activity increased 15% compared to last week
- Bug reports decreased by 20%
- Feature requests trending upward

### 🏷️ Top Labels

![Label Distribution](label-distribution.png)

1. **documentation** - 45 issues (community focus)
2. **bug** - 38 issues (stable rate)
3. **enhancement** - 32 issues (growing backlog)

### ⚠️ Action Items

**Needs Attention**:
- 🏷️ 12 issues need labels → [View](link)
- 👤 18 issues need assignment → [View](link)
- 💤 7 stale issues (>30d inactive) → [View](link)

**High Priority**:
- #1234 - Critical bug affecting production
- #1235 - Security vulnerability reported
- #1236 - Breaking change for v2.0

### 📊 Health Score: 85/100

**Strengths**:
- ✅ Fast response times
- ✅ High PR merge rate
- ✅ Active community

**Areas to Improve**:
- ⚠️ Growing backlog of enhancements
- ⚠️ Some issues lack triage

---
*Report generated by [Daily Report](run-url) • [View Previous Reports](discussions-url)*
```

## Related Examples

- **Production examples**:
  - `.github/workflows/daily-issues-report.md` - Comprehensive issues report
  - `.github/workflows/copilot-pr-merged-report.md` - PR activity report
  - `.github/workflows/daily-code-metrics.md` - Code quality metrics

## Tips

- **Cache data**: Store intermediate results to speed up reruns
- **Handle errors**: Add error handling for missing data
- **Test locally**: Use workflow_dispatch to test before scheduling
- **Optimize performance**: Limit data fetching to necessary time ranges
- **Make it actionable**: Include links to filtered views
- **Track trends**: Compare current metrics to previous periods
- **Keep it scannable**: Use charts and formatting to highlight key info

## Security Considerations

- This workflow reads repository data and creates discussions
- Uses `strict: true` for enhanced security
- Python execution is sandboxed
- All operations validated through safe-outputs

---

**Pattern Info**:
- Complexity: Intermediate
- Trigger: Scheduled + manual dispatch
- Safe Outputs: create_discussion, upload_asset
- Tools: GitHub (default, discussions), bash (Python)
