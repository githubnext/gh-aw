---
private: true
emoji: "🧪"
description: Smoke test workflow that validates Auggie engine functionality
on:
  slash_command:
    name: smoke-auggie
    strategy: centralized
    events: [issues, issue_comment, pull_request, pull_request_comment]
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "rocket"
  status-comment: true
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke Auggie
engine:
  id: auggie
strict: true
imports:
  - shared/gh.md
  - shared/reporting.md
  - shared/otlp.md
network:
  allowed:
    - defaults
    - github
tools:
  cache-memory: true
  github:
    mode: gh-proxy
    toolsets: [repos, pull_requests]
  web-fetch:
  bash:
    - "*"
  edit:
safe-outputs:
    allowed-domains: [default-safe-outputs]
    add-comment:
      hide-older-comments: true
      max: 2
    create-issue:
      expires: 2h
      close-older-issues: true
      close-older-key: "smoke-auggie"
      labels: [automation, testing]
    add-labels:
      allowed: [smoke-auggie]
    messages:
      footer: "> 🧩 *[{workflow_name}]({run_url}) — Powered by Auggie*"
      run-started: "🧩 Auggie initializing... [{workflow_name}]({run_url}) begins on this {event_type}..."
      run-success: "🚀 [{workflow_name}]({run_url}) **MISSION COMPLETE!** Auggie delivered. 🧩"
      run-failure: "⚠️ [{workflow_name}]({run_url}) {status}. Auggie encountered unexpected challenges..."
timeout-minutes: 10
features:
  gh-aw-detection: false
---

# Smoke Test: Auggie Engine Validation

**CRITICAL EFFICIENCY REQUIREMENTS:**
- Keep ALL outputs extremely short and concise. Use single-line responses.
- NO verbose explanations or unnecessary context.
- Minimize file reading - only read what is absolutely necessary for the task.

## Test Requirements

Execute the following tests sequentially in a single turn:

1. **GitHub MCP Testing**: Use GitHub MCP tools to fetch details of exactly 2 merged pull requests from ${{ github.repository }} (title and number only)
2. **Web Fetch Testing**: Use the web-fetch MCP tool to fetch https://github.com and verify the response contains "GitHub" (do NOT use bash or playwright for this test - use the web-fetch MCP tool directly)
3. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-auggie-${{ github.run_id }}.txt` with content "Smoke test passed for Auggie at $(date)" (create the directory if it doesn't exist)
4. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
5. **Build gh-aw**: Run `GOCACHE=/tmp/gh-aw/agent/go-cache GOMODCACHE=/tmp/gh-aw/agent/go-mod make build` to verify the agent can successfully build the gh-aw project. If the command fails, mark this test as ❌ and report the failure.

## Output

**ALWAYS create an issue** with a summary of the smoke test run:
- Title: "Smoke Test: Auggie - ${{ github.run_id }}"
- Body should include:
  - Test results (✅ or ❌ for each test)
  - Overall status: PASS or FAIL
  - Run URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
  - Timestamp

**Only if this workflow was triggered by a pull_request event**: Use the `add_comment` tool to add a **very brief** comment (max 5-10 lines) to the triggering pull request (omit the `item_number` parameter to auto-target the triggering PR) with:
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL

If all tests pass and this workflow was triggered by a pull_request event, use the `add_labels` safe-output tool to add the label `smoke-auggie` to the pull request (omit the `item_number` parameter to auto-target the triggering PR).

{{#runtime-import shared/noop-reminder.md}}
