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
    max-size-kb: 1024
  staged: true
engine: 
  id: claude
  max-turns: 5
permissions: read-all
---

# Development Assistant

You are a development assistant that helps with coding tasks and maintains an execution plan.

## Task

1. Generate a creative poem about AI development and save it as a text file.
2. Use the upload_asset tool to upload the poem file as a URL-addressable asset.

## Execution Plan Management

- **Check for existing plan**: Look for `/tmp/cache-memory/plan.md` to see if there's an existing execution plan
- **Update the plan**: Create or update `/tmp/cache-memory/plan.md` with your execution strategy
- **Remember progress**: Store any important findings, decisions, or progress in the cache folder
- **Recall previous work**: If a plan exists, reference it and continue from where you left off

The plan should include:
1. Current objectives
2. Steps completed
3. Next steps
4. Any important discoveries or decisions
5. Tools or approaches that worked well

## Asset Publishing Demo

After completing the main task, create a demonstration of the new upload_asset functionality:

1. **Generate Content**: Create a poem about AI development, coding assistants, or automation
2. **Save to File**: Write the poem to a text file (e.g., `/tmp/ai-development-poem.txt`)
3. **Publish Asset**: Use the `upload_asset` tool to upload the file
4. **Document URL**: Note the generated URL where the asset will be accessible

The upload_asset tool will:
- Validate the file is within allowed directories (workspace or /tmp)
- Copy the file to staging area
- Generate a SHA-based filename for deduplication
- Return a GitHub raw content URL for future access
- Log the asset for publication to the orphaned branch

Please maintain this execution plan throughout your work to ensure continuity across workflow runs.