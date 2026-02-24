---
description: Guidelines for creating agentic workflows that generate reports — output type selection, formatting style, and automatic cleanup.
---

# Report Generation

Consult this file when creating an agentic workflow that generates reports — recurring status updates, audits, analysis summaries, or any structured output posted as a GitHub issue, discussion, or comment.

## Choosing the Right Output Type

| Use case | Recommended output |
|---|---|
| Recurring status / audit report | `create-discussion` with `close-older-discussions` |
| One-off finding or analysis | `create-issue` with `close-older-issues` |
| Inline update on an existing issue or PR | `add-comment` with `hide-older-comments` |

Use `create-discussion` (not `create-issue`) for reports — discussions signal "status / informational" intent, keep the issue tracker clean, and support the `close-older-discussions` cleanup mechanism out of the box.

## Automatic Cleanup

Reports accumulate over time. Always configure automatic cleanup when the workflow runs on a schedule or recurs.

- **`expires`**: Auto-closes the issue or discussion after a time period (e.g. `7` days, `2w`, `1m`). Use when reports become stale after a fixed window.
- **`close-older-issues: true`**: Closes previous issues from the same workflow before creating a new one. Requires `title-prefix` or `labels` to identify matching issues.
- **`close-older-discussions: true`**: Closes older discussions with the same title prefix or labels as "OUTDATED". Requires `title-prefix` or `labels`.
- **`hide-older-comments: true`**: Minimizes previous comments from the same workflow before posting a new one. Useful for rolling status updates on the same issue or PR.

**Default recommendation for recurring reports:** use `create-discussion` with `close-older-discussions: true` and a stable `title-prefix` so only the latest report is visible.

```yaml
safe-outputs:
  create-discussion:
    title-prefix: "Weekly Status:"
    category: "Reports"
    close-older-discussions: true
    expires: 30
```

## Report Style and Structure

### Header Levels

**Use `###` or lower for all headers in your report to maintain proper document hierarchy.**

When creating GitHub issues or discussions:
- Use `###` (h3) for main sections (e.g., `### Test Summary`)
- Use `####` (h4) for subsections (e.g., `#### Device-Specific Results`)
- Never use `##` (h2) or `#` (h1) in reports — these are reserved for titles

### Progressive Disclosure

**Wrap detailed content in `<details><summary><b>Section Name</b></summary>` tags to improve readability and reduce scrolling.**

Use collapsible sections for:
- Verbose details (full logs, raw data)
- Secondary information (minor warnings, extra context)
- Per-item breakdowns when there are many items

Always keep critical information visible (summary, critical issues, key metrics).

### Report Structure Pattern

1. **Overview**: 1–2 paragraphs summarizing key findings
2. **Critical Information**: Show immediately (summary stats, critical issues)
3. **Details**: Use `<details><summary><b>Section Name</b></summary>` for expanded content
4. **Context**: Add helpful metadata (workflow run, date, trigger)

### Design Principles

Reports should:
- **Build trust through clarity**: Most important info immediately visible
- **Exceed expectations**: Add helpful context like trends, comparisons
- **Create delight**: Use progressive disclosure to reduce overwhelm
- **Maintain consistency**: Follow patterns across all reports

### Example Report Structure

```markdown
### Summary
- Key metric 1: value
- Key metric 2: value
- Status: ✅/⚠️/❌

### Critical Issues
[Always visible - these are important]

<details>
<summary><b>View Detailed Results</b></summary>

[Comprehensive details, logs, traces]

</details>

<details>
<summary><b>View All Warnings</b></summary>

[Minor issues and potential problems]

</details>

### Recommendations
[Actionable next steps - keep visible]
```

## Workflow Run References

- Format run IDs as links: `[§12345](https://github.com/owner/repo/actions/runs/12345)`
- Include up to 3 most relevant run URLs at the end under `**References:**`
- Do NOT add footer attribution — the system appends it automatically
