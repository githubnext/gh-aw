---
description: Demonstrates the `max-tool-denials` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine:
  id: copilot
  copilot-sdk: true
max-tool-denials: 5
timeout-minutes: 5
---

# Schema Demo: `max-tool-denials`

This workflow was auto-generated to demonstrate usage of the `max-tool-denials` field in the
gh-aw frontmatter schema. It exists solely to achieve 100% schema feature coverage.

## What `max-tool-denials` Does

Copilot SDK safeguard threshold for repeated tool denials before stopping inference.

## Task

Call `noop` -- this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `max-tool-denials` -- no action needed."}}
```
