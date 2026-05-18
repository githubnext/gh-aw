---
description: Demonstrates the `dependencies` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
dependencies:
  - microsoft/apm-sample-package
timeout-minutes: 5
---

# Schema Demo: `dependencies`

This workflow was auto-generated to demonstrate usage of the `dependencies` field in the
gh-aw frontmatter schema. It exists solely to achieve 100% schema feature coverage.

## What `dependencies` Does

Defines APM package references to install for the workflow.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `dependencies` -- no action needed."}}
```
