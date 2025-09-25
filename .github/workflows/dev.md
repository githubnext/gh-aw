---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot*
engine: copilot
tools:
  cache-memory: true
safe-outputs:
  create-issue:
  staged: true
---
Summarize the issue and post the summary in a comment.
