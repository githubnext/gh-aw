---
name: Stale PR Nudge
description: Comment on pull requests that have had no activity for 7 days to keep reviews moving.
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
tools:
  github:
    toolsets: [pull_requests]
safe-outputs:
  add-comment:
    max: 5
---

Calculate the date 7 days ago and search for open, non-draft pull requests with no recent activity using `search_pull_requests` with query `is:pr is:open is:unreviewed updated:<COMPUTED-DATE -is:draft`.
For each result (up to 5), post a polite nudge comment tagging the assignee or author asking for a status update.
Skip pull requests that already have the `on-hold` label.
