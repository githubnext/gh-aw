---
on: 
  workflow_dispatch:
name: Dev
description: Add a poem to the latest discussion
timeout-minutes: 5
strict: false
engine: copilot

permissions:
  contents: read
  discussions: write

tools:
  github:
    toolsets: [discussions]
imports:
  - shared/gh.md
safe-outputs:
  update-discussion:
    body:
---

Find the latest discussion in this repository and update its body by appending a short, creative poem about GitHub Agentic Workflows.

The poem should:
- Be 4-8 lines long
- Mention automation, AI agents, or workflow concepts
- Be uplifting and inspiring
- Use the **append** operation to add to the existing discussion body

Output as JSONL format using the update_discussion tool.
