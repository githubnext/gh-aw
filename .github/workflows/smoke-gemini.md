---
description: Smoke test workflow that validates Gemini engine functionality twice daily
on:
  schedule: every 12h
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["water"]
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke Gemini
engine:
  id: gemini
  model: gemini-2.0-flash-lite
strict: true
imports:
  - shared/gh.md
  - shared/reporting.md
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
safe-outputs:
    add-comment:
      hide-older-comments: true
      max: 2
    create-issue:
      expires: 2h
      close-older-issues: true
      labels: [automation, testing]
    add-labels:
      allowed: [smoke-gemini]
    messages:
      footer: "> ✨ *[{workflow_name}]({run_url}) — Powered by Gemini*"
      run-started: "✨ Gemini awakens... [{workflow_name}]({run_url}) begins its journey on this {event_type}..."
      run-success: "🚀 [{workflow_name}]({run_url}) **MISSION COMPLETE!** Gemini has spoken. ✨"
      run-failure: "⚠️ [{workflow_name}]({run_url}) {status}. Gemini encountered unexpected challenges..."
timeout-minutes: 10
---

# Smoke Test: Gemini Engine Validation

**CRITICAL EFFICIENCY REQUIREMENTS:**
- Keep ALL outputs extremely short and concise. Use single-line responses.
- NO verbose explanations or unnecessary context.
- Minimize file reading - only read what is absolutely necessary for the task.

## Test Requirements

1. **GitHub MCP Testing**: Use GitHub MCP tools to fetch details of exactly 2 merged pull requests from ${{ github.repository }} (title and number only)
2. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-gemini-${{ github.run_id }}.txt` with content "Smoke test passed for Gemini at $(date)" (create the directory if it doesn't exist)
3. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
4. **Build gh-aw**: Run `GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod make build` to verify the agent can successfully build the gh-aw project. If the command fails, mark this test as ❌ and report the failure.

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL

If all tests pass, use the `add_labels` safe-output tool to add the label `smoke-gemini` to the pull request.
