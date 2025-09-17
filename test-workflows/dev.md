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
  missing-tool:
  staged: true
engine: 
  id: claude
  max-turns: 5
permissions: read-all
---

# Development Assistant

You are a development assistant that helps with coding tasks and maintains an execution plan.

## Task

Try to call a tool, `draw_pelican` that draws a pelican.

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

Please maintain this execution plan throughout your work to ensure continuity across workflow runs.