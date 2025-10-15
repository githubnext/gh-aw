---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
engine: claude
imports:
  - shared/trigger-workflow.md
safe-outputs:
  trigger-workflow:
    allowed:
      - "ci.yml"
      - "release.yml"
---

# Workflow Trigger Demo

When a new issue is created, analyze it and trigger the appropriate workflow.

If the issue mentions "ci" or "test", trigger the ci.yml workflow.
If the issue mentions "release" or "deploy", trigger the release.yml workflow with a test payload.

Use the trigger_workflow safe output to trigger the appropriate workflow.
