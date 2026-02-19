---
name: issue-trigger-test
description: Workflow triggered by issue events
on:
  issues:
    types: [opened, labeled]
permissions:
  contents: read
  issues: read
engine: copilot
timeout-minutes: 10
---

# Mission

When an issue is opened or labeled, analyze its content and report findings.
