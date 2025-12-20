---
on: 
  workflow_dispatch:
name: Dev
description: Test MCP gateway with GitHub issues
timeout-minutes: 5
strict: true
engine: copilot

permissions:
  contents: read
  issues: read

sandbox:
  mcp:
    container: ghcr.io/githubnext/mcp-gateway
    port: 8080

tools:
  github:
    toolsets: [issues]
imports:
  - shared/gh.md
---

# Test MCP Gateway with GitHub Issues

List the last 2 issues from this repository and verify the answer is correct.

**Requirements:**
1. Use the GitHub tools to fetch the last 2 issues
2. Display the issue numbers and titles
3. Verify the data by checking:
   - Issue numbers are valid
   - Titles are present
   - Issues are sorted by most recent first

**Expected Output:**
- Issue #123: "Title of issue"
- Issue #122: "Title of another issue"

Confirm that you successfully retrieved the issues and the data looks correct.
