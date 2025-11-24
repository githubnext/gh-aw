---
description: Smoke test workflow that validates Codex engine functionality by reviewing recent PRs every 6 hours
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
name: Smoke Codex
engine: codex
strict: false
network:
  allowed:
    - defaults
    - github
    - playwright
tools:
  github:
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
---

# Smoke Test: Codex Engine Validation

This smoke test validates Codex engine functionality by testing core capabilities:

## Test Requirements

1. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}
2. **File Writing Testing**: Create a test file `/tmp/smoke-test-codex-${{ github.run_id }}.txt` with content "Smoke test passed for Codex at $(date)"
3. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
4. **Playwright MCP Testing**: Use playwright to navigate to https://github.com and verify the page title contains "GitHub"

## Output

Add a comment to the current pull request with:
- Summary of the 2 merged PRs
- File writing test result (success/failure)
- Bash tool test result (file content verification)
- Playwright test result (page title verification)
- Overall smoke test status for **Codex engine**