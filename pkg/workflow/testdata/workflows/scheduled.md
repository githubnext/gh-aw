---
on:
  schedule:
    - cron: "0 9 * * 1"
permissions:
  contents: read
  issues: write
engine: copilot
tools:
  web-search:
  github:
    toolsets: [default]
safe-outputs:
  create-issue:
    title-prefix: "[weekly] "
    labels: [automation, weekly-report]
timeout-minutes: 20
---

# Weekly Report Workflow

Generate a weekly summary report of repository activity.
