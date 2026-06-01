---
description: Demonstrates the `resources` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
resources:
  - ../.github/workflows/ci.yml
timeout-minutes: 5
---

# Schema Demo: `resources`

This workflow was auto-generated to demonstrate usage of the `resources` field in
the gh-aw frontmatter schema. It exists solely to achieve 100% schema feature
coverage.

## What `resources` Does

Optional list of additional workflow or action files fetched alongside this workflow.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `resources` -- no action needed."}}
```
