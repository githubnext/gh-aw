---
description: Generates a daily markdown report summarizing project activity, organized by project directories, and stores it in the repository
on:
  schedule:
    - cron: "0 7 * * 1-5"  # 7 AM UTC on weekdays
  workflow_dispatch:
    inputs:
      start_date:
        description: "Custom start date (YYYY-MM-DD format, optional)"
        required: false
        type: string
      end_date:
        description: "Custom end date (YYYY-MM-DD format, optional)"
        required: false
        type: string
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
network:
  allowed:
    - defaults
  firewall: true
tools:
  edit:
  bash:
    - "git log:*"
    - "git diff:*"
    - "git show:*"
    - "ls:*"
    - "find:*"
    - "cat:*"
    - "date:*"
    - "mkdir:*"
    - "head:*"
    - "tail:*"
  github:
    toolsets:
      - repos
      - issues
      - pull_requests
safe-outputs:
  create-pull-request:
    title-prefix: "[daily-report] "
    labels: [automation, daily-report]
    draft: false
---

# Daily Project Activity Report Generator

Generate a concise, human-readable markdown report summarizing recent activity in **${{ github.repository }}**, organized by project directories.

## Configuration

First, check if a configuration file exists at `.github/daily-report-config.json` or `config/daily-report.json`.

**Expected config format (if present):**

```json
{
  "projects": [
    {"name": "Project Name", "paths": ["path/to/project/"]},
    {"name": "Another Project", "paths": ["another/path/", "second/path/"]}
  ],
  "ignore_patterns": [".github/", "scripts/", "docs/", ".devcontainer/"],
  "reports_directory": "reports/daily",
  "include_index": true
}
```

**If no config exists**, infer projects from top-level directories, excluding common non-project directories: `.github`, `.git`, `scripts`, `docs`, `.devcontainer`, `.vscode`, `node_modules`, `vendor`.

## Reporting Window

Determine the time window for the report:

1. If custom dates provided via `${{ github.event.inputs.start_date }}` and `${{ github.event.inputs.end_date }}`, use those.
2. Otherwise, check for the most recent report in `reports/daily/` to find the last report date.
3. If no previous report, use the last 24 hours.

## Data Collection

For the determined time window, collect:

1. **Commits**: Use `git log` to find commits affecting the repository within the time window.
2. **Pull Requests**: Query GitHub API for PRs created, merged, or closed in the window.
3. **Issues**: Query GitHub API for issues opened, closed, or commented in the window.

For each activity item, determine which project it belongs to based on file paths affected.

## Report Structure

Create a markdown file at `reports/daily/YYYY-MM-DD-daily-report.md` with this structure:

```markdown
# Daily Activity Report - [Date]

**Time Window**: [Start Date/Time] to [End Date/Time]

## Summary

[3-7 bullet points highlighting the most notable changes across all projects]

## Per-Project Activity

### [Project Name]

[Brief description if available from config]

#### Commits

| SHA | Title | Author | Time |
|-----|-------|--------|------|
| [short_sha] | [commit title] | @author | [relative time] |

#### Pull Requests

| # | Title | State | Action |
|---|-------|-------|--------|
| [number] | [title] | [open/merged/closed] | [created/updated/merged] |

#### Issues

| # | Title | State | Action |
|---|-------|-------|--------|
| [number] | [title] | [open/closed] | [opened/closed/commented] |

### [Next Project Name]

[Same structure]

### Uncategorized

[Activity not matching any project]

---

## Notes / Follow-ups

[Optional: Unreviewed PRs, newly opened bugs, or blocked tasks if any]
```

## Index File (Optional)

If `include_index` is true in config (default: true), update `reports/daily-index.md`:

```markdown
# Daily Reports Index

| Date | Summary | Link |
|------|---------|------|
| YYYY-MM-DD | [One-line summary] | [View Report](./daily/YYYY-MM-DD-daily-report.md) |
```

## Output Requirements

1. Create the reports directory if it doesn't exist: `mkdir -p reports/daily`
2. Write the daily report markdown file
3. Update the index file (if configured)
4. Commit the changes via `create-pull-request` safe output

## Edge Cases

- **No activity**: Generate a report stating "No activity recorded for this period."
- **No projects detected**: Fall back to a single "Repository-wide" section.
- **Existing report for today**: Update the file idempotently (replace content, don't duplicate).

## Style Guidelines

- Use neutral, concise language
- Format dates consistently (YYYY-MM-DD for filenames, readable format in content)
- Keep summaries brief (1 sentence per bullet)
- Use tables for activity lists for easy scanning
- Include links to issues/PRs where possible: `#123`
