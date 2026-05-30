---
description: Demonstrates the `check-for-updates` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
check-for-updates: false
timeout-minutes: 5
---

# Schema Demo: `check-for-updates`

This workflow was auto-generated to demonstrate usage of the `check-for-updates` field in the
gh-aw frontmatter schema. It exists solely to achieve 100% schema feature coverage.

## What `check-for-updates` Does

Controls whether the compile-agentic version update check runs in the activation job.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `check-for-updates` -- no action needed."}}
```
