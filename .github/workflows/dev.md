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
Create a poem and post it as a new issue.