---
timeout-minutes: 10
strict: true
on:
  schedule:
  - cron: 0 9 * * 1-5
  stop-after: +1mo
  workflow_dispatch: null
permissions:
  contents: read
  issues: read
  pull-requests: read
tracker-id: daily-team-status
network: defaults
imports:
- githubnext/agentics/workflows/shared/reporting.md@d3422bf940923ef1d43db5559652b8e1e71869f3
safe-outputs:
  create-issue:
    expires: 1d
    title-prefix: "[team-status] "
description: |
  This workflow created daily team status reporter creating upbeat activity summaries.
  Gathers recent repository activity (issues, PRs, releases, code changes)
  and generates engaging GitHub issues with productivity insights, community
  highlights, and project recommendations. Uses a positive, encouraging tone with
  moderate emoji usage to boost team morale.
source: githubnext/agentics/workflows/daily-team-status.md@d3422bf940923ef1d43db5559652b8e1e71869f3
tools:
  github: null
---

{{#runtime-import? .github/shared-instructions.md}}

# Daily Team Status

Create an upbeat daily status report for the team as a GitHub issue.

## What to include

- Recent repository activity (issues, PRs, releases, code changes)
- Team productivity suggestions and improvement ideas
- Community engagement highlights
- Project investment and feature recommendations

## Style

- Be positive, encouraging, and helpful 🌟
- Use emojis moderately for engagement
- Keep it concise - adjust length based on actual activity

## Process

1. Gather recent activity from the repository
2. Create a new GitHub issue with your findings and insights
