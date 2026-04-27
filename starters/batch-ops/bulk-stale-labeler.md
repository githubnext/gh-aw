---
name: Bulk Stale Labeler
description: Apply a "stale" label to issues that have had no activity for 30 days, processed in weekly batches.
on:
  schedule:
    - cron: "0 3 * * 1"
  workflow_dispatch:
permissions:
  contents: read
  issues: read
tools:
  github:
    toolsets: [issues]
safe-outputs:
  add-labels:
    max: 20
  add-comment:
    max: 1
---

Calculate the date 30 days ago and search for open issues with no recent activity using `search_issues` with query `is:issue is:open -label:stale updated:<COMPUTED-DATE`.
For up to 20 results, apply the `stale` label and note that the issue will be closed in 14 days if there is no further activity.
Create a summary comment on this run's log issue listing how many issues were marked stale.
