---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot*
      - detection
      - codex*
engine: claude
safe-outputs:
    staged: true
    create-issue:
---
Write a poem and post it as an issue.
