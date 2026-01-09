---
description: Smoke test workflow that validates Codex engine functionality with AWF firewall enabled
on: 
  schedule: every 12h
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "rocket"
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke Codex Firewall
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
    - "https://api.github.com"  # Test HTTPS-only protocol filtering
safe-outputs:
    add-comment:
      hide-older-comments: true
    create-issue:
      expires: 2h
    add-labels:
      allowed: [smoke-codex-firewall]
    hide-comment:
    messages:
      footer: "> 🔥 *Firewall tested by [{workflow_name}]({run_url})*"
      run-started: "🔒 Initiating firewall smoke test... [{workflow_name}]({run_url}) is validating network sandboxing for {event_type}..."
      run-success: "✅ Firewall validation complete... [{workflow_name}]({run_url}) confirmed network sandboxing is operational. 🛡️"
      run-failure: "❌ Firewall validation failed... [{workflow_name}]({run_url}) {status}. Network sandboxing may not be working correctly."
timeout-minutes: 10
tools:
  github:
  bash:
    - "*"
---

# Smoke Test: Codex Engine with AWF Firewall

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

## Test Requirements

This workflow validates that the Codex engine works correctly with AWF (Application-level Firewall) network sandboxing enabled.

1. **OpenAI Domain Access**: Test that direct curl access to OpenAI APIs (api.openai.com, openai.com) is BLOCKED by the firewall - the Codex CLI itself can access OpenAI (it adds these domains automatically), but raw curl commands should fail since OpenAI is not in the `defaults` or `github` network ecosystems
2. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }} to verify GitHub MCP server works through the firewall
3. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-codex-firewall-${{ github.run_id }}.txt` with content "Firewall smoke test passed for Codex at $(date)"
4. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
5. **Blocked Domain Testing**: Attempt to access a domain NOT in the allowed list (e.g., example.com) using curl - this should fail or be blocked
6. **Protocol Filtering Testing**: Verify that the AWF command includes the protocol-specific domain `https://api.github.com` in the --allow-domains flag. Check logs to confirm HTTPS prefix is preserved
7. **MCP Gateway Testing**: Check that the MCP gateway started successfully by verifying `/tmp/gh-aw/mcp-logs/gateway/` contains startup logs and `/tmp/gh-aw/mcp-config/gateway-input.json` exists

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- PR titles only (no descriptions)
- ✅ or ❌ for each test result
- Network status: SANDBOXED or NOT SANDBOXED
- Overall status: PASS or FAIL

If all tests pass, add the label `smoke-codex-firewall` to the pull request.
