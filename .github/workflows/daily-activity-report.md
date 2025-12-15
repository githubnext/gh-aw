---
on:
  schedule:
    - cron: daily at 9:00
  workflow_dispatch:

permissions: read-all

safe-outputs:
  create-issue:
    title-prefix: "[Daily Report] "
---

# Daily Activity Report

Create a daily summary of repository activity from the last 24 hours.

## Task

1. Check for new issues, pull requests, and commits from yesterday
2. Count the total activity (issues opened, PRs created, PRs merged)
3. Identify any notable changes or discussions
4. Create an issue with a brief summary (3-5 sentences max)

Keep the report concise and highlight only the most important activity.
