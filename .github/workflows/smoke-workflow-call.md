---
name: Smoke Workflow Call
description: Reusable workflow to validate checkout from fork works correctly in workflow_call context
on:
  workflow_call:
  workflow_dispatch:
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
    - "git status"
    - "git log *"
    - "git branch *"
    - "git remote *"
    - "echo *"
safe-outputs:
  add-comment:
    hide-older-comments: true
    max: 1
  messages:
    append-only-comments: true
    footer: "> 🔁 *workflow_call smoke test by [{workflow_name}]({run_url})*"
    run-started: "🔁 [{workflow_name}]({run_url}) is validating workflow_call checkout..."
    run-success: "✅ [{workflow_name}]({run_url}) successfully validated workflow_call checkout."
    run-failure: "❌ [{workflow_name}]({run_url}) failed to validate workflow_call checkout. Check the logs."
timeout-minutes: 10
---

# Smoke Test: Workflow Call Checkout Validation

This workflow is designed to be called via `workflow_call` from another workflow (e.g., `smoke-trigger`).
It validates that the PR branch checkout works correctly when invoked in a `workflow_call` context.

## Test Requirements

1. **Git Status**: Run `git status` to verify the workspace is properly initialized.
2. **Branch Check**: Run `git branch --show-current` to confirm which branch is checked out.
3. **Remote Check**: Run `git remote -v` to confirm the remote configuration.
4. **Log Check**: Run `git log --oneline -3` to verify the commit history is available.

## Output

Add a comment summarizing the checkout validation results:
- Current branch name
- Whether the workspace is clean or has changes
- Whether the checkout succeeded (based on git commands working without errors)
- Overall status: ✅ PASS or ❌ FAIL

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
