---
# No frontmatter configuration needed - this is a pure instructions file
---

## Report Formatting Guidelines

When generating reports for GitHub discussions, follow these formatting guidelines to improve readability and reduce scrolling.

### Use HTML Details/Summary Tags

**Wrap your reports in HTML `<details>` and `<summary>` tags** to make them collapsible. This is especially important for:
- Scheduled reports (daily, weekly, monthly)
- Audit results with multiple sections
- Analysis reports with detailed findings
- Any report longer than 2-3 screens

**Basic Structure:**

```markdown
<details>
<summary>📊 Report Title - [Date]</summary>

## Executive Summary

Brief overview at the top...

## Detailed Sections

<details>
<summary>Section 1: Analysis</summary>

Detailed content...

</details>

<details>
<summary>Section 2: Findings</summary>

More detailed content...

</details>

</details>
```

### Summary Best Practices

- **Be Descriptive**: Use clear summary text that describes what's inside
- **Include Metrics**: Add key numbers in the summary (e.g., "5 issues found")
- **Use Emojis**: Make summaries scannable (📊 📈 ⚠️ ✅ 🔍 📝)
- **Include Date**: Always include the date or time period covered

**Examples:**

```markdown
<summary>📊 Daily Report - 2024-10-22 (5 findings)</summary>
<summary>⚠️ Error Analysis - Last 24 Hours (12 errors detected)</summary>
<summary>✅ Weekly Summary - Oct 15-22 (All systems healthy)</summary>
```

### Report Structure

1. **Executive Summary**: Start with the most important information visible at the top
2. **Nested Sections**: Use nested `<details>` tags for subsections
3. **Keep Tables Readable**: Place large tables inside collapsible sections
4. **Highlight Critical Items**: Keep urgent issues visible without expansion

### Complete Example

```markdown
<details>
<summary>📊 Weekly Analysis - Oct 15-22, 2024 (45 runs, 3 issues)</summary>

## Executive Summary

Analyzed 45 workflow runs. Found 3 issues requiring attention.

**Key Metrics:**
- Total runs: 45
- Success rate: 93.3%
- Failed runs: 3

## Issues Found

<details>
<summary>⚠️ Issue 1: Authentication Failures (2 occurrences)</summary>

### Description
Details about the issue...

### Recommendation
How to fix it...

</details>

<details>
<summary>⚠️ Issue 2: Timeout in Data Collection (1 occurrence)</summary>

### Description
Details...

### Recommendation
Solution...

</details>

## Detailed Analysis

<details>
<summary>📈 Performance Metrics</summary>

| Workflow | Runs | Avg Duration | Success Rate |
|----------|------|--------------|--------------|
| daily-report | 7 | 5m 23s | 100% |

</details>

---

*Report generated automatically*

</details>
```

### When Details Tags Are Optional

- Brief status updates (single paragraph)
- Critical alerts that should be immediately visible
- Short summaries without detailed analysis
