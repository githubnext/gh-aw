---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot*
      - detection
engine: codex
safe-outputs:
    staged: true
    create-issue:
---
Write a poem and post it as an issue.
