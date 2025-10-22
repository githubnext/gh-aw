# Reporting Instructions

This file contains instructions for generating reports in agentic workflows. Follow these guidelines when creating reports that will be posted as GitHub discussions or issues.

## Report Format

### Use HTML Details/Summary Tags

To prevent excessive scrolling and improve readability, **wrap your reports in HTML `<details>` and `<summary>` tags**. This allows users to expand and collapse sections as needed.

**Basic Structure:**

```markdown
<details>
<summary>📊 Report Title - [Date]</summary>

## Report Content

Your detailed report content goes here...

### Section 1

Content for section 1...

### Section 2

Content for section 2...

</details>
```

**Multiple Sections:**

For longer reports with multiple major sections, use nested details tags:

```markdown
<details>
<summary>📊 Overall Report Title - [Date]</summary>

## Executive Summary

Brief overview of the report...

<details>
<summary>Section 1: Detailed Analysis</summary>

### Subsection 1.1

Detailed content...

### Subsection 1.2

More content...

</details>

<details>
<summary>Section 2: Findings</summary>

### Finding 1

Details...

### Finding 2

More details...

</details>

<details>
<summary>Section 3: Recommendations</summary>

List of recommendations...

</details>

</details>
```

## Best Practices

### Summary Text

- **Be Descriptive**: Use clear, informative summary text that describes what's inside
- **Include Key Metrics**: Add important numbers or status in the summary (e.g., "Found 5 issues, 3 improvements")
- **Use Emojis**: Add relevant emojis to make summaries more scannable (📊 📈 ⚠️ ✅ 🔍 📝)
- **Include Date**: Always include the date or time period covered

**Examples:**

```markdown
<summary>📊 Daily Report - 2024-10-22 (5 findings)</summary>
<summary>⚠️ Error Analysis - Last 24 Hours (12 errors detected)</summary>
<summary>✅ Weekly Summary - Oct 15-22 (All systems healthy)</summary>
<summary>🔍 Audit Results - 15 workflows analyzed</summary>
```

### Report Structure

1. **Start with Executive Summary**: Place the most important information at the top, outside nested details tags
2. **Organize by Topic**: Group related information together
3. **Use Collapsible Sections**: Wrap detailed analysis, logs, or supplementary information in collapsible sections
4. **Keep Tables Readable**: Place large tables inside details tags
5. **Highlight Key Findings**: Keep critical issues visible in the main summary

### Markdown Formatting

- Use proper markdown headers (`##`, `###`, `####`)
- Use tables for structured data
- Use code blocks with syntax highlighting for code snippets
- Use bullet points and numbered lists for clarity
- Add horizontal rules (`---`) to separate major sections

### Example Complete Report

```markdown
<details>
<summary>📊 Weekly Workflow Analysis - Oct 15-22, 2024 (45 runs, 3 issues)</summary>

## Executive Summary

Analyzed 45 workflow runs from the past week. Found 3 issues requiring attention and identified 2 optimization opportunities.

**Key Metrics:**
- Total runs: 45
- Success rate: 93.3%
- Failed runs: 3
- Average duration: 8m 45s

## Issues Found

<details>
<summary>⚠️ Issue 1: Authentication Failures (2 occurrences)</summary>

### Description
Two workflows failed due to GitHub token expiration.

### Affected Workflows
- `daily-report.md` - Run #12345
- `weekly-analysis.md` - Run #12367

### Recommendation
Update token refresh logic in authentication step.

</details>

<details>
<summary>⚠️ Issue 2: Timeout in Data Collection (1 occurrence)</summary>

### Description
Workflow exceeded 15-minute timeout during data collection phase.

### Affected Workflow
- `data-collector.md` - Run #12389

### Recommendation
Increase timeout to 20 minutes or optimize data collection queries.

</details>

## Performance Analysis

<details>
<summary>📈 Detailed Performance Metrics</summary>

| Workflow | Runs | Avg Duration | Success Rate |
|----------|------|--------------|--------------|
| daily-report | 7 | 5m 23s | 100% |
| weekly-analysis | 4 | 12m 45s | 75% |
| data-collector | 5 | 8m 12s | 80% |

</details>

## Recommendations

1. ✅ Update authentication token management
2. ✅ Increase timeout for data-collector workflow
3. 💡 Consider caching frequently accessed data to improve performance

---

*Report generated automatically by Audit Agent*

</details>
```

## When to Use Details Tags

### Always Use

- For scheduled reports (daily, weekly, monthly)
- For audit results with multiple sections
- For analysis reports with detailed findings
- For any report longer than 2-3 screens

### Optional

- For brief status updates (single paragraph)
- For critical alerts that should be immediately visible
- For short summaries without detailed analysis

## Accessibility

- Ensure summary text is descriptive enough to understand content without expanding
- Don't nest details tags more than 2-3 levels deep
- Keep critical information accessible without requiring multiple expansions
- Test that details/summary tags work correctly in GitHub's markdown renderer
