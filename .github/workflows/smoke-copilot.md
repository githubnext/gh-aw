---
description: Smoke Copilot
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
  actions: read
name: Smoke Copilot
engine: copilot
imports:
  - shared/reporting.md
network:
  allowed:
    - defaults
    - node
    - github
    - playwright
tools:
  agentic-workflows:
  cache-memory: true
  edit:
  bash:
    - "*"
  github:
  playwright:
    allowed_domains:
      - github.com
  serena:
    languages:
      go: {}
  web-fetch:
runtimes:
  go:
    version: "1.25"
sandbox:
  mcp:
    container: "ghcr.io/githubnext/gh-aw-mcpg"
safe-outputs:
    add-comment:
      hide-older-comments: true
    create-issue:
      expires: 2h
      group: true
      close-older-issues: true
    add-labels:
      allowed: [smoke-copilot]
    remove-labels:
      allowed: [smoke]
    messages:
      append-only-comments: true
      footer: "> 📰 *BREAKING: Report filed by [{workflow_name}]({run_url})*"
      run-started: "📰 BREAKING: [{workflow_name}]({run_url}) is now investigating this {event_type}. Sources say the story is developing..."
      run-success: "📰 VERDICT: [{workflow_name}]({run_url}) has concluded. All systems operational. This is a developing story. 🎤"
      run-failure: "📰 DEVELOPING STORY: [{workflow_name}]({run_url}) reports {status}. Our correspondents are investigating the incident..."
timeout-minutes: 5
strict: true
---

# Smoke Test: Copilot Engine Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

## Test Requirements

1. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}
2. **Serena MCP Testing**: Use the Serena MCP server tool `activate_project` to initialize the workspace at `${{ github.workspace }}` and verify it succeeds (do NOT use bash to run go commands - use Serena's MCP tools)
3. **Playwright Testing**: Use playwright to navigate to <https://github.com> and verify the page title contains "GitHub"
4. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-copilot-${{ github.run_id }}.txt` with content "Smoke test passed for Copilot at $(date)" (create the directory if it doesn't exist)
5. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)

## Output

1. **Create an issue** with a summary of the smoke test run:
   - Title: "Smoke Test: Copilot - ${{ github.run_id }}"
   - Body should include:
     - Test results (✅ or ❌ for each test)
     - Overall status: PASS or FAIL
     - Run URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
     - Timestamp
     - Pull request author and assignees

2. Add a **very brief** comment (max 5-10 lines) to the current pull request with:
   - PR titles only (no descriptions)
   - ✅ or ❌ for each test result
   - Overall status: PASS or FAIL
   - Mention the pull request author and any assignees

If all tests pass:
- Add the label `smoke-copilot` to the pull request
- Remove the label `smoke` from the pull request
