---
description: Smoke test workflow that validates Codex engine functionality by reviewing recent PRs twice daily
on: 
  schedule: every 12h
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "hooray"
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke Codex
engine: codex
strict: true
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
    - github
    - playwright
tools:
  cache-memory: true
  github:
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
      allowed: [smoke-codex]
    hide-comment:
    messages:
      footer: "> 🔮 *The oracle has spoken through [{workflow_name}]({run_url})*"
      run-started: "🔮 The ancient spirits stir... [{workflow_name}]({run_url}) awakens to divine this {event_type}..."
      run-success: "✨ The prophecy is fulfilled... [{workflow_name}]({run_url}) has completed its mystical journey. The stars align. 🌟"
      run-failure: "🌑 The shadows whisper... [{workflow_name}]({run_url}) {status}. The oracle requires further meditation..."
timeout-minutes: 10
---

# Smoke Test: Codex Engine Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

## Test Requirements

1. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}
2. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-codex-${{ github.run_id }}.txt` with content "Smoke test passed for Codex at $(date)" (create the directory if it doesn't exist)
3. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
4. **Playwright MCP Testing**: Use playwright to navigate to https://github.com and verify the page title contains "GitHub"
5. **Cache Memory Testing**: Write a test file to `/tmp/gh-aw/cache-memory/smoke-test-${{ github.run_id }}.txt` with content "Cache memory test for run ${{ github.run_id }}" and verify it was created successfully
6. **Safe Input gh Tool Testing**: Use the `safeinputs-gh` tool to run "gh issues list --limit 3" to verify the tool can access GitHub issues

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- PR titles only (no descriptions)
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL

If all tests pass, add the label `smoke-codex` to the pull request.
