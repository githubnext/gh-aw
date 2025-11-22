---
description: Example daily workflow that creates an issue using GitHub App authentication
on:
  schedule:
    # Every day at 10am UTC
    - cron: "0 10 * * *"
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read
  actions: read

engine: copilot

timeout-minutes: 10

safe-outputs:
  create-issue:
    title-prefix: "[daily] "
    labels: [automation, daily-report]
    max: 1

imports:
  - shared/safe-output-app.md

tools:
  bash:
    - "date"
    - "echo"
---

# Daily Issue Creator

Create a daily issue summarizing repository activity.

## Task

1. Get the current date
2. Create an issue titled "Daily Report for [date]"
3. Include a summary of:
   - Recent commits
   - Open pull requests
   - Recent issues

Use the GitHub App authentication provided by the shared workflow to create the issue.
