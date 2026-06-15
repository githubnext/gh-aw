---
description: Demonstrates the `post-steps` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
post-steps:
  - name: Post-step
    run: echo done
timeout-minutes: 5
---

# Schema Demo: `post-steps`

This workflow was auto-generated to demonstrate usage of the `post-steps` field in the
gh-aw frontmatter schema. It exists solely to achieve 100% schema feature coverage.

## What `post-steps` Does

Custom workflow steps to run after AI execution.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `post-steps` -- no action needed."}}
```
