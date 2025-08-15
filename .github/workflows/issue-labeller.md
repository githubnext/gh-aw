---
on:
  issues:
    types: [opened, reopened]
permissions:
  issues: write
tools:
  github:
    allowed: [update_issue]
---
Assign labels to the issue #${{ github.event.issue.number }}.
