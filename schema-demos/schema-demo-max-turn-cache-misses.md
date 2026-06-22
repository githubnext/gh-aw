---
description: Demonstrates the `max-turn-cache-misses` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
max-turn-cache-misses: 5
timeout-minutes: 5
---

# Schema Demo: `max-turn-cache-misses`

This workflow was auto-generated to demonstrate usage of the
`max-turn-cache-misses` field in the gh-aw frontmatter schema. It exists solely
to achieve 100% schema feature coverage.

## What `max-turn-cache-misses` Does

Maximum number of consecutive AWF cache misses allowed before the API proxy
blocks further requests.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `max-turn-cache-misses` -- no action needed."}}
```
