---
on:
  workflow_dispatch: # do not remove this trigger
  push:
    branches-ignore:
      - main
safe-outputs:
  missing-tool:
  staged: true
engine: 
  id: claude
  max-turns: 5
permissions: read-all
---

Write a short poem.

<!-- This workflow tests the integration with the Claude AI engine. 
  Meant as a scratchpad in pull requests. -->