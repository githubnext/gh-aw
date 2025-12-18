---
on: 
  workflow_dispatch:
name: Dev
description: Add a poem to an issue
timeout-minutes: 5
strict: false
engine: copilot

permissions:
  contents: read
  issues: write

tools:
  github:
    toolsets: [issues]
imports:
  - shared/gh.md
safe-outputs:
  update-issue:
    body:
---

Find an open issue in this repository and update its body by appending a short, creative poem about GitHub Agentic Workflows.

The poem should:
- Be 4-8 lines long
- Mention automation, AI agents, or workflow concepts
- Be uplifting and inspiring
- Be added to the existing issue body

You MUST use the update_issue tool to update an issue with a poem in the body. This is required.
