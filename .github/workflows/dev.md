---
on: 
  workflow_dispatch:
name: Dev
description: Create a poem about GitHub and save it to repo-memory
timeout-minutes: 5
strict: false
engine: codex
permissions:
  contents: read
  issues: read
tools:
  repo-memory:
    branch-name: memory/poems
    description: "Poem collection"
  github: false
imports:
  - shared/gh.md
---
# Create a Poem and Save to Repo Memory

Create a creative poem about GitHub and agentic workflows, then save it to the repo-memory.

## Task

1. **Create a Poem**: Write a creative, fun poem about GitHub, automation, and agentic workflows.
   - The poem should be 8-12 lines
   - Include references to GitHub features like Issues, Pull Requests, Actions, etc.
   - Make it engaging and technical but fun

2. **Save to Repo Memory**: Save the poem to `/tmp/gh-aw/repo-memory-default/memory/default/poem_{{ github.run_number }}.md`
   - Use the run number in the filename to make it unique
   - Include a header with the date and run information
   - The file will be automatically committed and pushed to the `memory/poems` branch

3. **List Previous Poems**: If there are other poem files in the repo memory, list them to show the history.

## Example Poem Structure

```markdown
# Poem #{{ github.run_number }}
Date: {{ current date }}
Run ID: ${{ github.run_id }}

[Your poem here]
```