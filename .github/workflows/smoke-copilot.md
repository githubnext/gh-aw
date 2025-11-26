---
description: Smoke test workflow that validates Copilot engine functionality by reviewing recent PRs every 6 hours
on: 
  schedule:
    - cron: "0 0,6,12,18 * * *"  # Every 6 hours
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "eyes"
permissions:
  contents: read
  pull-requests: read
  issues: read
name: Smoke Copilot
engine: copilot
network:
  allowed:
    - defaults
    - node
    - github
    - playwright
  firewall: true
tools:
  edit:
  bash:
    - "*"
  github:
  playwright:
    allowed_domains:
      - github.com
  serena: ["go"]
safe-outputs:
    add-comment:
    create-issue:
    add-labels:
      allowed: [smoke-copilot]
timeout-minutes: 10
strict: true
---

# Smoke Test: Copilot Engine Validation

This smoke test validates Copilot engine functionality by testing core capabilities:

## Test Requirements

1. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}
2. **File Writing Testing**: Create a test file `/tmp/smoke-test-copilot-${{ github.run_id }}.txt` with content "Smoke test passed for Copilot at $(date)"
3. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
4. **Playwright MCP Testing**: Use playwright to navigate to https://github.com and verify the page title contains "GitHub"

## Output

Add a comment to the current pull request with:
- Summary of the 2 merged PRs
- File writing test result (success/failure)
- Bash tool test result (file content verification)
- Playwright test result (page title verification)
- Overall smoke test status for **Copilot engine**

If all tests pass successfully, add the label `smoke-copilot` to the pull request.