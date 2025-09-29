---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot/*
engine: copilot
safe-outputs:
    staged: true
    create-issue:
---
Write a poem and post it as an issue.