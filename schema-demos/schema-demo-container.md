---
description: Demonstrates the `container` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
container:
  image: ubuntu:latest
timeout-minutes: 5
---

# Schema Demo: `container`

This workflow was auto-generated to demonstrate usage of the `container` field in the
gh-aw frontmatter schema. It exists solely to achieve 100% schema feature coverage.

## What `container` Does

Configures the container used to run the job steps.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `container` -- no action needed."}}
```
