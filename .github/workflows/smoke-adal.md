---
private: true
emoji: "🩷"
description: Smoke test workflow that validates AdaL engine functionality
on:
  slash_command:
    name: smoke-adal
    strategy: centralized
    events: [issues, issue_comment, pull_request, pull_request_comment]
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["water"]
  reaction: "rocket"
  status-comment: true
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke AdaL
model: adal/gpt-5.6-terra
engine:
  id: adal
strict: true
imports:
  - shared/adal.md
  - shared/smoke-test-brevity.md
  - shared/reporting.md
network:
  allowed: []
tools:
  edit:
  bash:
    - "*"
safe-outputs:
  allowed-domains: [default-safe-outputs]
  add-comment:
    hide-older-comments: true
    max: 2
  create-issue:
    expires: 2h
    close-older-issues: true
    close-older-key: "smoke-adal"
    labels: [automation, testing]
  add-labels:
    allowed: [smoke-adal]
  messages:
    footer: "> 🩷 *[{workflow_name}]({run_url}) — Powered by AdaL*{ai_credits_suffix}{history_link}"
    run-started: "🩷 AdaL initializing... [{workflow_name}]({run_url}) begins on this {event_type}..."
    run-success: "🩷 [{workflow_name}]({run_url}) AdaL delivered."
    run-failure: "[{workflow_name}]({run_url}) {status}. AdaL encountered an unexpected error."
timeout-minutes: 15
features:
  gh-aw-detection: false
sandbox:
  agent:
    id: awf
---

# Smoke Test: AdaL Engine Validation

AdaL has no documented headless MCP configuration, so this smoke test exercises
prompt delivery, model selection, shell commands, and file editing. Safe outputs
are emitted by appending JSONL entries to the file referenced by
`$GH_AW_SAFE_OUTPUTS`.

## Test Requirements

1. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-adal-${{ github.run_id }}.txt` with content "Smoke test passed for AdaL" (create the directory if it doesn't exist)
2. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
3. **Repository Access Testing**: Run `git log --oneline -1` in the repository checkout and confirm a commit is reported

## Output

**ALWAYS create an issue** by appending a single line of JSON to the file at
`$GH_AW_SAFE_OUTPUTS`, of the form
`{"type":"create_issue","title":"...","body":"..."}`:
- Title: "Smoke Test: AdaL - ${{ github.run_id }}"
- Body should include:
  - Test results (PASS or FAIL for each test)
  - Overall status: PASS or FAIL
  - Run URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}

**Only if this workflow was triggered by a pull_request event**: append a
`{"type":"add_comment","body":"..."}` line to `$GH_AW_SAFE_OUTPUTS` with a
**very brief** comment (max 5-10 lines) containing:
- PASS or FAIL for each test result
- Overall status: PASS or FAIL

If all tests pass and this workflow was triggered by a pull_request event, also
append `{"type":"add_labels","labels":["smoke-adal"]}` to
`$GH_AW_SAFE_OUTPUTS`.
