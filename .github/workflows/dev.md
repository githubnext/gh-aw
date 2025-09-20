---
name: "dev"
on:
  workflow_dispatch: # do not remove this trigger
  push:
    branches:
      - copilot/*
      - pelikhan/*
tools:
  cache-memory: true
safe-outputs:
  upload-asset:
    branch: "dev-assets/${{ github.run_id }}"
    max-size: 1024
  create-issue:
    title-prefix: "[dev] "
    labels: [automation, dev-workflow]
  staged: true
engine: 
  id: claude
  max-turns: 5
permissions: read-all
---

# Development Assistant

1. **Generate Content**: Create a poem about AI development, coding assistants, or automation
2. **Save to File**: Write the poem to a text file (e.g., `/tmp/ai-development-poem.txt`)
3. **Publish Asset**: Use the `upload asset` tool to upload the file
4. **Create issue**: Use `create issue` to create a GitHub issue with a link to the uploaded asset

Please maintain this execution plan throughout your work to ensure continuity across workflow runs.