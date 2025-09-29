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
Do nothing.