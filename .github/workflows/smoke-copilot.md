---
description: Smoke Copilot
on: 
  schedule:
    - cron: "0 0,7,13,19 * * *"  # Every 6 hours
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "eyes"
permissions:
  contents: read
  pull-requests: read
  issues: read
name: Smoke Copilot
engine: copilot
network:
  allowed:
    - defaults
    - node
    - github
sandbox:
  agent: awf  # Firewall enabled
  mcp:
    port: 8080
tools:
  cache-memory: true
  edit:
  bash:
    - "*"
  github:
safe-outputs:
    add-comment:
      hide-older-comments: true
    create-issue:
      expires: 1d
    add-labels:
      allowed: [smoke-copilot]
    messages:
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
2. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-copilot-${{ github.run_id }}.txt` with content "Smoke test passed for Copilot at $(date)" (create the directory if it doesn't exist)
3. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
4. **GitHub MCP Default Toolset Testing**: Verify that the `get_me` tool is NOT available with default toolsets. Try to use it and confirm it fails with a tool not found error.
5. **Cache Memory Testing**: Write a test file to `/tmp/gh-aw/cache-memory/smoke-test-${{ github.run_id }}.txt` with content "Cache memory test for run ${{ github.run_id }}" and verify it was created successfully
6. **MCP Gateway Testing**: Verify that the MCP gateway is running by checking if the container is active and the health endpoint is accessible
7. **Available Tools Display**: List all available tools that you have access to in this workflow execution.

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- PR titles only (no descriptions)
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL
- Mention the pull request author and any assignees

If all tests pass, add the label `smoke-copilot` to the pull request.