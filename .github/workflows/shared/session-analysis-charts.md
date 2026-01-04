---
# Session Analysis Charts
# Reusable chart generation for Copilot session analysis workflows
#
# Usage:
#   imports:
#     - shared/session-analysis-charts.md
#
# This import provides:
# - Python environment for chart generation
# - Instructions for creating session analysis charts
# - Best practices for session trend visualization

imports:
  - shared/python-dataviz.md
---

# Session Analysis Chart Generation

You are an expert at creating session analysis trend charts that reveal insights about Copilot agent session patterns over time.

## 📊 Chart Generation Requirements

**IMPORTANT**: Generate exactly 2 trend charts that showcase Copilot agent session patterns over time.

### Chart Generation Process

**Phase 1: Data Collection**

Collect data for the past 30 days (or available data) from cache memory and session logs:

1. **Session Completion Data**:
   - Count of sessions completed successfully per day
   - Count of sessions failed/abandoned per day
   - Completion rate percentage per day

2. **Session Duration Data**:
   - Average session duration per day (in minutes)
   - Median session duration per day
   - Number of sessions with loops/retries

**Phase 2: Data Preparation**

1. Create CSV files in `/tmp/gh-aw/python/data/` with the collected data:
   - `session_completion.csv` - Daily completion counts and rates
   - `session_duration.csv` - Daily duration statistics

2. Each CSV should have a date column and metric columns with appropriate headers

**Phase 3: Chart Generation**

Generate exactly **2 high-quality trend charts**:

**Chart 1: Session Completion Trends**
- Multi-line chart showing:
  - Successful completions (line, green)
  - Failed/abandoned sessions (line, red)
  - Completion rate percentage (line with secondary y-axis)
- X-axis: Date (last 30 days)
- Y-axis: Count (left), Percentage (right)
- Save as: `/tmp/gh-aw/python/charts/session_completion_trends.png`

**Chart 2: Session Duration & Efficiency**
- Dual visualization showing:
  - Average session duration (line)
  - Median session duration (line)
  - Sessions with loops (bar chart overlay)
- X-axis: Date (last 30 days)
- Y-axis: Duration in minutes
- Save as: `/tmp/gh-aw/python/charts/session_duration_trends.png`

**Chart Quality Requirements**:
- DPI: 300 minimum
- Figure size: 12x7 inches for better readability
- Use seaborn styling with a professional color palette
- Include grid lines for easier reading
- Clear, large labels and legend
- Title with context (e.g., "Session Completion Rates - Last 30 Days")
- Annotations for significant changes or anomalies

**Phase 4: Upload Charts**

1. Upload both charts using the `upload asset` tool
2. Collect the returned URLs for embedding in the discussion

**Phase 5: Embed Charts in Discussion**

Include the charts in your analysis report with this structure:

```markdown
## 📈 Session Trends Analysis

### Completion Patterns
![Session Completion Trends](URL_FROM_UPLOAD_ASSET_CHART_1)

[Brief 2-3 sentence analysis of completion trends, highlighting improvements in success rates or concerning patterns]

### Duration & Efficiency
![Session Duration Trends](URL_FROM_UPLOAD_ASSET_CHART_2)

[Brief 2-3 sentence analysis of session duration patterns, noting efficiency improvements or areas needing attention]
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
