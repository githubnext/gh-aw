---
description: Demonstrates the `secrets` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
secrets:
  API_TOKEN: ${{ secrets.API_TOKEN }}
timeout-minutes: 5
---

# Schema Demo: `secrets`

This workflow was auto-generated to demonstrate usage of the `secrets` field in the
gh-aw frontmatter schema. It exists solely to achieve 100% schema feature coverage.

## What `secrets` Does

Secret values passed to workflow execution.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `secrets` -- no action needed."}}
```
