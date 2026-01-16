---
on: 
  issues:
    types: [labeled, opened]
    names: [dev]
  workflow_dispatch:
name: Dev
description: Read an issue and post a poem about it using OpenCode
timeout-minutes: 5
strict: false
sandbox: false
engine: opencode

permissions:
  contents: read
  issues: read

tools:
  github:
    toolsets: [issues]

safe-outputs:
  staged: true
  add-comment:
    max: 1
---

# Read Issue and Post Poem

Read the current issue and post a poem about it as a comment in staged mode.

**Requirements:**
1. Read the current issue from the GitHub context
2. Understand the issue's title, body, and context
3. Write a creative poem inspired by the issue content
4. Post the poem as a comment on the issue using `create_issue_comment` in staged mode
5. The poem should be relevant, creative, and engaging
