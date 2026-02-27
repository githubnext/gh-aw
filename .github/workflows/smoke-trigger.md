---
name: Smoke Trigger
description: Triggers smoke-workflow-call via workflow_call to test fork checkout in a workflow_call chain
on:
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke-workflow-call"]
permissions:
  contents: read
  pull-requests: read
engine: copilot
strict: true
network:
  allowed:
    - defaults
tools:
  bash:
    - "echo *"
jobs:
  call-smoke-workflow-call:
    uses: ./.github/workflows/smoke-workflow-call.lock.yml
    permissions:
      contents: read
      pull-requests: write
safe-outputs:
  add-comment:
    hide-older-comments: true
    max: 1
  messages:
    append-only-comments: true
    footer: "> 🚀 *smoke-trigger by [{workflow_name}]({run_url})*"
    run-started: "🚀 [{workflow_name}]({run_url}) is triggering the workflow_call smoke test..."
    run-success: "✅ [{workflow_name}]({run_url}) successfully triggered smoke-workflow-call."
    run-failure: "❌ [{workflow_name}]({run_url}) failed to trigger smoke-workflow-call. Check the logs."
timeout-minutes: 15
---

# Smoke Trigger: workflow_call Chain Test

This workflow tests the `workflow_call` chain by triggering `smoke-workflow-call` as a reusable workflow.
It validates that checkout from a fork PR works correctly when invoked via `workflow_call`.

## Test Requirements

1. **Confirm trigger**: Echo the current event type and PR number (if available) to confirm this workflow was triggered correctly.
2. **Report**: Add a brief comment confirming that the `smoke-workflow-call` workflow was triggered and this workflow completed successfully.

## Output

Add a comment summarizing:
- This workflow was triggered successfully
- The `call-smoke-workflow-call` job was dispatched
- Overall status: ✅ PASS or ❌ FAIL

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
