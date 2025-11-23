---
description: Smoke test workflow that validates Claude engine functionality by reviewing recent PRs every 6 hours
on: 
  schedule:
    - cron: "0 0,6,12,18 * * *"  # Every 6 hours
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
permissions:
  contents: read
  issues: read
  pull-requests: read
  
name: Smoke Claude
engine:
  id: claude
  max-turns: 15
imports:
  - shared/mcp-pagination.md
network:
  allowed:
    - defaults
    - github
    - playwright
tools:
  github:
    toolsets: [repos, pull_requests]
  playwright:
    allowed_domains:
      - github.com
  edit:
  bash:
    - "*"
  serena: ["go"]
safe-outputs:
    staged: true
    add-comment:
timeout-minutes: 10
strict: false
---

# Smoke Test: Claude Engine Validation

This smoke test validates Claude engine functionality by testing core capabilities:

## Test Requirements

1. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}
2. **File Writing Testing**: Create a test file `/tmp/smoke-test-claude-${{ github.run_id }}.txt` with content "Smoke test passed for Claude at $(date)"
3. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
4. **Playwright MCP Testing**: Use playwright to navigate to https://github.com and verify the page title contains "GitHub"

## Output

Add a comment to the current pull request with:
- Summary of the 2 merged PRs
- File writing test result (success/failure)
- Bash tool test result (file content verification)
- Playwright test result (page title verification)
- Overall smoke test status for **Claude engine**