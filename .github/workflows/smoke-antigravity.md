---
private: true
emoji: "🧪"
description: Smoke test workflow that validates Antigravity engine functionality twice daily
on:
  slash_command:
    name: smoke-antigravity
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
name: Smoke Antigravity
engine:
  id: antigravity
strict: true
imports:
  - shared/gh.md
  - shared/reporting-otlp.md
  - shared/otlp.md
network:
  allowed:
    - defaults
    - github
tools:
  cache-memory: true
  github:
    toolsets: [repos, pull_requests]
  edit:
  bash:
    - "*"
  web-fetch:
safe-outputs:
    allowed-domains: [default-safe-outputs]
    add-comment:
      hide-older-comments: true
      max: 2
    create-issue:
      expires: 2h
      close-older-issues: true
      close-older-key: "smoke-antigravity"
      labels: [automation, testing]
    add-labels:
      allowed: [smoke-antigravity]
    messages:
      footer: "> ✨ *[{workflow_name}]({run_url}) — Powered by Antigravity*{ai_credits_suffix}{history_link}"
      run-started: "✨ Antigravity awakens... [{workflow_name}]({run_url}) begins its journey on this {event_type}..."
      run-success: "🚀 [{workflow_name}]({run_url}) **MISSION COMPLETE!** Antigravity has spoken. ✨"
      run-failure: "⚠️ [{workflow_name}]({run_url}) {status}. Antigravity encountered unexpected challenges..."
timeout-minutes: 10
features:
  gh-aw-detection: false
---

# Smoke Test: Antigravity Engine Validation

**CRITICAL EFFICIENCY REQUIREMENTS:**
- Keep ALL outputs extremely short and concise. Use single-line responses.
- NO verbose explanations or unnecessary context.
- Minimize file reading - only read what is absolutely necessary for the task.

## Test Requirements

Execute all 5 tests sequentially in this agent:

1. **GitHub MCP Testing**: Use GitHub MCP tools to fetch details of exactly 2 merged pull requests from ${{ github.repository }} (title and number only)
2. **Web Fetch Testing**: Use the web-fetch MCP tool to fetch https://github.com and verify the response contains "GitHub" (do NOT use bash or playwright)
3. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-antigravity-${{ github.run_id }}.txt` with content "Smoke test passed for Antigravity at $(date)"
4. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
5. **Build gh-aw**: Run `GOCACHE=/tmp/gh-aw/agent/go-cache GOMODCACHE=/tmp/gh-aw/agent/go-mod make build` to verify the agent can successfully build the gh-aw project. If the command fails, mark this test as ❌ and report the failure.

After completing all tests, proceed to the Output section below.

## Output

**ALWAYS create an issue** with a summary of the smoke test run:
- Title: "Smoke Test: Antigravity - ${{ github.run_id }}"
- Body should include:
  - Test results (✅ or ❌ for each test)
  - Overall status: PASS or FAIL
  - Run URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
  - Timestamp

**Only if this workflow was triggered by a pull_request event**: Use the `add_comment` tool to add a **very brief** comment (max 5-10 lines) to the triggering pull request (omit the `item_number` parameter to auto-target the triggering PR) with:
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL

If all tests pass and this workflow was triggered by a pull_request event, use the `add_labels` safe-output tool to add the label `smoke-antigravity` to the pull request (omit the `item_number` parameter to auto-target the triggering PR).

{{#runtime-import shared/noop-reminder.md}}