---
description: Demonstrates the `github-app` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
github-app:
  client-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
timeout-minutes: 5
---

# Schema Demo: `github-app`

This workflow was auto-generated to demonstrate usage of the `github-app` field in the
gh-aw frontmatter schema. It exists solely to achieve 100% schema feature coverage.

## What `github-app` Does

Provides top-level GitHub App credentials used as a fallback for nested token minting operations.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `github-app` -- no action needed."}}
```
