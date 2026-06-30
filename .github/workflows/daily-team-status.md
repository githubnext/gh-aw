---
description: |
  This workflow is a daily team status reporter creating upbeat activity summaries.
  Gathers recent repository activity (issues, PRs, discussions, releases, code changes)
  and generates engaging GitHub issues with productivity insights, community
  highlights, and project recommendations. Uses a positive, encouraging tone with
  moderate emoji usage to boost team morale.

on:
  schedule: daily
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read

network: defaults

tools:
  github:
    min-integrity: none # This workflow is allowed to examine and comment on any issues

safe-outputs:
  mentions: false
  allowed-github-references: []
  create-issue:
    title-prefix: "[team-status] "
    labels: [report, daily-status]
    close-older-issues: true
source: githubnext/agentics/workflows/team-status.md@main
---

# Team Status

Create an upbeat daily status report for the team as a GitHub issue.

## What to include

- Recent repository activity (issues, PRs, discussions, releases, code changes)
- Team productivity suggestions and improvement ideas
- Community engagement highlights
- Project investment and feature recommendations

## Style

- Be positive, encouraging, and helpful 🌟
- Use emojis moderately for engagement
- Keep it concise - adjust length based on actual activity
- Use `###` (h3) or lower for all report headers; never use `#` or `##` inside the report body
- Wrap long activity item lists and recommendation lists in `<details><summary>...</summary>...</details>` to reduce scrolling

## Report Structure

Structure the report with this hierarchy:

- `### 🌟 Team Snapshot`
  - 1-2 paragraph overview of today's team momentum and notable signals
- `### 🛠️ Activity Highlights`
  - Keep concise highlights visible
  - Put detailed activity lists in `<details>` blocks
- `### 💡 Recommendations`
  - Put long recommendation lists in `<details>` blocks
- `### 📌 Next Steps`
  - Short actionable closing section

## Process

1. Gather recent activity from the repository
2. Create a new GitHub issue with your findings and insights
