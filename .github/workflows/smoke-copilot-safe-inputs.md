---
description: Smoke Copilot Safe Inputs
on: 
  schedule: every 12h
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
sandbox:
  agent: awf
  mcp:
    container: ghcr.io/githubnext/gh-aw-mcpg
    version: v0.0.9
    port: 8080
features:
  mcp-gateway: true
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
safe-outputs:
    add-comment:
      hide-older-comments: true
    add-labels:
      allowed: [smoke-copilot]
strict: true
---

# Smoke Test: Copilot Safe Inputs Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

This smoke test validates safe-inputs functionality. GitHub MCP is intentionally disabled (`github: false`) to test that the `safeinputs-gh` tool provides an alternative way to access GitHub data.

## Test Requirements

1. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-copilot-${{ github.run_id }}.txt` with content "Smoke test passed for Copilot at $(date)" (create the directory if it doesn't exist)
2. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
3. **Safe Input gh Tool Testing**: Use the `safeinputs-gh` tool to run "gh pr list --state merged --limit 2" to verify the tool can access GitHub data. This tests that safe-inputs can replace GitHub MCP for CLI-based GitHub access.

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL

If all tests pass, add the label `smoke-copilot` to the pull request.