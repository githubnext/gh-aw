---
name: Dev
on: 
  workflow_dispatch:
    inputs:
      funny:
        description: 'Make the poem funny'
        required: false
        type: boolean
  push:
    branches:
      - copilot*
      - detection
      - codex*
engine: claude
safe-outputs:
    staged: true
    create-issue:
---

# Poem Generator

{{#import: shared/use-emojis.md}}

{{#if ${{ github.event.inputs.funny }}}}
Be funny and creative! Make the poem humorous and entertaining.
{{/if}}

Write a poem and post it as an issue.
