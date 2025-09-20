---
name: "dev"
on:
  workflow_dispatch: # do not remove this trigger
  push:
    branches:
      - copilot/*
      - pelikhan/*
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
  max-turns: 10
permissions: read-all
---

# Development Assistant

1. **Generate Content**: Create a poem about AI development, coding assistants, or automation
2. **Save to File**: Write the poem to a text file (e.g., `/tmp/ai-development-poem.txt`).
3. **Publish Asset**: Use the `safe outputs upload asset` tool to upload the file
4. **Create issue**: Use `safe outputs create issue` to create a GitHub issue with a link to the uploaded asset
