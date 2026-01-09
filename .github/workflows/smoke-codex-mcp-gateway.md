---
description: Smoke test workflow that validates MCP Gateway functionality with Codex engine
on: 
  schedule: every 12h
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "rocket"
permissions:
  contents: read
  pull-requests: read
  issues: read
name: Smoke Codex MCP Gateway
engine: codex
features:
  mcp-gateway: true
sandbox:
  agent: awf
  mcp:
    container: ghcr.io/githubnext/gh-aw-mcpg
    version: v0.0.9
    port: 8080
network:
  allowed:
    - defaults
    - github
tools:
  edit:
  bash:
    - "*"
  github:
safe-outputs:
    add-comment:
      hide-older-comments: true
    add-labels:
      allowed: [smoke-mcp-gateway-codex]
    messages:
      footer: "> 🔮 *Gateway oracle report by [{workflow_name}]({run_url})*"
      run-started: "🔮 The gateway spirits awaken... [{workflow_name}]({run_url}) begins divination of {event_type}..."
      run-success: "✨ Gateway prophecy fulfilled... [{workflow_name}]({run_url}) confirms MCP gateway is aligned. 🌟"
      run-failure: "🌑 Gateway shadows linger... [{workflow_name}]({run_url}) {status}. The gateway requires further meditation."
timeout-minutes: 10
strict: true
---

# Smoke Test: Codex MCP Gateway Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

This workflow validates that the MCP Gateway integration works correctly with the Codex engine and AWF firewall.

## Test Requirements

1. **GitHub MCP via Gateway**: Review the last 2 merged pull requests in ${{ github.repository }} to verify GitHub MCP server works through the gateway
2. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-codex-mcp-gateway-${{ github.run_id }}.txt` with content "Codex MCP Gateway smoke test passed at $(date)"
3. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
4. **Gateway Configuration**: Read `/tmp/gh-aw/mcp-config/` to verify the gateway configuration was generated

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- PR titles only (no descriptions)
- ✅ or ❌ for each test result
- Gateway status: RUNNING or NOT RUNNING
- Overall status: PASS or FAIL

If all tests pass, add the label `smoke-mcp-gateway-codex` to the pull request.
