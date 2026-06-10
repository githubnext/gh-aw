---
description: Guidelines for creating agentic workflows that generate reports — output type selection, formatting style, and automatic cleanup.
---

# Report Generation

For workflows that generate reports — status updates, audits, summaries — posted as GitHub issues, discussions, or comments.

## Choosing the Output Type

| Use case | Recommended output |
|---|---|
| Report (default) | `create-issue` with `close-older-issues` |
| Inline update on an existing issue or PR | `add-comment` with `hide-older-comments` |
| Discussion-based report (only when explicitly requested) | `create-discussion` with `close-older-discussions` |

Default to `create-issue` — searchable, supports close/expire cleanup. Use `create-discussion` only when explicitly requested.

### PM/Stakeholder Digests

- `create-issue` for operational reports (backlog follow-ups, ownership tracking, recurring status).
- `create-discussion` only when the requester explicitly wants threaded collaboration / async feedback.
- If unclear, default to `create-issue`.

## Automatic Cleanup

Configure cleanup for scheduled or recurring reports.

- **`expires`** — auto-close after a window (e.g. `7`, `2w`, `1m`).
- **`close-older-issues: true`** — close previous issues from the same workflow. Requires `title-prefix` or `labels`.
- **`close-older-discussions: true`** — close older matching discussions as "OUTDATED". Requires `title-prefix` or `labels`.
- **`hide-older-comments: true`** — minimize previous comments. Useful for rolling status updates.

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

Wrap detail content in `<details><summary>Section Name</summary>`. Use for verbose logs/raw data, secondary info, per-item breakdowns. Keep summary, critical issues, and key metrics visible.

### Alerts Instead of Emojis

- `> [!NOTE]` — neutral status
- `> [!WARNING]` — warnings
- `> [!CAUTION]` — high-risk or blocking

Do not use emoji severity markers (`✅`, `⚠️`, `❌`, `🧪`).

### Structure Pattern

1. **Overview** — 1–2 paragraphs of key findings
2. **Critical info** — summary stats, critical issues (always visible)
3. **Details** — `<details><summary>...</summary>` for expanded content
4. **Context** — workflow run, date, trigger

### Example Report Structure

```markdown
### Summary
- Key metric 1: value
- Key metric 2: value

> [!WARNING]
> Status: degradation detected in one or more checks.

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

Without filtering, `@username` notifies users and `#123` creates cross-reference backlinks — noise every run.

- **`mentions: false`** — escapes all `@mentions`, no notifications.
- **`allowed-github-references: []`** — escapes `#123` / `owner/repo#123`, no backlinks.
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
