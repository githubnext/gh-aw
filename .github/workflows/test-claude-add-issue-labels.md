---
on:
  issues:
    types: [opened]
  reaction: eyes
engine: 
  id: claude
  model: claude-3-5-sonnet-20241022
timeout_minutes: 10
permissions:
  actions: read
  contents: read
safe-outputs:
  add-issue-labels:
    allowed: ["bug", "feature"]
---

Add the issue labels "quack" and "dog" to the issue.

