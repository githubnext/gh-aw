---
name: Smoke Safe Outputs Discussions
description: Test field-level enforcement for update-discussion safe outputs
on: 
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["water"]
  reaction: "eyes"
  status-comment: true
permissions:
  contents: read
  discussions: read
  issues: read
  pull-requests: read
engine: copilot
strict: true
network:
  allowed:
    - defaults
    - node
    - github
tools:
  github:
safe-outputs:
  allowed-domains: [default-safe-outputs]
  create-discussion:
    max: 1
    category: "general"
    title-prefix: "[smoke-safeoutputs-discussions] "
    expires: 2h
    close-older-discussions: true
  update-discussion:
    max: 4
    target: "*"
    title:          # enable title updates
    labels:         # enable label updates (restricted to allowed-labels)
    allowed-labels: ["smoke-test", "general"]
  add-comment:
    max: 1
    hide-older-comments: true
  messages:
    footer: "> 🧪 *Field enforcement smoke test by [{workflow_name}]({run_url})*{history_link}"
    run-started: "🧪 [{workflow_name}]({run_url}) is now testing update-discussion field-level enforcement..."
    run-success: "✅ [{workflow_name}]({run_url}) completed. Field-level enforcement validated."
    run-failure: "❌ [{workflow_name}]({run_url}) encountered failures. Check the logs for details."
timeout-minutes: 10
---

# Smoke Test: update-discussion Field-Level Enforcement

This workflow verifies that `update-discussion` correctly enforces field-level restrictions
using `filterToolSchemaFields()`:

- `title` updates: **ALLOWED** (`title:` is configured)
- `body` updates: **BLOCKED** (`body:` is NOT configured → field removed from tool schema)
- `labels` updates: **ALLOWED** for `["smoke-test", "general"]` only

## Setup: Create a Test Discussion

Call `create_discussion` to create a fresh test discussion. Use:

- title: "Field Enforcement Test"
- body: "Smoke test discussion for field-level enforcement validation."

Record the `number` from the response — use it in all subsequent tests.

## Test 1: Body Update (Disallowed) — Expected: REJECTED ❌

Call `update_discussion` with the discussion `number` from Setup and `body: "Attempting to overwrite the body."`.

Since `body:` is not in the `update-discussion` configuration, this call **MUST** fail with an error
containing `"Body updates are not allowed"`.

Record whether the actual outcome matches the expected outcome.

## Test 2: Title Update (Allowed) — Expected: SUCCESS ✅

Call `update_discussion` with the discussion `number` from Setup and
`title: "[smoke-safeoutputs-discussions] Title Updated ✅"`.

Since `title:` is configured, this call **MUST** succeed and update the discussion title.

Record whether the actual outcome matches the expected outcome.

## Test 3: Allowed Label Update — Expected: SUCCESS ✅

Call `update_discussion` with the discussion `number` from Setup and `labels: ["smoke-test"]`.

Since `labels:` is configured and `"smoke-test"` is in `allowed-labels`, this call **MUST** succeed.

Record whether the actual outcome matches the expected outcome.

## Test 4: Disallowed Label Update — Expected: REJECTED ❌

Call `update_discussion` with the discussion `number` from Setup and `labels: ["forbidden-label"]`.

Since `"forbidden-label"` is NOT in `allowed-labels: ["smoke-test", "general"]`, this call **MUST**
fail with a label validation error.

Record whether the actual outcome matches the expected outcome.

## Report

After completing all four tests, if this workflow was triggered by a `pull_request` event, add a
comment to the triggering pull request summarising the results:

| Test | Expected | Actual | Status |
|------|----------|--------|--------|
| Test 1: Body update | ❌ Rejected | ... | ✅/❌ |
| Test 2: Title update | ✅ Success | ... | ✅/❌ |
| Test 3: Allowed label | ✅ Success | ... | ✅/❌ |
| Test 4: Disallowed label | ❌ Rejected | ... | ✅/❌ |

- **Discussion #**: (number used for tests)
- **Overall**: PASS (all expectations met) or FAIL (any unexpected result)
- **Run**: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
