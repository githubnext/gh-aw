---
description: Smoke Copilot Safe Inputs
on: 
  schedule:
    - cron: "0 0,7,13,19 * * *"  # Every 6 hours
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
  firewall:
    log-level: debug  # Enable debug-level firewall logs
imports:
  - shared/gh.md
tools:
  edit:
  bash:
    - "*"
  github: false
safe-outputs:
    add-comment:
    create-issue:
    add-labels:
      allowed: [smoke-copilot]
    messages:
      footer: "📰🔥📋 [{run_url}]({run_url})"
      run-started: "📰🚀🔍👀📡🕵️ [{run_url}]({run_url})"
      run-success: "📰✅🎉🏁✨🎤 [{run_url}]({run_url})"
      run-failure: "📰⚠️🔥❌🚨🔧 [{run_url}]({run_url})"
timeout-minutes: 5
strict: true
---

# Smoke Test: Copilot Engine Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

## Test Requirements

1. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}
2. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-copilot-${{ github.run_id }}.txt` with content "Smoke test passed for Copilot at $(date)" (create the directory if it doesn't exist)
3. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
4. **GitHub MCP Default Toolset Testing**: Verify that the `get_me` tool is NOT available with default toolsets. Try to use it and confirm it fails with a tool not found error.

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- PR titles only (no descriptions)
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL

If all tests pass, add the label `smoke-copilot` to the pull request.
