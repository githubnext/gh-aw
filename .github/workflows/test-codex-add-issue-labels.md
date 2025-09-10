---
on:
  issues:
    types: [opened, reopened]
  reaction: eyes

engine: 
  id: codex

safe-outputs:
  add-issue-label:
---

If the title of the issue #${{ github.event.issue.number }} is "Hello from Codex" then add the issue label "codex-safe-output-label-test" to the issue.

