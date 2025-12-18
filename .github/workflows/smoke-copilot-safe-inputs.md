---
description: Smoke Copilot Safe Inputs
on: 
  schedule: daily
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "eyes"
permissions:
  contents: read
  pull-requests: read
  issues: read
name: Smoke Copilot Safe Inputs
engine: copilot
network:
  allowed:
    - defaults
    - node
    - github
imports:
  - shared/gh.md
tools:
  edit:
  bash:
    - "*"
  github: false
  serena: ["go"]
safe-outputs:
    add-comment:
      hide-older-comments: true
    create-issue:
      expires: 1d
    add-labels:
      allowed: [smoke-copilot]
strict: true
---

# Smoke Test: Copilot Engine Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

## Test Requirements

1. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}
2. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-copilot-${{ github.run_id }}.txt` with content "Smoke test passed for Copilot at $(date)" (create the directory if it doesn't exist)
3. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
4. **Serena MCP Testing**: Use Serena to list classes in the project
5. **Safe Input gh Tool Testing**: Use the `safeinputs-gh` tool to run "gh issues list --limit 3" to verify the tool can access GitHub issues

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- PR titles only (no descriptions)
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL

If all tests pass, add the label `smoke-copilot` to the pull request.