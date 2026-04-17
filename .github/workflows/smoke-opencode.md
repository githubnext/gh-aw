---
description: Smoke test workflow that validates OpenCode engine functionality
on:
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "eyes"
  status-comment: true
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke OpenCode
engine:
  id: opencode
  model: anthropic/claude-sonnet-4-20250514
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
  web-fetch:
safe-outputs:
    add-comment:
      hide-older-comments: true
      max: 2
    create-issue:
      expires: 2h
      close-older-issues: true
      labels: [automation, testing]
    add-labels:
      allowed: [smoke-opencode]
    messages:
      footer: "> ⚡ *[{workflow_name}]({run_url}) — Powered by OpenCode*"
      run-started: "⚡ OpenCode initializing... [{workflow_name}]({run_url}) begins on this {event_type}..."
      run-success: "🎯 [{workflow_name}]({run_url}) **MISSION COMPLETE!** OpenCode has delivered. ⚡"
      run-failure: "⚠️ [{workflow_name}]({run_url}) {status}. OpenCode encountered unexpected challenges..."
timeout-minutes: 15
---

# Smoke Test: OpenCode Engine Validation

**CRITICAL EFFICIENCY REQUIREMENTS:**
- Keep ALL outputs extremely short and concise. Use single-line responses.
- NO verbose explanations or unnecessary context.
- Minimize file reading - only read what is absolutely necessary for the task.

## Test Requirements

1. **GitHub MCP Testing**: Use GitHub MCP tools to fetch details of exactly 2 merged pull requests from ${{ github.repository }} (title and number only)
2. **Web Fetch Testing**: Use the web-fetch MCP tool to fetch https://github.com and verify the response contains "GitHub" (do NOT use bash or playwright for this test - use the web-fetch MCP tool directly)
3. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-opencode-${{ github.run_id }}.txt` with content "Smoke test passed for OpenCode at $(date)" (create the directory if it doesn't exist)
4. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
5. **Build gh-aw**: Run `GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod make build` to verify the agent can successfully build the gh-aw project. If the command fails, mark this test as ❌ and report the failure.

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL

If all tests pass, use the `add_labels` safe-output tool to add the label `smoke-opencode` to the pull request.

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
