---
description: Smoke test workflow that validates OpenCode custom engine functionality daily
on: 
  schedule: daily
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "rocket"
permissions:
  contents: read
  issues: read
  pull-requests: read
  
name: Smoke OpenCode
imports:
  - shared/opencode.md
strict: true
sandbox:
  mcp:
    container: "ghcr.io/githubnext/gh-aw-mcpg"
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
  serena:
    languages:
      go: {}
safe-outputs:
    add-comment:
      hide-older-comments: true
    create-issue:
      expires: 2h
      group: true
      close-older-issues: true
    add-labels:
      allowed: [smoke-opencode]
    messages:
      footer: "> 🚀 *[Liftoff Complete] — Powered by [{workflow_name}]({run_url})*"
      run-started: "🚀 **IGNITION!** [{workflow_name}]({run_url}) launching for this {event_type}! *[T-minus counting...]*"
      run-success: "🎯 **MISSION SUCCESS** — [{workflow_name}]({run_url}) **TARGET ACQUIRED!** All systems nominal! ✨"
      run-failure: "⚠️ **MISSION ABORT...** [{workflow_name}]({run_url}) {status}! Houston, we have a problem..."
timeout-minutes: 10
---

# Smoke Test: OpenCode Custom Engine Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

## Test Requirements

1. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}
2. **Serena Go Testing**: Use the `serena-go` tool to run a basic go command like "go version" to verify the tool is available
3. **Playwright Testing**: Use playwright to navigate to https://github.com and verify the page title contains "GitHub"
4. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-opencode-${{ github.run_id }}.txt` with content "Smoke test passed for OpenCode at $(date)" (create the directory if it doesn't exist)
5. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)

## Output

1. **Create an issue** with a summary of the smoke test run:
   - Title: "Smoke Test: OpenCode - ${{ github.run_id }}"
   - Body should include:
     - Test results (✅ or ❌ for each test)
     - Overall status: PASS or FAIL
     - Run URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
     - Timestamp

2. Add a **very brief** comment (max 5-10 lines) to the current pull request with:
   - PR titles only (no descriptions)
   - ✅ or ❌ for each test result
   - Overall status: PASS or FAIL

If all tests pass, add the label `smoke-opencode` to the pull request.
