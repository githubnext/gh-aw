---
description: Creates weekly summary of issue activity including trends, charts, and insights every Monday
timeout-minutes: 20
strict: true
on:
  schedule:
    - cron: "0 15 * * 1"  # Weekly on Mondays at 3 PM UTC
  workflow_dispatch:
permissions:
  issues: read
tracker-id: weekly-issue-summary
engine: copilot
network:
  allowed:
    - defaults
    - python
    - node
sandbox:
  agent: awf  # Firewall enabled (migrated from network.firewall)
tools:
  edit:
  bash:
    - "*"
  github:
    toolsets: 
      - issues
safe-outputs:
  upload-asset:
  create-discussion:
    title-prefix: "[Weekly Summary] "
    category: "Audits"
    close-older-discussions: true
imports:
  - shared/reporting.md
  - shared/trends.md
---

# Weekly Issue Summary

## 📊 Trend Charts Requirement

**IMPORTANT**: Generate exactly 2 trend charts that showcase issue activity patterns over time.

### Chart Generation Process

**Phase 1: Data Collection**

Collect data for the past 30 days (or available data) using GitHub API:

1. **Issue Activity Data**:
   - Count of issues opened per day
   - Count of issues closed per day
   - Running count of open issues

2. **Issue Resolution Data**:
   - Average time to close issues (in days)
   - Distribution of issue lifespans
   - Issues by label category over time

**Phase 2: Data Preparation**

1. Create CSV files in `/tmp/gh-aw/python/data/` with the collected data:
   - `issue_activity.csv` - Daily opened/closed counts and open count
   - `issue_resolution.csv` - Resolution time statistics

2. Each CSV should have a date column and metric columns with appropriate headers

**Phase 3: Chart Generation**

Generate exactly **2 high-quality trend charts**:

**Chart 1: Issue Activity Trends**
- Multi-line chart showing:
  - Issues opened per week (line or bar)
  - Issues closed per week (line or bar)
  - Net change (opened - closed) per week
  - Running total of open issues (line)
- X-axis: Week (last 12 weeks or 30 days)
- Y-axis: Count
- Save as: `/tmp/gh-aw/python/charts/issue_activity_trends.png`

**Chart 2: Issue Resolution Time Trends**
- Line chart with statistics showing:
  - Average time to close (in days, 7-day moving average)
  - Median time to close
  - Shaded area showing resolution time variance
- X-axis: Date (last 30 days)
- Y-axis: Days to resolution
- Save as: `/tmp/gh-aw/python/charts/issue_resolution_trends.png`

**Chart Quality Requirements**:
- DPI: 300 minimum
- Figure size: 12x7 inches for better readability
- Use seaborn styling with a professional color palette
- Include grid lines for easier reading
- Clear, large labels and legend
- Title with context (e.g., "Issue Activity - Last 12 Weeks")
- Annotations for notable patterns or changes

**Phase 4: Upload Charts**

1. Upload both charts using the `upload asset` tool
2. Collect the returned URLs for embedding in the discussion

**Phase 5: Embed Charts in Discussion**

Include the charts in your weekly summary with this structure:

```markdown
## 📈 Issue Activity Trends

### Weekly Activity Patterns
![Issue Activity Trends](URL_FROM_UPLOAD_ASSET_CHART_1)

[Brief 2-3 sentence analysis of issue activity trends, highlighting increases/decreases in activity or backlog growth]

### Resolution Time Analysis
![Issue Resolution Trends](URL_FROM_UPLOAD_ASSET_CHART_2)

[Brief 2-3 sentence analysis of how quickly issues are being resolved, noting improvements or slowdowns]
```

### Python Implementation Notes

- Use pandas for data manipulation and date handling
- Use matplotlib.pyplot and seaborn for visualization
- Set appropriate date formatters for x-axis labels
- Use `plt.xticks(rotation=45)` for readable date labels
- Apply `plt.tight_layout()` before saving
- Handle cases where data might be sparse or missing

### Error Handling

If insufficient data is available (less than 7 days):
- Generate the charts with available data
- Add a note in the analysis mentioning the limited data range
- Consider using a bar chart instead of line chart for very sparse data

---

## Weekly Analysis

Analyze all issues opened in the repository ${{ github.repository }} over the last 7 days.

Create a comprehensive summary that includes:
- Total number of issues opened
- List of issue titles with their numbers and authors
- Any notable patterns or trends (common labels, types of issues, etc.)

Format the summary clearly with proper markdown formatting for easy reading.