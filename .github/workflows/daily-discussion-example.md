---
description: Example daily workflow that creates a discussion using GitHub App authentication
on:
  schedule:
    # Every day at 11am UTC
    - cron: "0 11 * * *"
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read
  actions: read

engine: copilot

timeout-minutes: 10

safe-outputs:
  create-discussion:
    title-prefix: "[daily] "
    category: "General"
    max: 1

imports:
  - shared/safe-output-app.md

tools:
  bash:
    - "date"
    - "echo"
---

# Daily Discussion Creator

Create a daily discussion to engage the community.

## Task

1. Get the current date
2. Create a discussion titled "Daily Check-in for [date]"
3. Include content about:
   - Today's focus areas
   - Recent achievements
   - Questions for the community

Use the GitHub App authentication provided by the shared workflow to create the discussion.
