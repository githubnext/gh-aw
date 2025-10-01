---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot*
engine: github-models
safe-outputs:
    staged: true
    create-issue:
---
Generate a poem. Post result in an issue.
