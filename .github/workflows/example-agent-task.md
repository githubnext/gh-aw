---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
safe-outputs:
  create-agent-task:
    base: main
---

# Example: Create Agent Task

This is an example workflow that demonstrates the create-agent-task safe output.

When triggered manually, this workflow will:
1. Analyze the repository structure
2. Identify areas for improvement
3. Create a GitHub Copilot agent task to implement the improvements

The agent task will be configured to use the 'main' branch as its base.
