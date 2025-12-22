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

# Test MCP Gateway: Read Last Issue and Write Poem in Staged Mode

Read the most recent issue from the repository and write a creative poem about it in a new issue using **staged mode** (preview mode).

**Requirements:**
1. Use the GitHub tools to fetch the most recent issue from this repository
2. Read the issue title and body to understand what it's about
3. Write a short, creative poem (4-6 lines) inspired by the content of that issue
4. Create a new issue with:
   - Title: Start with the prefix "[Poetry Test]" followed by a creative title that relates to the original issue
   - Body: Your poem about the issue, plus a reference to the original issue number
5. **IMPORTANT**: Use staged mode (add `staged: true` to your create-issue call) so the issue is previewed with the 🎭 indicator but not actually created
6. Confirm that:
   - You successfully read the last issue
   - You created a poem inspired by it
   - The new issue was created in staged mode with the 🎭 indicator
