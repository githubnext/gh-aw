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
engine:
  id: copilot
  model: gpt-5
network:
  firewall:
    log-level: debug  # Enable debug-level firewall logs
tools:
  edit:
  bash:
    - "*"
  github:
    toolsets: 
      - issues
safe-outputs:
    add-comment:
    create-issue:
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

0. **List of tools**: List all tools available to this workflow.
1. **GitHub MCP Testing**: List the last 2 issues in ${{ github.repository }}. Make sure to only ask for 2.
2. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-copilot-${{ github.run_id }}.txt` with content "Smoke test passed for Copilot at $(date)" (create the directory if it doesn't exist)
3. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- Issues titles only (no descriptions)
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL

If all tests pass, add the label `smoke-copilot` to the pull request.