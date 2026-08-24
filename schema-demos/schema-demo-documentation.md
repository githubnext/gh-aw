---
description: Demonstrates the `documentation` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
documentation: https://docs.example.com/automation/repository-health
timeout-minutes: 5
---

# Schema Demo: `documentation`

This workflow was auto-generated to demonstrate usage of the `documentation` field in
the gh-aw frontmatter schema. It exists solely to achieve 100% schema feature
coverage.

## What `documentation` Does

Optional absolute HTTPS URL for human-facing workflow documentation.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `documentation` -- no action needed."}}
```
