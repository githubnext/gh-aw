---
description: Smoke test workflow that validates Claude engine functionality by reviewing recent PRs twice daily
on: 
  schedule: every 12h
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "heart"
permissions:
  contents: read
  issues: read
  pull-requests: read
  
name: Smoke Claude
engine:
  id: claude
  max-turns: 15
strict: true
features:
  mcp-gateway: true
imports:
  - shared/mcp-pagination.md
network:
  allowed:
    - defaults
    - github
    - playwright
sandbox:
  agent: awf  # Firewall enabled
  mcp:
    container: ghcr.io/githubnext/gh-aw-mcpg
    # version defaults to constants.DefaultMCPGatewayVersion (v0.0.9)
tools:
  cache-memory: true
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
    add-comment:
      hide-older-comments: true
    create-issue:
      expires: 2h
    add-labels:
      allowed: [smoke-claude]
    messages:
      footer: "> 💥 *[THE END] — Illustrated by [{workflow_name}]({run_url})*"
      run-started: "💥 **WHOOSH!** [{workflow_name}]({run_url}) springs into action on this {event_type}! *[Panel 1 begins...]*"
      run-success: "🎬 **THE END** — [{workflow_name}]({run_url}) **MISSION: ACCOMPLISHED!** The hero saves the day! ✨"
      run-failure: "💫 **TO BE CONTINUED...** [{workflow_name}]({run_url}) {status}! Our hero faces unexpected challenges..."
timeout-minutes: 10
---

# Smoke Test: Claude Engine Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

## Test Requirements

1. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}
2. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-claude-${{ github.run_id }}.txt` with content "Smoke test passed for Claude at $(date)" (create the directory if it doesn't exist)
3. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
4. **Playwright MCP Testing**: Use playwright to navigate to https://github.com and verify the page title contains "GitHub"
5. **Cache Memory Testing**: Write a test file to `/tmp/gh-aw/cache-memory/smoke-test-${{ github.run_id }}.txt` with content "Cache memory test for run ${{ github.run_id }}" and verify it was created successfully
6. **Safe Input gh Tool Testing**: Use the `safeinputs-gh` tool to run "gh issues list --limit 3" to verify the tool can access GitHub issues
7. **MCP Gateway Testing**: Verify the MCP gateway is running by checking `/tmp/gh-aw/mcp-logs/gateway/stderr.log` for gateway startup messages. Confirm the gateway health endpoint is accessible and MCP servers are proxied through the gateway.

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- PR titles only (no descriptions)
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL

If all tests pass, add the label `smoke-claude` to the pull request.
