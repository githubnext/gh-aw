---
on: 
  workflow_dispatch:
    inputs:
      issue_number:
        description: Issue number to read
        required: true
        type: string
name: Dev
description: Read an issue and post a poem about it
timeout-minutes: 5
strict: true
engine: copilot

permissions:
  contents: read
  issues: read

features:
  action-mode: "script"

tools:
  github:
    toolsets: [issues]

safe-outputs:
  staged: true
  add-comment:
    max: 1
---

# Read Issue and Post Poem

Read a single issue and post a poem about it as a comment in staged mode.

**Requirements:**
1. Read the issue specified by the `issue_number` input
2. Understand the issue's title, body, and context
3. Write a creative poem inspired by the issue content
4. Post the poem as a comment on the issue using `create_issue_comment` in staged mode
5. The poem should be relevant, creative, and engaging
