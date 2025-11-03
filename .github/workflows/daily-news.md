---
on:
  schedule:
    # Every day at 9am UTC, all days except Saturday and Sunday
    - cron: "0 9 * * 1-5"
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read
  discussions: read
  actions: read

engine: copilot

network:
  firewall: true

safe-outputs:
  upload-assets:
  create-discussion:
    category: "daily-news"
    max: 1

tools:
  cache-memory:
  edit:
  bash:
    - "*"
  web-fetch:
  github:
    toolsets:
      - default
      - discussions

imports:
  - shared/mcp/tavily.md
  - shared/jqschema.md
  - shared/reporting.md
  - shared/trends.md
---

# Daily News

Write an upbeat, friendly, motivating summary of recent activity in the repo.

## 📊 Trend Charts Requirement

**IMPORTANT**: Generate exactly 2 trend charts that showcase key metrics of the project. These charts should visualize trends over time to give the team insights into project health and activity patterns.

### Chart Generation Process

**Phase 1: Data Collection**

Collect data for the past 30 days (or available data) using GitHub API:

1. **Issues Activity Data**: 
   - Count of issues opened per day
   - Count of issues closed per day
   - Running count of open issues

2. **Pull Requests Activity Data**:
   - Count of PRs opened per day
   - Count of PRs merged per day
   - Count of PRs closed per day

3. **Commit Activity Data**:
   - Count of commits per day on main branches
   - Number of contributors per day

4. **Additional Metrics** (optional for enrichment):
   - Discussion activity
   - Code review comments
   - CI/CD run statistics

**Phase 2: Data Preparation**

1. Create CSV files in `/tmp/gh-aw/python/data/` with the collected data:
   - `issues_prs_activity.csv` - Daily counts of issues and PRs
   - `commit_activity.csv` - Daily commit counts and contributors

2. Each CSV should have a date column and metric columns with appropriate headers

**Phase 3: Chart Generation**

Generate exactly **2 high-quality trend charts**:

**Chart 1: Issues & Pull Requests Activity**
- Multi-line chart showing:
  - Issues opened (line)
  - Issues closed (line)
  - PRs opened (line)
  - PRs merged (line)
- X-axis: Date (last 30 days)
- Y-axis: Count
- Include a 7-day moving average overlay if data is noisy
- Save as: `/tmp/gh-aw/python/charts/issues_prs_trends.png`

**Chart 2: Commit Activity & Contributors**
- Dual-axis chart or stacked visualization showing:
  - Daily commit count (bar chart or line)
  - Number of unique contributors (line with markers)
- X-axis: Date (last 30 days)
- Y-axis: Count
- Save as: `/tmp/gh-aw/python/charts/commit_trends.png`

**Chart Quality Requirements**:
- DPI: 300 minimum
- Figure size: 12x7 inches for better readability
- Use seaborn styling with a professional color palette
- Include grid lines for easier reading
- Clear, large labels and legend
- Title with context (e.g., "Issues & PR Activity - Last 30 Days")
- Annotations for significant peaks or patterns

**Phase 4: Upload Charts**

1. Upload both charts using the `upload asset` tool
2. Collect the returned URLs for embedding in the discussion

**Phase 5: Embed Charts in Discussion**

Include the charts in your daily news discussion report with this structure:

```markdown
## 📈 Trend Analysis

### Issues & Pull Requests Activity
![Issues and PR Trends](URL_FROM_UPLOAD_ASSET_CHART_1)

[Brief 2-3 sentence analysis of the trends shown in this chart, highlighting notable patterns, increases, decreases, or insights]

### Commit Activity & Contributors
![Commit Activity Trends](URL_FROM_UPLOAD_ASSET_CHART_2)

[Brief 2-3 sentence analysis of the trends shown in this chart, noting developer engagement, busy periods, or collaboration patterns]
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

- Include some or all of the following:
  * Recent issues activity
  * Recent pull requests
  * Recent discussions
  * Recent releases
  * Recent comments
  * Recent code reviews
  * Recent code changes
  * Recent failed CI runs
  * Look at the changesets in ./changeset folder

- If little has happened, don't write too much.

- Give some deep thought to ways the team can improve their productivity, and suggest some ways to do that.

- Include a description of open source community engagement, if any.

- Highlight suggestions for possible investment, ideas for features and project plan, ways to improve community engagement, and so on.

- Be helpful, thoughtful, respectful, positive, kind, and encouraging.

- Use emojis to make the report more engaging and fun, but don't overdo it.

- Include a short haiku at the end of the report to help orient the team to the season of their work.

- In a note at the end of the report, include a log of
  * all search queries (web, issues, pulls, content) you used to generate the data for the report
  * all commands you used to generate the data for the report
  * all files you read to generate the data for the report
  * places you didn't have time to read or search, but would have liked to

Create a new GitHub discussion with a title containing today's date (e.g., "Daily Status - 2024-10-10") containing a markdown report with your findings. Use links where appropriate.

Only a new discussion should be created, do not close or update any existing discussions.
