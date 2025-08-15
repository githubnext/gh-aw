---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [update_issue]
---
Assign labels to the issue #${{ github.event.issue.number }}.
