---
on: 
  workflow_dispatch:
  reaction: "eyes"
  push:
    branches:
      - copilot/*
engine: copilot
safe-outputs:
  create-issue:
  staged: true
---
Summarize the issue.
