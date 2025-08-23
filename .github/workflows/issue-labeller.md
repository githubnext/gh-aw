---
on:
  issues:
    types: [opened]
output:
  labels:
    allowed: [bug, feature]
---
## Issue Labeller
- analyze issue #${{ github.event.issue.number }} content
- categorize the content as 'bug' or 'feature'
- label the issue accordingly
