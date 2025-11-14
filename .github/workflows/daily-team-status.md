---
on:
  schedule:
    # Every day at 9am UTC, all days except Saturday and Sunday
    - cron: "0 9 * * 1-5"
  workflow_dispatch:
  # workflow will no longer trigger after 30 days. Remove this and recompile to run indefinitely
  stop-after: +30d 
permissions:
  contents: read
  issues: read
  pull-requests: read
network: defaults
tools:
  github:
safe-outputs:
  create-discussion:
    title-prefix: "[team-status] "
    category: "announcements"
imports:
- githubnext/agentics/shared/reporting.md@09e77ed2e49f0612e258db12839e86e8e2a6c692
source: githubnext/agentics/workflows/daily-team-status.md@09e77ed2e49f0612e258db12839e86e8e2a6c692
---
# Daily Team Status

Create an upbeat daily status report for the team as a GitHub discussion.

## What to include

- Recent repository activity (issues, PRs, discussions, releases, code changes)
- Team productivity suggestions and improvement ideas
- Community engagement highlights
- Project investment and feature recommendations

## Style

- Be positive, encouraging, and helpful 🌟
- Use emojis moderately for engagement
- Keep it concise - adjust length based on actual activity

## Process

1. Gather recent activity from the repository
2. Create a new GitHub discussion with your findings and insights
