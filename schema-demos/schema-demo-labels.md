---
description: Demonstrates the `labels` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
labels:
  - automation
  - demo
timeout-minutes: 5
---

# Schema Demo: `labels`

This workflow was auto-generated to demonstrate usage of the `labels` field in the
gh-aw frontmatter schema. It exists solely to achieve 100% schema feature coverage.

## What `labels` Does

Categorizes and organizes workflows with labels that can be used for filtering.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `labels` -- no action needed."}}
```
