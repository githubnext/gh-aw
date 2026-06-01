---
description: Demonstrates the `runs-on-slim` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
runs-on-slim: ubuntu-slim
timeout-minutes: 5
---

# Schema Demo: `runs-on-slim`

This workflow was auto-generated to demonstrate usage of the `runs-on-slim` field in
the gh-aw frontmatter schema. It exists solely to achieve 100% schema feature
coverage.

## What `runs-on-slim` Does

Runner for all framework-generated jobs when a slim runner override is needed.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `runs-on-slim` -- no action needed."}}
```
