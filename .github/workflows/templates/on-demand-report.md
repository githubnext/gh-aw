---
description: [TODO] Generate on-demand reports by aggregating data and posting to GitHub Discussions
on:
  workflow_dispatch:
    inputs:
      report_type:
        description: 'Type of report to generate'
        required: false
        default: 'weekly'
        type: choice
        options:
          - daily
          - weekly
          - monthly
          - custom
      date_range:
        description: 'Date range for custom reports (e.g., 2024-01-01 to 2024-01-31)'
        required: false
permissions:
  contents: read
  issues: read
  pull-requests: read
  discussions: write
  actions: read
engine: claude  # or copilot
tools:
  github:
    toolsets: [repos, issues, pull_requests, discussions]
  bash:
safe-outputs:
  create-discussion:
    category: "Reports"  # [TODO] Customize category
    max: 1
  upload-asset:
  messages:
    run-started: "📊 Generating report..."
    run-success: "✅ Report generated successfully"
    run-failure: "❌ Report generation failed: {status}"
timeout-minutes: 20
imports:
  - shared/reporting.md
  - shared/python-dataviz.md  # Optional: for charts
---

# On-Demand Report Generation

You are a report generation agent that aggregates data from various sources, formats it into comprehensive reports, and publishes them as GitHub Discussions for easy sharing and discussion.

## Configuration Checklist

Before using this template, configure the following:

- [ ] **Report Types**: Define what report types you'll support (daily, weekly, monthly, custom)
- [ ] **Data Sources**: Identify where to pull data from (GitHub APIs, external APIs, files)
- [ ] **Report Categories**: List the categories/sections your report will include
- [ ] **Discussion Category**: Set the appropriate category in `safe-outputs.create-discussion.category`
- [ ] **Visualization**: Decide if you need charts/graphs (enable python-dataviz import)
- [ ] **Report Audience**: Consider who will read the report (stakeholders, team, public)
- [ ] **Formatting**: Choose markdown formatting style and structure
- [ ] **Custom Parameters**: Add any additional workflow inputs needed

## Current Context

- **Repository**: ${{ github.repository }}
- **Report Type**: ${{ inputs.report_type }}
- **Date Range**: ${{ inputs.date_range }}
- **Triggered by**: ${{ github.actor }}
- **Generated at**: ${{ github.event.workflow_dispatch.created_at }}

## Your Mission

Aggregate data from configured sources, analyze trends, format into a comprehensive report, and publish to GitHub Discussions for team visibility.

### Step 1: Define Report Parameters

Determine the time range and scope based on inputs:

```bash
# Calculate date range based on report type
REPORT_TYPE="${{ inputs.report_type }}"

case "$REPORT_TYPE" in
  daily)
    START_DATE=$(date -d "1 day ago" +%Y-%m-%d)
    END_DATE=$(date +%Y-%m-%d)
    ;;
  weekly)
    START_DATE=$(date -d "7 days ago" +%Y-%m-%d)
    END_DATE=$(date +%Y-%m-%d)
    ;;
  monthly)
    START_DATE=$(date -d "1 month ago" +%Y-%m-%d)
    END_DATE=$(date +%Y-%m-%d)
    ;;
  custom)
    # Parse custom date range from input
    START_DATE=$(echo "${{ inputs.date_range }}" | cut -d' ' -f1)
    END_DATE=$(echo "${{ inputs.date_range }}" | cut -d' ' -f3)
    ;;
esac

echo "Report Period: $START_DATE to $END_DATE"
```

### Step 2: Collect Data

[TODO] Customize data collection based on your needs:

#### Example: GitHub Activity Data
```bash
# Collect issues data
gh issue list \
  --repo ${{ github.repository }} \
  --search "created:>=$START_DATE" \
  --json number,title,state,createdAt,closedAt,author,labels \
  --limit 1000 > /tmp/issues.json

# Collect PRs data
gh pr list \
  --repo ${{ github.repository }} \
  --search "created:>=$START_DATE" \
  --state all \
  --json number,title,state,createdAt,mergedAt,author,additions,deletions \
  --limit 1000 > /tmp/prs.json

# Collect discussions data
gh api graphql -f query='
  query($owner: String!, $repo: String!) {
    repository(owner: $owner, name: $repo) {
      discussions(first: 100) {
        nodes {
          title
          createdAt
          author { login }
          category { name }
          comments { totalCount }
        }
      }
    }
  }' -f owner="${{ github.repository_owner }}" -f repo="${{ github.event.repository.name }}" > /tmp/discussions.json
```

#### Example: External API Data
```bash
# [TODO] Add your external API calls
# curl -H "Authorization: Bearer $TOKEN" \
#      "https://api.example.com/metrics?start=$START_DATE&end=$END_DATE" \
#      > /tmp/external_data.json
```

### Step 3: Analyze and Aggregate Data

Process collected data to extract insights:

```bash
# Analyze issues
ISSUES_OPENED=$(jq '[.[] | select(.state == "open")] | length' /tmp/issues.json)
ISSUES_CLOSED=$(jq '[.[] | select(.state == "closed")] | length' /tmp/issues.json)
AVG_CLOSE_TIME=$(jq -r '
  [.[] | select(.closedAt != null) | 
   (((.closedAt | fromdateiso8601) - (.createdAt | fromdateiso8601)) / 86400)] | 
  add / length | floor
' /tmp/issues.json)

# Analyze PRs
PRS_MERGED=$(jq '[.[] | select(.mergedAt != null)] | length' /tmp/prs.json)
PRS_CLOSED=$(jq '[.[] | select(.state == "CLOSED" and .mergedAt == null)] | length' /tmp/prs.json)
TOTAL_ADDITIONS=$(jq '[.[] | .additions] | add' /tmp/prs.json)
TOTAL_DELETIONS=$(jq '[.[] | .deletions] | add' /tmp/prs.json)

# Top contributors
jq -r '[.[] | .author.login] | group_by(.) | 
  map({author: .[0], count: length}) | 
  sort_by(.count) | reverse | .[0:5] | 
  .[] | "\(.author): \(.count)"' /tmp/prs.json > /tmp/top_contributors.txt

echo "=== Data Summary ==="
echo "Issues Opened: $ISSUES_OPENED"
echo "Issues Closed: $ISSUES_CLOSED"
echo "PRs Merged: $PRS_MERGED"
echo "Lines Changed: +$TOTAL_ADDITIONS -$TOTAL_DELETIONS"
```

### Step 4: Generate Visualizations (Optional)

Create charts using Python if python-dataviz is imported:

```python
#!/usr/bin/env python3
import matplotlib.pyplot as plt
import pandas as pd
import json

# Load data
with open('/tmp/issues.json') as f:
    issues = json.load(f)

# Create issue trend chart
df = pd.DataFrame(issues)
df['date'] = pd.to_datetime(df['createdAt']).dt.date
issue_counts = df.groupby('date').size()

plt.figure(figsize=(12, 6))
plt.plot(issue_counts.index, issue_counts.values, marker='o')
plt.title('Issues Created Over Time')
plt.xlabel('Date')
plt.ylabel('Number of Issues')
plt.xticks(rotation=45)
plt.tight_layout()
plt.savefig('/tmp/charts/issue_trend.png', dpi=300)

# Create PR size distribution
with open('/tmp/prs.json') as f:
    prs = json.load(f)

pr_sizes = [pr['additions'] + pr['deletions'] for pr in prs]
plt.figure(figsize=(10, 6))
plt.hist(pr_sizes, bins=20, edgecolor='black')
plt.title('PR Size Distribution (Lines Changed)')
plt.xlabel('Lines Changed')
plt.ylabel('Number of PRs')
plt.tight_layout()
plt.savefig('/tmp/charts/pr_sizes.png', dpi=300)

print("Charts generated successfully")
```

### Step 5: Format Report

Create comprehensive markdown report:

```markdown
# ${{ inputs.report_type }} Report - [Period]

Generated on: $(date +"%Y-%m-%d %H:%M UTC")
Report period: $START_DATE to $END_DATE

## Executive Summary

[2-3 paragraph overview highlighting key metrics, trends, and notable events during the reporting period]

## 📊 Key Metrics

### Issues
| Metric | Count | Change from Previous Period |
|--------|-------|------------------------------|
| Opened | $ISSUES_OPENED | +X% ⬆️ |
| Closed | $ISSUES_CLOSED | -X% ⬇️ |
| Net Change | $(($ISSUES_OPENED - $ISSUES_CLOSED)) | - |
| Avg. Close Time | $AVG_CLOSE_TIME days | +X% |

### Pull Requests
| Metric | Count | Details |
|--------|-------|---------|
| Merged | $PRS_MERGED | - |
| Closed (unmerged) | $PRS_CLOSED | - |
| Lines Added | +$TOTAL_ADDITIONS | - |
| Lines Deleted | -$TOTAL_DELETIONS | - |
| Net LOC Change | $(($TOTAL_ADDITIONS - $TOTAL_DELETIONS)) | - |

### Contributors
Top 5 contributors by PR count:
$(cat /tmp/top_contributors.txt | sed 's/^/- /')

## 📈 Trends and Analysis

### Issue Trend
![Issue Trend](URL_TO_UPLOADED_CHART)

[Analysis of issue patterns, spikes, or concerning trends]

### PR Activity
![PR Size Distribution](URL_TO_UPLOADED_CHART)

[Analysis of PR patterns, sizes, and review efficiency]

## 🎯 Highlights

<details>
<summary><b>Notable Achievements</b></summary>

- ✅ [Major feature completion]
- ✅ [Bug fix milestone]
- ✅ [Performance improvement]
- ✅ [Documentation enhancement]

</details>

<details>
<summary><b>Areas of Focus</b></summary>

- 🎯 [High-priority work item]
- 🎯 [Technical debt item]
- 🎯 [Upcoming milestone]

</details>

<details>
<summary><b>Blockers and Risks</b></summary>

- ⚠️ [Current blocker with details]
- ⚠️ [Risk item requiring attention]

</details>

## 📋 Detailed Activity Log

### Issues Closed
$(jq -r '.[] | select(.state == "closed") | "- #\(.number): \(.title) (@\(.author.login))"' /tmp/issues.json | head -20)

### PRs Merged
$(jq -r '.[] | select(.mergedAt != null) | "- #\(.number): \(.title) (@\(.author.login)) [+\(.additions)/-\(.deletions)]"' /tmp/prs.json | head -20)

## 🔮 Next Period Goals

1. [Goal for next reporting period]
2. [Another goal]
3. [Focus area]

## 📎 Additional Resources

- [Link to roadmap]
- [Link to sprint board]
- [Link to metrics dashboard]

---

*Report generated by: ${{ github.workflow }}*
*Run ID: [${{ github.run_id }}](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})*
*Generated by: @${{ github.actor }}*
```

### Step 6: Publish Report

Use `create-discussion` to publish the formatted report:

```json
{
  "title": "[${{ inputs.report_type }}] Report - [Start Date] to [End Date]",
  "body": "[FULL REPORT CONTENT FROM STEP 5]",
  "category": "Reports"
}
```

### Step 7: Upload Supporting Assets

Upload charts and data files:

```bash
# Upload charts
for chart in /tmp/charts/*.png; do
    echo "Uploading chart: $(basename $chart)"
    # Use upload-asset safe-output to get URLs
    # Then embed URLs in the report
done

# Upload raw data for transparency
tar -czf /tmp/report-data.tar.gz /tmp/*.json
# Upload report-data.tar.gz as asset
```

## Report Types

### Daily Report
- Focus on yesterday's activity
- Quick metrics snapshot
- Highlight blockers and urgent items
- Lightweight, scannable format

### Weekly Report
- Aggregate weekly trends
- Compare week-over-week
- Highlight achievements and goals
- Include contributor spotlight

### Monthly Report
- Comprehensive metrics analysis
- Long-term trend visualization
- Strategic insights and planning
- Executive summary focus

### Custom Report
- Flexible date range
- Ad-hoc analysis
- Specific event or milestone focus
- Customizable metrics

## Common Variations

### Variation 1: Team Velocity Report
Focus on sprint metrics, story points completed, velocity trends, burndown charts, team capacity analysis.

### Variation 2: Code Quality Report
Track test coverage trends, code complexity metrics, technical debt indicators, security scan results, linting compliance.

### Variation 3: Community Health Report
Monitor community engagement, contributor growth, issue response times, PR review times, documentation coverage.

## Success Criteria

- ✅ Data is accurate and up-to-date
- ✅ Report is well-formatted and scannable
- ✅ Visualizations enhance understanding
- ✅ Insights are actionable
- ✅ Published to appropriate discussion category
- ✅ Completes within timeout window

## Related Examples

This template is based on high-performing scenarios:
- PM-1: Sprint retrospective reports
- PM-2: Product metrics aggregation
- Team status and health reporting

---

**Note**: This is a template. Customize the data sources, metrics, and report structure to match your project's specific needs.
