---
description: Smoke test workflow that validates Copilot engine functionality without firewall by reviewing recent PRs every 6 hours
on: 
  schedule:
    - cron: "0 0,6,12,18 * * *"  # Every 6 hours
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "+1"
permissions:
  contents: read
  pull-requests: read
  issues: read
name: Smoke Copilot No Firewall
engine: copilot
network:
  allowed:
    - defaults
    - node
    - github
    - playwright
  firewall: false
tools:
  edit:
  bash:
    - "*"
  github:
  playwright:
    allowed_domains:
      - github.com
  serena: ["go"]
safe-outputs:
    add-comment:
    create-issue:
    add-labels:
      allowed: [smoke-copilot-no-firewall]
timeout-minutes: 10
strict: false
---

# Smoke Test: Copilot Engine Validation (No Firewall)

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

## Test Requirements

1. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}
2. **File Writing Testing**: Create a test file `/tmp/smoke-test-copilot-${{ github.run_id }}.txt` with content "Smoke test passed for Copilot at $(date)"
3. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
4. **Playwright MCP Testing**: Use playwright to navigate to https://github.com and verify the page title contains "GitHub"

## Output

**ROBOTIC DESCRIPTIVE STYLE**: Write your output like a systematic robot giving a technical status report. Use precise, mechanical language.

Add a **very brief** comment (max 5-10 lines) to the current pull request in robotic style:
- Use systematic prefixes: "UNIT_001:", "SUBSYSTEM:", "MODULE:"
- Report in precise technical format: "STATUS: OPERATIONAL", "RESULT: NOMINAL"
- Use ✅ or ❌ as "diagnostic indicators"
- End with a system summary: "DIAGNOSTIC COMPLETE. ALL UNITS OPERATIONAL." or "ALERT: ANOMALIES DETECTED. REPAIR REQUIRED."

Example tone: "UNIT_001 [GITHUB_MCP]: QUERY EXECUTED. 2 PULL_REQUESTS RETRIEVED. STATUS: ✅ NOMINAL."

If all tests pass, add the label `smoke-copilot-no-firewall` to the pull request.
