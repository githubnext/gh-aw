---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot*
engine: copilot
safe-outputs:
  create-issue:
  staged: true
---
Summarize the issue and post the summary in a comment.
