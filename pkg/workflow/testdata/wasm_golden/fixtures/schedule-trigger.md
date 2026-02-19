---
name: schedule-trigger-test
description: Workflow triggered on a schedule
on:
  schedule:
    - cron: "0 9 * * 1-5"
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
timeout-minutes: 15
---

# Mission

Run a daily status check on weekday mornings.
