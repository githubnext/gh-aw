---
description: Demonstrates the `metadata` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
metadata:
  author: schema-coverage
  version: "1.0.0"
timeout-minutes: 5
---

# Schema Demo: `metadata`

This workflow was auto-generated to demonstrate usage of the `metadata` field in the
gh-aw frontmatter schema. It exists solely to achieve 100% schema feature coverage.

## What `metadata` Does

Stores custom key-value pairs compatible with the custom agent spec.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `metadata` -- no action needed."}}
```
