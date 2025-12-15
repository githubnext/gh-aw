---
on:
  schedule:
    - cron: "0 9 * * 1-5"  # 9 AM UTC, weekdays only
  workflow_dispatch:

permissions: read-all

safe-outputs:
  create-issue:
    title-prefix: "[Daily Report] "
    labels: [report, automated]
    expires: 7d
---

# Daily Activity Report

Create a daily summary of repository activity from the last 24 hours.

## Task

1. Check for new issues, pull requests, and commits from yesterday
2. Count the total activity (issues opened, PRs created, PRs merged)
3. Identify any notable changes or discussions
4. Create an issue with a brief summary (3-5 sentences max)

Keep the report concise and highlight only the most important activity.
