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

Create a new poem and save it to file poem.txt, then push the changes to a pull request.