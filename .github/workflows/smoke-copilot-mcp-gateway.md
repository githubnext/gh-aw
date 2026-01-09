---
description: Smoke test workflow that validates MCP Gateway functionality with AWF firewall
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
name: Smoke Copilot MCP Gateway
engine: copilot
features:
  mcp-gateway: true
sandbox:
  agent: awf
  mcp:
    container: ghcr.io/githubnext/gh-aw-mcpg
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
      allowed: [smoke-mcp-gateway]
    messages:
      footer: "> 🌐 *Gateway tested by [{workflow_name}]({run_url})*"
      run-started: "🌐 MCP Gateway smoke test initiated... [{workflow_name}]({run_url}) is validating gateway routing for {event_type}..."
      run-success: "✅ Gateway validation complete... [{workflow_name}]({run_url}) confirmed MCP gateway is operational. 🚀"
      run-failure: "❌ Gateway validation failed... [{workflow_name}]({run_url}) {status}. MCP gateway may not be working correctly."
timeout-minutes: 10
strict: true
---

# Smoke Test: Copilot MCP Gateway Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

This workflow validates that the MCP Gateway integration works correctly with the AWF firewall. The gateway routes MCP server calls through a unified HTTP endpoint using `host.docker.internal`.

## Test Requirements

1. **GitHub MCP via Gateway**: Review the last 2 merged pull requests in ${{ github.repository }} to verify GitHub MCP server works through the gateway
2. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-mcp-gateway-${{ github.run_id }}.txt` with content "MCP Gateway smoke test passed at $(date)"
3. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
4. **Gateway Logs Check**: Check `/tmp/gh-aw/mcp-logs/gateway/` for gateway startup logs to confirm the gateway started successfully
5. **Gateway Config Check**: Read `/tmp/gh-aw/mcp-config/gateway-input.json` to verify the gateway received proper configuration

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- PR titles only (no descriptions)
- ✅ or ❌ for each test result
- Gateway status: RUNNING or NOT RUNNING
- Overall status: PASS or FAIL

If all tests pass, add the label `smoke-mcp-gateway` to the pull request.
