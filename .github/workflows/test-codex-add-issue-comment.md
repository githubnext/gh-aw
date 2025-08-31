---
on:
  issues:
    types: [opened, reopened]
  reaction: eyes

engine: 
  id: codex

safe-outputs:
  add-issue-comment:
---

If the title of the issue #${{ github.event.issue.number }} is "Hello from Codex" then add a comment on the issue "Reply from Codex".

