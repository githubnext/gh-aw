---
on:
  issues:
    types: [opened, reopened]
  reaction: eyes

engine: 
  id: codex

safe-outputs:
  update-issue:
    status:
    title:
    body:
---

If the title of the issue #${{ github.event.issue.number }} is exactly "[codex-test] Update Issue Test" then:

1. Change the status to "closed"
2. Update the title to "[UPDATED] Update Issue Test - Processed by Codex"
3. Add a line to the end of the body saying "This issue was automatically updated by the Codex agentic workflow."
