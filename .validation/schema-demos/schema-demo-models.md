---
description: Demonstrates the `models` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
models:
  demo:
    - gpt-5-codex
timeout-minutes: 5
---

# Schema Demo: `models`

This workflow was auto-generated to demonstrate usage of the `models` field in the
gh-aw frontmatter schema. It exists solely to achieve 100% schema feature coverage.

## What `models` Does

Defines named model aliases with ordered fallback lists.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `models` -- no action needed."}}
```
