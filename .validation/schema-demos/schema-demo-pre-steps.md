---
description: Demonstrates the `pre-steps` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
pre-steps:
  - name: Pre-step
    run: echo preparing
timeout-minutes: 5
---

# Schema Demo: `pre-steps`

This workflow was auto-generated to demonstrate usage of the `pre-steps` field in the
gh-aw frontmatter schema. It exists solely to achieve 100% schema feature coverage.

## What `pre-steps` Does

Runs custom workflow steps at the very beginning of the agent job.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `pre-steps` -- no action needed."}}
```
