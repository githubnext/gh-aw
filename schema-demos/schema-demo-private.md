---
description: Demonstrates the `private` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
private: true
timeout-minutes: 5
---

# Schema Demo: `private`

This workflow was auto-generated to demonstrate usage of the `private` field in
the gh-aw frontmatter schema. It exists solely to achieve 100% schema feature
coverage.

## What `private` Does

Marks the workflow as private so it is not meant to be shared outside its repository.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `private` -- no action needed."}}
```
