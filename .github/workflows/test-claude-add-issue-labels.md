---
on:
  issues:
    types: [opened, reopened]
  reaction: eyes

engine: 
  id: claude

safe-outputs:
  add-issue-labels:
---

If the title of the issue #${{ github.event.issue.number }} is "Hello" then add the issue labels "claude-safe-output-label-test" to the issue.

