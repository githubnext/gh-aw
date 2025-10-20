---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
safe-outputs:
  create-agent-task:
    base: main
---

# Test Agent Task Creation

When triggered, create a GitHub Copilot agent task to improve the code quality in this repository.
