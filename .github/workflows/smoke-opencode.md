---
description: Smoke OpenCode
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
name: Smoke OpenCode
engine: opencode
tools:
  cache-memory: true
  edit:
  bash:
    - "*"
  github:
  playwright:
    allowed_domains:
      - github.com
  serena: ["go"]
  web-fetch:
sandbox:
  mcp:
    container: "ghcr.io/githubnext/gh-aw-mcpg"
    version: latest
safe-outputs:
    add-comment:
      hide-older-comments: true
    create-issue:
      expires: 2h
    add-labels:
      allowed: [smoke-opencode]
    messages:
      footer: "> 🚀 *Report filed by [{workflow_name}]({run_url})*"
      run-started: "🚀 [{workflow_name}]({run_url}) is now investigating this {event_type}..."
      run-success: "🚀 [{workflow_name}]({run_url}) has concluded. All systems operational. ✅"
      run-failure: "🚀 [{workflow_name}]({run_url}) reports {status}. Investigation ongoing... ⚠️"
timeout-minutes: 5
strict: true
---

# Smoke Test: OpenCode Engine Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

## Test Requirements

1. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}
2. **Serena Go Testing**: Use the `serena-go` tool to run a basic go command like "go version" to verify the tool is available
3. **Playwright Testing**: Use playwright to navigate to <https://github.com> and verify the page title contains "GitHub"
4. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-opencode-${{ github.run_id }}.txt` with content "Smoke test passed for OpenCode at $(date)" (create the directory if it doesn't exist)
5. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- PR titles only (no descriptions)
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL
- Mention the pull request author and any assignees

If all tests pass, add the label `smoke-opencode` to the pull request.
