---
private: true
emoji: "🐍"
description: Smoke test workflow that validates Pydantic AI engine functionality
on:
  slash_command:
    name: smoke-pydantic
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
name: Smoke Pydantic AI
model: copilot/claude-sonnet-4-5
engine:
  id: pydantic-ai
strict: true
imports:
  - shared/pydantic.md
  - shared/smoke-test-brevity.md
  - shared/reporting.md
network:
  allowed: []
safe-outputs:
  allowed-domains: [default-safe-outputs]
  add-comment:
    hide-older-comments: true
    max: 2
  create-issue:
    expires: 2h
    close-older-issues: true
    close-older-key: "smoke-pydantic"
    labels: [automation, testing]
  add-labels:
    allowed: [smoke-pydantic]
  messages:
    footer: "> 🐍 *[{workflow_name}]({run_url}) — Powered by Pydantic AI*{ai_credits_suffix}{history_link}"
    run-started: "🐍 Pydantic AI initializing... [{workflow_name}]({run_url}) begins on this {event_type}..."
    run-success: "🐍 [{workflow_name}]({run_url}) Pydantic AI delivered."
    run-failure: "⚠️ [{workflow_name}]({run_url}) {status}. Pydantic AI encountered unexpected challenges..."
timeout-minutes: 10
features:
  gh-aw-detection: false
sandbox:
  agent:
    id: awf
    sudo: false
---

# Smoke Test: Pydantic AI Engine Validation

## Test Requirements

1. **Model Connectivity Testing**: Answer the question "What is 2 + 2?" in a single short line.
2. **MCP Tool Testing**: Confirm that the `safeoutputs` MCP tools are available to you.

## Output

**ALWAYS create an issue** by appending a single line of JSON to the file at `$GH_AW_SAFE_OUTPUTS`, of the form
`{"type":"create_issue","title":"...","body":"..."}`:
- Title: "Smoke Test: Pydantic AI - ${{ github.run_id }}"
- Body should include:
  - Test results (✅ or ❌ for each test)
  - Overall status: PASS or FAIL
  - Run URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}

**Only if this workflow was triggered by a pull_request event**: append a
`{"type":"add_comment","body":"..."}` line to `$GH_AW_SAFE_OUTPUTS` with a **very brief**
comment (max 5-10 lines) containing:
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL

If all tests pass and this workflow was triggered by a pull_request event, also append
`{"type":"add_labels","labels":["smoke-pydantic"]}` to `$GH_AW_SAFE_OUTPUTS`.
