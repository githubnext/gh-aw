---
on:
  schedule:
    - cron: "0 9 * * 1-5"
  workflow_dispatch:
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
---
# Daily Team Status (Cached)

Create an upbeat daily status report for the team as a GitHub discussion.

This is a CACHED version of the workflow to demonstrate offline compilation!

## What to include

- Recent repository activity
- Team productivity suggestions
- Community engagement highlights

## Style

- Be positive and encouraging 🌟
- Use emojis moderately
- Keep it concise

## Process

1. Gather recent activity
2. Create discussion with findings
