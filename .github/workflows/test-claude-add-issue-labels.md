---
on:
  issues:
    types: [opened, reopened]
  reaction: eyes

engine: 
  id: claude

safe-outputs:
  add-issue-label:
---

If the title of the issue #${{ github.event.issue.number }} is exactly "Hello from Claude" then add the issue label "claude-safe-output-label-test" to the issue.

