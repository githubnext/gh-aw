---
description: Demonstrates the `environment` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
environment: production
timeout-minutes: 5
---

# Schema Demo: `environment`

This workflow was auto-generated to demonstrate usage of the `environment` field in the
gh-aw frontmatter schema. It exists solely to achieve 100% schema feature coverage.

## What `environment` Does

References a GitHub Actions environment for protected environments and deployments.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `environment` -- no action needed."}}
```
