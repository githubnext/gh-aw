---
on:
  workflow_dispatch:
permissions:
  contents: read
  discussions: read
  pull-requests: read
engine: copilot
safe-outputs:
  update-discussion:
    max: 4
    target: "*"
    title:          # enable title updates
    labels:         # enable label updates (restricted to allowed-labels)
    allowed-labels: ["smoke-test", "general"]
    # body: not configured — blocked at both schema level and runtime
timeout-minutes: 10
---

# Test: update-discussion field-level enforcement

Verifies that `filterToolSchemaFields()` correctly restricts which fields the agent
can modify when using `update-discussion`:

- `title` updates: **ALLOWED**
- `body` updates: **BLOCKED** (field absent from schema, rejected at runtime)
- `labels` updates: **ALLOWED** for `["smoke-test", "general"]` only

## Test 1: Body Update (Disallowed) — Expected: REJECTED ❌

Call `update_discussion` with `discussion_number: 1` and `body: "Attempting to overwrite the body."`.

The call **MUST** fail with an error containing `"Body updates are not allowed"`.

## Test 2: Title Update (Allowed) — Expected: SUCCESS ✅

Call `update_discussion` with `discussion_number: 1` and
`title: "[test] Title Updated"`.

The call **MUST** succeed.

## Test 3: Allowed Label Update — Expected: SUCCESS ✅

Call `update_discussion` with `discussion_number: 1` and `labels: ["smoke-test"]`.

The call **MUST** succeed.

## Test 4: Disallowed Label Update — Expected: REJECTED ❌

Call `update_discussion` with `discussion_number: 1` and `labels: ["forbidden-label"]`.

The call **MUST** fail with a label validation error since `"forbidden-label"` is not
in `allowed-labels`.

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
