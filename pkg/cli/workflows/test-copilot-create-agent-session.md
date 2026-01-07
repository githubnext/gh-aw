---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
safe-outputs:
  create-agent-session:
    base: main
---

# Test: Create Agent Task

Test workflow for the create-agent-session safe output.

Create a GitHub Copilot agent session to improve code quality in the repository.
