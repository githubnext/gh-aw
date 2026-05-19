---
description: Guidelines for creating agentic workflows that generate reports — output type selection, formatting style, and automatic cleanup.
---

# Report Generation

For agentic workflows that generate reports — status updates, audits, summaries, or any structured output posted as a GitHub issue, discussion, or comment.

## Choosing the Right Output Type

| Use case | Recommended output |
|---|---|
| Report (default) | `create-issue` with `close-older-issues` |
| Inline update on an existing issue or PR | `add-comment` with `hide-older-comments` |
| Discussion-based report (only when explicitly requested) | `create-discussion` with `close-older-discussions` |

Default to `create-issue` — issues are searchable and support full close/expire cleanup. Only use `create-discussion` when explicitly requested.

## Automatic Cleanup

Always configure cleanup for scheduled or recurring reports.

- **`expires`**: Auto-closes after a time period (e.g. `7`, `2w`, `1m`). Use when reports go stale after a fixed window.
- **`close-older-issues: true`**: Closes previous issues from the same workflow. Requires `title-prefix` or `labels`.
- **`close-older-discussions: true`**: Closes older matching discussions as "OUTDATED". Requires `title-prefix` or `labels`.
- **`hide-older-comments: true`**: Minimizes previous comments from the same workflow. Useful for rolling status updates.

**Recommended for recurring reports**: `create-issue` with `close-older-issues: true` and a stable `title-prefix`.

```yaml
safe-outputs:
  create-issue:
    title-prefix: "Weekly Status:"
    labels: [report]
    close-older-issues: true
    expires: 30
```

## Report Style and Structure

### Header Levels

- Use `###` (h3) for main sections — e.g., `### Test Summary`
- Use `####` (h4) for subsections — e.g., `#### Device-Specific Results`
- Never use `##` (h2) or `#` (h1) — those are reserved for titles

### Progressive Disclosure

Wrap detailed content in `<details><summary>Section Name</summary>` tags. Use for:
- Verbose details (full logs, raw data)
- Secondary information (minor warnings, extra context)
- Per-item breakdowns when there are many items

Keep critical information visible (summary, critical issues, key metrics).

### Report Structure Pattern

1. **Overview**: 1–2 paragraphs summarizing key findings
2. **Critical Information**: Show immediately (summary stats, critical issues)
3. **Details**: Use `<details><summary>Section Name</summary>` for expanded content
4. **Context**: Add helpful metadata (workflow run, date, trigger)

### Example Report Structure

```markdown
### Summary
- Key metric 1: value
- Key metric 2: value
- Status: ✅/⚠️/❌

### Critical Issues
[Always visible - these are important]

<details>
<summary>View Detailed Results</summary>

[Comprehensive details, logs, traces]

</details>

<details>
<summary>View All Warnings</summary>

[Minor issues and potential problems]

</details>

### Recommendations
[Actionable next steps - keep visible]
```

## Workflow Run References

- Format run IDs as links: `[§12345](https://github.com/owner/repo/actions/runs/12345)`
- Include up to 3 most relevant run URLs at the end under `**References:**`
- Do NOT add footer attribution — the system appends it automatically

## Avoiding Mentions and Backlinks

Without filtering, `@username` notifies users and `#123` creates cross-reference backlinks on that issue/PR — noise every run.

- **`mentions: false`** — escapes all `@mentions`, no notifications sent.
- **`allowed-github-references: []`** — escapes all `#123` / `owner/repo#123` references, no backlinks.
- **`max-bot-mentions: 0`** — neutralizes bot-trigger phrases like `fixes #123` / `closes #456`.

```yaml
safe-outputs:
  mentions: false
  allowed-github-references: []
  max-bot-mentions: 0
  create-issue:
    title-prefix: "Weekly Status:"
    labels: [report]
    close-older-issues: true
    expires: 30
```

Applies globally to all safe-output types (issues, comments, discussions).
