---
description: Core smoke test workflow that validates essential functionality across all AI engines (Copilot, Claude, Codex)
on: 
  schedule:
    - cron: "0 0,6,12,18 * * *"  # Every 6 hours
  workflow_dispatch:
    inputs:
      engine:
        description: 'AI engine to test (all, copilot, claude, codex)'
        required: false
        default: 'all'
        type: choice
        options:
          - all
          - copilot
          - claude
          - codex
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "rocket"
permissions:
  contents: read
  pull-requests: read
  issues: read
name: Smoke Core
engine: copilot
network:
  allowed:
    - defaults
    - node
    - github
    - playwright
sandbox:
  agent: awf  # Firewall enabled for security testing
tools:
  cache-memory: true
  edit:
  bash:
    - "*"
  github:
    toolsets: [repos, pull_requests]
  playwright:
    allowed_domains:
      - github.com
safe-outputs:
  add-comment:
    hide-older-comments: true
  create-issue:
    expires: 1d
  add-labels:
    allowed: [smoke-core, smoke-copilot, smoke-claude, smoke-codex]
  messages:
    footer: "> 🎯 *Core smoke test by [{workflow_name}]({run_url})*"
    run-started: "🎯 SMOKE CORE: [{workflow_name}]({run_url}) testing {event_type}. Running comprehensive validation..."
    run-success: "✅ SMOKE CORE: [{workflow_name}]({run_url}) PASSED. All core functionality validated. 🎉"
    run-failure: "❌ SMOKE CORE: [{workflow_name}]({run_url}) {status}. Core functionality issues detected..."
timeout-minutes: 10
strict: true
---

# Smoke Test: Core Engine Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

This workflow validates core functionality across all AI engines. It tests essential features that should work consistently regardless of the engine being used.

## Engine Selection

Current engine: **copilot** (default for smoke-core)

This workflow uses the Copilot engine by default. When triggered via `workflow_dispatch` with the `engine` input parameter, the AI agent will be instructed to simulate testing for the specified engine(s):
- `all` - Report as if tests were run for Copilot, Claude, and Codex (this workflow runs with Copilot engine)
- `copilot` - Report results for Copilot engine
- `claude` - Report results as if tested with Claude engine
- `codex` - Report results as if tested with Codex engine

**Note:** The actual execution uses the Copilot engine. The `engine` input controls the reporting and test scenario coverage.

## Core Test Requirements

### 1. GitHub MCP Testing
Review the last 2 merged pull requests in ${{ github.repository }} using the GitHub MCP server. Verify that the MCP integration is working correctly.

### 2. File Writing Testing
Create a test file `/tmp/gh-aw/agent/smoke-test-core-${{ github.run_id }}.txt` with content "Core smoke test passed at $(date)" (create the directory if it doesn't exist).

### 3. Bash Tool Testing
Execute bash commands to verify file creation was successful (use `cat` to read the file back and confirm the content).

### 4. Cache Memory Testing
Write a test file to `/tmp/gh-aw/cache-memory/smoke-test-${{ github.run_id }}.txt` with content "Cache memory test for run ${{ github.run_id }}" and verify it was created successfully.

### 5. Playwright MCP Testing
Use playwright to navigate to https://github.com and verify the page title contains "GitHub". This validates browser automation works correctly.

### 6. GitHub MCP Toolsets Testing
Verify that the GitHub MCP server is using the correct toolsets (repos, pull_requests) and confirm these tools are available.

## Multi-Engine Testing

If `${{ inputs.engine }}` is set to `all`, perform the above tests for each engine in sequence:

1. **Copilot Engine** - Run all tests with Copilot
2. **Claude Engine** - Run all tests with Claude  
3. **Codex Engine** - Run all tests with Codex

For each engine, report results separately.

## Output

Add a **very brief** comment (max 10-15 lines) to the current pull request with:

### Format:
```
## Smoke Core Test Results - Run ${{ github.run_id }}

**Engine(s) Tested:** [engine list]

| Test | Result |
|------|--------|
| GitHub MCP | ✅/❌ |
| File Writing | ✅/❌ |
| Bash Tools | ✅/❌ |
| Cache Memory | ✅/❌ |
| Playwright | ✅/❌ |
| MCP Toolsets | ✅/❌ |

**Overall Status:** PASS/FAIL

**Latest PRs:** [Brief titles only]
```

If all tests pass, add the label `smoke-core` to the pull request. For specific engines, also add engine-specific labels (`smoke-copilot`, `smoke-claude`, or `smoke-codex`).
