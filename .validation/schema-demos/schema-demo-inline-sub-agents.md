---
description: Demonstrates the `inline-sub-agents` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
inline-sub-agents: true
timeout-minutes: 5
---

# Schema Demo: `inline-sub-agents`

This workflow was auto-generated to demonstrate usage of the `inline-sub-agents` field in the
gh-aw frontmatter schema. It exists solely to achieve 100% schema feature coverage.

## What `inline-sub-agents` Does

Retains the deprecated switch for inline sub-agent support, which is enabled by default.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `inline-sub-agents` -- no action needed."}}
```
