---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot/*
      - collect-guards
engine: copilot
safe-outputs:
    staged: true
    create-issue:
    create-pull-request:
---
Create a poem and post it as a new issue.