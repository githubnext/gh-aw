---
on:
  issues:
    types: [opened]
  reaction: eyes

engine: 
  id: claude

permissions:
  issues: read

safe-outputs:
  add-issue-labels:
---

Add the issue labels "quack" and "dog" to the issue.

