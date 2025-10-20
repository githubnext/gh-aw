---
on: 
  workflow_dispatch:
concurrency:
  group: dev-workflow-${{ github.ref }}
  cancel-in-progress: true
name: Dev
engine: claude
permissions:
  contents: read
  actions: read

safe-outputs:
  create-agent-task:
    base: main
---

# Generate a Poem

Generate a poem of exactly 3 words and create an agent task with it.
