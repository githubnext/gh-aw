---
description: Demonstrates the `run-name` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
run-name: Schema Demo Run
timeout-minutes: 5
---

# Schema Demo: `run-name`

This workflow was auto-generated to demonstrate usage of the `run-name` field in
the gh-aw frontmatter schema. It exists solely to achieve 100% schema feature
coverage.

## What `run-name` Does

Custom name for workflow runs that appears in the GitHub Actions interface.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `run-name` -- no action needed."}}
```
