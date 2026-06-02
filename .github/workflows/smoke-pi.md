---
emoji: "🧪"
description: Smoke test workflow that validates Pi engine functionality
on:
  slash_command:
    name: smoke-pi
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
name: Smoke Pi
experiments:
  sub_agent_decomposition:
    variants: [single_agent, parallel_sub_agents]
    description: "Test whether decomposing smoke tests into parallel sub-agents reduces token cost"
    hypothesis: "H0: no change in effective token consumption. H1: parallel sub-agents reduce tokens by 15-25% by eliminating unnecessary context sharing"
    metric: effective_token_count
    secondary_metrics: [run_duration_seconds, test_pass_rate, false_failure_rate]
    guardrail_metrics:
      - name: test_completion_rate
        threshold: ">=0.95"
      - name: overall_pass_rate
        threshold: ">=0.80"
    min_samples: 20
    weight: [50, 50]
    start_date: "2026-05-22"
    analysis_type: mann_whitney
    tags: [cost_optimization, smoke_tests, pi_engine]
    # issue: PLACEHOLDER_ISSUE_NUMBER
engine:
  id: pi
  model: copilot/gpt-5.4
strict: true
sandbox:
  agent:
    config:
      filesystem:
        allowWrite:
          - ${{ github.workspace }}
          - /tmp/gh-aw/agent
runtimes:
  node: {}
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
    mode: gh-proxy
  edit:
  bash:
    - "*"
  web-fetch:
  cli-proxy: true
safe-outputs:
    allowed-domains: [default-safe-outputs]
    add-comment:
      hide-older-comments: true
      max: 2
    create-issue:
      expires: 2h
      close-older-issues: true
      close-older-key: "smoke-pi"
      labels: [automation, testing]
    add-labels:
      allowed: [smoke-pi]
    messages:
      footer: "> 🥧 *[{workflow_name}]({run_url}) — Powered by Pi*{effective_tokens_suffix}{history_link}"
      run-started: "🥧 Pi initializing... [{workflow_name}]({run_url}) begins on this {event_type}..."
      run-success: "🚀 [{workflow_name}]({run_url}) **MISSION COMPLETE!** Pi delivered. 🥧"
      run-failure: "⚠️ [{workflow_name}]({run_url}) {status}. Pi encountered unexpected challenges..."
timeout-minutes: 10

---

# Smoke Test: Pi Engine Validation

**CRITICAL EFFICIENCY REQUIREMENTS:**
- Keep ALL outputs extremely short and concise. Use single-line responses.
- NO verbose explanations or unnecessary context.
- Minimize file reading - only read what is absolutely necessary for the task.

## Test Requirements

{{#if experiments.sub_agent_decomposition == 'parallel_sub_agents'}}
Launch five parallel `task` agents using mode: "background" to execute each smoke test independently. Use the `task` agent type with `description` field for each:

1. **GitHub MCP Test Agent**: Fetch 2 merged PR titles from ${{ github.repository }}
2. **Web Fetch Test Agent**: Fetch https://github.com and verify "GitHub" in response using web-fetch MCP
3. **File I/O Test Agent**: Create `/tmp/gh-aw/agent/smoke-test-pi-${{ github.run_id }}.txt` with timestamp
4. **Bash Test Agent**: Verify file creation with `cat` command
5. **Build Test Agent**: Run `GOCACHE=/tmp/gh-aw/agent/go-cache GOMODCACHE=/tmp/gh-aw/agent/go-mod make build`

Wait for all five agents to complete (you'll receive notifications). Read each agent's result using `read_agent`. Aggregate the results into a unified report with ✅/❌ status for each test.

{{else}}
Execute the following tests sequentially in a single turn:

1. **GitHub MCP Testing**: Use GitHub MCP tools to fetch details of exactly 2 merged pull requests from ${{ github.repository }} (title and number only)
2. **Web Fetch Testing**: Use the web-fetch MCP tool to fetch https://github.com and verify the response contains "GitHub" (do NOT use bash or playwright for this test - use the web-fetch MCP tool directly)
3. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-pi-${{ github.run_id }}.txt` with content "Smoke test passed for Pi at $(date)" (create the directory if it doesn't exist)
4. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
5. **Build gh-aw**: Run `GOCACHE=/tmp/gh-aw/agent/go-cache GOMODCACHE=/tmp/gh-aw/agent/go-mod make build` to verify the agent can successfully build the gh-aw project. If the command fails, mark this test as ❌ and report the failure.
{{/if}}

## Output

**ALWAYS create an issue** with a summary of the smoke test run:
- Title: "Smoke Test: Pi - ${{ github.run_id }}"
- Body should include:
  - Test results (✅ or ❌ for each test)
  - Overall status: PASS or FAIL
  - Run URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
  - Timestamp

**Only if this workflow was triggered by a pull_request event**: Use the `add_comment` tool to add a **very brief** comment (max 5-10 lines) to the triggering pull request (omit the `item_number` parameter to auto-target the triggering PR) with:
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL

If all tests pass and this workflow was triggered by a pull_request event, use the `add_labels` safe-output tool to add the label `smoke-pi` to the pull request (omit the `item_number` parameter to auto-target the triggering PR).

{{#runtime-import shared/noop-reminder.md}}
