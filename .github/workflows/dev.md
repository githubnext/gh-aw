---
on:
  workflow_dispatch: # do not remove this trigger
  push:
    branches:
      - copilot/*
      - pelikhan/*
safe-outputs:
  missing-tool:
  staged: true
engine: 
  id: claude
  max-turns: 5
permissions: read-all
---

Try to call a tool that draws a pelican.