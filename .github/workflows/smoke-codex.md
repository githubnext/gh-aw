---
description: Smoke test workflow that validates Codex engine functionality by reviewing recent PRs twice daily
on: 
  schedule: every 12h
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "hooray"
  status-comment: true
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke Codex
engine: codex
strict: false
imports:
  - shared/gh.md
  - shared/reporting.md
network:
  allowed:
    - defaults
    - github
    - playwright
tools:
  cache-memory: true
  github:
  playwright:
  edit:
  bash:
    - "*"
  serena:
    languages:
      go: {}
  web-fetch:
  qmd:
    checkouts:
      - name: docs
        paths:
          - docs/src/**/*.md
          - docs/src/**/*.mdx
        context: "gh-aw project documentation"
    searches:
      - name: issues
        type: issues
        max: 500
        github-token: ${{ secrets.GITHUB_TOKEN }}
runtimes:
  go:
    version: "1.25"
safe-outputs:
    allowed-domains: [default-safe-outputs]
    add-comment:
      hide-older-comments: true
      max: 2
    create-issue:
      expires: 2h
      close-older-issues: true
      labels: [automation, testing]
    add-labels:
      allowed: [smoke-codex]
    remove-labels:
      allowed: [smoke]
    unassign-from-user:
      allowed: [githubactionagent]
      max: 1
    hide-comment:
    messages:
      footer: "> 🔮 *The oracle has spoken through [{workflow_name}]({run_url})*{history_link}"
      run-started: "🔮 The ancient spirits stir... [{workflow_name}]({run_url}) awakens to divine this {event_type}..."
      run-success: "✨ The prophecy is fulfilled... [{workflow_name}]({run_url}) has completed its mystical journey. The stars align. 🌟"
      run-failure: "🌑 The shadows whisper... [{workflow_name}]({run_url}) {status}. The oracle requires further meditation..."
    actions:
      add-smoked-label:
        uses: actions-ecosystem/action-add-labels@v1.1.3
        description: Add the 'smoked' label to the current pull request
        env:
          GITHUB_TOKEN: ${{ github.token }}
timeout-minutes: 15
checkout:
  - fetch-depth: 2
    current: true
---

# Smoke Test: Codex Engine Validation

**CRITICAL EFFICIENCY REQUIREMENTS:**
- Keep ALL outputs extremely short and concise. Use single-line responses.
- NO verbose explanations or unnecessary context.
- Minimize file reading - only read what is absolutely necessary for the task.
- Use targeted, specific queries - avoid broad searches or large data retrievals.

## Test Requirements

1. **GitHub MCP Testing**: Use GitHub MCP tools to fetch details of exactly 2 merged pull requests from ${{ github.repository }} (title and number only, no descriptions)
2. **Serena MCP Testing**: 
   - Use the Serena MCP server tool `activate_project` to initialize the workspace at `${{ github.workspace }}` and verify it succeeds (do NOT use bash to run go commands)
   - After initialization, use the `find_symbol` tool to search for symbols and verify that at least 3 symbols are found in the results
3. **Playwright Testing**: Use the playwright tools to navigate to https://github.com and verify the page title contains "GitHub" (do NOT try to install playwright - use the provided MCP tools)
4. **Web Fetch Testing**: Use the web-fetch MCP tool to fetch https://github.com and verify the response contains "GitHub" (do NOT use bash or playwright for this test - use the web-fetch MCP tool directly)
5. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-codex-${{ github.run_id }}.txt` with content "Smoke test passed for Codex at $(date)" (create the directory if it doesn't exist)
6. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
7. **Build gh-aw**: Run `GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod make build` to verify the agent can successfully build the gh-aw project (both caches must be set to /tmp because the default cache locations are not writable). If the command fails, mark this test as ❌ and report the failure.

## Output

**Check `${{ github.event_name }}` to determine the correct output action:**

- **If `${{ github.event_name }}` is `pull_request`**: Use the `add_comment` safe-output tool to add a **very brief** comment (max 5-10 lines) to the triggering pull request, specifying `item_number: ${{ github.event.pull_request.number }}` (use this exact number — do NOT search GitHub for a PR):
  - PR titles only (no descriptions)
  - ✅ or ❌ for each test result
  - Overall status: PASS or FAIL

- **If `${{ github.event_name }}` is `schedule` or `workflow_dispatch`**: Call the `noop` safe-output tool with a brief summary of the test results (there is no pull request to comment on for scheduled or manually triggered runs).

If all tests pass and this workflow was triggered by a `pull_request` event:
- Use the `add_labels` safe-output tool to add the label `smoke-codex` to the pull request (use `item_number: ${{ github.event.pull_request.number }}`)
- Use the `remove_labels` safe-output tool to remove the label `smoke` from the pull request (use `item_number: ${{ github.event.pull_request.number }}`)
- Use the `unassign_from_user` safe-output tool to unassign the user `githubactionagent` from the pull request (this is a fictitious user used for testing; use `item_number: ${{ github.event.pull_request.number }}`)
- Use the `add_smoked_label` safe-output action tool to add the label `smoked` to the pull request (call it with `{"labels": "smoked", "number": "${{ github.event.pull_request.number }}"}`)

**Important**: You **MUST** always call exactly one safe-output tool. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
