---
description: Smoke test for blocked domains with Claude engine
on: 
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke Blocked Domains Claude
engine: claude
network:
  firewall: true
  allowed:
    - defaults
    - github
  blocked:
    - npmjs.org
    - registry.npmjs.org
safe-outputs:
    add-comment:
      hide-older-comments: true
    create-issue:
      expires: 2h
    add-labels:
      allowed: [smoke-blocked-domains-claude]
    messages:
      footer: "> 🚫 *Blocked domains tested by [{workflow_name}]({run_url})*"
      run-started: "🚫 Testing blocked domains... [{workflow_name}]({run_url}) is validating domain blocking for {event_type}..."
      run-success: "✅ Blocked domains test complete... [{workflow_name}]({run_url}) confirmed domain blocking is operational. 🛡️"
      run-failure: "❌ Blocked domains test failed... [{workflow_name}]({run_url}) {status}. Domain blocking may not be working correctly."
timeout-minutes: 5
tools:
  github:
  bash:
    - "*"
---

# Smoke Test: Blocked Domains with Claude

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible.**

## Test Requirements

This workflow validates that the blocked domains feature works correctly with the Claude engine and AWF firewall.

1. **Allowed Domain Testing**: Test that GitHub domains (allowed) are accessible using bash tools - this should succeed
2. **Blocked Domain Testing**: Attempt to access NPM domains (blocked) using `curl https://registry.npmjs.org` - this should FAIL or be blocked by the firewall
3. **Blocked Ecosystem Testing**: Verify that `npmjs.org` is also blocked (part of Node ecosystem blocking)
4. **GitHub MCP Testing**: Verify GitHub MCP server works (allowed domains should not affect GitHub toolset functionality)
5. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-blocked-domains-claude-${{ github.run_id }}.txt` with content "Blocked domains test for Claude run ${{ github.run_id }}"

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- ✅ or ❌ for each test result
- List which domains were blocked successfully
- List which domains were allowed successfully
- Overall status: PASS or FAIL

If all tests pass (GitHub allowed, NPM blocked), add the label `smoke-blocked-domains-claude` to the pull request.
