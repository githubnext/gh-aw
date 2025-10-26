---
# No frontmatter configuration needed - this is a pure instructions file
---

## Report Formatting

Structure your report with an overview followed by detailed content:

1. **Content Overview**: Start with 1-2 paragraphs that summarize the key findings, highlights, or main points of your report. This should give readers a quick understanding of what the report contains without needing to expand the details.

2. **Detailed Content**: Place the rest of your report inside HTML `<details>` and `<summary>` tags to allow readers to expand and view the full information.

**Example format:**

```markdown
Brief overview paragraph 1 introducing the report and its main findings.

Optional overview paragraph 2 with additional context or highlights.

<details>
<summary>Full Report Details</summary>

## Detailed Analysis

Full report content with all sections, tables, and detailed information goes here.

### Section 1
[Content]

### Section 2
[Content]

</details>
```

## Reporting Workflow Run Information

When analyzing workflow run logs or reporting information from GitHub Actions runs:

### 1. Workflow Run ID Formatting

**Always render workflow run IDs as clickable URLs** when mentioning them in your report. The workflow run data includes a `url` field that provides the full GitHub Actions run page URL.

**Format:**
```markdown
[Run §12345](https://github.com/owner/repo/actions/runs/12345)
```

**Example:**
```markdown
Analysis based on [Run §456789](https://github.com/githubnext/gh-aw/actions/runs/456789)
```

### 2. Document References for Workflow Runs

When your analysis is based on information mined from one or more workflow runs, **include up to 3 workflow run URLs as document references** at the end of your report.

**Format:**
```markdown
---

**References:**
- [Run §12345](https://github.com/owner/repo/actions/runs/12345)
- [Run §12346](https://github.com/owner/repo/actions/runs/12346)
- [Run §12347](https://github.com/owner/repo/actions/runs/12347)
```

**Guidelines:**
- Include **maximum 3 references** to keep reports concise
- Choose the most relevant or representative runs (e.g., failed runs, high-cost runs, or runs with significant findings)
- Always use the actual URL from the workflow run data (available in `url`, `RunURL`, or similar fields)
- If analyzing more than 3 runs, select the most important ones for references
