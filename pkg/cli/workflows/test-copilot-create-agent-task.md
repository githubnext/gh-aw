---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
safe-outputs:
  create-agent-task:
    base: main
---

# Test: Create Agent Task

Test workflow for the create-agent-task safe output.

Create a GitHub Copilot agent task to improve code quality in the repository.
