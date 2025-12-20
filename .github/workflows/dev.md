---
on: 
  workflow_dispatch:
name: Dev
description: Test MCP gateway with issue creation in staged mode
timeout-minutes: 5
strict: true
engine: copilot

permissions:
  contents: read
  issues: read

sandbox:
  mcp:
    port: 8080

tools:
  github:
    toolsets: [issues]
safe-outputs:
  create-issue:
    title-prefix: "[Poetry Test] "
    max: 1
imports:
  - shared/gh.md
---

# Test MCP Gateway with Issue Creation in Staged Mode

Create an issue with a poem about GitHub Copilot in **staged mode** (preview mode).

**Requirements:**
1. Write a short, creative poem (4-6 lines) about GitHub Copilot
2. Create an issue with:
   - Title: Start with the prefix "[Poetry Test]" followed by a creative title for your poem
   - Body: The poem you created
3. **IMPORTANT**: Use staged mode (add `staged: true` to your create-issue call) so the issue is previewed with the 🎭 indicator but not actually created
4. Confirm that the issue creation was successful in staged mode
