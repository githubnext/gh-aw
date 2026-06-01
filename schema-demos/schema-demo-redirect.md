---
description: Demonstrates the `redirect` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
redirect: https://github.com/github/gh-aw/blob/main/.github/workflows/ci.yml
timeout-minutes: 5
---

# Schema Demo: `redirect`

This workflow was auto-generated to demonstrate usage of the `redirect` field in
the gh-aw frontmatter schema. It exists solely to achieve 100% schema feature
coverage.

## What `redirect` Does

Optional workflow location redirect for updates.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `redirect` -- no action needed."}}
```
