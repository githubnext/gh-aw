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
  create-pull-request:
  staged: true
---

Improve README.md and push the changes to a pull request.